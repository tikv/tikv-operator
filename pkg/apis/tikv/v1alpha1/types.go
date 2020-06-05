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

package v1alpha1

import (
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	// TiKVStateUp represents status of Up of TiKV
	TiKVStateUp string = "Up"
	// TiKVStateDown represents status of Down of TiKV
	TiKVStateDown string = "Down"
	// TiKVStateOffline represents status of Offline of TiKV
	TiKVStateOffline string = "Offline"
	// TiKVStateTombstone represents status of Tombstone of TiKV
	TiKVStateTombstone string = "Tombstone"
)

// MemberType represents member type
type MemberType string

const (
	// PDMemberType is pd container type
	PDMemberType MemberType = "pd"
	// TiKVMemberType is tikv container type
	TiKVMemberType MemberType = "tikv"
)

func (p MemberType) String() string {
	return string(p)
}

// MemberPhase is the current state of member
type MemberPhase string

const (
	// NormalPhase represents normal state of TiDB cluster.
	NormalPhase MemberPhase = "Normal"
	// UpgradePhase represents the upgrade state of TiDB cluster.
	UpgradePhase MemberPhase = "Upgrade"
)

// ConfigUpdateStrategy represents the strategy to update configuration
type ConfigUpdateStrategy string

const (
	// ConfigUpdateStrategyInPlace update the configmap without changing the name
	ConfigUpdateStrategyInPlace ConfigUpdateStrategy = "InPlace"
	// ConfigUpdateStrategyRollingUpdate generate different configmap on configuration update and
	// try to rolling-update the pod controller (e.g. statefulset) to apply updates.
	ConfigUpdateStrategyRollingUpdate ConfigUpdateStrategy = "RollingUpdate"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// +k8s:openapi-gen=true
// TikvCluster is the control script's spec
type TikvCluster struct {
	metav1.TypeMeta `json:",inline"`
	// +k8s:openapi-gen=false
	metav1.ObjectMeta `json:"metadata"`

	// Spec defines the behavior of a tikv cluster
	Spec TikvClusterSpec `json:"spec"`

	// +k8s:openapi-gen=false
	// Most recently observed status of the tikv cluster
	Status TikvClusterStatus `json:"status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// +k8s:openapi-gen=true
// TikvClusterList is TikvCluster list
type TikvClusterList struct {
	metav1.TypeMeta `json:",inline"`
	// +k8s:openapi-gen=false
	metav1.ListMeta `json:"metadata"`

	Items []TikvCluster `json:"items"`
}

// +k8s:openapi-gen=true
// TikvClusterSpec describes the attributes that a user creates on a tikv cluster
type TikvClusterSpec struct {
	// Discovery spec
	Discovery DiscoverySpec `json:"discovery,omitempty"`

	// PD cluster spec
	PD PDSpec `json:"pd"`

	// TiKV cluster spec
	TiKV TiKVSpec `json:"tikv"`

	// Indicates that the tikv cluster is paused and will not be processed by
	// the controller.
	// +optional
	Paused bool `json:"paused,omitempty"`

	// Cluster version
	// +optional
	Version string `json:"version"`

	// SchedulerName of TiKV cluster Pods
	SchedulerName string `json:"schedulerName,omitempty"`

	// ImagePullPolicy of TiDB cluster Pods
	// +kubebuilder:default=IfNotPresent
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`

	// ConfigUpdateStrategy determines how the configuration change is applied to the cluster.
	// UpdateStrategyInPlace will update the ConfigMap of configuration in-place and an extra rolling-update of the
	// cluster component is needed to reload the configuration change.
	// UpdateStrategyRollingUpdate will create a new ConfigMap with the new configuration and rolling-update the
	// related components to use the new ConfigMap, that is, the new configuration will be applied automatically.
	// +kubebuilder:validation:Enum=InPlace,RollingUpdate
	// +kubebuilder:default=InPlacne
	ConfigUpdateStrategy ConfigUpdateStrategy `json:"configUpdateStrategy,omitempty"`

	// Whether Hostnetwork is enabled for TiDB cluster Pods
	// Optional: Defaults to false
	// +optional
	HostNetwork *bool `json:"hostNetwork,omitempty"`

	// Affinity of TiDB cluster Pods
	// +optional
	Affinity *corev1.Affinity `json:"affinity,omitempty"`

	// PriorityClassName of TiDB cluster Pods
	// Optional: Defaults to omitted
	// +optional
	PriorityClassName *string `json:"priorityClassName,omitempty"`

	// Base node selectors of TiDB cluster Pods, components may add or override selectors upon this respectively
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Base annotations of TiDB cluster Pods, components may add or override selectors upon this respectively
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// Base tolerations of TiDB cluster Pods, components may add more tolerations upon this respectively
	// +optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// Time zone of TiDB cluster Pods
	// Optional: Defaults to UTC
	// +optional
	Timezone string `json:"timezone,omitempty"`
}

// TikvClusterStatus represents the current status of a tikv cluster.
type TikvClusterStatus struct {
	ClusterID string     `json:"clusterID,omitempty"`
	PD        PDStatus   `json:"pd,omitempty"`
	TiKV      TiKVStatus `json:"tikv,omitempty"`
	// Represents the latest available observations of a tikv cluster's state.
	// +optional
	Conditions []TikvClusterCondition `json:"conditions,omitempty"`
}

// TikvClusterCondition describes the state of a tikv cluster at a certain point.
type TikvClusterCondition struct {
	// Type of the condition.
	Type TikvClusterConditionType `json:"type"`
	// Status of the condition, one of True, False, Unknown.
	Status corev1.ConditionStatus `json:"status"`
	// The last time this condition was updated.
	LastUpdateTime metav1.Time `json:"lastUpdateTime,omitempty"`
	// Last time the condition transitioned from one status to another.
	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	// The reason for the condition's last transition.
	// +optional
	Reason string `json:"reason,omitempty"`
	// A human readable message indicating details about the transition.
	// +optional
	Message string `json:"message,omitempty"`
}

// TikvClusterConditionType represents a tikv cluster condition value.
type TikvClusterConditionType string

const (
	// TikvClusterReady indicates that the tikv cluster is ready or not.
	// This is defined as:
	// - All statefulsets are up to date (currentRevision == updateRevision).
	// - All PD members are healthy.
	// - All TiDB pods are healthy.
	// - All TiKV stores are up.
	// - All TiFlash stores are up.
	TikvClusterReady TikvClusterConditionType = "Ready"
)

// +k8s:openapi-gen=true
// DiscoverySpec contains details of Discovery members
type DiscoverySpec struct {
	corev1.ResourceRequirements `json:",inline"`
}

// +k8s:openapi-gen=true
// PDSpec contains details of PD members
type PDSpec struct {
	ComponentSpec               `json:",inline"`
	corev1.ResourceRequirements `json:",inline"`

	// The desired ready replicas
	// +kubebuilder:validation:Minimum=1
	Replicas int32 `json:"replicas"`

	// TODO: remove optional after defaulting introduced
	// Base image of the component, image tag is now allowed during validation
	// +kubebuilder:default=pingcap/pd
	// +optional
	BaseImage string `json:"baseImage"`

	// Service defines a Kubernetes service of PD cluster.
	// Optional: Defaults to `.spec.services` in favor of backward compatibility
	// +optional
	Service *ServiceSpec `json:"service,omitempty"`

	// MaxFailoverCount limit the max replicas could be added in failover, 0 means no failover.
	// Optional: Defaults to 3
	// +kubebuilder:validation:Minimum=0
	// +optional
	MaxFailoverCount *int32 `json:"maxFailoverCount,omitempty"`

	// The storageClassName of the persistent volume for PD data storage.
	// Defaults to Kubernetes default storage class.
	// +optional
	StorageClassName *string `json:"storageClassName,omitempty"`

	// Config is the Configuration of pd-servers
	// +optional
	Config *PDConfig `json:"config,omitempty"`

	// TLSClientSecretName is the name of secret which stores tikv server client certificate
	// which used by Dashboard.
	// +optional
	TLSClientSecretName *string `json:"tlsClientSecretName,omitempty"`
}

// +k8s:openapi-gen=true
// TiKVSpec contains details of TiKV members
type TiKVSpec struct {
	ComponentSpec               `json:",inline"`
	corev1.ResourceRequirements `json:",inline"`

	// Specify a Service Account for tikv
	ServiceAccount string `json:"serviceAccount,omitempty"`

	// The desired ready replicas
	// +kubebuilder:validation:Minimum=1
	Replicas int32 `json:"replicas"`

	// TODO: remove optional after defaulting introduced
	// Base image of the component, image tag is now allowed during validation
	// +kubebuilder:default=pingcap/tikv
	// +optional
	BaseImage string `json:"baseImage"`

	// Whether create the TiKV container in privileged mode, it is highly discouraged to enable this in
	// critical environment.
	// Optional: defaults to false
	// +optional
	Privileged *bool `json:"privileged,omitempty"`

	// MaxFailoverCount limit the max replicas could be added in failover, 0 means no failover
	// Optional: Defaults to 3
	// +kubebuilder:validation:Minimum=0
	// +optional
	MaxFailoverCount *int32 `json:"maxFailoverCount,omitempty"`

	// The storageClassName of the persistent volume for TiKV data storage.
	// Defaults to Kubernetes default storage class.
	// +optional
	StorageClassName *string `json:"storageClassName,omitempty"`

	// Config is the Configuration of tikv-servers
	// +optional
	Config *TiKVConfig `json:"config,omitempty"`
}

// +k8s:openapi-gen=true
// ComponentSpec is the base spec of each component, the fields should always accessed by the Basic<Component>Spec() method to respect the cluster-level properties
type ComponentSpec struct {
	// Image of the component, override baseImage and version if present
	// Deprecated
	// +k8s:openapi-gen=false
	Image string `json:"image,omitempty"`

	// Version of the component. Override the cluster-level version if non-empty
	// Optional: Defaults to cluster-level setting
	// +optional
	Version *string `json:"version,omitempty"`

	// ImagePullPolicy of the component. Override the cluster-level imagePullPolicy if present
	// Optional: Defaults to cluster-level setting
	// +optional
	ImagePullPolicy *corev1.PullPolicy `json:"imagePullPolicy,omitempty"`

	// Whether Hostnetwork of the component is enabled. Override the cluster-level setting if present
	// Optional: Defaults to cluster-level setting
	// +optional
	HostNetwork *bool `json:"hostNetwork,omitempty"`

	// Affinity of the component. Override the cluster-level one if present
	// Optional: Defaults to cluster-level setting
	// +optional
	Affinity *corev1.Affinity `json:"affinity,omitempty"`

	// PriorityClassName of the component. Override the cluster-level one if present
	// Optional: Defaults to cluster-level setting
	// +optional
	PriorityClassName *string `json:"priorityClassName,omitempty"`

	// SchedulerName of the component. Override the cluster-level one if present
	// Optional: Defaults to cluster-level setting
	// +optional
	SchedulerName *string `json:"schedulerName,omitempty"`

	// NodeSelector of the component. Merged into the cluster-level nodeSelector if non-empty
	// Optional: Defaults to cluster-level setting
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Annotations of the component. Merged into the cluster-level annotations if non-empty
	// Optional: Defaults to cluster-level setting
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// Tolerations of the component. Override the cluster-level tolerations if non-empty
	// Optional: Defaults to cluster-level setting
	// +optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// PodSecurityContext of the component
	// +optional
	PodSecurityContext *corev1.PodSecurityContext `json:"podSecurityContext,omitempty"`

	// ConfigUpdateStrategy of the component. Override the cluster-level updateStrategy if present
	// Optional: Defaults to cluster-level setting
	// +optional
	ConfigUpdateStrategy *ConfigUpdateStrategy `json:"configUpdateStrategy,omitempty"`

	// List of environment variables to set in the container, like
	// v1.Container.Env.
	Env []corev1.EnvVar `json:"env,omitempty"`
}

// +k8s:openapi-gen=true
type ServiceSpec struct {
	// Type of the real kubernetes service
	Type corev1.ServiceType `json:"type,omitempty"`

	// Additional annotations of the kubernetes service object
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// LoadBalancerIP is the loadBalancerIP of service
	// Optional: Defaults to omitted
	// +optional
	LoadBalancerIP *string `json:"loadBalancerIP,omitempty"`

	// ClusterIP is the clusterIP of service
	// +optional
	ClusterIP *string `json:"clusterIP,omitempty"`

	// PortName is the name of service port
	// +optional
	PortName *string `json:"portName,omitempty"`
}

// PDStatus is PD status
type PDStatus struct {
	Synced          bool                       `json:"synced,omitempty"`
	Phase           MemberPhase                `json:"phase,omitempty"`
	StatefulSet     *apps.StatefulSetStatus    `json:"statefulSet,omitempty"`
	Members         map[string]PDMember        `json:"members,omitempty"`
	Leader          PDMember                   `json:"leader,omitempty"`
	FailureMembers  map[string]PDFailureMember `json:"failureMembers,omitempty"`
	UnjoinedMembers map[string]UnjoinedMember  `json:"unjoinedMembers,omitempty"`
	Image           string                     `json:"image,omitempty"`
}

// PDMember is PD member
type PDMember struct {
	Name string `json:"name"`
	// member id is actually a uint64, but apimachinery's json only treats numbers as int64/float64
	// so uint64 may overflow int64 and thus convert to float64
	ID        string `json:"id"`
	ClientURL string `json:"clientURL"`
	Health    bool   `json:"health"`
	// Last time the health transitioned from one to another.
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
}

// PDFailureMember is the pd failure member information
type PDFailureMember struct {
	PodName       string      `json:"podName,omitempty"`
	MemberID      string      `json:"memberID,omitempty"`
	PVCUID        types.UID   `json:"pvcUID,omitempty"`
	MemberDeleted bool        `json:"memberDeleted,omitempty"`
	CreatedAt     metav1.Time `json:"createdAt,omitempty"`
}

// UnjoinedMember is the pd unjoin cluster member information
type UnjoinedMember struct {
	PodName   string      `json:"podName,omitempty"`
	PVCUID    types.UID   `json:"pvcUID,omitempty"`
	CreatedAt metav1.Time `json:"createdAt,omitempty"`
}

// TiKVStatus is TiKV status
type TiKVStatus struct {
	Synced          bool                        `json:"synced,omitempty"`
	Phase           MemberPhase                 `json:"phase,omitempty"`
	StatefulSet     *apps.StatefulSetStatus     `json:"statefulSet,omitempty"`
	Stores          map[string]TiKVStore        `json:"stores,omitempty"`
	TombstoneStores map[string]TiKVStore        `json:"tombstoneStores,omitempty"`
	FailureStores   map[string]TiKVFailureStore `json:"failureStores,omitempty"`
	Image           string                      `json:"image,omitempty"`
}

// TiKVStores is either Up/Down/Offline/Tombstone
type TiKVStore struct {
	// store id is also uint64, due to the same reason as pd id, we store id as string
	ID                string      `json:"id"`
	PodName           string      `json:"podName"`
	IP                string      `json:"ip"`
	LeaderCount       int32       `json:"leaderCount"`
	State             string      `json:"state"`
	LastHeartbeatTime metav1.Time `json:"lastHeartbeatTime"`
	// Last time the health transitioned from one to another.
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
}

// TiKVFailureStore is the tikv failure store information
type TiKVFailureStore struct {
	PodName   string      `json:"podName,omitempty"`
	StoreID   string      `json:"storeID,omitempty"`
	CreatedAt metav1.Time `json:"createdAt,omitempty"`
}
