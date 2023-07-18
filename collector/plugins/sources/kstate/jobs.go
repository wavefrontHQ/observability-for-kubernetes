// Copyright 2020 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package kstate

import (
	"reflect"
	"time"

	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/wf"

	log "github.com/sirupsen/logrus"

	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/configuration"
	batchv1 "k8s.io/api/batch/v1"
)

func pointsForJob(item interface{}, transforms configuration.Transforms) []wf.Metric {
	job, ok := item.(*batchv1.Job)
	if !ok {
		log.Errorf("invalid type: %s", reflect.TypeOf(item).String())
		return nil
	}

	tags := buildTags("job", job.Name, job.Namespace, transforms.Tags)
	now := time.Now().Unix()
	active := float64(job.Status.Active)
	failed := float64(job.Status.Failed)
	succeeded := float64(job.Status.Succeeded)
	completions := floatValOrDefault(job.Spec.Completions, -1.0)
	parallelism := floatValOrDefault(job.Spec.Parallelism, -1.0)

	points := []wf.Metric{
		metricPoint(transforms.Prefix, "job.active", active, now, transforms.Source, tags),
		metricPoint(transforms.Prefix, "job.failed", failed, now, transforms.Source, tags),
		metricPoint(transforms.Prefix, "job.succeeded", succeeded, now, transforms.Source, tags),
		metricPoint(transforms.Prefix, "job.completions", completions, now, transforms.Source, tags),
		metricPoint(transforms.Prefix, "job.parallelism", parallelism, now, transforms.Source, tags),
	}

	var workloadKind, workloadName string

	if job.OwnerReferences == nil || len(job.OwnerReferences) == 0 {
		workloadKind = workloadKindJob
		workloadName = job.Name
	} else {
		workloadKind = job.OwnerReferences[0].Kind
		workloadName = job.OwnerReferences[0].Name
	}
	workloadTags := buildWorkloadTags(workloadKind, workloadName, job.Namespace, transforms.Tags)
	status := workloadReady
	if job.Status.Failed > 0 {
		status = workloadNotReady
	}
	points = append(points, metricPoint(transforms.Prefix, workloadStatusMetric, status, now, transforms.Source, workloadTags))

	return points
}
