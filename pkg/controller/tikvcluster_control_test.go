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
	"errors"
	"testing"

	"time"

	. "github.com/onsi/gomega"
	"github.com/tikv/tikv-operator/pkg/apis/tikv/v1alpha1"
	"github.com/tikv/tikv-operator/pkg/client/clientset/versioned/fake"
	listers "github.com/tikv/tikv-operator/pkg/client/listers/tikv/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	core "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
)

func TestTikvClusterControlUpdateTikvCluster(t *testing.T) {
	g := NewGomegaWithT(t)
	recorder := record.NewFakeRecorder(10)
	tc := newTikvCluster()
	tc.Spec.PD.Replicas = int32(5)
	fakeClient := &fake.Clientset{}
	control := NewRealTikvClusterControl(fakeClient, nil, recorder)
	fakeClient.AddReactor("update", "tikvclusters", func(action core.Action) (bool, runtime.Object, error) {
		update := action.(core.UpdateAction)
		return true, update.GetObject(), nil
	})
	updateTC, err := control.UpdateTikvCluster(tc, &v1alpha1.TikvClusterStatus{}, &v1alpha1.TikvClusterStatus{})
	g.Expect(err).To(Succeed())
	g.Expect(updateTC.Spec.PD.Replicas).To(Equal(int32(5)))
}

func TestTikvClusterControlUpdateTikvClusterConflictSuccess(t *testing.T) {
	g := NewGomegaWithT(t)
	recorder := record.NewFakeRecorder(10)
	tc := newTikvCluster()
	fakeClient := &fake.Clientset{}
	indexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
	tcLister := listers.NewTikvClusterLister(indexer)
	control := NewRealTikvClusterControl(fakeClient, tcLister, recorder)
	conflict := false
	fakeClient.AddReactor("update", "tikvclusters", func(action core.Action) (bool, runtime.Object, error) {
		update := action.(core.UpdateAction)
		if !conflict {
			conflict = true
			return true, update.GetObject(), apierrors.NewConflict(action.GetResource().GroupResource(), tc.Name, errors.New("conflict"))
		}
		return true, update.GetObject(), nil
	})
	_, err := control.UpdateTikvCluster(tc, &v1alpha1.TikvClusterStatus{}, &v1alpha1.TikvClusterStatus{})
	g.Expect(err).To(Succeed())
}

func TestDeepEqualExceptHeartbeatTime(t *testing.T) {
	g := NewGomegaWithT(t)

	new := &v1alpha1.TikvClusterStatus{
		TiKV: v1alpha1.TiKVStatus{
			Synced: true,
			Stores: map[string]v1alpha1.TiKVStore{
				"1": {
					LastHeartbeatTime: metav1.Now(),
					ID:                "1",
				},
			},
		},
	}
	time.Sleep(1 * time.Second)
	old := &v1alpha1.TikvClusterStatus{
		TiKV: v1alpha1.TiKVStatus{
			Synced: true,
			Stores: map[string]v1alpha1.TiKVStore{
				"1": {
					LastHeartbeatTime: metav1.Now(),
					ID:                "1",
				},
			},
		},
	}
	g.Expect(deepEqualExceptHeartbeatTime(new, old)).To(Equal(true))
}
