// Copyright 2020 TiKV Project Authors.
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
	utiltikvcluster "github.com/tikv/tikv-operator/pkg/util/tikvcluster"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
)

// TikvClusterConditionUpdater interface that translates cluster state into
// into tikv cluster status conditions.
type TikvClusterConditionUpdater interface {
	Update(*v1alpha1.TikvCluster) error
}

type tikvClusterConditionUpdater struct {
}

var _ TikvClusterConditionUpdater = &tikvClusterConditionUpdater{}

func (u *tikvClusterConditionUpdater) Update(tc *v1alpha1.TikvCluster) error {
	u.updateReadyCondition(tc)
	// in the future, we may return error when we need to Kubernetes API, etc.
	return nil
}

func allStatefulSetsAreUpToDate(tc *v1alpha1.TikvCluster) bool {
	isUpToDate := func(status *appsv1.StatefulSetStatus, requireExist bool) bool {
		if status == nil {
			return !requireExist
		}
		return status.CurrentRevision == status.UpdateRevision
	}
	return isUpToDate(tc.Status.PD.StatefulSet, true) &&
		isUpToDate(tc.Status.TiKV.StatefulSet, true)
}

func (u *tikvClusterConditionUpdater) updateReadyCondition(tc *v1alpha1.TikvCluster) {
	status := v1.ConditionFalse
	reason := ""
	message := ""

	switch {
	case !allStatefulSetsAreUpToDate(tc):
		reason = utiltikvcluster.StatfulSetNotUpToDate
		message = "Statefulset(s) are in progress"
	case !tc.PDAllMembersReady():
		reason = utiltikvcluster.PDUnhealthy
		message = "PD(s) are not healthy"
	case !tc.TiKVAllStoresReady():
		reason = utiltikvcluster.TiKVStoreNotUp
		message = "TiKV store(s) are not up"
	default:
		status = v1.ConditionTrue
		reason = utiltikvcluster.Ready
		message = "TiKV cluster is fully up and running"
	}
	cond := utiltikvcluster.NewTikvClusterCondition(v1alpha1.TikvClusterReady, status, reason, message)
	utiltikvcluster.SetTikvClusterCondition(&tc.Status, *cond)
}
