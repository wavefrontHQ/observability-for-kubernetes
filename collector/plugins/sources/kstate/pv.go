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

func pointsForPV(item interface{}, transforms configuration.Transforms) []wf.Metric {
	persistentVolume, ok := item.(*corev1.PersistentVolume)
	if !ok {
		log.Errorf("invalid type: %s", reflect.TypeOf(item).String())
		return nil
	}

	sharedTags := buildTags("PV_name", persistentVolume.Name, persistentVolume.Namespace, transforms.Tags)
	copyLabels(persistentVolume.GetLabels(), sharedTags)

	now := time.Now().Unix()
	points := buildPVCapacityBytes(persistentVolume, transforms, now, sharedTags)
	points = append(points, buildPVInfo(persistentVolume, transforms, now, sharedTags))
	points = append(points, buildPVPhase(persistentVolume, transforms, now, sharedTags))
	points = append(points, buildPVClaimRef(persistentVolume, transforms, now, sharedTags))

	return points
}

func buildPVPhase(persistentVolume *corev1.PersistentVolume, transforms configuration.Transforms, now int64, sharedTags map[string]string) wf.Metric {
	tags := make(map[string]string, len(sharedTags))
	copyTags(sharedTags, tags)

	tags["phase"] = string(persistentVolume.Status.Phase)
	phaseValue := util.ConvertPVPhase(persistentVolume.Status.Phase)
	return metricPoint(transforms.Prefix, "pv.status.phase", float64(phaseValue), now, transforms.Source, tags)
}

func buildPVClaimRef(persistentVolume *corev1.PersistentVolume, transforms configuration.Transforms, now int64, sharedTags map[string]string) wf.Metric {
	tags := make(map[string]string, len(sharedTags))
	copyTags(sharedTags, tags)

	claimRef := persistentVolume.Spec.ClaimRef

	tags["claimref_name"] = "-"
	tags["claimref_namespace"] = "-"
	if claimRef != nil {
		tags["claimref_name"] = claimRef.Name
		tags["claimref_namespace"] = claimRef.Namespace
	}
	return metricPoint(transforms.Prefix, "pv.claim_ref",
		1.0, now, transforms.Source, tags)
}

func buildPVCapacityBytes(persistentVolume *corev1.PersistentVolume, transforms configuration.Transforms, now int64, sharedTags map[string]string) []wf.Metric {
	tags := make(map[string]string, len(sharedTags))
	copyTags(sharedTags, tags)

	var capacity = persistentVolume.Spec.Capacity[corev1.ResourceStorage]
	return []wf.Metric{
		metricPoint(transforms.Prefix, "PV.request.storage_bytes", float64(capacity.Value()), now, transforms.Source, tags),
	}
}

func buildPVInfo(persistentVolume *corev1.PersistentVolume, transforms configuration.Transforms, now int64, sharedTags map[string]string) wf.Metric {
	tags := make(map[string]string, len(sharedTags))
	copyTags(sharedTags, tags)

	tags["volumename"] = claim.Spec.VolumeName
	// Use beta annotation first
	if class, found := claim.Annotations[corev1.BetaStorageClassAnnotation]; found {
		tags["storageclassname"] = class
	}
	tags["storageclassname"] = *claim.Spec.StorageClassName

	return metricPoint(transforms.Prefix, "PV.info", 1.0, now, transforms.Source, tags)
}
