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

func pointsForReplicaSet(item interface{}, transforms configuration.Transforms) []wf.Metric {
	rs, ok := item.(*appsv1.ReplicaSet)
	if !ok {
		log.Errorf("invalid type: %s", reflect.TypeOf(item).String())
		return nil
	}

	tags := buildTags("replicaset", rs.Name, rs.Namespace, transforms.Tags)
	now := time.Now().Unix()
	desired := floatValOrDefault(rs.Spec.Replicas, 1.0)
	available := float64(rs.Status.AvailableReplicas)
	ready := float64(rs.Status.ReadyReplicas)

	points := []wf.Metric{
		metricPoint(transforms.Prefix, "replicaset.desired_replicas", desired, now, transforms.Source, tags),
		metricPoint(transforms.Prefix, "replicaset.available_replicas", available, now, transforms.Source, tags),
		metricPoint(transforms.Prefix, "replicaset.ready_replicas", ready, now, transforms.Source, tags),
	}
	if rs.OwnerReferences == nil || len(rs.OwnerReferences) == 0 {
		workloadStatus := getWorkloadStatusForReplicaSet(desired, rs.Status.AvailableReplicas)
		workloadTags := buildWorkloadTags(workloadKindReplicaSet, rs.Name, rs.Namespace, transforms.Tags)
		points = append(points, buildWorkloadStatusMetric(transforms.Prefix, workloadStatus, now, transforms.Source, workloadTags))
	}

	return points
}

func getWorkloadStatusForReplicaSet(desiredReplicas float64, availableReplicas int32) float64 {
	// number of available replicas for this replica set match the number of desired replicas
	if availableReplicas == int32(desiredReplicas) {
		return workloadReady
	}
	return workloadNotReady
}
