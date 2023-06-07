// Copyright 2018-2019 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package wavefront

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/events"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/wf"
	"github.com/wavefronthq/observability-for-kubernetes/collector/plugins/sinks"

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
	assert.Equal(t, 0, len(getReceivedLines(fakeSink)))
}

func TestName(t *testing.T) {
	fakeSink := NewTestWavefrontSink()
	name := fakeSink.Name()
	assert.Equal(t, name, "wavefront_sink")
}

func TestCreateWavefrontSinkWithNoEmptyInputs(t *testing.T) {
	cfg := configuration.SinkConfig{
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

func TestPrefix(t *testing.T) {
	cfg := configuration.SinkConfig{
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
	assert.True(t, strings.Contains(getReceivedLines(sink), "test.cpu.idle"))
}

func TestNilPointDataBatch(t *testing.T) {
	cfg := configuration.SinkConfig{
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
	assert.True(t, strings.Contains(getReceivedLines(sink), "test.cpu.idle"))
}

func TestCleansTagsBeforeSending(t *testing.T) {
	cfg := configuration.SinkConfig{
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
	assert.NotContains(t, getReceivedLines(sink), "emptyTag")
}

func TestEvents(t *testing.T) {
	cfg := configuration.SinkConfig{
		ProxyAddress: "wavefront-proxy:2878",
		TestMode:     true,
	}

	event := &events.Event{
		Message: "Pulled image",
	}

	t.Run("events disabled", func(t *testing.T) {
		cfg.EventsEnabled = false
		sink, err := NewWavefrontSink(cfg)
		assert.NoError(t, err)

		sink.ExportEvent(event)

		assert.Empty(t, getReceivedLines(sink))
	})

	t.Run("events enabled", func(t *testing.T) {
		cfg.EventsEnabled = true
		sink, err := NewWavefrontSink(cfg)
		assert.NoError(t, err)

		sink.ExportEvent(event)

		assert.NotEmpty(t, getReceivedLines(sink))
		assert.Contains(t, getReceivedLines(sink), "Pulled image")
	})
}

func getReceivedLines(sink sinks.Sink) string {
	return strings.TrimSpace(sink.(*wavefrontSink).WavefrontClient.(*TestSender).GetReceivedLines())
}
