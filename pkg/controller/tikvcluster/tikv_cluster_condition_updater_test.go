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
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/tikv/tikv-operator/pkg/apis/tikv/v1alpha1"
	utiltikvcluster "github.com/tikv/tikv-operator/pkg/util/tikvcluster"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
)

func TestTikvClusterConditionUpdater_Ready(t *testing.T) {
	tests := []struct {
		name        string
		tc          *v1alpha1.TikvCluster
		wantStatus  v1.ConditionStatus
		wantReason  string
		wantMessage string
	}{
		{
			name: "statfulset(s) not up to date",
			tc: &v1alpha1.TikvCluster{
				Status: v1alpha1.TikvClusterStatus{
					PD: v1alpha1.PDStatus{
						StatefulSet: &appsv1.StatefulSetStatus{
							CurrentRevision: "1",
							UpdateRevision:  "2",
						},
					},
					TiKV: v1alpha1.TiKVStatus{
						StatefulSet: &appsv1.StatefulSetStatus{
							CurrentRevision: "1",
							UpdateRevision:  "2",
						},
					},
				},
			},
			wantStatus:  v1.ConditionFalse,
			wantReason:  utiltikvcluster.StatfulSetNotUpToDate,
			wantMessage: "Statefulset(s) are in progress",
		},
		{
			name: "pd(s) not healthy",
			tc: &v1alpha1.TikvCluster{
				Spec: v1alpha1.TikvClusterSpec{
					PD: v1alpha1.PDSpec{
						Replicas: 1,
					},
				},
				Status: v1alpha1.TikvClusterStatus{
					PD: v1alpha1.PDStatus{
						Members: map[string]v1alpha1.PDMember{
							"pd-1": {
								Health: false,
							},
						},
						StatefulSet: &appsv1.StatefulSetStatus{
							CurrentRevision: "2",
							UpdateRevision:  "2",
						},
					},
					TiKV: v1alpha1.TiKVStatus{
						StatefulSet: &appsv1.StatefulSetStatus{
							CurrentRevision: "2",
							UpdateRevision:  "2",
						},
					},
				},
			},
			wantStatus:  v1.ConditionFalse,
			wantReason:  utiltikvcluster.PDUnhealthy,
			wantMessage: "PD(s) are not healthy",
		},
		{
			name: "tikv(s) not healthy",
			tc: &v1alpha1.TikvCluster{
				Spec: v1alpha1.TikvClusterSpec{
					PD: v1alpha1.PDSpec{
						Replicas: 1,
					},
					TiKV: v1alpha1.TiKVSpec{
						Replicas: 1,
					},
				},
				Status: v1alpha1.TikvClusterStatus{
					PD: v1alpha1.PDStatus{
						Members: map[string]v1alpha1.PDMember{
							"pd-0": {
								Health: true,
							},
						},
						StatefulSet: &appsv1.StatefulSetStatus{
							CurrentRevision: "2",
							UpdateRevision:  "2",
						},
					},
					TiKV: v1alpha1.TiKVStatus{
						Stores: map[string]v1alpha1.TiKVStore{
							"tikv-0": {
								State: "Down",
							},
						},
						StatefulSet: &appsv1.StatefulSetStatus{
							CurrentRevision: "2",
							UpdateRevision:  "2",
						},
					},
				},
			},
			wantStatus:  v1.ConditionFalse,
			wantReason:  utiltikvcluster.TiKVStoreNotUp,
			wantMessage: "TiKV store(s) are not up",
		},
		{
			name: "all ready",
			tc: &v1alpha1.TikvCluster{
				Spec: v1alpha1.TikvClusterSpec{
					PD: v1alpha1.PDSpec{
						Replicas: 1,
					},
					TiKV: v1alpha1.TiKVSpec{
						Replicas: 1,
					},
				},
				Status: v1alpha1.TikvClusterStatus{
					PD: v1alpha1.PDStatus{
						Members: map[string]v1alpha1.PDMember{
							"pd-0": {
								Health: true,
							},
						},
						StatefulSet: &appsv1.StatefulSetStatus{
							CurrentRevision: "2",
							UpdateRevision:  "2",
						},
					},
					TiKV: v1alpha1.TiKVStatus{
						Stores: map[string]v1alpha1.TiKVStore{
							"tikv-0": {
								State: "Up",
							},
						},
						StatefulSet: &appsv1.StatefulSetStatus{
							CurrentRevision: "2",
							UpdateRevision:  "2",
						},
					},
				},
			},
			wantStatus:  v1.ConditionTrue,
			wantReason:  utiltikvcluster.Ready,
			wantMessage: "TiKV cluster is fully up and running",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conditionUpdater := &tikvClusterConditionUpdater{}
			conditionUpdater.Update(tt.tc)
			cond := utiltikvcluster.GetTikvClusterCondition(tt.tc.Status, v1alpha1.TikvClusterReady)
			if diff := cmp.Diff(tt.wantStatus, cond.Status); diff != "" {
				t.Errorf("unexpected status (-want, +got): %s", diff)
			}
			if diff := cmp.Diff(tt.wantReason, cond.Reason); diff != "" {
				t.Errorf("unexpected reason (-want, +got): %s", diff)
			}
			if diff := cmp.Diff(tt.wantMessage, cond.Message); diff != "" {
				t.Errorf("unexpected message (-want, +got): %s", diff)
			}
		})
	}
}
