package events

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/configuration"
	"k8s.io/client-go/kubernetes/fake"
)

func TestAnnotateEvent(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()
	fakePod := createFakePod(t, fakeClient, nil)
	workloadCache := &testWorkloadCache{
		workloadName: "some-workload-name",
		workloadKind: "some-workload-kind",
		nodeName:     "some-node-name",
	}
	sink := &MockExport{}
	cfg := configuration.EventsConfig{ClusterName: "some-cluster-name", ClusterUUID: "some-cluster-uuid"}
	event := fakeEvent()
	event.InvolvedObject.Kind = fakePod.Kind
	event.InvolvedObject.Namespace = fakePod.Namespace
	event.InvolvedObject.Name = fakePod.Name
	er := NewEventRouter(fakeClient, cfg, sink, true, workloadCache)

	annotateEvent(event, er)

	require.Equal(t, "some-cluster-name", event.ObjectMeta.Annotations["aria/cluster-name"])
	require.Equal(t, "some-cluster-uuid", event.ObjectMeta.Annotations["aria/cluster-uuid"])
	require.Equal(t, "some-workload-kind", event.ObjectMeta.Annotations["aria/workload-kind"])
	require.Equal(t, "some-workload-name", event.ObjectMeta.Annotations["aria/workload-name"])
	require.Equal(t, "some-node-name", event.ObjectMeta.Annotations["aria/node-name"])

	require.Equal(t, "Creation", event.ObjectMeta.Annotations["aria/category"])
}
