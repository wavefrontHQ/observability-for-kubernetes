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
	Job        = "Job"
)

// subcategory
const (
	ImagePullBackOff      = "ImagePullBackOff"
	CrashLoopBackOff      = "CrashLoopBackOff"
	FailedMount           = "FailedMount"
	Unhealthy             = "Unhealthy"
	InsufficientResources = "InsufficientResources"
	FailedCreate          = "FailedCreate"
	OOMKilled             = "OOMKilled"
	ProvisioningFailed    = "ProvisioningFailed"
	BackoffLimitExceeded  = "BackoffLimitExceeded"
	NodeNotReady          = "NodeNotReady"
)

type match func(event *v1.Event) bool

type eventMatcher struct {
	match       match
	category    string
	subcategory string
}

func (em *eventMatcher) categorize(event *v1.Event) bool {
	if em.match(event) {
		event.ObjectMeta.Annotations["aria/category"] = em.getCategory(event)
		event.ObjectMeta.Annotations["aria/subcategory"] = em.getSubcategory(event)
		event.ObjectMeta.Annotations["important"] = "true"
		return true
	} else {
		event.ObjectMeta.Annotations["important"] = "false"
		return false
	}
}

func (em *eventMatcher) getCategory(event *v1.Event) string {
	if len(em.category) > 0 {
		return em.category
	}
	return event.InvolvedObject.Kind
}

func (em *eventMatcher) getSubcategory(event *v1.Event) string {
	if len(em.subcategory) > 0 {
		return em.subcategory
	}
	return event.InvolvedObject.Kind
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
		if matcher.categorize(event) {
			break
		}
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
	annotator.eventMatchers = append(annotator.eventMatchers, annotator.jobMatchers()...)
	annotator.eventMatchers = append(annotator.eventMatchers, annotator.defaultMatcher())

	return annotator
}

func (ea *EventAnnotator) schedulingMatchers() []eventMatcher {
	return []eventMatcher{
		{
			match: func(event *v1.Event) bool {
				return event.Type == v1.EventTypeWarning && event.Reason == "FailedScheduling" && strings.Contains(strings.ToLower(event.Message), "insufficient")
			},
			category:    Scheduling,
			subcategory: InsufficientResources,
		},
		{
			match: func(event *v1.Event) bool {
				return event.Reason == "NodeNotReady" && event.InvolvedObject.Kind == "Pod" && strings.Contains(strings.ToLower(event.Message), "not ready")
			},
			category:    Scheduling,
			subcategory: NodeNotReady,
		},
	}
}

func (ea *EventAnnotator) creationMatchers() []eventMatcher {
	return []eventMatcher{
		{
			match: func(event *v1.Event) bool {
				return event.Type == v1.EventTypeWarning && event.Reason == "Failed" && strings.Contains(strings.ToLower(event.Message), "image")
			},
			category:    Creation,
			subcategory: ImagePullBackOff,
		},
		{
			match: func(event *v1.Event) bool {
				return event.Type == v1.EventTypeNormal && event.Reason == "BackOff" && strings.Contains(strings.ToLower(event.Message), "back-off pulling image")
			},
			category:    Creation,
			subcategory: ImagePullBackOff,
		},
		{
			match: func(event *v1.Event) bool {
				return event.Type == v1.EventTypeWarning && event.Reason == "FailedMount"
			},
			category:    Creation,
			subcategory: FailedMount,
		},
	}
}

func (ea *EventAnnotator) runtimeMatchers() []eventMatcher {
	return []eventMatcher{
		{
			match: func(event *v1.Event) bool {
				return event.Type == v1.EventTypeWarning && event.Reason == "BackOff" && strings.Contains(strings.ToLower(event.Message), "back-off restarting")
			},
			category:    Runtime,
			subcategory: CrashLoopBackOff,
		},
		{
			match: func(event *v1.Event) bool {
				return event.Type == v1.EventTypeWarning && event.Reason == "Unhealthy"
			},
			category:    Runtime,
			subcategory: Unhealthy,
		},
		{
			match: func(event *v1.Event) bool {
				return event.Type == v1.EventTypeWarning && event.Reason == "OOMKilling"
			},
			category:    Runtime,
			subcategory: OOMKilled,
		},
	}
}

func (ea *EventAnnotator) storageMatchers() []eventMatcher {
	return []eventMatcher{
		{
			match: func(event *v1.Event) bool {
				return event.Type == v1.EventTypeWarning && event.Reason == "FailedCreate" && strings.Contains(strings.ToLower(event.Message), "volumemounts")
			},
			category:    Storage,
			subcategory: FailedCreate,
		},
		{
			match: func(event *v1.Event) bool {
				return event.Type == v1.EventTypeWarning && event.Reason == "ProvisioningFailed"
			},
			category:    Storage,
			subcategory: ProvisioningFailed,
		},
	}
}

func (ea *EventAnnotator) jobMatchers() []eventMatcher {
	return []eventMatcher{
		{
			match: func(event *v1.Event) bool {
				return event.InvolvedObject.Kind == "Job" && event.Reason == "BackoffLimitExceeded"
			},
			category:    Job,
			subcategory: BackoffLimitExceeded,
		},
	}
}

func (ea *EventAnnotator) defaultMatcher() eventMatcher {
	return eventMatcher{
		match: func(event *v1.Event) bool {
			return event.Type == v1.EventTypeWarning
		},
	}
}
