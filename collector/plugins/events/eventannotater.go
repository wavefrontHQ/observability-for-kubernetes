// Copyright 2019 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
package events

import (
	"strings"

	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/util"
	v1 "k8s.io/api/core/v1"
)

// category
const (
	Creation   = "Creation"
	Runtime    = "Runtime"
	Scheduling = "Scheduling"
	Storage    = "Storage"
)

// subcategory
const (
	ImagePullBackOff      = "ImagePullBackOff"
	CrashLoopBackOff      = "CrashLoopBackOff"
	FailedMount           = "FailedMount"
	Unhealthy             = "Unhealthy"
	InsufficientResources = "InsufficientResources"
	FailedCreate          = "FailedCreate"
	Terminating           = "Terminating"
)

func annotateEvent(event *v1.Event, workloadCache util.WorkloadCache, clusterName, clusterUUID string) {
	if event.ObjectMeta.Annotations == nil {
		event.ObjectMeta.Annotations = map[string]string{}
	}
	event.ObjectMeta.Annotations["aria/cluster-name"] = clusterName
	event.ObjectMeta.Annotations["aria/cluster-uuid"] = clusterUUID

	if event.InvolvedObject.Kind == "Pod" {
		workloadName, workloadKind, nodeName := workloadCache.GetWorkloadForPodName(event.InvolvedObject.Name, event.InvolvedObject.Namespace)
		event.ObjectMeta.Annotations["aria/workload-name"] = workloadName
		event.ObjectMeta.Annotations["aria/workload-kind"] = workloadKind
		if len(nodeName) > 0 {
			event.ObjectMeta.Annotations["aria/node-name"] = nodeName
		}
	}
	categorizeEvent(event)
}

func categorizeEvent(event *v1.Event) {
	if event.Reason == "Failed" && strings.Contains(strings.ToLower(event.Message), "image") {
		event.ObjectMeta.Annotations["aria/category"] = Creation
		event.ObjectMeta.Annotations["aria/subcategory"] = ImagePullBackOff
	} else if event.Reason == "BackOff" && strings.Contains(event.Message, "Back-off restarting") {
		event.ObjectMeta.Annotations["aria/category"] = Runtime
		event.ObjectMeta.Annotations["aria/subcategory"] = CrashLoopBackOff
	} else if event.Reason == "FailedMount" {
		event.ObjectMeta.Annotations["aria/category"] = Creation
		event.ObjectMeta.Annotations["aria/subcategory"] = FailedMount
	} else if event.Reason == "Unhealthy" {
		event.ObjectMeta.Annotations["aria/category"] = Runtime
		event.ObjectMeta.Annotations["aria/subcategory"] = Unhealthy
	} else if event.Reason == "FailedScheduling" && strings.Contains(strings.ToLower(event.Message), "insufficient") {
		event.ObjectMeta.Annotations["aria/category"] = Scheduling
		event.ObjectMeta.Annotations["aria/subcategory"] = InsufficientResources
	} else if event.Reason == "FailedCreate" {
		event.ObjectMeta.Annotations["aria/category"] = Storage
		event.ObjectMeta.Annotations["aria/subcategory"] = FailedCreate
	}
}
