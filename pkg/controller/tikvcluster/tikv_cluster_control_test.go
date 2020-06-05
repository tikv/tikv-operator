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
	"fmt"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/tikv/tikv-operator/pkg/apis/tikv/v1alpha1"
	"github.com/tikv/tikv-operator/pkg/client/clientset/versioned/fake"
	informers "github.com/tikv/tikv-operator/pkg/client/informers/externalversions"
	"github.com/tikv/tikv-operator/pkg/controller"
	mm "github.com/tikv/tikv-operator/pkg/manager/member"
	"github.com/tikv/tikv-operator/pkg/manager/meta"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
)

func TestTikvClusterControlUpdateTikvCluster(t *testing.T) {
	g := NewGomegaWithT(t)

	type testcase struct {
		name                     string
		update                   func(cluster *v1alpha1.TikvCluster)
		orphanPodCleanerErr      bool
		syncPDMemberManagerErr   bool
		syncTiKVMemberManagerErr bool
		syncMetaManagerErr       bool
		updateTCStatusErr        bool
		errExpectFn              func(*GomegaWithT, error)
	}
	testFn := func(test *testcase, t *testing.T) {
		t.Log(test.name)

		tc := newTikvClusterForTikvClusterControl()
		if test.update != nil {
			test.update(tc)
		}
		control, orphanPodCleaner, pdMemberManager, tikvMemberManager, metaManager, tcUpdater := newFakeTikvClusterControl()

		if test.orphanPodCleanerErr {
			orphanPodCleaner.SetnOrphanPodCleanerError(fmt.Errorf("clean orphan pod error"))
		}
		if test.syncPDMemberManagerErr {
			pdMemberManager.SetSyncError(fmt.Errorf("pd member manager sync error"))
		}
		if test.syncTiKVMemberManagerErr {
			tikvMemberManager.SetSyncError(fmt.Errorf("tikv member manager sync error"))
		}
		if test.syncMetaManagerErr {
			metaManager.SetSyncError(fmt.Errorf("meta manager sync error"))
		}
		if test.updateTCStatusErr {
			tcUpdater.SetUpdateTikvClusterError(fmt.Errorf("update tikvcluster status error"), 0)
		}

		err := control.UpdateTikvCluster(tc)
		if test.errExpectFn != nil {
			test.errExpectFn(g, err)
		}
	}
	tests := []testcase{
		{
			name:                     "clean orphan pod error",
			update:                   nil,
			orphanPodCleanerErr:      true,
			syncPDMemberManagerErr:   false,
			syncTiKVMemberManagerErr: false,
			syncMetaManagerErr:       false,
			updateTCStatusErr:        false,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "clean orphan pod error")).To(Equal(true))
			},
		},
		{
			name:                     "pd member manager sync error",
			update:                   nil,
			orphanPodCleanerErr:      false,
			syncPDMemberManagerErr:   true,
			syncTiKVMemberManagerErr: false,
			syncMetaManagerErr:       false,
			updateTCStatusErr:        false,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "pd member manager sync error")).To(Equal(true))
			},
		},
		{
			name:                     "tikv member manager sync error",
			update:                   nil,
			orphanPodCleanerErr:      false,
			syncPDMemberManagerErr:   false,
			syncTiKVMemberManagerErr: true,
			syncMetaManagerErr:       false,
			updateTCStatusErr:        false,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "tikv member manager sync error")).To(Equal(true))
			},
		},
		{
			name:                     "meta manager sync error",
			update:                   nil,
			orphanPodCleanerErr:      false,
			syncPDMemberManagerErr:   false,
			syncTiKVMemberManagerErr: false,
			syncMetaManagerErr:       true,
			updateTCStatusErr:        false,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "meta manager sync error")).To(Equal(true))
			},
		},
		{
			name:                     "tikvcluster status is not updated",
			update:                   nil,
			orphanPodCleanerErr:      false,
			syncPDMemberManagerErr:   false,
			syncTiKVMemberManagerErr: false,
			syncMetaManagerErr:       false,
			updateTCStatusErr:        false,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).NotTo(HaveOccurred())
			},
		},
		{
			name: "tikvcluster status update failed",
			update: func(cluster *v1alpha1.TikvCluster) {
				cluster.Status.PD.Members = map[string]v1alpha1.PDMember{
					"pd-0": {Name: "pd-0", Health: true},
					"pd-1": {Name: "pd-1", Health: true},
					"pd-2": {Name: "pd-2", Health: true},
				}
				cluster.Status.PD.StatefulSet = &apps.StatefulSetStatus{ReadyReplicas: 3}
				cluster.Status.TiKV.Stores = map[string]v1alpha1.TiKVStore{
					"tikv-0": {PodName: "tikv-0", State: v1alpha1.TiKVStateUp},
					"tikv-1": {PodName: "tikv-1", State: v1alpha1.TiKVStateUp},
					"tikv-2": {PodName: "tikv-2", State: v1alpha1.TiKVStateUp},
				}
				cluster.Status.TiKV.StatefulSet = &apps.StatefulSetStatus{ReadyReplicas: 3}
			},
			orphanPodCleanerErr:      false,
			syncPDMemberManagerErr:   false,
			syncTiKVMemberManagerErr: false,
			syncMetaManagerErr:       false,
			updateTCStatusErr:        true,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).To(HaveOccurred())
				g.Expect(strings.Contains(err.Error(), "update tikvcluster status error")).To(Equal(true))
			},
		},
		{
			name: "normal",
			update: func(cluster *v1alpha1.TikvCluster) {
				cluster.Status.PD.Members = map[string]v1alpha1.PDMember{
					"pd-0": {Name: "pd-0", Health: true},
					"pd-1": {Name: "pd-1", Health: true},
					"pd-2": {Name: "pd-2", Health: true},
				}
				cluster.Status.PD.StatefulSet = &apps.StatefulSetStatus{ReadyReplicas: 3}
				cluster.Status.TiKV.Stores = map[string]v1alpha1.TiKVStore{
					"tikv-0": {PodName: "tikv-0", State: v1alpha1.TiKVStateUp},
					"tikv-1": {PodName: "tikv-1", State: v1alpha1.TiKVStateUp},
					"tikv-2": {PodName: "tikv-2", State: v1alpha1.TiKVStateUp},
				}
				cluster.Status.TiKV.StatefulSet = &apps.StatefulSetStatus{ReadyReplicas: 3}
			},
			orphanPodCleanerErr:      false,
			syncPDMemberManagerErr:   false,
			syncTiKVMemberManagerErr: false,
			syncMetaManagerErr:       false,
			updateTCStatusErr:        false,
			errExpectFn: func(g *GomegaWithT, err error) {
				g.Expect(err).NotTo(HaveOccurred())
			},
		},
	}

	for i := range tests {
		testFn(&tests[i], t)
	}
}

