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

package defaulting

import (
	"github.com/tikv/tikv-operator/pkg/apis/tikv/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"
)

const (
	defaultTiKVImage = "pingcap/tikv"
	defaultPDImage   = "pingcap/pd"
)

func SetTikvClusterDefault(tc *v1alpha1.TikvCluster) {
	setTikvClusterSpecDefault(tc)
	setPdSpecDefault(tc)
	setTikvSpecDefault(tc)
}

// setTikvClusterSpecDefault is only managed the property under Spec
func setTikvClusterSpecDefault(tc *v1alpha1.TikvCluster) {
	if string(tc.Spec.ImagePullPolicy) == "" {
		tc.Spec.ImagePullPolicy = corev1.PullIfNotPresent
	}
}

func setTikvSpecDefault(tc *v1alpha1.TikvCluster) {
	if len(tc.Spec.Version) > 0 || tc.Spec.TiKV.Version != nil {
		if tc.Spec.TiKV.BaseImage == "" {
			tc.Spec.TiKV.BaseImage = defaultTiKVImage
		}
	}
	if tc.Spec.TiKV.MaxFailoverCount == nil {
		tc.Spec.TiKV.MaxFailoverCount = pointer.Int32Ptr(3)
	}
}

func setPdSpecDefault(tc *v1alpha1.TikvCluster) {
	if len(tc.Spec.Version) > 0 || tc.Spec.PD.Version != nil {
		if tc.Spec.PD.BaseImage == "" {
			tc.Spec.PD.BaseImage = defaultPDImage
		}
	}
	if tc.Spec.PD.MaxFailoverCount == nil {
		tc.Spec.PD.MaxFailoverCount = pointer.Int32Ptr(3)
	}
}
