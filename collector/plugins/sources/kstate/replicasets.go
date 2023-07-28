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
	replicaset, ok := item.(*appsv1.ReplicaSet)
	if !ok {
		log.Errorf("invalid type: %s", reflect.TypeOf(item).String())
		return nil
	}

	tags := buildTags("replicaset", replicaset.Name, replicaset.Namespace, transforms.Tags)
	now := time.Now().Unix()
	desired := floatValOrDefault(replicaset.Spec.Replicas, 1.0)
	available := float64(replicaset.Status.AvailableReplicas)
	ready := float64(replicaset.Status.ReadyReplicas)

	points := []wf.Metric{
		metricPoint(transforms.Prefix, "replicaset.desired_replicas", desired, now, transforms.Source, tags),
		metricPoint(transforms.Prefix, "replicaset.available_replicas", available, now, transforms.Source, tags),
		metricPoint(transforms.Prefix, "replicaset.ready_replicas", ready, now, transforms.Source, tags),
	}

	// emit workload.status metric for replica sets with no owner references
	if !util.HasOwnerReference(replicaset.OwnerReferences) {
		workloadStatus := getWorkloadStatus(int32(desired), int32(available))
		reason, message := getWorkloadReasonAndMessageForReplicaSet(workloadStatus, replicaset)
		workloadTags := buildWorkloadTags(workloadKindReplicaSet, replicaset.Name, replicaset.Namespace, int32(desired), int32(available), reason, message, transforms.Tags)
		points = append(points, buildWorkloadStatusMetric(transforms.Prefix, workloadStatus, now, transforms.Source, workloadTags))
	}

	return points
}

func getWorkloadReasonAndMessageForReplicaSet(status float64, replicaset *appsv1.ReplicaSet) (reason, message string) {
	if status == workloadReady {
		return "", ""
	}
	for _, condition := range replicaset.Status.Conditions {
		if condition.Type == appsv1.ReplicaSetReplicaFailure && condition.Status == corev1.ConditionTrue {
			return condition.Reason, truncateMessage(condition.Message)
		}
	}
	return reason, message
}
