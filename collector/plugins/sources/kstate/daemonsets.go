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

func pointsForDaemonSet(item interface{}, transforms configuration.Transforms) []wf.Metric {
	daemonset, ok := item.(*appsv1.DaemonSet)
	if !ok {
		log.Errorf("invalid type: %s", reflect.TypeOf(item).String())
		return nil
	}

	tags := buildTags("daemonset", daemonset.Name, daemonset.Namespace, transforms.Tags)
	now := time.Now().Unix()
	currentScheduled := float64(daemonset.Status.CurrentNumberScheduled)
	desiredScheduled := float64(daemonset.Status.DesiredNumberScheduled)
	misScheduled := float64(daemonset.Status.NumberMisscheduled)
	ready := float64(daemonset.Status.NumberReady)

	desired := daemonset.Status.DesiredNumberScheduled
	available := daemonset.Status.NumberAvailable
	workloadStatus := getWorkloadStatus(desired, available)
	reason, message := getWorkloadReasonAndMessageForDaemonSet(workloadStatus, daemonset)
	workloadTags := buildWorkloadTags(workloadKindDaemonSet, daemonset.Name, daemonset.Namespace, desired, available, reason, message, transforms.Tags)

	return []wf.Metric{
		metricPoint(transforms.Prefix, "daemonset.current_scheduled", currentScheduled, now, transforms.Source, tags),
		metricPoint(transforms.Prefix, "daemonset.desired_scheduled", desiredScheduled, now, transforms.Source, tags),
		metricPoint(transforms.Prefix, "daemonset.misscheduled", misScheduled, now, transforms.Source, tags),
		metricPoint(transforms.Prefix, "daemonset.ready", ready, now, transforms.Source, tags),
		buildWorkloadStatusMetric(transforms.Prefix, workloadStatus, now, transforms.Source, workloadTags),
	}
}

func getWorkloadReasonAndMessageForDaemonSet(status float64, daemonset *appsv1.DaemonSet) (reason, message string) {
	if status == workloadReady {
		return "", ""
	}
	// TODO: return failure reason and message for DaemonSet
	reason = "MinimumNodesUnavailable"
	message = "DaemonSet does not have minimum number of nodes that should be running the daemon pod."
	return reason, message
}
