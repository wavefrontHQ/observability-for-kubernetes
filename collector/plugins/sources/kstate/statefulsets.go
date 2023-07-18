// Copyright 2020 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package kstate

import (
	"reflect"
	"time"

	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/wf"

	log "github.com/sirupsen/logrus"

	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/configuration"
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

	workloadTags := buildWorkloadTags(workloadKindStatefulSet, statefulset.Name, statefulset.Namespace, transforms.Tags)
	workloadPoint := buildWorkloadStatusMetric(transforms.Prefix, desired, ready, now, transforms.Source, workloadTags)

	return []wf.Metric{
		metricPoint(transforms.Prefix, "statefulset.desired_replicas", desired, now, transforms.Source, tags),
		metricPoint(transforms.Prefix, "statefulset.current_replicas", current, now, transforms.Source, tags),
		metricPoint(transforms.Prefix, "statefulset.ready_replicas", ready, now, transforms.Source, tags),
		metricPoint(transforms.Prefix, "statefulset.updated_replicas", updated, now, transforms.Source, tags),
		workloadPoint,
	}
}
