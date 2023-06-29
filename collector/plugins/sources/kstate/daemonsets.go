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

	workloadTags := buildWorkloadTags("daemonset", daemonset.Name, daemonset.Namespace, transforms.Tags)
	workloadPoint := buildWorkloadStatusMetric(transforms.Prefix, desiredScheduled, ready, now, transforms.Source, workloadTags)

	return []wf.Metric{
		metricPoint(transforms.Prefix, "daemonset.current_scheduled", currentScheduled, now, transforms.Source, tags),
		metricPoint(transforms.Prefix, "daemonset.desired_scheduled", desiredScheduled, now, transforms.Source, tags),
		metricPoint(transforms.Prefix, "daemonset.misscheduled", misScheduled, now, transforms.Source, tags),
		metricPoint(transforms.Prefix, "daemonset.ready", ready, now, transforms.Source, tags),
		workloadPoint,
	}
}
