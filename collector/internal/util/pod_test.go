package util

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestIsStuckInTerminating(t *testing.T) {
	t.Run("Pod is not stuck in terminating if deletion timestamp is nil", func(t *testing.T) {
		assert.False(t, IsStuckInTerminating(fakePod()))
	})

	t.Run("Pod is not stuck in terminating if pod is gracefully deleted", func(t *testing.T) {
		pod := fakePod()
		pod.ObjectMeta.DeletionTimestamp = &metav1.Time{Time: time.Now()}
		assert.False(t, IsStuckInTerminating(pod))
	})

	t.Run("Pod is stuck in terminating", func(t *testing.T) {
		pod := GetPodStuckInTerminating()
		assert.True(t, IsStuckInTerminating(pod))
	})

}

func fakePod() *corev1.Pod {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "a-pod",
			Namespace: "a-ns",
		},
		Spec: corev1.PodSpec{NodeName: "some-node-name"},
	}
	return pod
}
