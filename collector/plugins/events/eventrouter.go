// Copyright 2019 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
package events

import (
	"context"
	"strings"
	"time"

	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/configuration"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/events"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/leadership"
	"github.com/wavefronthq/observability-for-kubernetes/collector/plugins/sinks/wavefront"
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
	sink              wavefront.WavefrontSink
	sharedInformers   informers.SharedInformerFactory
	stop              chan struct{}
	scrapeCluster     bool
	leadershipManager *leadership.Manager
	filters           eventFilter
}

func NewEventRouter(clientset kubernetes.Interface, cfg configuration.EventsConfig, sink wavefront.WavefrontSink, scrapeCluster bool) *EventRouter {
	sharedInformers := informers.NewSharedInformerFactory(clientset, time.Minute)
	eventsInformer := sharedInformers.Core().V1().Events()

	er := &EventRouter{
		kubeClient:      clientset,
		sink:            sink,
		scrapeCluster:   scrapeCluster,
		sharedInformers: sharedInformers,
		filters:         newEventFilter(cfg.Filters),
	}

	eventsInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: er.addEvent,
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
func (er *EventRouter) addEvent(obj interface{}) {
	e, ok := obj.(*v1.Event)
	if !ok {
		return // prevent unlikely panic
	}

	// ignore events older than a minute to prevent surge on startup
	if e.LastTimestamp.Time.Before(time.Now().Add(-1 * time.Minute)) {
		Log.WithField("event", e.Message).Trace("Ignoring older event")
		return
	}

	ns := e.InvolvedObject.Namespace
	if len(ns) == 0 {
		ns = "default"
	}

	workloadName, workloadKind, err := er.workloadFromEvent(e.InvolvedObject)
	if e.InvolvedObject.Kind == "Pod" { // TODO: can we make this a constant?
		workloadName, workloadKind = util.GetWorkloadForPod(er.kubeClient, e.InvolvedObject.Name, ns)
	}
	if err != nil {
		log.Info(err)
	}

	tags := map[string]string{
		"namespace_name": ns,
		"kind":           e.InvolvedObject.Kind,
		"reason":         e.Reason,
		"component":      e.Source.Component,
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

	eType := e.Type
	if len(eType) == 0 {
		eType = "Normal"
	}

	er.sink.ExportEvent(newEvent(
		e.Message,
		e.LastTimestamp.Time,
		e.Source.Host,
		workloadName,
		workloadKind,
		tags,
		*e,
		event.Type(eType),
	))
}

func newEvent(message string, ts time.Time, host string, workloadName string, workloadKind string, tags map[string]string, coreV1Event v1.Event, options ...event.Option) *events.Event {
	// convert tags to annotations
	for k, v := range tags {
		options = append(options, event.Annotate(k, v))
	}

	labels := make(map[string]string)
	labels["workloadName"] = workloadName
	labels["workloadKind"] = workloadKind

	return &events.Event{
		Message: message,
		Ts:      ts,
		Host:    host,
		Options: options,
		Event:   coreV1Event,
		Labels:  labels,
	}
}

func (er *EventRouter) workloadFromEvent(o v1.ObjectReference) (workload string, workloadKind string, err error) {
	name := o.Name
	ns := o.Namespace
	if len(ns) == 0 {
		ns = "default"
	}
	kind := o.Kind

	result := ""
	var owner []metav1.OwnerReference

	// TODO: error handling
	for result == "" {
		log.Infof("NAME:%s", name)
		switch kind {
		case "Pod":
			obj, err := er.kubeClient.CoreV1().Pods(ns).Get(context.Background(), name, metav1.GetOptions{})
			if err != nil {
				log.Infof("Object:%v /// Error:%s", obj.OwnerReferences, err)
			}
			owner = obj.GetOwnerReferences()
		case "ReplicaSet":
			obj, err := er.kubeClient.AppsV1().ReplicaSets(ns).Get(context.Background(), name, metav1.GetOptions{})
			if err != nil {
				log.Infof("Object:%v /// Error:%s", obj.OwnerReferences, err)
			}
			owner = obj.GetOwnerReferences()
		case "Job":
			// TODO: Handle CronJob and Job rec
			obj, err := er.kubeClient.BatchV1().Jobs(ns).Get(context.Background(), name, metav1.GetOptions{})
			if err != nil {
				log.Infof("Error: %s", err)
			} else {
				log.Infof("Object: %v", obj.OwnerReferences)
			}
			owner = obj.GetOwnerReferences()
		case "DaemonSet", "Deployment", "StatefulSet":
			return name, kind, nil
		default:
			log.Infof("Unknown object type: %v", kind)
			// TODO: maybe not nil
			return name, kind, nil
		}

		if len(owner) == 0 {
			log.Info("DONE with our recursion wohoo")
			log.Infof("Found workload with name: %s", name)
			result = name
		} else {
			name = owner[0].Name
			kind = owner[0].Kind
		}
	}

	return result, kind, nil
}
