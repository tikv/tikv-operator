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
	"reflect"

	"github.com/tikv/tikv-operator/pkg/apis/tikv/v1alpha1"
	corev1 "k8s.io/api/core/v1"

	apivalidation "k8s.io/apimachinery/pkg/api/validation"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// ValidateTikvCluster validates a TikvCluster, it performs basic validation for all TikvClusters despite it is legacy
// or not
func ValidateTikvCluster(tc *v1alpha1.TikvCluster) field.ErrorList {
	allErrs := field.ErrorList{}
	// validate metadata
	fldPath := field.NewPath("metadata")
	// validate metadata/annotations
	allErrs = append(allErrs, validateAnnotations(tc.ObjectMeta.Annotations, fldPath.Child("annotations"))...)
	// validate spec
	allErrs = append(allErrs, validateTiKVClusterSpec(&tc.Spec, field.NewPath("spec"))...)
	return allErrs
}

func validateAnnotations(anns map[string]string, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, apivalidation.ValidateAnnotations(anns, fldPath)...)
	return allErrs
}

func validateTiKVClusterSpec(spec *v1alpha1.TikvClusterSpec, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, validatePDSpec(&spec.PD, fldPath.Child("pd"))...)
	allErrs = append(allErrs, validateTiKVSpec(&spec.TiKV, fldPath.Child("tikv"))...)
	return allErrs
}

func validatePDSpec(spec *v1alpha1.PDSpec, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, validateComponentSpec(&spec.ComponentSpec, fldPath)...)
	allErrs = append(allErrs, validateRequestsStorage(spec.ResourceRequirements.Requests, fldPath)...)
	return allErrs
}

func validateTiKVSpec(spec *v1alpha1.TiKVSpec, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, validateComponentSpec(&spec.ComponentSpec, fldPath)...)
	allErrs = append(allErrs, validateRequestsStorage(spec.ResourceRequirements.Requests, fldPath)...)
	return allErrs
}

func validateComponentSpec(spec *v1alpha1.ComponentSpec, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	// TODO validate other fields
	allErrs = append(allErrs, validateEnv(spec.Env, fldPath.Child("env"))...)
	return allErrs
}

// validateRequestsStorage validates resources requests storage
func validateRequestsStorage(requests corev1.ResourceList, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	if _, ok := requests[corev1.ResourceStorage]; !ok {
		allErrs = append(allErrs, field.Required(fldPath.Child("requests.storage").Key((string(corev1.ResourceStorage))), "storage request must not be empty"))
	}
	return allErrs
}

// validateEnv validates env vars
func validateEnv(vars []corev1.EnvVar, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	for i, ev := range vars {
		idxPath := fldPath.Index(i)
		if len(ev.Name) == 0 {
			allErrs = append(allErrs, field.Required(idxPath.Child("name"), ""))
		} else {
			for _, msg := range validation.IsEnvVarName(ev.Name) {
				allErrs = append(allErrs, field.Invalid(idxPath.Child("name"), ev.Name, msg))
			}
		}
		allErrs = append(allErrs, validateEnvVarValueFrom(ev, idxPath.Child("valueFrom"))...)
	}
	return allErrs
}

func validateEnvVarValueFrom(ev corev1.EnvVar, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if ev.ValueFrom == nil {
		return allErrs
	}

	numSources := 0

	if ev.ValueFrom.FieldRef != nil {
		numSources++
		allErrs = append(allErrs, field.Invalid(fldPath.Child("fieldRef"), "", "fieldRef is not supported"))
	}
	if ev.ValueFrom.ResourceFieldRef != nil {
		numSources++
		allErrs = append(allErrs, field.Invalid(fldPath.Child("resourceFieldRef"), "", "resourceFieldRef is not supported"))
	}
	if ev.ValueFrom.ConfigMapKeyRef != nil {
		numSources++
		allErrs = append(allErrs, validateConfigMapKeySelector(ev.ValueFrom.ConfigMapKeyRef, fldPath.Child("configMapKeyRef"))...)
	}
	if ev.ValueFrom.SecretKeyRef != nil {
		numSources++
		allErrs = append(allErrs, validateSecretKeySelector(ev.ValueFrom.SecretKeyRef, fldPath.Child("secretKeyRef"))...)
	}

	if numSources == 0 {
		allErrs = append(allErrs, field.Invalid(fldPath, "", "must specify one of: `configMapKeyRef` or `secretKeyRef`"))
	} else if len(ev.Value) != 0 {
		if numSources != 0 {
			allErrs = append(allErrs, field.Invalid(fldPath, "", "may not be specified when `value` is not empty"))
		}
	} else if numSources > 1 {
		allErrs = append(allErrs, field.Invalid(fldPath, "", "may not have more than one field specified at a time"))
	}

	return allErrs
}

func validateConfigMapKeySelector(s *corev1.ConfigMapKeySelector, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	for _, msg := range apivalidation.NameIsDNSSubdomain(s.Name, false) {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("name"), s.Name, msg))
	}
	if len(s.Key) == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child("key"), ""))
	} else {
		for _, msg := range validation.IsConfigMapKey(s.Key) {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("key"), s.Key, msg))
		}
	}

	return allErrs
}

