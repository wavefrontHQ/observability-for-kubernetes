// Copyright 2020 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package kstate

import (
	"fmt"
	"sync"
	"time"

	"k8s.io/api/autoscaling/v2beta2"

	log "github.com/sirupsen/logrus"

	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/leadership"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

const (
	daemonSets               = "daemonsets"
	deployments              = "deployments"
	replicaSets              = "replicasets"
	replicationControllers   = "replicationcontrollers"
	statefulSets             = "statefulsets"
	jobs                     = "jobs"
	cronJobs                 = "cronjobs"
	horizontalPodAutoscalers = "horizontalpodautoscalers"
	nodes                    = "nodes"
	nonRunningPods           = "pods"
)

var (
	doOnce    sync.Once
	singleton *lister
)

type lister struct {
	kubeClient kubernetes.Interface
	informers  map[string]cache.SharedInformer
	stopCh     chan struct{}
}

func newLister(kubeClient kubernetes.Interface) *lister {
	doOnce.Do(func() {
		singleton = &lister{
			kubeClient: kubeClient,
			informers:  buildInformers(kubeClient),
		}
		leadership.NewManager(singleton, "kstate", kubeClient).Start()
	})
	return singleton
}

func buildInformers(kubeClient kubernetes.Interface) map[string]cache.SharedInformer {
	m := make(map[string]cache.SharedInformer)
	m[daemonSets] = buildInformer(daemonSets, &appsv1.DaemonSet{}, kubeClient.AppsV1().RESTClient())
	m[deployments] = buildInformer(deployments, &appsv1.Deployment{}, kubeClient.AppsV1().RESTClient())
	m[statefulSets] = buildInformer(statefulSets, &appsv1.StatefulSet{}, kubeClient.AppsV1().RESTClient())
	m[replicaSets] = buildInformer(replicaSets, &appsv1.ReplicaSet{}, kubeClient.AppsV1().RESTClient())
	m[jobs] = buildInformer(jobs, &batchv1.Job{}, kubeClient.BatchV1().RESTClient())
	m[cronJobs] = buildInformer(cronJobs, &batchv1beta1.CronJob{}, kubeClient.BatchV1beta1().RESTClient())
	m[horizontalPodAutoscalers] = buildInformer(horizontalPodAutoscalers, &v2beta2.HorizontalPodAutoscaler{}, kubeClient.AutoscalingV2beta2().RESTClient())
	m[nodes] = buildInformer(nodes, &v1.Node{}, kubeClient.CoreV1().RESTClient())
	m[replicationControllers] = buildInformer(replicationControllers, &v1.ReplicationController{}, kubeClient.CoreV1().RESTClient())
	m[nonRunningPods] = buildInformerWithFieldsSelector(nonRunningPods, &v1.Pod{}, kubeClient.CoreV1().RESTClient(), fields.OneTermNotEqualSelector("status.phase", "Running"))
	return m
}

func buildInformer(resource string, resType runtime.Object, getter cache.Getter) cache.SharedInformer {
	return buildInformerWithFieldsSelector(resource, resType, getter, fields.Everything())
}

func buildInformerWithFieldsSelector(resource string, resType runtime.Object, getter cache.Getter, selector fields.Selector) cache.SharedInformer {
	lw := cache.NewListWatchFromClient(getter, resource, v1.NamespaceAll, selector)
	return cache.NewSharedInformer(lw, resType, time.Hour)
}

func (l *lister) List(resource string) ([]interface{}, error) {
	if informer, exists := l.informers[resource]; exists {
		return informer.GetStore().List(), nil
	} else {
		return nil, fmt.Errorf("unsupported resource type: %s", resource)
	}
}

func (l *lister) Resume() {
	log.Infof("starting kstate lister")
	l.stopCh = make(chan struct{})
	for k, informer := range l.informers {
		log.Debugf("starting %s informer", k)
		go informer.Run(l.stopCh)
	}
}

func (l *lister) Pause() {
	log.Infof("pausing kstate lister")
	if l.stopCh != nil {
		close(l.stopCh)
	}
}