func TestTikvClusterStatusEquality(t *testing.T) {
	g := NewGomegaWithT(t)
	tcStatus := v1alpha1.TikvClusterStatus{}

	tcStatusCopy := tcStatus.DeepCopy()
	tcStatusCopy.PD = v1alpha1.PDStatus{}
	g.Expect(apiequality.Semantic.DeepEqual(&tcStatus, tcStatusCopy)).To(Equal(true))

	tcStatusCopy = tcStatus.DeepCopy()
	tcStatusCopy.PD.Phase = v1alpha1.NormalPhase
	g.Expect(apiequality.Semantic.DeepEqual(&tcStatus, tcStatusCopy)).To(Equal(false))
}

func newFakeTikvClusterControl() (
	ControlInterface,
	*mm.FakeOrphanPodsCleaner,
	*mm.FakePDMemberManager,
	*mm.FakeTiKVMemberManager,
	*meta.FakeMetaManager,
	*controller.FakeTikvClusterControl) {
	cli := fake.NewSimpleClientset()
	tcInformer := informers.NewSharedInformerFactory(cli, 0).Tikv().V1alpha1().TikvClusters()
	recorder := record.NewFakeRecorder(10)

	tcUpdater := controller.NewFakeTikvClusterControl(tcInformer)
	pdMemberManager := mm.NewFakePDMemberManager()
	tikvMemberManager := mm.NewFakeTiKVMemberManager()
	metaManager := meta.NewFakeMetaManager()
	orphanPodCleaner := mm.NewFakeOrphanPodsCleaner()
	discoveryManager := mm.NewFakeDiscoveryManger()
	control := NewDefaultTikvClusterControl(
		tcUpdater,
		pdMemberManager,
		tikvMemberManager,
		metaManager,
		orphanPodCleaner,
		discoveryManager,
		&tikvClusterConditionUpdater{},
		recorder,
	)

	return control, orphanPodCleaner, pdMemberManager, tikvMemberManager, metaManager, tcUpdater
}

func newTikvClusterForTikvClusterControl() *v1alpha1.TikvCluster {
	return &v1alpha1.TikvCluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "TikvCluster",
			APIVersion: "pingcap.com/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pd",
			Namespace: corev1.NamespaceDefault,
			UID:       types.UID("test"),
		},
		Spec: v1alpha1.TikvClusterSpec{
			Version: "v3.0.8",
			PD: v1alpha1.PDSpec{
				Replicas:  3,
				BaseImage: "pingcap/pd",
				Config:    &v1alpha1.PDConfig{},
				ResourceRequirements: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: resource.MustParse("10G"),
					},
				},
			},
			TiKV: v1alpha1.TiKVSpec{
				Replicas:  3,
				BaseImage: "pingcap/tikv",
				Config:    &v1alpha1.TiKVConfig{},
				ResourceRequirements: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: resource.MustParse("10G"),
					},
				},
			},
		},
	}
}
