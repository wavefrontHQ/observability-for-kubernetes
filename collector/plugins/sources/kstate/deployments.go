// Copyright 2020 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package kstate

import (
	"reflect"
	"time"

	log "github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/configuration"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/wf"
)

func pointsForDeployment(item interface{}, transforms configuration.Transforms) []wf.Metric {
	deployment, ok := item.(*appsv1.Deployment)
	if !ok {
		log.Errorf("invalid type: %s", reflect.TypeOf(item).String())
		return nil
	}

	tags := buildTags("deployment", deployment.Name, deployment.Namespace, transforms.Tags)
	now := time.Now().Unix()
	desired := floatValOrDefault(deployment.Spec.Replicas, 1.0)
	available := float64(deployment.Status.AvailableReplicas)
	ready := float64(deployment.Status.ReadyReplicas)

	workloadStatus := getWorkloadStatus(int32(desired), int32(available))
	reason, message := getWorkloadReasonAndMessageForDeployment(workloadStatus, deployment)
	workloadTags := buildWorkloadTags(workloadKindDeployment, deployment.Name, deployment.Namespace, int32(desired), int32(available), reason, message, transforms.Tags)

	return []wf.Metric{
		metricPoint(transforms.Prefix, "deployment.desired_replicas", desired, now, transforms.Source, tags),
		metricPoint(transforms.Prefix, "deployment.available_replicas", available, now, transforms.Source, tags),
		metricPoint(transforms.Prefix, "deployment.ready_replicas", ready, now, transforms.Source, tags),
		buildWorkloadStatusMetric(transforms.Prefix, workloadStatus, now, transforms.Source, workloadTags),
	}
}

func getWorkloadReasonAndMessageForDeployment(status float64, deployment *appsv1.Deployment) (reason, message string) {
	if status == workloadReady {
		return "", ""
	}

	for _, condition := range deployment.Status.Conditions {
		if condition.Type == appsv1.DeploymentAvailable && condition.Status == corev1.ConditionFalse {
			return condition.Reason, truncateMessage(condition.Message)
		}
	}
	return reason, message
}
