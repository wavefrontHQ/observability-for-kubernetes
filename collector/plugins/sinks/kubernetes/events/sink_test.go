package events

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/events"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/stretchr/testify/assert"

	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/configuration"
)

func NewTestSink() *k8sEventSink {
	return &k8sEventSink{
		ClusterName:               "testCluster",
		eventsExternalEndpointURL: "test",
	}
}

func TestName(t *testing.T) {
	fakeSink := NewTestSink()
	name := fakeSink.Name()
	assert.Equal(t, name, "k8s_events_sink")
}

func TestCreateWavefrontSinkWithEventsExternalEndpointURL(t *testing.T) {
	cfg := configuration.SinkConfig{
		ProxyAddress: "wavefront-proxy:2878",
		ClusterName:  "testCluster",
		TestMode:     true,
		Transforms: configuration.Transforms{
			Prefix: "testPrefix",
		},
		EventsExternalEndpointURL: "https://example.com",
	}
	sink, err := NewK8sEventsOnlySink(cfg)
	assert.NoError(t, err)
	assert.NotNil(t, sink)
	k8sSink, ok := sink.(*k8sEventSink)
	assert.Equal(t, true, ok)
	assert.Equal(t, "testCluster", k8sSink.ClusterName)
	assert.Equal(t, "https://example.com", k8sSink.eventsExternalEndpointURL)
}

func TestEventsSendToExternalEndpointURL(t *testing.T) {
	var requestBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		b, _ := io.ReadAll(r.Body)
		requestBody = string(b)
	}))
	defer server.Close()

	cfg := configuration.SinkConfig{
		ProxyAddress: "wavefront-proxy:2878",
		ClusterName:  "testCluster",
		Transforms: configuration.Transforms{
			Prefix: "testPrefix",
		},
		EventsExternalEndpointURL: server.URL,
	}

	event := &events.Event{Event: v1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "wavefront-proxy-66b4d9dd94-cfqrr.1764aab366800a9c",
			Namespace:         "observability-system",
			UID:               "37122603-ee03-4ecc-b866-5a2122703207",
			ResourceVersion:   "737",
			CreationTimestamp: metav1.NewTime(time.Now()),
		},
		InvolvedObject: v1.ObjectReference{
			Namespace:       "observability-system",
			Kind:            "Pod",
			Name:            "wavefront-proxy-66b4d9dd94-cfqrr",
			UID:             "0a3c1f18-4680-479b-bb54-9a82b9e3f997",
			APIVersion:      "v1",
			ResourceVersion: "662",
			FieldPath:       "spec.containers{wavefront-proxy}",
		},
		Reason:         "Pulled",
		Message:        "Successfully pulled image projects.registry.vmware.com/tanzu_observability_keights_saas/proxy:12.4.1 in 52.454993525s",
		FirstTimestamp: metav1.NewTime(time.Now()),
		LastTimestamp:  metav1.NewTime(time.Now()),
		Count:          1,
		Type:           "Normal",
		Source: v1.EventSource{
			Host:      "kind-control-plane",
			Component: "kubelet",
		},
	},
	}

	sink, err := NewK8sEventsOnlySink(cfg)
	assert.NoError(t, err)

	sink.ExportEvent(event)
	assert.Contains(t, requestBody, "\"kind\":\"Pod\"")
	assert.Contains(t, requestBody, "\"name\":\"wavefront-proxy-66b4d9dd94-cfqrr.1764aab366800a9c\"")
	assert.Contains(t, requestBody, "\"clusterName\":\"testCluster\"")
}
