// Copyright 2019 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
package events

import (
	"strings"

	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/util"
	v1 "k8s.io/api/core/v1"
	corev1listers "k8s.io/client-go/listers/core/v1"
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

type match func(event *v1.Event) bool
type eventMatcher struct {
	match       match
	category    string
	subcategory string
	podLister   corev1listers.PodLister
}

type eventAnnotater struct {
	workloadCache util.WorkloadCache
	clusterName   string
	clusterUUID   string
	eventMatchers []eventMatcher
}

func (em *eventMatcher) matches(event *v1.Event) bool {
	return em.match(event)
}

func NewEventAnnotator(workloadCache util.WorkloadCache, clusterName, clusterUUID string) eventAnnotater {
	matchers := make([]eventMatcher, 5)

	imageMatcher := eventMatcher{
		match: func(event *v1.Event) bool {
			return event.Reason == "Failed" && strings.Contains(strings.ToLower(event.Message), "image")
		},
		category:    Creation,
		subcategory: ImagePullBackOff,
		podLister:   nil,
	}
	matchers = append(matchers, imageMatcher)

	backoffMatcher := eventMatcher{
		match: func(event *v1.Event) bool {
			return event.Reason == "BackOff" && strings.Contains(event.Message, "Back-off restarting")
		},
		category:    Runtime,
		subcategory: CrashLoopBackOff,
		podLister:   nil,
	}

	matchers = append(matchers, backoffMatcher)

	failedMount := eventMatcher{
		match: func(event *v1.Event) bool {
			return event.Reason == "FailedMount"
		},
		category:    Creation,
		subcategory: FailedMount,
		podLister:   nil,
	}

	unhealthy := eventMatcher{
		match: func(event *v1.Event) bool {
			return event.Reason == "Unhealthy"
		},
		category:    Runtime,
		subcategory: Unhealthy,
		podLister:   nil,
	}

	matchers = append(matchers, unhealthy)

	scheduling := eventMatcher{
		match: func(event *v1.Event) bool {
			return event.Reason == "FailedScheduling" && strings.Contains(strings.ToLower(event.Message), "insufficient")
		},
		category:    Scheduling,
		subcategory: InsufficientResources,
		podLister:   nil,
	}

	matchers = append(matchers, scheduling)

	failedCreate := eventMatcher{
		match: func(event *v1.Event) bool {
			return event.Reason == "FailedCreate"
		},
		category:    Scheduling,
		subcategory: InsufficientResources,
		podLister:   nil,
	}

	matchers = append(matchers, failedCreate)

	return eventAnnotater{
		workloadCache: workloadCache,
		clusterName:   clusterName,
		clusterUUID:   clusterUUID,
		eventMatchers: matchers,
	}
}

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
