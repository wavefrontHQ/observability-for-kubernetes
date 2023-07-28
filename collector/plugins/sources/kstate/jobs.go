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

	workloadName, workloadKind := getWorkloadNameAndKindForJob(job)
	workloadStatus := getWorkloadStatusForJob(job.Status.Conditions)
	reason, message := getWorkloadReasonAndMessageForJob(workloadStatus, job)
	desired := intValOrDefault(job.Spec.Completions, 1)
	workloadTags := buildWorkloadTags(workloadKind, workloadName, job.Namespace, desired, int32(succeeded), reason, message, transforms.Tags)
	points = append(points, buildWorkloadStatusMetric(transforms.Prefix, workloadStatus, now, transforms.Source, workloadTags))

	return points
}

func getWorkloadNameAndKindForJob(job *batchv1.Job) (name, kind string) {
	if util.HasOwnerReference(job.OwnerReferences) {
		return job.OwnerReferences[0].Name, job.OwnerReferences[0].Kind
	}
	return job.Name, workloadKindJob
}

func getWorkloadStatusForJob(jobConditions []batchv1.JobCondition) float64 {
	if len(jobConditions) == 0 {
		return workloadNotReady
	}

	for _, jobCondition := range jobConditions {
		if util.JobConditionIsFailed(jobCondition) {
			return workloadNotReady
		}
	}
	return workloadReady
}

func getWorkloadReasonAndMessageForJob(status float64, job *batchv1.Job) (reason, message string) {
	if status == workloadReady {
		return "", ""
	}

	if job.Status.Failed > 0 {
		for _, condition := range job.Status.Conditions {
			if util.JobConditionIsFailed(condition) {
				return condition.Reason, condition.Message
			}
		}
	}
	return reason, message
}
