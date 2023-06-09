// Copyright 2020 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package kstate

import (
	"reflect"
	"time"

	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/util"

	corev1 "k8s.io/api/core/v1"

	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/wf"

	log "github.com/sirupsen/logrus"

	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/configuration"
)

func pointsForPVC(item interface{}, transforms configuration.Transforms) []wf.Metric {
	persistentVolumeClaim, ok := item.(*corev1.PersistentVolumeClaim)
	if !ok {
		log.Errorf("invalid type: %s", reflect.TypeOf(item).String())
		return nil
	}

	sharedTags := buildTags("pvc_name", persistentVolumeClaim.Name, persistentVolumeClaim.Namespace, transforms.Tags)
	copyLabels(persistentVolumeClaim.GetLabels(), sharedTags)

	now := time.Now().Unix()
	points := buildPVCRequestStorage(persistentVolumeClaim, transforms, now, sharedTags)
	points = append(points, buildPVCInfo(persistentVolumeClaim, transforms, now, sharedTags))
	points = append(points, buildPVCPhaseMetric(persistentVolumeClaim, transforms, now, sharedTags))
	points = append(points, buildPVCConditions(persistentVolumeClaim, transforms, now, sharedTags)...)
	points = append(points, buildPVCAccessModes(persistentVolumeClaim, transforms, now, sharedTags)...)

	return points
}

func buildPVCRequestStorage(claim *corev1.PersistentVolumeClaim, transforms configuration.Transforms, now int64, sharedTags map[string]string) []wf.Metric {
	tags := make(map[string]string, len(sharedTags))
	copyTags(sharedTags, tags)

	var resourceStorage = claim.Spec.Resources.Requests[corev1.ResourceStorage]
	return []wf.Metric{
		metricPoint(transforms.Prefix, "pvc.request.storage_bytes", float64(resourceStorage.Value()), now, transforms.Source, tags),
	}
}

func buildPVCInfo(claim *corev1.PersistentVolumeClaim, transforms configuration.Transforms, now int64, sharedTags map[string]string) wf.Metric {
	tags := make(map[string]string, len(sharedTags))
	copyTags(sharedTags, tags)

	tags["volume_name"] = claim.Spec.VolumeName
	// Use beta annotation first

	tags["storage_class_name"] = *claim.Spec.StorageClassName
	if class, found := claim.Annotations[corev1.BetaStorageClassAnnotation]; found {
		tags["storage_class_name"] = class
	}

	return metricPoint(transforms.Prefix, "pvc.info", 1.0, now, transforms.Source, tags)
}

func buildPVCPhaseMetric(claim *corev1.PersistentVolumeClaim, transforms configuration.Transforms, now int64, sharedTags map[string]string) wf.Metric {
	tags := make(map[string]string, len(sharedTags))
	copyTags(sharedTags, tags)

	tags["phase"] = string(claim.Status.Phase)
	phaseValue := util.ConvertPVCPhase(claim.Status.Phase)
	return metricPoint(transforms.Prefix, "pvc.status.phase", float64(phaseValue), now, transforms.Source, tags)
}

func buildPVCConditions(claim *corev1.PersistentVolumeClaim, transforms configuration.Transforms, now int64, sharedTags map[string]string) []wf.Metric {
	points := make([]wf.Metric, len(claim.Status.Conditions))
	for i, condition := range claim.Status.Conditions {
		tags := make(map[string]string, len(sharedTags))
		copyTags(sharedTags, tags)

		tags["status"] = string(condition.Status)
		tags["condition"] = string(condition.Type)

		points[i] = metricPoint(transforms.Prefix, "pvc.status.condition",
			util.ConditionStatusFloat64(condition.Status), now, transforms.Source, tags)
	}
	return points
}

func buildPVCAccessModes(claim *corev1.PersistentVolumeClaim, transforms configuration.Transforms, now int64, sharedTags map[string]string) []wf.Metric {
	points := make([]wf.Metric, len(claim.Spec.AccessModes))
	for i, accessMode := range claim.Spec.AccessModes {
		tags := make(map[string]string, len(sharedTags))
		copyTags(sharedTags, tags)

		tags["access_mode"] = string(accessMode)

		points[i] = metricPoint(transforms.Prefix, "pvc.access_mode",
			1.0, now, transforms.Source, tags)
	}
	return points
}