func validateSecretKeySelector(s *corev1.SecretKeySelector, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	for _, msg := range apivalidation.NameIsDNSSubdomain(s.Name, false) {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("name"), s.Name, msg))
	}
	if len(s.Key) == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child("key"), ""))
	} else {
		for _, msg := range validation.IsConfigMapKey(s.Key) {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("key"), s.Key, msg))
		}
	}

	return allErrs
}

// ValidateCreateTikvCLuster validates a newly created TikvCluster
func ValidateCreateTikvCluster(tc *v1alpha1.TikvCluster) field.ErrorList {
	allErrs := field.ErrorList{}
	// basic validation
	allErrs = append(allErrs, ValidateTikvCluster(tc)...)
	allErrs = append(allErrs, validateNewTikvClusterSpec(&tc.Spec, field.NewPath("spec"))...)
	return allErrs
}

// ValidateUpdateTikvCluster validates a new TikvCluster against an existing TikvCluster to be updated
func ValidateUpdateTikvCluster(old, tc *v1alpha1.TikvCluster) field.ErrorList {

	allErrs := field.ErrorList{}
	// basic validation
	allErrs = append(allErrs, ValidateTikvCluster(tc)...)
	allErrs = append(allErrs, validateUpdatePDConfig(old.Spec.PD.Config, tc.Spec.PD.Config, field.NewPath("spec.pd.config"))...)
	allErrs = append(allErrs, disallowUsingLegacyAPIInNewCluster(old, tc)...)

	return allErrs
}

// For now we limit some validations only in Create phase to keep backward compatibility
func validateNewTikvClusterSpec(spec *v1alpha1.TikvClusterSpec, path *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	if spec.Version == "" {
		allErrs = append(allErrs, field.Invalid(path.Child("version"), spec.Version, "version must not be empty"))
	}
	if spec.PD.BaseImage == "" {
		allErrs = append(allErrs, field.Invalid(path.Child("pd.baseImage"), spec.PD.BaseImage, "baseImage of PD must not be empty"))
	}
	if spec.TiKV.BaseImage == "" {
		allErrs = append(allErrs, field.Invalid(path.Child("tikv.baseImage"), spec.TiKV.BaseImage, "baseImage of TiKV must not be empty"))
	}
	if spec.TiKV.Image != "" {
		allErrs = append(allErrs, field.Invalid(path.Child("tikv.image"), spec.TiKV.Image, "image has been deprecated, use baseImage instead"))
	}
	if spec.PD.Image != "" {
		allErrs = append(allErrs, field.Invalid(path.Child("pd.image"), spec.PD.Image, "image has been deprecated, use baseImage instead"))
	}
	return allErrs
}

// disallowUsingLegacyAPIInNewCluster checks if user use the legacy API in newly create cluster during update
// TODO(aylei): this could be removed after we enable validateTikvCluster() in update, which is more strict
func disallowUsingLegacyAPIInNewCluster(old, tc *v1alpha1.TikvCluster) field.ErrorList {
	allErrs := field.ErrorList{}
	path := field.NewPath("spec")
	if old.Spec.Version != "" && tc.Spec.Version == "" {
		allErrs = append(allErrs, field.Invalid(path.Child("version"), tc.Spec.Version, "version must not be empty"))
	}
	if old.Spec.PD.BaseImage != "" && tc.Spec.PD.BaseImage == "" {
		allErrs = append(allErrs, field.Invalid(path.Child("pd.baseImage"), tc.Spec.PD.BaseImage, "baseImage of PD must not be empty"))
	}
	if old.Spec.TiKV.BaseImage != "" && tc.Spec.TiKV.BaseImage == "" {
		allErrs = append(allErrs, field.Invalid(path.Child("tikv.baseImage"), tc.Spec.TiKV.BaseImage, "baseImage of TiKV must not be empty"))
	}
	if old.Spec.TiKV.Config != nil && tc.Spec.TiKV.Config == nil {
		allErrs = append(allErrs, field.Invalid(path.Child("tikv.config"), tc.Spec.TiKV.Config, "TiKV.config must not be nil"))
	}
	if old.Spec.PD.Config != nil && tc.Spec.PD.Config == nil {
		allErrs = append(allErrs, field.Invalid(path.Child("pd.config"), tc.Spec.PD.Config, "PD.config must not be nil"))
	}
	return allErrs
}

func validateUpdatePDConfig(old, conf *v1alpha1.PDConfig, path *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	// for newly created cluster, both old and new are non-nil, guaranteed by validation
	if old == nil || conf == nil {
		return allErrs
	}

	if conf.Security != nil && len(conf.Security.CertAllowedCN) > 1 {
		allErrs = append(allErrs, field.Invalid(path.Child("security.cert-allowed-cn"), conf.Security.CertAllowedCN,
			"Only one CN is currently supported"))
	}

	if !reflect.DeepEqual(old.Schedule, conf.Schedule) {
		allErrs = append(allErrs, field.Invalid(path.Child("schedule"), conf.Schedule,
			"PD Schedule Config is immutable through CRD, please modify with pd-ctl instead."))
	}
	if !reflect.DeepEqual(old.Replication, conf.Replication) {
		allErrs = append(allErrs, field.Invalid(path.Child("replication"), conf.Replication,
			"PD Replication Config is immutable through CRD, please modify with pd-ctl instead."))
	}
	return allErrs
}
