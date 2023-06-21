package util

import (
	"time"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	appsv1listers "k8s.io/client-go/listers/apps/v1"
	batchv1listers "k8s.io/client-go/listers/batch/v1"
	corev1listers "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
)

type WorkloadCache interface {
	GetWorkloadForPodName(podName, ns string) (name, kind string)
	GetWorkloadForPod(pod *corev1.Pod) (string, string)
}

type workloadCache struct {
	podLister corev1listers.PodLister
	rsLister  appsv1listers.ReplicaSetLister
	jobLister batchv1listers.JobLister
}

func NewWorkloadCache(kubeClient kubernetes.Interface) (WorkloadCache, error) {
	singletonPodLister, err := GetPodLister(kubeClient)
	if err != nil {
		return nil, err
	}
	replicaSetLister, err := getReplicaSetLister(kubeClient)
	if err != nil {
		return nil, err
	}
	jobLister, err := getJobLister(kubeClient)
	if err != nil {
		return nil, err
	}
	return &workloadCache{
		podLister: singletonPodLister,
		rsLister:  replicaSetLister,
		jobLister: jobLister,
	}, nil
}

func getReplicaSetLister(kubeClient kubernetes.Interface) (appsv1listers.ReplicaSetLister, error) {
	lw := cache.NewListWatchFromClient(kubeClient.AppsV1().RESTClient(), "replicaSets", corev1.NamespaceAll, fields.Everything())
	store := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
	replicaSetLister := appsv1listers.NewReplicaSetLister(store)
	go cache.NewReflector(lw, &appsv1.ReplicaSet{}, store, time.Hour).Run(NeverStop)
	return replicaSetLister, nil
}

func getJobLister(kubeClient kubernetes.Interface) (batchv1listers.JobLister, error) {
	lw := cache.NewListWatchFromClient(kubeClient.BatchV1().RESTClient(), "jobs", corev1.NamespaceAll, fields.Everything())
	store := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
	jobLister := batchv1listers.NewJobLister(store)
	go cache.NewReflector(lw, &batchv1.Job{}, store, time.Hour).Run(NeverStop)
	return jobLister, nil
}

func (wc workloadCache) GetWorkloadForPodName(podName, ns string) (name, kind string) {
	pod, err := wc.podLister.Pods(ns).Get(podName)
	if err != nil {
		return "", ""
	}
	return wc.GetWorkloadForPod(pod)
}

func (wc workloadCache) GetWorkloadForPod(pod *corev1.Pod) (string, string) {
	if len(pod.OwnerReferences) == 0 {
		return pod.Name, pod.Kind
	}

	podOwner := pod.OwnerReferences[0]
	var parentOwners []metav1.OwnerReference

	switch podOwner.Kind {
	case "ReplicaSet":
		rs, _ := wc.rsLister.ReplicaSets(pod.Namespace).Get(podOwner.Name)
		parentOwners = rs.OwnerReferences
	case "Job":
		job, _ := wc.jobLister.Jobs(pod.Namespace).Get(podOwner.Name)
		parentOwners = job.OwnerReferences
	}

	if len(parentOwners) != 0 {
		// the ReplicaSet or Job has a parent (likely a Deployment or CronJob), so return that
		return parentOwners[0].Name, parentOwners[0].Kind
	} else {
		// Otherwise return the owner of the Pod
		return podOwner.Name, podOwner.Kind
	}
}
