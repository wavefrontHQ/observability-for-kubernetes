package events

import (
	"context"
	"testing"
	"time"

	"k8s.io/client-go/kubernetes/fake"

	"github.com/stretchr/testify/assert"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/events"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/metrics"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAddEvent(t *testing.T) {
	sink := &MockExport{}

	er := &EventRouter{
		sink: sink,
	}
	event := &v1.Event{
		Message:       "Test message for events",
		LastTimestamp: metav1.NewTime(time.Now()),
		Source: v1.EventSource{
			Host:      "test Host",
			Component: "some-component",
		},
		InvolvedObject: v1.ObjectReference{
			Namespace: "test-namespace",
			Kind:      "some-kind",
			Name:      "test-name",
		},
		Reason: "some-reason",
	}
	er.addEvent(event)
	assert.Equal(t, event.Message, sink.Message)
	assert.Equal(t, event.LastTimestamp, metav1.NewTime(sink.Ts))
	assert.Equal(t, event.Source.Host, sink.Host)
	assert.Equal(t, "Normal", sink.Annotations["type"])
	assert.Equal(t, "some-kind", sink.Annotations["kind"])
	assert.Equal(t, "some-reason", sink.Annotations["reason"])
	assert.Equal(t, "some-component", sink.Annotations["component"])
	assert.Equal(t, "test-name", sink.Annotations["resource_name"])
	assert.NotContains(t, sink.Annotations, "pod_name")
	assert.Equal(t, "test-namespace", event.InvolvedObject.Namespace)
}

func TestAddEventHasWorkload(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()
	fakePod := createFakePod(t, fakeClient, nil)

	sink := &MockExport{}

	er := &EventRouter{
		kubeClient: fakeClient,
		sink:       sink,
	}
	event := &v1.Event{
		Message:       "Test message for events",
		LastTimestamp: metav1.NewTime(time.Now()),
		Source: v1.EventSource{
			Host:      "test Host",
			Component: "some-component",
		},
		InvolvedObject: v1.ObjectReference{
			Namespace: fakePod.Namespace,
			Kind:      fakePod.Kind,
			Name:      fakePod.Name,
		},
		Reason: "some-reason",
	}
	er.addEvent(event)
	assert.Equal(t, event.Message, sink.Message)
	assert.Equal(t, event.LastTimestamp, metav1.NewTime(sink.Ts))
	assert.Equal(t, event.Source.Host, sink.Host)
	assert.Equal(t, "Normal", sink.Annotations["type"])
	assert.Equal(t, fakePod.Kind, sink.Annotations["kind"])
	assert.Equal(t, "some-reason", sink.Annotations["reason"])
	assert.Equal(t, "some-component", sink.Annotations["component"])
	assert.Equal(t, fakePod.Name, sink.Annotations["pod_name"])
	assert.NotContains(t, sink.Annotations, "resource_name")
	assert.Equal(t, fakePod.Namespace, event.InvolvedObject.Namespace)
}

type MockExport struct {
	events.Event
	Annotations map[string]string
}

func (m *MockExport) ExportEvent(event *events.Event) {
	tagMap := map[string]interface{}{
		"annotations": map[string]string{},
	}
	for _, e := range event.Options {
		e(tagMap)
	}
	m.Message = event.Message
	m.Ts = event.Ts
	m.Host = event.Host
	m.Annotations = tagMap["annotations"].(map[string]string)
	m.Labels = event.Labels
}
func (m *MockExport) Export(batch *metrics.Batch) {}
func (m *MockExport) Name() string                { return "" }
func (m *MockExport) Stop()                       {}

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
