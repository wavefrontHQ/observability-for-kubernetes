// Copyright 2020 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package kstate

import (
	"reflect"
	"time"

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
	//tags := buildTags("pvc", persistentVolumeClaim.Name, persistentVolumeClaim.Namespace, transforms.Tags)
	now := time.Now().Unix()
	//var resourceStorage = persistentVolumeClaim.Spec.Resources.Requests[corev1.ResourceStorage]
	//rsValue := float64(resourceStorage.Value())
	points := buildPVCConditions(persistentVolumeClaim, transforms, now)
	//points = append(points, metricPoint(transforms.Prefix, "pvc.request.storage_bytes"), rsValue, now, transforms.Source, tags)
	//points = append(points, buildPVCPhase(persistentVolumeClaim, transforms, now)...)
	//points = append(points, buildPVCInfo(persistentVolumeClaim, transforms, now))

	return points
}


func buildPVCConditions(claim *corev1.PersistentVolumeClaim, transforms configuration.Transforms, ts int64) []wf.Metric {
	points := make([]wf.Metric, len(claim.Status.Conditions))
	for i, condition := range claim.Status.Conditions {
		tags := buildTags("pvc", claim.Name, claim.Namespace, transforms.Tags)
		copyLabels(claim.GetLabels(), tags)
		tags["status"] = string(condition.Status)
		tags["condition"] = string(condition.Type)

		log.Printf("pvc.Status: %+v", claim.Status)

		// add status and condition (condition.status and condition.type)
		points[i] = metricPoint(transforms.Prefix, "pvc.status.condition",
			ConditionStatusFloat64(condition.Status), ts, transforms.Source, tags)
	}
	return points
}

// TODO write tests
