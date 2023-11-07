package events

import (
	"os"
	"testing"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/yaml"

	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes/fake"
)

func TestAnnotateEventNonCategory(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()
	fakePod := createFakePod(t, fakeClient, nil)
	workloadCache := &testWorkloadCache{
		workloadName: "some-workload-name",
		workloadKind: "some-workload-kind",
		nodeName:     "some-node-name",
	}
	event := fakeEvent()
	event.InvolvedObject.Kind = fakePod.Kind
	event.InvolvedObject.Namespace = fakePod.Namespace
	event.InvolvedObject.Name = fakePod.Name
	ea := NewEventAnnotator(workloadCache, "some-cluster-name", "some-cluster-uuid")
	ea.annotateEvent(event)

	require.Equal(t, "some-cluster-name", event.ObjectMeta.Annotations["aria/cluster-name"])
	require.Equal(t, "some-cluster-uuid", event.ObjectMeta.Annotations["aria/cluster-uuid"])
	require.Equal(t, "some-workload-kind", event.ObjectMeta.Annotations["aria/workload-kind"])
	require.Equal(t, "some-workload-name", event.ObjectMeta.Annotations["aria/workload-name"])
	require.Equal(t, "some-node-name", event.ObjectMeta.Annotations["aria/node-name"])
}

func TestAnnotateCategories(t *testing.T) {
	t.Run("Failed to pull image", func(t *testing.T) {
		validateCategorySubcategory(t, "examples/failed_to_pull.yaml", Creation, ImagePullBackOff)
	})
	t.Run("Crash loop backoff", func(t *testing.T) {
		validateCategorySubcategory(t, "examples/crash_loop_backoff.yaml", Runtime, CrashLoopBackOff)
	})
	t.Run("Failed Mount", func(t *testing.T) {
		validateCategorySubcategory(t, "examples/failed_mount.yaml", Creation, FailedMount)
	})
	t.Run("Unhealthy", func(t *testing.T) {
		validateCategorySubcategory(t, "examples/unhealthy.yaml", Runtime, Unhealthy)
	})
	t.Run("FailedScheduling", func(t *testing.T) {
		validateCategorySubcategory(t, "examples/failed_scheduling.yaml", Scheduling, InsufficientResources)
	})
	t.Run("FailedCreate", func(t *testing.T) {
		validateCategorySubcategory(t, "examples/failed_create.yaml", Storage, FailedCreate)
	})
	t.Run("Stuck in Terminating", func(t *testing.T) {
		validateCategorySubcategory(t, "examples/stuck_in_terminating.yaml", Runtime, Terminating)
	})
}

func validateCategorySubcategory(t *testing.T, file, category, subcategory string) {
	workloadCache := &testWorkloadCache{
		workloadName: "some-workload-name",
		workloadKind: "some-workload-kind",
		nodeName:     "some-node-name",
	}
	eventList := getEventList(t, file)
	ea := NewEventAnnotator(workloadCache, "some-cluster-name", "some-cluster-uuid")
	for _, event := range eventList.Items {
		ea.annotateEvent(&event)
		require.Equal(t, category, event.ObjectMeta.Annotations["aria/category"])
		require.Equal(t, subcategory, event.ObjectMeta.Annotations["aria/subcategory"])
	}
}

func getEventList(t *testing.T, fileName string) v1.EventList {
	failedToPull, err := os.ReadFile(fileName)
	require.NoError(t, err)
	var event v1.EventList
	err = yaml.Unmarshal(failedToPull, &event)
	require.NotNil(t, event)
	return event
}
