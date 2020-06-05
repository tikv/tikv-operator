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

package registry

import (
	"context"

	"github.com/tikv/tikv-operator/pkg/apis/tikv/v1alpha1"
	"github.com/tikv/tikv-operator/pkg/apis/tikv/v1alpha1/defaulting"
	"github.com/tikv/tikv-operator/pkg/apis/tikv/v1alpha1/validation"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/klog"
)

// +k8s:deepcopy-gen=false
type TikvClusterStrategy struct{}

func (TikvClusterStrategy) NewObject() runtime.Object {
	return &v1alpha1.TikvCluster{}
}

func (TikvClusterStrategy) PrepareForCreate(ctx context.Context, obj runtime.Object) {
	if tc, ok := castTikvCluster(obj); ok {
		defaulting.SetTikvClusterDefault(tc)
	}
}

func (TikvClusterStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
	// no op to not affect the cluster managed by old versions of the helm chart
}

func (TikvClusterStrategy) Validate(ctx context.Context, obj runtime.Object) field.ErrorList {
	if tc, ok := castTikvCluster(obj); ok {
		return validation.ValidateCreateTikvCluster(tc)
	}
	return field.ErrorList{}
}

func (TikvClusterStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	oldTc, oldOk := castTikvCluster(old)
	tc, ok := castTikvCluster(obj)
	if ok && oldOk {
		return validation.ValidateUpdateTikvCluster(oldTc, tc)
	}
	return field.ErrorList{}
}

func castTikvCluster(obj runtime.Object) (*v1alpha1.TikvCluster, bool) {
	tc, ok := obj.(*v1alpha1.TikvCluster)
	if !ok {
		// impossible for non-malicious request, this usually indicates a client error when the strategy is used by webhook,
		// we simply ignore error requests
		klog.Errorf("Object %T is not v1alpah1.TikvCluster, cannot processed by TikvClusterStrategy", obj)
		return nil, false
	}
	return tc, true
}
