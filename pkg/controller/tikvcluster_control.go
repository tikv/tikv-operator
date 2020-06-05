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

package controller

import (
	"fmt"
	"strings"

	"github.com/tikv/tikv-operator/pkg/apis/tikv/v1alpha1"
	"github.com/tikv/tikv-operator/pkg/client/clientset/versioned"
	tcinformers "github.com/tikv/tikv-operator/pkg/client/informers/externalversions/tikv/v1alpha1"
	listers "github.com/tikv/tikv-operator/pkg/client/listers/tikv/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog"
)

// TikvClusterControlInterface manages TikvClusters
type TikvClusterControlInterface interface {
	UpdateTikvCluster(*v1alpha1.TikvCluster, *v1alpha1.TikvClusterStatus, *v1alpha1.TikvClusterStatus) (*v1alpha1.TikvCluster, error)
}

type realTikvClusterControl struct {
	cli      versioned.Interface
	tcLister listers.TikvClusterLister
	recorder record.EventRecorder
}

// NewRealTikvClusterControl creates a new TikvClusterControlInterface
func NewRealTikvClusterControl(cli versioned.Interface,
	tcLister listers.TikvClusterLister,
	recorder record.EventRecorder) TikvClusterControlInterface {
	return &realTikvClusterControl{
		cli,
		tcLister,
		recorder,
	}
}

func (rtc *realTikvClusterControl) UpdateTikvCluster(tc *v1alpha1.TikvCluster, newStatus *v1alpha1.TikvClusterStatus, oldStatus *v1alpha1.TikvClusterStatus) (*v1alpha1.TikvCluster, error) {
	ns := tc.GetNamespace()
	tcName := tc.GetName()

	status := tc.Status.DeepCopy()
	var updateTC *v1alpha1.TikvCluster

	// don't wait due to limited number of clients, but backoff after the default number of steps
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		var updateErr error
		updateTC, updateErr = rtc.cli.TikvV1alpha1().TikvClusters(ns).Update(tc)
		if updateErr == nil {
			klog.Infof("TikvCluster: [%s/%s] updated successfully", ns, tcName)
			return nil
		}
		klog.Errorf("failed to update TikvCluster: [%s/%s], error: %v", ns, tcName, updateErr)

		if updated, err := rtc.tcLister.TikvClusters(ns).Get(tcName); err == nil {
			// make a copy so we don't mutate the shared cache
			tc = updated.DeepCopy()
			tc.Status = *status
		} else {
			utilruntime.HandleError(fmt.Errorf("error getting updated TikvCluster %s/%s from lister: %v", ns, tcName, err))
		}

		return updateErr
	})
	return updateTC, err
}

func (rtc *realTikvClusterControl) recordTikvClusterEvent(verb string, tc *v1alpha1.TikvCluster, err error) {
	tcName := tc.GetName()
	if err == nil {
		reason := fmt.Sprintf("Successful%s", strings.Title(verb))
		msg := fmt.Sprintf("%s TikvCluster %s successful",
			strings.ToLower(verb), tcName)
		rtc.recorder.Event(tc, corev1.EventTypeNormal, reason, msg)
	} else {
		reason := fmt.Sprintf("Failed%s", strings.Title(verb))
		msg := fmt.Sprintf("%s TikvCluster %s failed error: %s",
			strings.ToLower(verb), tcName, err)
		rtc.recorder.Event(tc, corev1.EventTypeWarning, reason, msg)
	}
}

func deepEqualExceptHeartbeatTime(newStatus *v1alpha1.TikvClusterStatus, oldStatus *v1alpha1.TikvClusterStatus) bool {
	sweepHeartbeatTime(newStatus.TiKV.Stores)
	sweepHeartbeatTime(newStatus.TiKV.TombstoneStores)
	sweepHeartbeatTime(oldStatus.TiKV.Stores)
	sweepHeartbeatTime(oldStatus.TiKV.TombstoneStores)

	return apiequality.Semantic.DeepEqual(newStatus, oldStatus)
}

func sweepHeartbeatTime(stores map[string]v1alpha1.TiKVStore) {
	for id, store := range stores {
		store.LastHeartbeatTime = metav1.Time{}
		stores[id] = store
	}
}

// FakeTikvClusterControl is a fake TikvClusterControlInterface
type FakeTikvClusterControl struct {
	TcLister                 listers.TikvClusterLister
	TcIndexer                cache.Indexer
	updateTikvClusterTracker RequestTracker
}

// NewFakeTikvClusterControl returns a FakeTikvClusterControl
func NewFakeTikvClusterControl(tcInformer tcinformers.TikvClusterInformer) *FakeTikvClusterControl {
	return &FakeTikvClusterControl{
		tcInformer.Lister(),
		tcInformer.Informer().GetIndexer(),
		RequestTracker{},
	}
}

// SetUpdateTikvClusterError sets the error attributes of updateTikvClusterTracker
func (ssc *FakeTikvClusterControl) SetUpdateTikvClusterError(err error, after int) {
	ssc.updateTikvClusterTracker.SetError(err).SetAfter(after)
}

// UpdateTikvCluster updates the TikvCluster
func (ssc *FakeTikvClusterControl) UpdateTikvCluster(tc *v1alpha1.TikvCluster, _ *v1alpha1.TikvClusterStatus, _ *v1alpha1.TikvClusterStatus) (*v1alpha1.TikvCluster, error) {
	defer ssc.updateTikvClusterTracker.Inc()
	if ssc.updateTikvClusterTracker.ErrorReady() {
		defer ssc.updateTikvClusterTracker.Reset()
		return tc, ssc.updateTikvClusterTracker.GetError()
	}

	return tc, ssc.TcIndexer.Update(tc)
}
