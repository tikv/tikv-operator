// Copyright 2017 TiKV Project Authors.
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

package label

import (
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

const (
	// The following labels are recommended by kubernetes https://kubernetes.io/docs/concepts/overview/working-with-objects/common-labels/

	// ManagedByLabelKey is Kubernetes recommended label key, it represents the tool being used to manage the operation of an application
	// For resources managed by TiDB Operator, its value is always tikv-operator
	ManagedByLabelKey string = "app.kubernetes.io/managed-by"
	// ComponentLabelKey is Kubernetes recommended label key, it represents the component within the architecture
	ComponentLabelKey string = "app.kubernetes.io/component"
	// NameLabelKey is Kubernetes recommended label key, it represents the name of the application
	// It should always be tikv-cluster in our case.
	NameLabelKey string = "app.kubernetes.io/name"
	// InstanceLabelKey is Kubernetes recommended label key, it represents a unique name identifying the instance of an application
	// It's set by helm when installing a release
	InstanceLabelKey string = "app.kubernetes.io/instance"

	// NamespaceLabelKey is label key used in PV for easy querying
	NamespaceLabelKey string = "app.kubernetes.io/namespace"

	// ClusterIDLabelKey is cluster id label key
	ClusterIDLabelKey string = "tikv.org/cluster-id"

	// StoreIDLabelKey is TiKV store id label key
	StoreIDLabelKey string = "tikv.org/store-id"

	// MemberIDLabelKey is member id label key
	MemberIDLabelKey string = "tikv.org/member-id"

	// AnnForceUpgradeKey is tc annotation key to indicate whether force upgrade should be done
	AnnForceUpgradeKey = "tikv.org/force-upgrade"

	// AnnPDDeferDeleting is pd pod annotation key  in pod for defer for deleting pod
	AnnPDDeferDeleting = "tikv.org/pd-defer-deleting"

	// AnnEvictLeaderBeginTime is pod annotation key to indicate the begin time for evicting region leader
	AnnEvictLeaderBeginTime = "tikv.org/evictLeaderBeginTime"

	// AnnPodDeferDeleting is pod annotation key to indicate the pod which need to be restarted
	AnnPodDeferDeleting = "tikv.org/pod-defer-deleting"

	// AnnPodNameKey is pod name annotation key used in PV/PVC for synchronizing tidb cluster meta info
	AnnPodNameKey string = "tikv.org/pod-name"

	// AnnPVCDeferDeleting is pvc defer deletion annotation key used in PVC for defer deleting PVC
	AnnPVCDeferDeleting = "tikv.org/pvc-defer-deleting"

	// AnnPVCPodScheduling is pod scheduling annotation key, it represents whether the pod is scheduling
	AnnPVCPodScheduling = "tikv.org/pod-scheduling"

	// AnnPDDeleteSlots is annotation key of pd delete slots.
	AnnPDDeleteSlots = "pd.tikv.org/delete-slots"

	// TiKVDeleteSlots is annotation key of tikv delete slots.
	AnnTiKVDeleteSlots = "tikv.tikv.org/delete-slots"

	// AnnSysctlInit is pod annotation key to indicate whether configuring sysctls with init container
	AnnSysctlInit = "tikv.org/sysctl-init"

	// AnnSysctlInitVal is pod annotation value to indicate whether configuring sysctls with init container
	AnnSysctlInitVal = "true"

	// AnnForceUpgradeVal is tc annotation value to indicate whether force upgrade should be done
	AnnForceUpgradeVal = "true"

	// PDLabelVal is PD label value
	PDLabelVal string = "pd"

	// TiKVLabelVal is TiKV label value
	TiKVLabelVal string = "tikv"

	// DiscoveryLabelVal is Discovery label value
	DiscoveryLabelVal string = "discovery"

	// TiKVOperator is ManagedByLabelKey label value
	TiKVOperator string = "tikv-operator"
)

// Label is the label field in metadata
type Label map[string]string

// New initialize a new Label for components of tikv cluster
func New() Label {
	return Label{
		NameLabelKey:      "tikv-cluster",
		ManagedByLabelKey: TiKVOperator,
	}
}

// Instance adds instance kv pair to label
func (l Label) Instance(name string) Label {
	l[InstanceLabelKey] = name
	return l
}

// Namespace adds namespace kv pair to label
func (l Label) Namespace(name string) Label {
	l[NamespaceLabelKey] = name
	return l
}

// Component adds component kv pair to label
func (l Label) Component(name string) Label {
	l[ComponentLabelKey] = name
	return l
}

// ComponentType returns component type
func (l Label) ComponentType() string {
	return l[ComponentLabelKey]
}

// PD assigns pd to component key in label
func (l Label) PD() Label {
	l.Component(PDLabelVal)
	return l
}

// Discovery assigns discovery to component key in label
func (l Label) Discovery() Label {
	l.Component(DiscoveryLabelVal)
	return l
}

// IsPD returns whether label is a PD
func (l Label) IsPD() bool {
	return l[ComponentLabelKey] == PDLabelVal
}

// TiKV assigns tikv to component key in label
func (l Label) TiKV() Label {
	l.Component(TiKVLabelVal)
	return l
}

// IsTiKV returns whether label is a TiKV
func (l Label) IsTiKV() bool {
	return l[ComponentLabelKey] == TiKVLabelVal
}

// Selector gets labels.Selector from label
func (l Label) Selector() (labels.Selector, error) {
	return metav1.LabelSelectorAsSelector(l.LabelSelector())
}

// LabelSelector gets LabelSelector from label
func (l Label) LabelSelector() *metav1.LabelSelector {
	return &metav1.LabelSelector{MatchLabels: l}
}

// Labels converts label to map[string]string
func (l Label) Labels() map[string]string {
	return l
}

// String converts label to a string
func (l Label) String() string {
	var arr []string

	for k, v := range l {
		arr = append(arr, fmt.Sprintf("%s=%s", k, v))
	}

	return strings.Join(arr, ",")
}

// IsManagedByTiKVOperator returns whether label is a Managed by tikv-operator
func (l Label) IsManagedByTiKVOperator() bool {
	return l[ManagedByLabelKey] == TiKVOperator
}
