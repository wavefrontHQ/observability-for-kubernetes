package events

import (
	"os"
	"testing"
	"time"

	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/testhelper"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"

	"github.com/stretchr/testify/require"
)

func TestAnnotateEventNonCategory(t *testing.T) {
	t.Run("adds cluster name and cluster uuid annotations", func(t *testing.T) {
		eventAnnotator := setupAnnotator()
		event := getFakePodEvent()
		eventAnnotator.annotate(event)
		require.Equal(t, "some-cluster-name", event.ObjectMeta.Annotations["aria/cluster-name"])
		require.Equal(t, "some-cluster-uuid", event.ObjectMeta.Annotations["aria/cluster-uuid"])
	})

	t.Run("adds workload annotations for Pod events", func(t *testing.T) {
		pod := getFakePod()
		workloadCache := testhelper.NewFakeWorkloadCache(pod.Name, pod.Kind, pod.Spec.NodeName, pod)
		eventAnnotator := NewEventAnnotator(workloadCache, "some-cluster-name", "some-cluster-uuid")
		event := getFakePodEvent()
		eventAnnotator.annotate(event)
		require.Equal(t, pod.Name, event.ObjectMeta.Annotations["aria/workload-name"])
		require.Equal(t, "Pod", event.ObjectMeta.Annotations["aria/workload-kind"])
		require.Equal(t, "some-node-name", event.ObjectMeta.Annotations["aria/node-name"])
	})
}

func TestCategorizeMatching(t *testing.T) {
	// Creation
	t.Run("Failed to pull image and ErrImagePull", func(t *testing.T) {
		validateCategorySubcategory(t, "examples/failed_to_pull.yaml", Creation, ImagePullBackOff, "true")
	})
	t.Run("Failed to pull image for Job", func(t *testing.T) {
		validateCategorySubcategory(t, "examples/job_failed_to_pull.yaml", Creation, ImagePullBackOff, "true")
	})
	t.Run("Back-off pulling image for Normal event type", func(t *testing.T) {
		validateCategorySubcategory(t, "examples/normal_backoff.yaml", Creation, ImagePullBackOff, "true")
	})
	t.Run("FailedMount", func(t *testing.T) {
		validateCategorySubcategory(t, "examples/failed_mount.yaml", Creation, FailedMount, "true")
	})

	// Runtime
	t.Run("Crash loop back-off", func(t *testing.T) {
		validateCategorySubcategory(t, "examples/crash_loop_backoff.yaml", Runtime, CrashLoopBackOff, "true")
	})
	t.Run("Unhealthy", func(t *testing.T) {
		validateCategorySubcategory(t, "examples/unhealthy.yaml", Runtime, Unhealthy, "true")
	})
	t.Run("Out-of-memory killed", func(t *testing.T) {
		validateCategorySubcategory(t, "examples/oom_killed.yaml", Runtime, OOMKilled, "true")
	})

	// Scheduling
	t.Run("FailedScheduling", func(t *testing.T) {
		validateCategorySubcategory(t, "examples/failed_scheduling.yaml", Scheduling, InsufficientResources, "true")
	})
	t.Run("Pod can't be scheduled as Node is not in ready state", func(t *testing.T) {
		validateCategorySubcategory(t, "examples/node_not_ready.yaml", Scheduling, NodeNotReady, "true")
	})

	// Storage
	t.Run("FailedCreate", func(t *testing.T) {
		validateCategorySubcategory(t, "examples/failed_create.yaml", Storage, FailedCreate, "true")
	})
	t.Run("Provisioning failed", func(t *testing.T) {
		validateCategorySubcategory(t, "examples/pv_provisioning_failed.yaml", Storage, ProvisioningFailed, "true")
	})

	// Job
	t.Run("BackoffLimitExceeded", func(t *testing.T) {
		validateCategorySubcategory(t, "examples/job_failed.yaml", Job, BackoffLimitExceeded, "true")
	})

	// Other
	t.Run("HPA", func(t *testing.T) {
		validateCategorySubcategory(t, "examples/hpa.yaml", "HorizontalPodAutoscaler", "HorizontalPodAutoscaler", "true")
	})
}

func TestCategorizeNonMatching(t *testing.T) {
	t.Run("When normal event is not important", func(t *testing.T) {
		validateCategorySubcategory(t, "examples/unimportant.yaml", "", "", "false")
	})

	t.Run("When normal event that shouldn't match", func(t *testing.T) {
		validateCategorySubcategory(t, "examples/normal_pulling_image.yaml", "", "", "false")
	})
}

func validateCategorySubcategory(t *testing.T, file, category, subcategory, important string) {
	eventAnnotator := setupAnnotator()
	eventList := getEventList(t, file)
	validateAnnotations(t, eventAnnotator, eventList, category, subcategory, important)
}

func validateAnnotations(t *testing.T, ea *EventAnnotator, eventList v1.EventList, category, subcategory, important string) {
	for _, event := range eventList.Items {
		ea.annotate(&event)
		require.Equal(t, category, event.ObjectMeta.Annotations["aria/category"])
		require.Equal(t, subcategory, event.ObjectMeta.Annotations["aria/subcategory"])
		require.Equal(t, important, event.ObjectMeta.Annotations["important"])
	}
}

func setupAnnotator() *EventAnnotator {
	workloadCache := testhelper.NewEmptyFakeWorkloadCache()
	return NewEventAnnotator(workloadCache, "some-cluster-name", "some-cluster-uuid")
}

func getEventList(t *testing.T, fileName string) v1.EventList {
	data, err := os.ReadFile(fileName)
	require.NoError(t, err)
	var event v1.EventList
	require.NoError(t, yaml.Unmarshal(data, &event))
	require.NotNil(t, event)
	return event
}

func getFakePod() *v1.Pod {
	return &v1.Pod{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "a-pod",
			Namespace: "a-ns",
		},
		Spec: v1.PodSpec{NodeName: "some-node-name"},
	}
}

func getFakePodEvent() *v1.Event {
	return &v1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "a-ns",
		},
		InvolvedObject: v1.ObjectReference{
			Kind:      "Pod",
			Name:      "a-pod",
			Namespace: "a-ns",
		},
		LastTimestamp: metav1.NewTime(time.Now()),
		Reason:        "SomeReason",
		Message:       "Some message",
		Type:          "Warning",
	}
}
