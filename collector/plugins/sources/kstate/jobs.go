// Copyright 2020 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package kstate

import (
	"reflect"
	"time"

	log "github.com/sirupsen/logrus"
	batchv1 "k8s.io/api/batch/v1"

	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/configuration"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/util"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/wf"
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

	var workloadKind, workloadName, workloadFailedReason string

	if len(job.OwnerReferences) == 0 {
		workloadKind = workloadKindJob
		workloadName = job.Name
	} else {
		workloadKind = job.OwnerReferences[0].Kind
		workloadName = job.OwnerReferences[0].Name
	}
	workloadStatus := getWorkloadStatusForJob(job.Status.Conditions)
	if failed != 0 {
		workloadFailedReason = job.Status.Conditions[0].Reason
	} else {
		workloadFailedReason = ""
	}
	completionsCount := int32(1)
	if job.Spec.Completions != nil {
		completionsCount = *job.Spec.Completions
	}
	workloadTags := buildWorkloadTags(workloadKind, workloadName, job.Namespace, completionsCount, int32(succeeded), workloadFailedReason, transforms.Tags)
	points = append(points, buildWorkloadStatusMetric(transforms.Prefix, workloadStatus, now, transforms.Source, workloadTags))

	return points
}

func getWorkloadStatusForJob(jobConditions []batchv1.JobCondition) float64 {
	if len(jobConditions) == 0 {
		return workloadNotReady
	}

	for _, jobCondition := range jobConditions {
		// When a specified number of successful completions is reached, the Job is complete.
		if jobCondition.Type == batchv1.JobComplete && jobCondition.Status == corev1.ConditionTrue {
			return workloadReady
		}
	}
	return workloadNotReady
}
