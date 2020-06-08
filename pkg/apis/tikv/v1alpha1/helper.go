// Copyright 2019 TiKV Project Authors.
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

package v1alpha1

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pingcap/advanced-statefulset/client/apis/apps/v1/helper"
	"github.com/tikv/tikv-operator/pkg/label"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

const (
	defaultHelperImage = "busybox:1.26.2"
	defaultTimeZone    = "UTC"
)

func (tc *TikvCluster) PDImage() string {
	image := tc.Spec.PD.Image
	baseImage := tc.Spec.PD.BaseImage
	// base image takes higher priority
	if baseImage != "" {
		version := tc.Spec.PD.Version
		if version == nil {
			version = &tc.Spec.Version
		}
		image = fmt.Sprintf("%s:%s", baseImage, *version)
	}
	return image
}

func (tc *TikvCluster) PDVersion() string {
	image := tc.PDImage()
	colonIdx := strings.LastIndexByte(image, ':')
	if colonIdx >= 0 {
		return image[colonIdx+1:]
	}

	return "latest"
}

func (tc *TikvCluster) TiKVImage() string {
	image := tc.Spec.TiKV.Image
	baseImage := tc.Spec.TiKV.BaseImage
	// base image takes higher priority
	if baseImage != "" {
		version := tc.Spec.TiKV.Version
		if version == nil {
			version = &tc.Spec.Version
		}
		image = fmt.Sprintf("%s:%s", baseImage, *version)
	}
	return image
}

func (tc *TikvCluster) GetInstanceName() string {
	return tc.Name
}

func (tc *TikvCluster) IsTLSClusterEnabled() bool {
	return false
}

func (tc *TikvCluster) Timezone() string {
	tz := tc.Spec.Timezone
	if tz == "" {
		return defaultTimeZone
	}
	return tz
}

func (tc *TikvCluster) PDAllPodsStarted() bool {
	return tc.PDStsDesiredReplicas() == tc.PDStsActualReplicas()
}

func (tc *TikvCluster) PDAllMembersReady() bool {
	if int(tc.PDStsDesiredReplicas()) != len(tc.Status.PD.Members) {
		return false
	}

	for _, member := range tc.Status.PD.Members {
		if !member.Health {
			return false
		}
	}
	return true
}

func (tc *TikvCluster) PDAutoFailovering() bool {
	if len(tc.Status.PD.FailureMembers) == 0 {
		return false
	}

	for _, failureMember := range tc.Status.PD.FailureMembers {
		if !failureMember.MemberDeleted {
			return true
		}
	}
	return false
}

func (tc *TikvCluster) PDStsDesiredReplicas() int32 {
	return tc.Spec.PD.Replicas + int32(len(tc.Status.PD.FailureMembers))
}

func (tc *TikvCluster) PDStsActualReplicas() int32 {
	stsStatus := tc.Status.PD.StatefulSet
	if stsStatus == nil {
		return 0
	}
	return stsStatus.Replicas
}

func (tc *TikvCluster) PDStsDesiredOrdinals(excludeFailover bool) sets.Int32 {
	replicas := tc.Spec.PD.Replicas
	if !excludeFailover {
		replicas = tc.PDStsDesiredReplicas()
	}
	return helper.GetPodOrdinalsFromReplicasAndDeleteSlots(replicas, tc.getDeleteSlots(label.PDLabelVal))
}

func (tc *TikvCluster) TiKVAllPodsStarted() bool {
	return tc.TiKVStsDesiredReplicas() == tc.TiKVStsActualReplicas()
}

func (tc *TikvCluster) TiKVAllStoresReady() bool {
	if int(tc.TiKVStsDesiredReplicas()) != len(tc.Status.TiKV.Stores) {
		return false
	}

	for _, store := range tc.Status.TiKV.Stores {
		if store.State != TiKVStateUp {
			return false
		}
	}

	return true
}

func (tc *TikvCluster) TiKVStsDesiredReplicas() int32 {
	return tc.Spec.TiKV.Replicas + int32(len(tc.Status.TiKV.FailureStores))
}

func (tc *TikvCluster) TiKVStsActualReplicas() int32 {
	stsStatus := tc.Status.TiKV.StatefulSet
	if stsStatus == nil {
		return 0
	}
	return stsStatus.Replicas
}

func (tc *TikvCluster) TiKVStsDesiredOrdinals(excludeFailover bool) sets.Int32 {
	replicas := tc.Spec.TiKV.Replicas
	if !excludeFailover {
		replicas = tc.TiKVStsDesiredReplicas()
	}
	return helper.GetPodOrdinalsFromReplicasAndDeleteSlots(replicas, tc.getDeleteSlots(label.TiKVLabelVal))
}

func (tc *TikvCluster) getDeleteSlots(component string) (deleteSlots sets.Int32) {
	deleteSlots = sets.NewInt32()
	annotations := tc.GetAnnotations()
	if annotations == nil {
		return deleteSlots
	}
	var key string
	if component == label.PDLabelVal {
		key = label.AnnPDDeleteSlots
	} else if component == label.TiKVLabelVal {
		key = label.AnnTiKVDeleteSlots
	} else {
		return
	}
	value, ok := annotations[key]
	if !ok {
		return
	}
	var slice []int32
	err := json.Unmarshal([]byte(value), &slice)
	if err != nil {
		return
	}
	deleteSlots.Insert(slice...)
	return
}

func (tc *TikvCluster) Scheme() string {
	if tc.IsTLSClusterEnabled() {
		return "https"
	}
	return "http"
}

func (tc *TikvCluster) PDUpgrading() bool {
	return tc.Status.PD.Phase == UpgradePhase
}

func (tc *TikvCluster) TiKVUpgrading() bool {
	return tc.Status.TiKV.Phase == UpgradePhase
}

func (tc *TikvCluster) PDIsAvailable() bool {
	lowerLimit := tc.Spec.PD.Replicas/2 + 1
	if int32(len(tc.Status.PD.Members)) < lowerLimit {
		return false
	}

	var availableNum int32
	for _, pdMember := range tc.Status.PD.Members {
		if pdMember.Health {
			availableNum++
		}
	}

	if availableNum < lowerLimit {
		return false
	}

	if tc.Status.PD.StatefulSet == nil || tc.Status.PD.StatefulSet.ReadyReplicas < lowerLimit {
		return false
	}

	return true
}

func (tc *TikvCluster) HelperImage() string {
	return defaultHelperImage
}

func (tc *TikvCluster) HelperImagePullPolicy() corev1.PullPolicy {
	return tc.Spec.ImagePullPolicy
}

func (tc *TikvCluster) TiKVContainerPrivilege() *bool {
	if tc.Spec.TiKV.Privileged == nil {
		pri := false
		return &pri
	}
	return tc.Spec.TiKV.Privileged
}
