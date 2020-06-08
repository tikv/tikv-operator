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
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// Reasons for TikvCluster conditions.

	// Ready
	Ready = "Ready"
	// StatefulSetNotUpToDate is added when one of statefulsets is not up to date.
	StatfulSetNotUpToDate = "StatefulSetNotUpToDate"
	// PDUnhealthy is added when one of pd members is unhealthy.
	PDUnhealthy = "PDUnhealthy"
	// TiKVStoreNotUp is added when one of tikv stores is not up.
	TiKVStoreNotUp = "TiKVStoreNotUp"
)

// NewTikvClusterCondition creates a new tikvcluster condition.
func NewTikvClusterCondition(condType v1alpha1.TikvClusterConditionType, status v1.ConditionStatus, reason, message string) *v1alpha1.TikvClusterCondition {
	return &v1alpha1.TikvClusterCondition{
		Type:               condType,
		Status:             status,
		LastUpdateTime:     metav1.Now(),
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	}
}

// GetTikvClusterCondition returns the condition with the provided type.
func GetTikvClusterCondition(status v1alpha1.TikvClusterStatus, condType v1alpha1.TikvClusterConditionType) *v1alpha1.TikvClusterCondition {
	for i := range status.Conditions {
		c := status.Conditions[i]
		if c.Type == condType {
			return &c
		}
	}
	return nil
}

// SetTikvClusterCondition updates the tikv cluster to include the provided condition. If the condition that
// we are about to add already exists and has the same status and reason then we are not going to update.
func SetTikvClusterCondition(status *v1alpha1.TikvClusterStatus, condition v1alpha1.TikvClusterCondition) {
	currentCond := GetTikvClusterCondition(*status, condition.Type)
	if currentCond != nil && currentCond.Status == condition.Status && currentCond.Reason == condition.Reason {
		return
	}
	// Do not update lastTransitionTime if the status of the condition doesn't change.
	if currentCond != nil && currentCond.Status == condition.Status {
		condition.LastTransitionTime = currentCond.LastTransitionTime
	}
	newConditions := filterOutCondition(status.Conditions, condition.Type)
	status.Conditions = append(newConditions, condition)
}

// filterOutCondition returns a new slice of tikvcluster conditions without conditions with the provided type.
func filterOutCondition(conditions []v1alpha1.TikvClusterCondition, condType v1alpha1.TikvClusterConditionType) []v1alpha1.TikvClusterCondition {
	var newConditions []v1alpha1.TikvClusterCondition
	for _, c := range conditions {
		if c.Type == condType {
			continue
		}
		newConditions = append(newConditions, c)
	}
	return newConditions
}

// GetTikvClusterReadyCondition extracts the tikvcluster ready condition from the given status and returns that.
// Returns nil if the condition is not present.
func GetTikvClusterReadyCondition(status v1alpha1.TikvClusterStatus) *v1alpha1.TikvClusterCondition {
	return GetTikvClusterCondition(status, v1alpha1.TikvClusterReady)
}
