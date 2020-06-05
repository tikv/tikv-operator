// Copyright 2018 TiKV Project Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package tikvcluster

import (
	"github.com/tikv/tikv-operator/pkg/apis/tikv/v1alpha1"
	"github.com/tikv/tikv-operator/pkg/apis/tikv/v1alpha1/defaulting"
	v1alpha1validation "github.com/tikv/tikv-operator/pkg/apis/tikv/v1alpha1/validation"
	"github.com/tikv/tikv-operator/pkg/controller"
	"github.com/tikv/tikv-operator/pkg/manager"
	"github.com/tikv/tikv-operator/pkg/manager/member"
	v1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	errorutils "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog"
)

// ControlInterface implements the control logic for updating TikvClusters and their children StatefulSets.
// It is implemented as an interface to allow for extensions that provide different semantics.
// Currently, there is only one implementation.
type ControlInterface interface {
	// UpdateTikvCluster implements the control logic for StatefulSet creation, update, and deletion
	UpdateTikvCluster(*v1alpha1.TikvCluster) error
}

// NewDefaultTikvClusterControl returns a new instance of the default implementation TikvClusterControlInterface that
// implements the documented semantics for TikvClusters.
func NewDefaultTikvClusterControl(
	tcControl controller.TikvClusterControlInterface,
	pdMemberManager manager.Manager,
	tikvMemberManager manager.Manager,
	metaManager manager.Manager,
	orphanPodsCleaner member.OrphanPodsCleaner,
	discoveryManager member.PDDiscoveryManager,
	conditionUpdater TikvClusterConditionUpdater,
	recorder record.EventRecorder) ControlInterface {
	return &defaultTikvClusterControl{
		tcControl,
		pdMemberManager,
		tikvMemberManager,
		metaManager,
		orphanPodsCleaner,
		discoveryManager,
		conditionUpdater,
		recorder,
	}
}

type defaultTikvClusterControl struct {
	tcControl         controller.TikvClusterControlInterface
	pdMemberManager   manager.Manager
	tikvMemberManager manager.Manager
	metaManager       manager.Manager
	orphanPodsCleaner member.OrphanPodsCleaner
	discoveryManager  member.PDDiscoveryManager
	conditionUpdater  TikvClusterConditionUpdater
	recorder          record.EventRecorder
}

// UpdateStatefulSet executes the core logic loop for a tikvcluster.
func (tcc *defaultTikvClusterControl) UpdateTikvCluster(tc *v1alpha1.TikvCluster) error {
	tcc.defaulting(tc)
	if !tcc.validate(tc) {
		return nil // fatal error, no need to retry on invalid object
	}

	var errs []error
	oldStatus := tc.Status.DeepCopy()

	if err := tcc.updateTikvCluster(tc); err != nil {
		errs = append(errs, err)
	}

	if err := tcc.conditionUpdater.Update(tc); err != nil {
		errs = append(errs, err)
	}

	if apiequality.Semantic.DeepEqual(&tc.Status, oldStatus) {
		return errorutils.NewAggregate(errs)
	}
	if _, err := tcc.tcControl.UpdateTikvCluster(tc.DeepCopy(), &tc.Status, oldStatus); err != nil {
		errs = append(errs, err)
	}

	return errorutils.NewAggregate(errs)
}

func (tcc *defaultTikvClusterControl) validate(tc *v1alpha1.TikvCluster) bool {
	errs := v1alpha1validation.ValidateTikvCluster(tc)
	if len(errs) > 0 {
		aggregatedErr := errs.ToAggregate()
		klog.Errorf("tikv cluster %s/%s is not valid and must be fixed first, aggregated error: %v", tc.GetNamespace(), tc.GetName(), aggregatedErr)
		tcc.recorder.Event(tc, v1.EventTypeWarning, "FailedValidation", aggregatedErr.Error())
		return false
	}
	return true
}

func (tcc *defaultTikvClusterControl) defaulting(tc *v1alpha1.TikvCluster) {
	defaulting.SetTikvClusterDefault(tc)
}

func (tcc *defaultTikvClusterControl) updateTikvCluster(tc *v1alpha1.TikvCluster) error {
	// cleaning all orphan pods managed by operator
	if _, err := tcc.orphanPodsCleaner.Clean(tc); err != nil {
		return err
	}

	// reconcile PD discovery service
	if err := tcc.discoveryManager.Reconcile(tc); err != nil {
		return err
	}

	// works that should do to making the pd cluster current state match the desired state:
	//   - create or update the pd service
	//   - create or update the pd headless service
	//   - create the pd statefulset
	//   - sync pd cluster status from pd to TikvCluster object
	//   - set two annotations to the first pd member:
	// 	   - label.Bootstrapping
	// 	   - label.Replicas
	//   - upgrade the pd cluster
	//   - scale out/in the pd cluster
	//   - failover the pd cluster
	if err := tcc.pdMemberManager.Sync(tc); err != nil {
		return err
	}

	// works that should do to making the tikv cluster current state match the desired state:
	//   - waiting for the pd cluster available(pd cluster is in quorum)
	//   - create or update tikv headless service
	//   - create the tikv statefulset
	//   - sync tikv cluster status from pd to TikvCluster object
	//   - set scheduler labels to tikv stores
	//   - upgrade the tikv cluster
	//   - scale out/in the tikv cluster
	//   - failover the tikv cluster
	if err := tcc.tikvMemberManager.Sync(tc); err != nil {
		return err
	}

	// syncing the labels from Pod to PVC and PV, these labels include:
	//   - label.StoreIDLabelKey
	//   - label.MemberIDLabelKey
	//   - label.NamespaceLabelKey
	if err := tcc.metaManager.Sync(tc); err != nil {
		return err
	}

	return nil
}

var _ ControlInterface = &defaultTikvClusterControl{}

type FakeTikvClusterControlInterface struct {
	err error
}

func NewFakeTikvClusterControlInterface() *FakeTikvClusterControlInterface {
	return &FakeTikvClusterControlInterface{}
}

func (ftcc *FakeTikvClusterControlInterface) SetUpdateTCError(err error) {
	ftcc.err = err
}

func (ftcc *FakeTikvClusterControlInterface) UpdateTikvCluster(_ *v1alpha1.TikvCluster) error {
	if ftcc.err != nil {
		return ftcc.err
	}
	return nil
}

var _ ControlInterface = &FakeTikvClusterControlInterface{}
