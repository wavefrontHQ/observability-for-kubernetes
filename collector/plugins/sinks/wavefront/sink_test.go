// Copyright 2018-2019 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package wavefront

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
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
	require.Equal(t, 0, len(getReceivedLines(fakeSink)))
}

func TestName(t *testing.T) {
	fakeSink := NewTestWavefrontSink()
	name := fakeSink.Name()
	require.Equal(t, name, "wavefront_sink")
}

func TestCreateWavefrontSinkWithNoEmptyInputs(t *testing.T) {
	config := defaultSinkConfig()
	config.Prefix = "testPrefix"
	sink, err := NewWavefrontSink(config)
	require.NoError(t, err)
	require.NotNil(t, sink)
	wfSink, ok := sink.(*wavefrontSink)
	require.Equal(t, true, ok)
	require.NotNil(t, wfSink.WavefrontClient)
	require.Equal(t, "testCluster", wfSink.ClusterName)
	require.Equal(t, "testPrefix", wfSink.Prefix)
}

func TestPrefix(t *testing.T) {
	cfg := defaultSinkConfig()
	cfg.Transforms.Prefix = "test."
	sink, err := NewWavefrontSink(cfg)
	require.NoError(t, err)

	db := metrics.Batch{
		Metrics: []wf.Metric{
			wf.NewPoint("cpu.idle", 1.0, 0, "fakeSource", nil),
		},
	}
	sink.Export(&db)
	require.True(t, strings.Contains(getReceivedLines(sink), "test.cpu.idle"))
}

func TestUseNodeNameForSource(t *testing.T) {
	sink, err := NewWavefrontSink(defaultSinkConfig())
	require.NoError(t, err)
	tags := map[string]string{"nodename": "fakeNode"}
	db := metrics.Batch{
		Metrics: []wf.Metric{
			nil,
			wf.NewPoint("cpu.idle", 1.0, 0, "fakeSource", tags),
		},
	}
	sink.Export(&db)
	require.Contains(t, getReceivedLines(sink), `source="fakeNode"`)
}

func TestLeaveSourceAloneIfNoNodeName(t *testing.T) {
	sink, err := NewWavefrontSink(defaultSinkConfig())
	require.NoError(t, err)
	db := metrics.Batch{
		Metrics: []wf.Metric{
			nil,
			wf.NewPoint("cpu.idle", 1.0, 0, "fakeSource", nil),
		},
	}
	sink.Export(&db)
	require.Contains(t, getReceivedLines(sink), `source="fakeSource"`)
}

func TestNilPointDataBatch(t *testing.T) {
	sink, err := NewWavefrontSink(defaultSinkConfig())
	require.NoError(t, err)

	db := metrics.Batch{
		Metrics: []wf.Metric{
			nil,
			wf.NewPoint("cpu.idle", 1.0, 0, "fakeSource", nil),
		},
	}
	sink.Export(&db)
	require.Contains(t, getReceivedLines(sink), "cpu.idle")
}

func TestCleansTagsBeforeSending(t *testing.T) {
	sink, err := NewWavefrontSink(defaultSinkConfig())
	require.NoError(t, err)

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
	require.NotContains(t, getReceivedLines(sink), "emptyTag")
}

func TestEvents(t *testing.T) {
	event := &events.Event{
		Message: "Pulled image",
	}

	t.Run("events disabled", func(t *testing.T) {
		cfg := defaultSinkConfig()
		sink, err := NewWavefrontSink(cfg)
		require.NoError(t, err)

		sink.ExportEvent(event)

		require.Empty(t, getReceivedLines(sink))
	})

	t.Run("events enabled", func(t *testing.T) {
		cfg := defaultSinkConfig()
		*cfg.EnableEvents = true
		sink, err := NewWavefrontSink(cfg)
		require.NoError(t, err)

		sink.ExportEvent(event)

		require.NotEmpty(t, getReceivedLines(sink))
		require.Contains(t, getReceivedLines(sink), "Pulled image")
	})
}

func defaultSinkConfig() configuration.SinkConfig {
	eventsEnabled := false
	cfg := configuration.SinkConfig{
		ProxyAddress:      "wavefront-proxy:2878",
		ClusterName:       "testCluster",
		TestMode:          true,
		EnableEvents:      &eventsEnabled,
		HeartbeatInterval: 1 * time.Minute,
	}
	return cfg
}

func getReceivedLines(sink sinks.Sink) string {
	return strings.TrimSpace(sink.(*wavefrontSink).WavefrontClient.(*TestSender).GetReceivedLines())
}
