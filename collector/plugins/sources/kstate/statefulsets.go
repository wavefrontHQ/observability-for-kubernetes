// Copyright 2020 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package kstate

import (
	"reflect"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/configuration"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/wf"
	appsv1 "k8s.io/api/apps/v1"
)

func pointsForStatefulSet(item interface{}, transforms configuration.Transforms) []wf.Metric {
	statefulset, ok := item.(*appsv1.StatefulSet)
	if !ok {
		log.Errorf("invalid type: %s", reflect.TypeOf(item).String())
		return nil
	}

	tags := buildTags("statefulset", statefulset.Name, statefulset.Namespace, transforms.Tags)
	now := time.Now().Unix()
	desired := floatValOrDefault(statefulset.Spec.Replicas, 1.0)
	ready := float64(statefulset.Status.ReadyReplicas)
	current := float64(statefulset.Status.CurrentReplicas)
	updated := float64(statefulset.Status.UpdatedReplicas)

	workloadStatus := getWorkloadStatus(int32(desired), int32(ready))
	reason, message := getWorkloadReasonAndMessageForStatefulSet(workloadStatus, statefulset)
	workloadTags := buildWorkloadTags(workloadKindStatefulSet, statefulset.Name, statefulset.Namespace, int32(desired), int32(ready), reason, message, transforms.Tags)

	return []wf.Metric{
		metricPoint(transforms.Prefix, "statefulset.desired_replicas", desired, now, transforms.Source, tags),
		metricPoint(transforms.Prefix, "statefulset.current_replicas", current, now, transforms.Source, tags),
		metricPoint(transforms.Prefix, "statefulset.ready_replicas", ready, now, transforms.Source, tags),
		metricPoint(transforms.Prefix, "statefulset.updated_replicas", updated, now, transforms.Source, tags),
		buildWorkloadStatusMetric(transforms.Prefix, workloadStatus, now, transforms.Source, workloadTags),
	}
}

func getWorkloadReasonAndMessageForStatefulSet(status float64, statefulset *appsv1.StatefulSet) (reason, message string) {
	if status == workloadReady {
		return "", ""
	}
	// TODO: return failure reason and message for StatefulSet
	reason = "MinimumReplicasUnavailable"
	message = "StatefulSet does not have minimum availability."
	return reason, message
}
