// Copyright 2020 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package kstate

import (
	"reflect"
	"time"

	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"

	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/configuration"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/metrics"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/util"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/wf"
)

func pointsForNonRunningPods(workloadCache util.WorkloadCache) func(item interface{}, transforms configuration.Transforms) []wf.Metric {
	return func(item interface{}, transforms configuration.Transforms) []wf.Metric {
		pod, ok := item.(*v1.Pod)
		if !ok {
			log.Errorf("invalid type: %s", reflect.TypeOf(item).String())
			return nil
		}

		workloadName, workloadKind := workloadCache.GetWorkloadForPod(pod)
		sharedTags := make(map[string]string, len(pod.GetLabels())+2)
		sharedTags[workloadNameTag] = workloadName
		sharedTags[workloadKindTag] = workloadKind

		copyLabels(pod.GetLabels(), sharedTags)
		now := time.Now().Unix()

		points := buildPodPhaseMetrics(pod, transforms, sharedTags, now)
		if util.IsStuckInTerminating(pod) {
			points = append(points, buildPodTerminatingMetrics(pod, sharedTags, transforms, now)...)
		}

		points = append(points, buildContainerStatusMetrics(pod, sharedTags, transforms, now)...)

		// emit workload.status metric for single pods with no owner references
		if !util.HasOwnerReference(pod.OwnerReferences) {
			workloadStatus := getWorkloadStatusForNonRunningPod(pod.Status.ContainerStatuses)
			reason, message := getWorkloadReasonAndMessageForNonRunningPod(workloadStatus, pod)
			// non-running pods has a default available value of 0 because they are not running
			workloadTags := buildWorkloadTags(workloadKindPod, pod.Name, pod.Namespace, 1, 0, reason, message, transforms.Tags)
			points = append(points, buildWorkloadStatusMetric(transforms.Prefix, workloadStatus, now, transforms.Source, workloadTags))
		}
		return points
	}
}

func truncateMessage(message string) string {
	maxPointTagLength := 255 - len("=") - len("message")
	if len(message) >= maxPointTagLength {
		return message[0:maxPointTagLength]
	}
	return message
}

func getWorkloadStatusForNonRunningPod(containerStatuses []v1.ContainerStatus) float64 {
	if len(containerStatuses) == 0 {
		return workloadNotReady
	}

	for _, containerStatus := range containerStatuses {
		containerStateInfo := util.NewContainerStateInfo(containerStatus.State)
		if containerStateInfo.Value == util.CONTAINER_STATE_TERMINATED && containerStateInfo.Reason == "Completed" {
			return workloadReady
		}
	}
	return workloadNotReady
}

func getWorkloadReasonAndMessageForNonRunningPod(status float64, pod *v1.Pod) (reason, message string) {
	if status == workloadReady {
		return "", ""
	}

	podPhase := util.ConvertPodPhase(pod.Status.Phase)
	if len(pod.Status.ContainerStatuses) > 0 {
		for _, containerStatus := range pod.Status.ContainerStatuses {
			switch podPhase {
			case util.POD_PHASE_PENDING:
				if !containerStatus.Ready && containerStatus.State.Waiting != nil {
					reason = containerStatus.State.Waiting.Reason
					message = truncateMessage(containerStatus.State.Waiting.Message)
					return reason, message
				}
			case util.POD_PHASE_FAILED:
				if !containerStatus.Ready && containerStatus.State.Terminated != nil {
					reason = containerStatus.State.Terminated.Reason
					message = truncateMessage(containerStatus.State.Terminated.Message)
					return reason, message
				}
			}
		}
	} else if len(pod.Status.Conditions) > 0 {
		for _, condition := range pod.Status.Conditions {
			switch podPhase {
			case util.POD_PHASE_PENDING, util.POD_PHASE_FAILED:
				if util.PodConditionIsUnchedulable(condition) {
					reason, message = condition.Reason, truncateMessage(condition.Message)
					if pod.DeletionTimestamp != nil {
						// TODO: Add failure message for pods stuck in terminating
						reason = "Terminating"
					}
					return reason, message
				}
			}
		}
	}

	return reason, message
}

