package util

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"testing"
)

func TestGetPodWorkload(t *testing.T) {
	t.Run("Pod with no owner", func(t *testing.T) {
		client := fake.NewSimpleClientset()
		pod := &v1.Pod{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Pod",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "a-pod",
				Namespace: "a-ns",
			},
		}

		pods := client.CoreV1().Pods("a-ns")
		_, err := pods.Create(context.Background(), pod, metav1.CreateOptions{})
		if err != nil {
			fmt.Print(err.Error())
		}
		name, kind := GetWorkloadForPod(client, "a-pod", "a-ns")
		assert.Equal(t, "a-pod", name)
		assert.Equal(t, "Pod", kind)
	})

	t.Run("Pod with ReplicaSet owner", func(t *testing.T) {
		client := fake.NewSimpleClientset()
		rs := &appsv1.ReplicaSet{
			TypeMeta: metav1.TypeMeta{Kind: "ReplicaSet"},
			ObjectMeta: metav1.ObjectMeta{
				Name:            "a-rs",
				Namespace:       "a-ns",
				OwnerReferences: []metav1.OwnerReference{},
			},
		}
		podOwner := metav1.OwnerReference{
			Kind: rs.Kind,
			Name: rs.Name,
		}
		pod := &v1.Pod{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Pod",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:            "a-pod",
				Namespace:       "a-ns",
				OwnerReferences: []metav1.OwnerReference{podOwner},
			},
		}

		pods := client.CoreV1().Pods("a-ns")
		_, err := pods.Create(context.Background(), pod, metav1.CreateOptions{})
		if err != nil {
			fmt.Print(err.Error())
		}
		replicaSets := client.AppsV1().ReplicaSets("a-ns")
		_, err = replicaSets.Create(context.Background(), rs, metav1.CreateOptions{})
		assert.NoError(t, err)

		name, kind := GetWorkloadForPod(client, "a-pod", "a-ns")
		assert.Equal(t, "a-rs", name)
		assert.Equal(t, "ReplicaSet", kind)
	})

	t.Run("Pod with Deployment owner", func(t *testing.T) {
		client := fake.NewSimpleClientset()
		dep := &appsv1.Deployment{
			TypeMeta: metav1.TypeMeta{
				Kind: "Deployment",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:            "a-dep",
				Namespace:       "a-ns",
				OwnerReferences: nil,
			},
		}
		rsOwner := metav1.OwnerReference{Kind: dep.Kind, Name: dep.Name}
		rs := &appsv1.ReplicaSet{
			TypeMeta: metav1.TypeMeta{Kind: "ReplicaSet"},
			ObjectMeta: metav1.ObjectMeta{
				Name:            "a-rs",
				Namespace:       "a-ns",
				OwnerReferences: []metav1.OwnerReference{rsOwner},
			},
		}
		podOwner := metav1.OwnerReference{
			Kind: rs.Kind,
			Name: rs.Name,
		}
		pod := &v1.Pod{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Pod",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:            "a-pod",
				Namespace:       "a-ns",
				OwnerReferences: []metav1.OwnerReference{podOwner},
			},
		}

		pods := client.CoreV1().Pods("a-ns")
		_, err := pods.Create(context.Background(), pod, metav1.CreateOptions{})
		assert.NoError(t, err)
		replicaSets := client.AppsV1().ReplicaSets("a-ns")
		_, err = replicaSets.Create(context.Background(), rs, metav1.CreateOptions{})
		assert.NoError(t, err)

		name, kind := GetWorkloadForPod(client, "a-pod", "a-ns")
		assert.Equal(t, dep.Name, name)
		assert.Equal(t, dep.Kind, kind)
	})
	t.Run("Pod with StatefulSet owner", func(t *testing.T) {
	})
	t.Run("Pod with DaemonSet owner", func(t *testing.T) {
	})
	t.Run("Pod with Job owner", func(t *testing.T) {
	})
	t.Run("Pod with CronJob owner", func(t *testing.T) {
	})
}
