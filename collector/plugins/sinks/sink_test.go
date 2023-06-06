// Copyright 2018-2019 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package sinks

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/events"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/wf"

	"github.com/stretchr/testify/assert"

	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/configuration"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/metrics"
)

func NewTestWavefrontSink() *wavefrontSink {
	return &wavefrontSink{
		WavefrontClient: NewTestSender(),
		ClusterName:     "testCluster",
	}
}

func TestStoreTimeseriesEmptyInput(t *testing.T) {
	fakeSink := NewTestWavefrontSink()
	db := metrics.Batch{}
	fakeSink.Export(&db)
	assert.Equal(t, 0, len(getMetrics(fakeSink)))
}

func TestName(t *testing.T) {
	fakeSink := NewTestWavefrontSink()
	name := fakeSink.Name()
	assert.Equal(t, name, "wavefront_sink")
}

func TestCreateWavefrontSinkWithNoEmptyInputs(t *testing.T) {
	cfg := configuration.WavefrontSinkConfig{
		ProxyAddress: "wavefront-proxy:2878",
		ClusterName:  "testCluster",
		TestMode:     true,
		Transforms: configuration.Transforms{
			Prefix: "testPrefix",
		},
	}
	sink, err := NewWavefrontSink(cfg)
	assert.NoError(t, err)
	assert.NotNil(t, sink)
	wfSink, ok := sink.(*wavefrontSink)
	assert.Equal(t, true, ok)
	assert.NotNil(t, wfSink.WavefrontClient)
	assert.Equal(t, "testCluster", wfSink.ClusterName)
	assert.Equal(t, "testPrefix", wfSink.Prefix)
}

func TestCreateWavefrontSinkWithEventsExternalEndpointURL(t *testing.T) {
	cfg := configuration.WavefrontSinkConfig{
		ProxyAddress: "wavefront-proxy:2878",
		ClusterName:  "testCluster",
		TestMode:     true,
		Transforms: configuration.Transforms{
			Prefix: "testPrefix",
		},
		EventsExternalEndpointURL: "https://example.com",
	}
	sink, err := NewWavefrontSink(cfg)
	assert.NoError(t, err)
	assert.NotNil(t, sink)
	wfSink, ok := sink.(*wavefrontSink)
	assert.Equal(t, true, ok)
	assert.NotNil(t, wfSink.WavefrontClient)
	assert.Equal(t, "testCluster", wfSink.ClusterName)
	assert.Equal(t, "testPrefix", wfSink.Prefix)
	assert.Equal(t, "https://example.com", wfSink.eventsExternalEndpointURL)
}

func TestEventsSendToExternalEndpointURL(t *testing.T) {
	var requestBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		b, _ := io.ReadAll(r.Body)
		requestBody = string(b)
	}))
	defer server.Close()

	cfg := configuration.WavefrontSinkConfig{
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

	sink, err := NewWavefrontSink(cfg)
	assert.NoError(t, err)

	sink.ExportEvent(event)
	assert.Contains(t, requestBody, "\"kind\":\"Pod\"")
	assert.Contains(t, requestBody, "\"name\":\"wavefront-proxy-66b4d9dd94-cfqrr.1764aab366800a9c\"")
	assert.Contains(t, requestBody, "\"clusterName\":\"testCluster\"")
}

func TestPrefix(t *testing.T) {
	cfg := configuration.WavefrontSinkConfig{
		ProxyAddress: "wavefront-proxy:2878",
		TestMode:     true,
		Transforms: configuration.Transforms{
			Prefix: "test.",
		},
	}
	sink, err := NewWavefrontSink(cfg)
	assert.NoError(t, err)

	db := metrics.Batch{
		Metrics: []wf.Metric{
			wf.NewPoint("cpu.idle", 1.0, 0, "fakeSource", nil),
		},
	}
	sink.Export(&db)
	assert.True(t, strings.Contains(getMetrics(sink), "test.cpu.idle"))
}

func TestNilPointDataBatch(t *testing.T) {
	cfg := configuration.WavefrontSinkConfig{
		ProxyAddress: "wavefront-proxy:2878",
		TestMode:     true,
		Transforms: configuration.Transforms{
			Prefix: "test.",
		},
	}
	sink, err := NewWavefrontSink(cfg)
	assert.NoError(t, err)

	db := metrics.Batch{
		Metrics: []wf.Metric{
			nil,
			wf.NewPoint("cpu.idle", 1.0, 0, "fakeSource", nil),
		},
	}
	sink.Export(&db)
	assert.True(t, strings.Contains(getMetrics(sink), "test.cpu.idle"))
}

func TestCleansTagsBeforeSending(t *testing.T) {
	cfg := configuration.WavefrontSinkConfig{
		ProxyAddress: "wavefront-proxy:2878",
		TestMode:     true,
		Transforms: configuration.Transforms{
			Prefix: "test.",
		},
	}
	sink, err := NewWavefrontSink(cfg)
	assert.NoError(t, err)

	db := metrics.Batch{
		Metrics: []wf.Metric{
			wf.NewPoint(
				"cpu.idle",
				1.0,
				0,
				"fakeSource",
				map[string]string{"emptyTag": ""},
			),
		},
	}
	sink.Export(&db)
	assert.NotContains(t, getMetrics(sink), "emptyTag")
}

func getMetrics(sink Sink) string {
	return strings.TrimSpace(sink.(*wavefrontSink).WavefrontClient.(*TestSender).GetReceivedLines())
}
