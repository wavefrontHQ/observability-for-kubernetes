package events

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"os"
	"testing"

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
	annotateEvent(event, workloadCache, "some-cluster-name", "some-cluster-uuid")

	require.Equal(t, "some-cluster-name", event.ObjectMeta.Annotations["aria/cluster-name"])
	require.Equal(t, "some-cluster-uuid", event.ObjectMeta.Annotations["aria/cluster-uuid"])
	require.Equal(t, "some-workload-kind", event.ObjectMeta.Annotations["aria/workload-kind"])
	require.Equal(t, "some-workload-name", event.ObjectMeta.Annotations["aria/workload-name"])
	require.Equal(t, "some-node-name", event.ObjectMeta.Annotations["aria/node-name"])
}

func TestAnnotateCategories(t *testing.T) {
	t.Run("Failed to pull image", func(t *testing.T) {
		workloadCache := &testWorkloadCache{
			workloadName: "daemonset-not-ready",
			workloadKind: "DaemonSet",
			nodeName:     "gke-cluster-default-pool-0816c2b3-zwkx",
		}
		event := getEvent(t, "examples/failed_to_pull.yaml")
		annotateEvent(&event, workloadCache, "some-cluster-name", "some-cluster-uuid")
		require.Equal(t, CREATION, event.ObjectMeta.Annotations["aria/category"])
	})
	t.Run("Crash loop backoff", func(t *testing.T) {
		workloadCache := &testWorkloadCache{
			workloadName: "pod-crash-loop-backoff",
			workloadKind: "Pod",
			nodeName:     "gke-cluster-default-pool-0816c2b3-jt7j",
		}
		event := getEvent(t, "examples/crash_loop_backoff.yaml")
		annotateEvent(&event, workloadCache, "some-cluster-name", "some-cluster-uuid")
		require.Equal(t, RUNTIME, event.ObjectMeta.Annotations["aria/category"])
	})
}

func getEvent(t *testing.T, fileName string) v1.Event {
	failedToPull, err := os.ReadFile(fileName)
	require.NoError(t, err)
	var event v1.Event
	err = yaml.Unmarshal(failedToPull, &event)
	require.NotNil(t, event)
	return event
}
