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
	workloadStatus := getWorkloadStatusForJob(job.Spec.Completions, job.Status.Succeeded)
	points = append(points, buildWorkloadStatusMetric(transforms.Prefix, workloadStatus, now, transforms.Source, workloadTags))

	return points
}

// When a specified number of successful completions is reached, the Job is complete.
func getWorkloadStatusForJob(completions *int32, succeeded int32) float64 {
	// 1. Non-parallel Job (completion count of 1), the Job is complete as soon as its Pod terminates successfully.
	// 2. Fixed completion count Job (completion count of N > 1), is complete when there are N successful Pods.
	// 3. Parallel Jobs with a work queue (completions is nil), the success of any pod signals the success of all pods.
	if completions != nil && *completions == succeeded {
		return workloadReady
	} else if completions == nil && succeeded > 0 {
		return workloadReady
	}
	return workloadNotReady
}
