package util

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestGetPodWorkload(t *testing.T) {
	t.Run("Pod with no owner", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset()
		fakePod := createFakePod(t, fakeClient, nil)

		name, kind := GetWorkloadForPod(fakeClient, fakePod.Name, fakePod.Namespace)
		assert.Equal(t, fakePod.Name, name)
		assert.Equal(t, fakePod.Kind, kind)
	})

	t.Run("Pod with ReplicaSet owner", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset()
		fakeReplicaSet := createFakeReplicaSet(t, fakeClient, nil)
		podOwner := metav1.OwnerReference{
			Kind: fakeReplicaSet.Kind,
			Name: fakeReplicaSet.Name,
		}
		fakePod := createFakePod(t, fakeClient, &podOwner)

		name, kind := GetWorkloadForPod(fakeClient, fakePod.Name, fakePod.Namespace)
		assert.Equal(t, fakeReplicaSet.Name, name)
		assert.Equal(t, fakeReplicaSet.Kind, kind)
	})

	t.Run("Pod with Deployment owner", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset()
		rsOwner := metav1.OwnerReference{Kind: "Deployment", Name: "a-deployment"}
		fakeReplicaSet := createFakeReplicaSet(t, fakeClient, &rsOwner)
		podOwner := metav1.OwnerReference{
			Kind: fakeReplicaSet.Kind,
			Name: fakeReplicaSet.Name,
		}
		fakePod := createFakePod(t, fakeClient, &podOwner)

		name, kind := GetWorkloadForPod(fakeClient, fakePod.Name, fakePod.Namespace)
		assert.Equal(t, "a-deployment", name)
		assert.Equal(t, "Deployment", kind)
	})

	t.Run("Pod with StatefulSet owner", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset()
		podOwner := metav1.OwnerReference{
			Kind: "StatefulSet",
			Name: "a-statefulset",
		}
		fakePod := createFakePod(t, fakeClient, &podOwner)

		name, kind := GetWorkloadForPod(fakeClient, fakePod.Name, fakePod.Namespace)
		assert.Equal(t, "a-statefulset", name)
		assert.Equal(t, "StatefulSet", kind)
	})

	t.Run("Pod with DaemonSet owner", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset()
		podOwner := metav1.OwnerReference{
			Kind: "DaemonSet",
			Name: "a-daemonset",
		}
		fakePod := createFakePod(t, fakeClient, &podOwner)

		name, kind := GetWorkloadForPod(fakeClient, fakePod.Name, fakePod.Namespace)
		assert.Equal(t, "a-daemonset", name)
		assert.Equal(t, "DaemonSet", kind)
	})

	t.Run("Pod with Job owner", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset()
		fakeJob := createFakeJob(t, fakeClient, nil)
		podOwner := metav1.OwnerReference{
			Kind: fakeJob.Kind,
			Name: fakeJob.Name,
		}
		fakePod := createFakePod(t, fakeClient, &podOwner)

		name, kind := GetWorkloadForPod(fakeClient, fakePod.Name, fakePod.Namespace)
		assert.Equal(t, fakeJob.Name, name)
		assert.Equal(t, fakeJob.Kind, kind)
	})

	t.Run("Pod with CronJob owner", func(t *testing.T) {
		fakeClient := fake.NewSimpleClientset()
		jobOwner := metav1.OwnerReference{Kind: "CronJob", Name: "a-cronjob"}
		fakeJob := createFakeJob(t, fakeClient, &jobOwner)
		podOwner := metav1.OwnerReference{
			Kind: fakeJob.Kind,
			Name: fakeJob.Name,
		}
		fakePod := createFakePod(t, fakeClient, &podOwner)

		name, kind := GetWorkloadForPod(fakeClient, fakePod.Name, fakePod.Namespace)
		assert.Equal(t, "a-cronjob", name)
		assert.Equal(t, "CronJob", kind)
	})
}

func createFakeJob(t *testing.T, fakeClient *fake.Clientset, owner *metav1.OwnerReference) *batchv1.Job {
	jobSpec := &batchv1.Job{
		TypeMeta: metav1.TypeMeta{Kind: "Job"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "a-job",
			Namespace: "a-ns",
		},
	}
	if owner != nil {
		jobSpec.OwnerReferences = []metav1.OwnerReference{*owner}
	}

	jobsClient := fakeClient.BatchV1().Jobs("a-ns")
	job, err := jobsClient.Create(context.Background(), jobSpec, metav1.CreateOptions{})
	assert.NoError(t, err)
	return job
}

func createFakeReplicaSet(t *testing.T, fakeClient *fake.Clientset, owner *metav1.OwnerReference) *appsv1.ReplicaSet {
	replicaSetSpec := &appsv1.ReplicaSet{
		TypeMeta: metav1.TypeMeta{Kind: "ReplicaSet"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "a-rs",
			Namespace: "a-ns",
		},
	}
	if owner != nil {
		replicaSetSpec.OwnerReferences = []metav1.OwnerReference{*owner}
	}

	replicaSetsClient := fakeClient.AppsV1().ReplicaSets("a-ns")
	rs, err := replicaSetsClient.Create(context.Background(), replicaSetSpec, metav1.CreateOptions{})
	assert.NoError(t, err)
	return rs
}

func createFakePod(t *testing.T, fakeClient *fake.Clientset, owner *metav1.OwnerReference) *v1.Pod {
	podSpec := &v1.Pod{
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
		podSpec.OwnerReferences = []metav1.OwnerReference{*owner}
	}

	podsClient := fakeClient.CoreV1().Pods("a-ns")
	pod, err := podsClient.Create(context.Background(), podSpec, metav1.CreateOptions{})
	assert.NoError(t, err)
	return pod
}
