package external

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/events"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/configuration"
)

func NewTestSink() *ExternalSink {
	return &ExternalSink{
		externalEndpointURL: "test",
	}
}

func TestName(t *testing.T) {
	fakeSink := NewTestSink()
	name := fakeSink.Name()
	require.Equal(t, name, "k8s_events_sink")
}

func TestCreateWavefrontSinkWithEventsExternalEndpointURL(t *testing.T) {
	sink, err := NewExternalSink(defaultSyncConfig("https://example.com"))
	require.NoError(t, err)
	require.NotNil(t, sink)
	k8sSink, ok := sink.(*ExternalSink)
	require.Equal(t, true, ok)
	require.Equal(t, "https://example.com", k8sSink.externalEndpointURL)
}

func TestEventsSendToExternalEndpointURL(t *testing.T) {
	t.Run("sends the event in json format", func(t *testing.T) {
		var requestBody, requestContentType, authorizationHeader string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			b, _ := io.ReadAll(r.Body)
			requestBody = string(b)
			requestContentType = r.Header.Get("Content-Type")
			authorizationHeader = r.Header.Get("Authorization")
		}))
		defer server.Close()

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
			Message:        "Successfully pulled image projects.registry.vmware.com/tanzu_observability_keights_saas/proxy:13.0 in 52.454993525s",
			FirstTimestamp: metav1.NewTime(time.Now()),
			LastTimestamp:  metav1.NewTime(time.Now()),
			Count:          1,
			Type:           "Normal",
			Source: v1.EventSource{
				Host:      "kind-control-plane",
				Component: "kubelet",
			},
		}}

		sink, err := NewExternalSink(defaultSyncConfig(server.URL))
		require.NoError(t, err)

		sink.ExportEvent(event)
		require.Contains(t, requestBody, "\"kind\":\"Pod\"")
		require.Contains(t, requestBody, "\"name\":\"wavefront-proxy-66b4d9dd94-cfqrr.1764aab366800a9c\"")
		require.Contains(t, requestContentType, "application/json")
		require.Contains(t, authorizationHeader, "Bearer some-key")
	})
}

func TestDisablingEvents(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	defer server.Close()

	cfg := defaultSyncConfig(server.URL)
	*cfg.EnableEvents = false

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
		Message:        "Successfully pulled image tobs/prxy:5.5 in 52.454993525s",
		FirstTimestamp: metav1.NewTime(time.Now()),
		LastTimestamp:  metav1.NewTime(time.Now()),
		Count:          1,
		Type:           "Normal",
		Source: v1.EventSource{
			Host:      "kind-control-plane",
			Component: "kubelet",
		},
	}}

	sink, _ := NewExternalSink(cfg)

	sink.ExportEvent(event)
	require.False(t, called, "expect event to not be exported")
}

func TestBadScenarios(t *testing.T) {
	t.Run("non-successful HTTP code handled", func(t *testing.T) {
		called := false
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			w.WriteHeader(404)
		}))
		defer server.Close()

		cfg := defaultSyncConfig(server.URL)
		sink, err := NewExternalSink(cfg)
		require.NoError(t, err)

		sink.ExportEvent(&events.Event{})
		require.True(t, called, "expect server to be called")
	})

	t.Run("server error handled", func(t *testing.T) {
		server := httptest.NewServer(nil)
		defer server.Close()

		cfg := defaultSyncConfig(server.URL)
		sink, err := NewExternalSink(cfg)
		require.NoError(t, err)

		sink.ExportEvent(&events.Event{})
	})

	t.Run("invalid server URL handled", func(t *testing.T) {
		cfg := defaultSyncConfig("\x00")

		sink, err := NewExternalSink(cfg)
		require.NoError(t, err)

		sink.ExportEvent(&events.Event{})
	})
}

func defaultSyncConfig(url string) configuration.SinkConfig {
	eventsEnabled := true
	return configuration.SinkConfig{
		Type:                      configuration.ExternalSinkType,
		ClusterName:               "testCluster",
		ExternalEndpointURL:       url,
		EnableEvents:              &eventsEnabled,
		ExternalEndpointAccessKey: "some-key",
	}
}
