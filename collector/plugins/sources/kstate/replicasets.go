// Copyright 2020 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package kstate

import (
	"reflect"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/configuration"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/util"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/wf"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
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

	// emit workload.status metric for replica sets with no owner references
	if len(rs.OwnerReferences) == 0 {
		workloadStatus := getWorkloadStatus(int32(desired), rs.Status.AvailableReplicas)
		workloadTags := buildWorkloadTags(workloadKindReplicaSet, rs.Name, rs.Namespace, int32(desired), rs.Status.AvailableReplicas, "", transforms.Tags)
		points = append(points, buildWorkloadStatusMetric(transforms.Prefix, workloadStatus, now, transforms.Source, workloadTags))
	}

	return points
}
