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
	OOMKilled             = "OOMKilled"
)

type match func(event *v1.Event) bool

type eventMatcher struct {
	match       match
	category    string
	subcategory string
	podLister   corev1listers.PodLister
}

func (em *eventMatcher) matches(event *v1.Event) bool {
	return em.match(event)
}

type EventAnnotator struct {
	workloadCache util.WorkloadCache
	clusterName   string
	clusterUUID   string
	eventMatchers []eventMatcher
}

func (ea *EventAnnotator) annotate(event *v1.Event) {
	if event.ObjectMeta.Annotations == nil {
		event.ObjectMeta.Annotations = map[string]string{}
	}
	event.ObjectMeta.Annotations["aria/cluster-name"] = ea.clusterName
	event.ObjectMeta.Annotations["aria/cluster-uuid"] = ea.clusterUUID

	if event.InvolvedObject.Kind == "Pod" {
		workloadName, workloadKind, nodeName := ea.workloadCache.GetWorkloadForPodName(event.InvolvedObject.Name, event.InvolvedObject.Namespace)
		event.ObjectMeta.Annotations["aria/workload-name"] = workloadName
		event.ObjectMeta.Annotations["aria/workload-kind"] = workloadKind
		if len(nodeName) > 0 {
			event.ObjectMeta.Annotations["aria/node-name"] = nodeName
		}
	}
	ea.categorize(event)
}

func (ea *EventAnnotator) categorize(event *v1.Event) {
	for _, matcher := range ea.eventMatchers {
		if matcher.matches(event) {
			event.ObjectMeta.Annotations["aria/category"] = matcher.category
			event.ObjectMeta.Annotations["aria/subcategory"] = matcher.subcategory
			break
		}
	}

	if _, found := event.ObjectMeta.Annotations["aria/category"]; !found {
		event.ObjectMeta.Annotations["aria/category"] = event.InvolvedObject.Kind
		event.ObjectMeta.Annotations["aria/subcategory"] = event.InvolvedObject.Kind
	}
}

func NewEventAnnotator(workloadCache util.WorkloadCache, clusterName, clusterUUID string) *EventAnnotator {
	annotator := &EventAnnotator{
		workloadCache: workloadCache,
		clusterName:   clusterName,
		clusterUUID:   clusterUUID,
		eventMatchers: make([]eventMatcher, 0),
	}

	annotator.eventMatchers = append(annotator.eventMatchers, annotator.schedulingMatchers()...)
	annotator.eventMatchers = append(annotator.eventMatchers, annotator.creationMatchers()...)
	annotator.eventMatchers = append(annotator.eventMatchers, annotator.runtimeMatchers()...)
	annotator.eventMatchers = append(annotator.eventMatchers, annotator.storageMatchers()...)

	return annotator
}

func (ea *EventAnnotator) schedulingMatchers() []eventMatcher {
	return []eventMatcher{
		{
			match: func(event *v1.Event) bool {
				return event.Reason == "FailedScheduling" && strings.Contains(strings.ToLower(event.Message), "insufficient")
			},
			category:    Scheduling,
			subcategory: InsufficientResources,
			podLister:   nil,
		},
	}
}

func (ea *EventAnnotator) creationMatchers() []eventMatcher {
	return []eventMatcher{
		{
			match: func(event *v1.Event) bool {
				return event.Reason == "Failed" && strings.Contains(strings.ToLower(event.Message), "image")
			},
			category:    Creation,
			subcategory: ImagePullBackOff,
			podLister:   nil,
		},
		{
			match: func(event *v1.Event) bool {
				return event.Reason == "FailedMount"
			},
			category:    Creation,
			subcategory: FailedMount,
			podLister:   nil,
		},
	}
}

func (ea *EventAnnotator) runtimeMatchers() []eventMatcher {
	return []eventMatcher{
		{
			match: func(event *v1.Event) bool {
				return event.Reason == "BackOff" && strings.Contains(strings.ToLower(event.Message), "back-off restarting")
			},
			category:    Runtime,
			subcategory: CrashLoopBackOff,
			podLister:   nil,
		},
		{
			match: func(event *v1.Event) bool {
				return event.Reason == "Unhealthy"
			},
			category:    Runtime,
			subcategory: Unhealthy,
			podLister:   nil,
		},
		{
			match: func(event *v1.Event) bool {
				return event.Reason == "Killing" && strings.Contains(strings.ToLower(event.Message), "stopping")
			},
			category:    Runtime,
			subcategory: Terminating,
			podLister:   nil,
		},
		{
			match: func(event *v1.Event) bool {
				return event.Reason == "OOMKilling"
			},
			category:    Runtime,
			subcategory: OOMKilled,
			podLister:   nil,
		},
	}
}

func (ea *EventAnnotator) storageMatchers() []eventMatcher {
	return []eventMatcher{
		{
			match: func(event *v1.Event) bool {
				return event.Reason == "FailedCreate"
			},
			category:    Storage,
			subcategory: FailedCreate,
			podLister:   nil,
		},
	}
}
