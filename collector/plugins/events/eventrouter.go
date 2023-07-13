// Copyright 2019 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
package events

import (
	"reflect"
	"strings"
	"sync/atomic"
	"time"

	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/configuration"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/events"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/leadership"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/util"
	"github.com/wavefronthq/observability-for-kubernetes/collector/plugins/sinks"
	"github.com/wavefronthq/wavefront-sdk-go/event"

	gometrics "github.com/rcrowley/go-metrics"
	log "github.com/sirupsen/logrus"

	v1 "k8s.io/api/core/v1"

	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
)

var Log = log.WithField("system", "events")
var leadershipName = "eventRouter"
var filteredEvents = gometrics.GetOrRegisterCounter("events.filtered", gometrics.DefaultRegistry)
var receivedEvents = gometrics.GetOrRegisterCounter("events.received", gometrics.DefaultRegistry)
var sentEvents = gometrics.GetOrRegisterCounter("events.sent", gometrics.DefaultRegistry)

type EventRouter struct {
	kubeClient        kubernetes.Interface
	eLister           corelisters.EventLister
	eListerSynced     cache.InformerSynced
	sink              sinks.Sink
	sharedInformers   informers.SharedInformerFactory
	stop              chan struct{}
	scrapeCluster     bool
	leadershipManager *leadership.Manager
	filters           eventFilter
	clusterName       string
	clusterUUID       string
	workloadCache     util.WorkloadCache
	informerSynced    atomic.Bool
}

func NewEventRouter(clientset kubernetes.Interface, cfg configuration.EventsConfig, sink sinks.Sink, scrapeCluster bool, workloadCache util.WorkloadCache) *EventRouter {
	sharedInformers := informers.NewSharedInformerFactory(clientset, time.Minute)
	eventsInformer := sharedInformers.Core().V1().Events()

	er := &EventRouter{
		kubeClient:      clientset,
		sink:            sink,
		scrapeCluster:   scrapeCluster,
		sharedInformers: sharedInformers,
		filters:         newEventFilter(cfg.Filters),
		clusterName:     cfg.ClusterName,
		clusterUUID:     cfg.ClusterUUID,
		workloadCache:   workloadCache,
	}

	eventsInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			er.addEvent(obj, !er.informerSynced.Load())
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			if reflect.DeepEqual(oldObj, newObj) {
				return
			}
			er.addEvent(newObj, false)
		},
	})
	er.eLister = eventsInformer.Lister()
	er.eListerSynced = eventsInformer.Informer().HasSynced
	er.leadershipManager = leadership.NewManager(er, leadershipName, clientset)

	return er
}

func (er *EventRouter) Start() {
	if er.scrapeCluster {
		er.leadershipManager.Start()
	}
}

func (er *EventRouter) Resume() {
	er.stop = make(chan struct{})

	Log.Infof("Starting EventRouter")

	go func() { er.sharedInformers.Start(er.stop) }()

	// here is where we kick the caches into gear
	if !cache.WaitForCacheSync(er.stop, er.eListerSynced) {
		log.Error("timed out waiting for caches to sync")
		return
	}
	er.informerSynced.Store(true)
	<-er.stop

	Log.Infof("Shutting down EventRouter")
}

func (er *EventRouter) Pause() {
	if er.stop != nil {
		close(er.stop)
	}
}

func (er *EventRouter) Stop() {
	if er.scrapeCluster {
		er.leadershipManager.Stop()
	}
	er.Pause()
}

// addEvent is called when an event is created, or during the initial list
func (er *EventRouter) addEvent(obj interface{}, isInInitialList bool) {
	if isInInitialList {
		return
	}

	e, ok := obj.(*v1.Event)
	if !ok {
		return // prevent unlikely panic
	}

	e = e.DeepCopy()

	if e.ObjectMeta.Annotations == nil {
		e.ObjectMeta.Annotations = map[string]string{}
	}
	e.ObjectMeta.Annotations["aria/cluster-name"] = er.clusterName
	e.ObjectMeta.Annotations["aria/cluster-uuid"] = er.clusterUUID

	if e.InvolvedObject.Kind == "Pod" {
		workloadName, workloadKind, nodeName := er.workloadCache.GetWorkloadForPodName(e.InvolvedObject.Name, e.InvolvedObject.Namespace)
		e.ObjectMeta.Annotations["aria/workload-name"] = workloadName
		e.ObjectMeta.Annotations["aria/workload-kind"] = workloadKind
		if len(nodeName) > 0 {
			e.ObjectMeta.Annotations["aria/node-name"] = nodeName
		}
	}

	ns := e.InvolvedObject.Namespace
	if len(ns) == 0 {
		ns = "default"
	}

	tags := map[string]string{
		"namespace_name": ns,
		"kind":           e.InvolvedObject.Kind,
		"reason":         e.Reason,
		"component":      e.Source.Component,
		"type":           e.Type,
	}

	resourceName := e.InvolvedObject.Name
	if resourceName != "" {
		if strings.ToLower(e.InvolvedObject.Kind) == "pod" {
			tags["pod_name"] = resourceName
		} else {
			tags["resource_name"] = resourceName
		}
	}

	receivedEvents.Inc(1)
	if !er.filters.matches(tags) {
		if log.IsLevelEnabled(log.TraceLevel) {
			Log.WithField("event", e.Message).Trace("Dropping event")
		}
		filteredEvents.Inc(1)
		return
	}
	sentEvents.Inc(1)

	er.sink.ExportEvent(newEvent(
		e.Message,
		e.LastTimestamp.Time,
		e.Source.Host,
		tags,
		*e,
	))
}

func newEvent(message string, ts time.Time, host string, tags map[string]string, coreV1Event v1.Event, options ...event.Option) *events.Event {
	// convert tags to annotations
	for k, v := range tags {
		options = append(options, event.Annotate(k, v))
	}

	return &events.Event{
		Message: message,
		Ts:      ts,
		Host:    host,
		Options: options,
		Event:   coreV1Event,
	}
}
