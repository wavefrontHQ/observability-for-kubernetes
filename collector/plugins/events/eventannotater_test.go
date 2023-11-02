package events

import (
	"github.com/stretchr/testify/require"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/configuration"
	"k8s.io/client-go/kubernetes/fake"
	"testing"
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

	require.Equal(t, "some-cluster-name", sink.ObjectMeta.Annotations["aria/cluster-name"])
	require.Equal(t, "some-cluster-uuid", sink.ObjectMeta.Annotations["aria/cluster-uuid"])
	require.Equal(t, "some-workload-kind", sink.ObjectMeta.Annotations["aria/workload-kind"])
	require.Equal(t, "some-workload-name", sink.ObjectMeta.Annotations["aria/workload-name"])
	require.Equal(t, "some-node-name", sink.ObjectMeta.Annotations["aria/node-name"])
}
