package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	appsv1listers "k8s.io/client-go/listers/apps/v1"
	batchv1listers "k8s.io/client-go/listers/batch/v1"
	corev1listers "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
)

type stores struct {
	podStore cache.Indexer
	rsStore  cache.Indexer
	jobStore cache.Indexer
}

func workloadCacheWithFakeListers() (workloadCache, stores) {
	s := stores{
		podStore: cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}),
		rsStore:  cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}),
		jobStore: cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}),
	}
	return workloadCache{
		podLister: corev1listers.NewPodLister(s.podStore),
		rsLister:  appsv1listers.NewReplicaSetLister(s.rsStore),
		jobLister: batchv1listers.NewJobLister(s.jobStore),
	}, s
}

func TestGetPodWorkloadForPod(t *testing.T) {
	t.Run("Pod with no owner", func(t *testing.T) {
		wc, s := workloadCacheWithFakeListers()
		fakePod := createFakePod(s.podStore, nil)

		name, kind := wc.GetWorkloadForPod(fakePod)
		assert.Equal(t, fakePod.Name, name)
		assert.Equal(t, fakePod.Kind, kind)
	})

	t.Run("Pod with ReplicaSet owner", func(t *testing.T) {
		wc, s := workloadCacheWithFakeListers()
		fakeReplicaSet := createFakeReplicaSet(s.rsStore, nil)
		podOwner := metav1.OwnerReference{
			Kind: fakeReplicaSet.Kind,
			Name: fakeReplicaSet.Name,
		}
		fakePod := createFakePod(s.podStore, &podOwner)

		name, kind := wc.GetWorkloadForPod(fakePod)
		assert.Equal(t, fakeReplicaSet.Name, name)
		assert.Equal(t, fakeReplicaSet.Kind, kind)
	})

	t.Run("Pod with Deployment owner", func(t *testing.T) {
		wc, s := workloadCacheWithFakeListers()
		rsOwner := metav1.OwnerReference{Kind: "Deployment", Name: "a-deployment"}
		fakeReplicaSet := createFakeReplicaSet(s.rsStore, &rsOwner)
		podOwner := metav1.OwnerReference{
			Kind: fakeReplicaSet.Kind,
			Name: fakeReplicaSet.Name,
		}
		fakePod := createFakePod(s.podStore, &podOwner)

		name, kind := wc.GetWorkloadForPod(fakePod)
		assert.Equal(t, "a-deployment", name)
		assert.Equal(t, "Deployment", kind)
	})

	t.Run("Pod with StatefulSet owner", func(t *testing.T) {
		wc, s := workloadCacheWithFakeListers()
		podOwner := metav1.OwnerReference{
			Kind: "StatefulSet",
			Name: "a-statefulset",
		}
		fakePod := createFakePod(s.podStore, &podOwner)

		name, kind := wc.GetWorkloadForPod(fakePod)
		assert.Equal(t, "a-statefulset", name)
		assert.Equal(t, "StatefulSet", kind)
	})

	t.Run("Pod with DaemonSet owner", func(t *testing.T) {
		wc, s := workloadCacheWithFakeListers()
		podOwner := metav1.OwnerReference{
			Kind: "DaemonSet",
			Name: "a-daemonset",
		}
		fakePod := createFakePod(s.podStore, &podOwner)

		name, kind := wc.GetWorkloadForPod(fakePod)
		assert.Equal(t, "a-daemonset", name)
		assert.Equal(t, "DaemonSet", kind)
	})

	t.Run("Pod with Job owner", func(t *testing.T) {
		wc, s := workloadCacheWithFakeListers()
		fakeJob := createFakeJob(s.jobStore, nil)
		podOwner := metav1.OwnerReference{
			Kind: fakeJob.Kind,
			Name: fakeJob.Name,
		}
		fakePod := createFakePod(s.podStore, &podOwner)

		name, kind := wc.GetWorkloadForPod(fakePod)
		assert.Equal(t, fakeJob.Name, name)
		assert.Equal(t, fakeJob.Kind, kind)
	})

	t.Run("Pod with CronJob owner", func(t *testing.T) {
		wc, s := workloadCacheWithFakeListers()
		jobOwner := metav1.OwnerReference{Kind: "CronJob", Name: "a-cronjob"}
		fakeJob := createFakeJob(s.jobStore, &jobOwner)
		podOwner := metav1.OwnerReference{
			Kind: fakeJob.Kind,
			Name: fakeJob.Name,
		}
		fakePod := createFakePod(s.podStore, &podOwner)

		name, kind := wc.GetWorkloadForPod(fakePod)
		assert.Equal(t, "a-cronjob", name)
		assert.Equal(t, "CronJob", kind)
	})

	t.Run("Pod with no owner", func(t *testing.T) {
		wc, s := workloadCacheWithFakeListers()
		fakePod := createFakePod(s.podStore, nil)

		name, kind := wc.GetWorkloadForPod(fakePod)
		assert.NotNil(t, name)
		assert.Equal(t, "Pod", kind)
	})
}

func TestGetPodWorkloadForPodName(t *testing.T) {
	t.Run("Pod with deployment owner", func(t *testing.T) {
		wc, s := workloadCacheWithFakeListers()
		rsOwner := metav1.OwnerReference{Kind: "Deployment", Name: "a-deployment"}
		fakeReplicaSet := createFakeReplicaSet(s.rsStore, &rsOwner)
		podOwner := metav1.OwnerReference{
			Kind: fakeReplicaSet.Kind,
			Name: fakeReplicaSet.Name,
		}
		fakePod := createFakePod(s.podStore, &podOwner)

		name, kind := wc.GetWorkloadForPodName(fakePod.Name, fakePod.Namespace)
		assert.Equal(t, "a-deployment", name)
		assert.Equal(t, "Deployment", kind)
	})

	t.Run("Returns empty strings on error", func(t *testing.T) {
		wc, _ := workloadCacheWithFakeListers()

		name, kind := wc.GetWorkloadForPodName("not-exist", "default")
		assert.Empty(t, name)
		assert.Empty(t, kind)
	})
}

func createFakeJob(jobStore cache.Indexer, owner *metav1.OwnerReference) *batchv1.Job {
	job := &batchv1.Job{
		TypeMeta: metav1.TypeMeta{Kind: "Job"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "a-job",
			Namespace: "a-ns",
		},
	}
	if owner != nil {
		job.OwnerReferences = []metav1.OwnerReference{*owner}
	}

	jobStore.Add(job)
	return job
}

func createFakeReplicaSet(rsStore cache.Indexer, owner *metav1.OwnerReference) *appsv1.ReplicaSet {
	replicaSet := &appsv1.ReplicaSet{
		TypeMeta: metav1.TypeMeta{Kind: "ReplicaSet"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "a-rs",
			Namespace: "a-ns",
		},
	}
	if owner != nil {
		replicaSet.OwnerReferences = []metav1.OwnerReference{*owner}
	}

	rsStore.Add(replicaSet)
	return replicaSet
}

func createFakePod(podStore cache.Indexer, owner *metav1.OwnerReference) *corev1.Pod {
	pod := &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "a-pod",
			Namespace: "a-ns",
		},
	}
	if owner != nil {
		pod.OwnerReferences = []metav1.OwnerReference{*owner}
	}

	podStore.Add(pod)
	return pod
}
