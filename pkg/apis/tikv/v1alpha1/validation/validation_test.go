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

package validation

import (
	"testing"

	. "github.com/onsi/gomega"
	"github.com/tikv/tikv-operator/pkg/apis/tikv/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestValidateRequestsStorage(t *testing.T) {
	g := NewGomegaWithT(t)
	tests := []struct {
		name                 string
		haveRequest          bool
		resourceRequirements corev1.ResourceRequirements
		expectedErrors       int
	}{
		{
			name:        "has request storage",
			haveRequest: true,
			resourceRequirements: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse("10G"),
				},
			},
			expectedErrors: 0,
		},
		{
			name:        "Empty request storage",
			haveRequest: false,
			resourceRequirements: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{},
			},
			expectedErrors: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tc := newTikvCluster()
			if tt.haveRequest {
				tc.Spec.PD.ResourceRequirements = tt.resourceRequirements
				tc.Spec.TiKV.ResourceRequirements = tt.resourceRequirements
			}
			err := ValidateTikvCluster(tc)
			r := len(err)
			g.Expect(r).Should(Equal(tt.expectedErrors))
		})
	}
}

func newTikvCluster() *v1alpha1.TikvCluster {
	tc := &v1alpha1.TikvCluster{}
	tc.Name = "test-validate-requests-storage"
	tc.Namespace = "default"
	return tc
}