func buildPodPhaseMetrics(pod *v1.Pod, transforms configuration.Transforms, sharedTags map[string]string, now int64) []wf.Metric {
	tags := buildTags("pod_name", pod.Name, pod.Namespace, transforms.Tags)
	tags[metrics.LabelMetricSetType.Key] = metrics.MetricSetTypePod
	tags[metrics.LabelPodId.Key] = string(pod.UID)
	tags["phase"] = string(pod.Status.Phase)

	phaseValue := util.ConvertPodPhase(pod.Status.Phase)
	if phaseValue == util.POD_PHASE_PENDING {
		for _, condition := range pod.Status.Conditions {
			if condition.Type == v1.PodScheduled && condition.Status == "False" {
				tags[metrics.LabelNodename.Key] = "none"
				tags["reason"] = condition.Reason
				tags["message"] = truncateMessage(condition.Message)
			} else if condition.Type == v1.ContainersReady && condition.Status == "False" {
				tags["reason"] = condition.Reason
				tags["message"] = truncateMessage(condition.Message)
			}
		}
	}

	if phaseValue == util.POD_PHASE_FAILED {
		for _, condition := range pod.Status.Conditions {
			if condition.Type == v1.PodReady {
				tags["reason"] = condition.Reason
				tags["message"] = truncateMessage(condition.Message)
			}
		}
	}

	nodeName := pod.Spec.NodeName
	if len(nodeName) > 0 {
		sharedTags[metrics.LabelNodename.Key] = nodeName
	}
	copyTags(sharedTags, tags)
	points := []wf.Metric{
		metricPoint(transforms.Prefix, "pod.status.phase", float64(phaseValue), now, transforms.Source, tags),
	}
	return points
}

func buildPodTerminatingMetrics(pod *v1.Pod, sharedTags map[string]string, transforms configuration.Transforms, now int64) []wf.Metric {
	tags := buildTags("pod_name", pod.Name, pod.Namespace, transforms.Tags)
	tags[metrics.LabelMetricSetType.Key] = metrics.MetricSetTypePod
	tags[metrics.LabelPodId.Key] = string(pod.UID)
	tags["DeletionTimestamp"] = pod.DeletionTimestamp.Format(time.RFC3339)
	tags["reason"] = "Terminating"

	for _, condition := range pod.Status.Conditions {
		if condition.Type == v1.PodScheduled && condition.Status == "False" {
			tags[metrics.LabelNodename.Key] = "none"
		}
	}

	nodeName := pod.Spec.NodeName
	if len(nodeName) > 0 {
		sharedTags[metrics.LabelNodename.Key] = nodeName
	}

	copyTags(sharedTags, tags)
	points := []wf.Metric{
		metricPoint(transforms.Prefix, "pod.terminating", 1, now, transforms.Source, tags),
	}
	return points
}

func buildContainerStatusMetrics(pod *v1.Pod, sharedTags map[string]string, transforms configuration.Transforms, now int64) []wf.Metric {
	statuses := pod.Status.ContainerStatuses
	if len(statuses) == 0 {
		return []wf.Metric{}
	}

	points := make([]wf.Metric, len(statuses))
	for i, status := range statuses {
		containerStateInfo := util.NewContainerStateInfo(status.State)
		tags := buildTags("pod_name", pod.Name, pod.Namespace, transforms.Tags)
		tags[metrics.LabelMetricSetType.Key] = metrics.MetricSetTypePodContainer
		tags[metrics.LabelContainerName.Key] = status.Name
		tags[metrics.LabelContainerBaseImage.Key] = status.Image

		copyTags(sharedTags, tags)
		containerStateInfo.AddMetricTags(tags)

		points[i] = metricPoint(transforms.Prefix, "pod_container.status", float64(containerStateInfo.Value), now, transforms.Source, tags)
	}
	return points
}
