package factory

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/configuration"
)

func TestSinkFactoryBuild(t *testing.T) {
	factory := NewSinkFactory()

	t.Run("build with wavefront sink configuration", func(t *testing.T) {
		cfg := defaultWavefrontConfig()

		sink, err := factory.Build(*cfg)

		require.NoError(t, err)
		require.NotNil(t, sink)
		require.Equal(t, "wavefront_sink", sink.Name())
	})

	t.Run("build with k8s event sink configuration", func(t *testing.T) {
		cfg := defaultExternalSinkConfig()

		sink, err := factory.Build(*cfg)

		require.NoError(t, err)
		require.NotNil(t, sink)
		require.Equal(t, "k8s_events_sink", sink.Name())
	})

}

func TestSinkFactoryBuildAll(t *testing.T) {
	factory := NewSinkFactory()

	t.Run("build with wavefront only sink", func(t *testing.T) {
		sinkConfigs := []*configuration.SinkConfig{
			defaultWavefrontConfig(),
		}

		sinks := factory.BuildAll(sinkConfigs)

		require.Equal(t, "wavefront_sink", sinks[0].Name())
	})

	t.Run("build with multiple sinks", func(t *testing.T) {
		sinkConfigs := []*configuration.SinkConfig{
			defaultWavefrontConfig(),
			defaultExternalSinkConfig(),
		}

		sinks := factory.BuildAll(sinkConfigs)

		require.Equal(t, 2, len(sinks))
		require.Equal(t, "wavefront_sink", sinks[0].Name())
		require.Equal(t, "k8s_events_sink", sinks[1].Name())
	})
}

func defaultWavefrontConfig() *configuration.SinkConfig {
	eventsEnabled := true
	return &configuration.SinkConfig{
		ProxyAddress:  "wavefront-proxy:2878",
		TestMode:      true,
		EventsEnabled: &eventsEnabled,
		Transforms: configuration.Transforms{
			Prefix: "test.",
		},
	}
}
func defaultExternalSinkConfig() *configuration.SinkConfig {
	eventsEnabled := false
	return &configuration.SinkConfig{
		Type:                configuration.ExternalSinkType,
		ExternalEndpointURL: "http://example.com",
		EventsEnabled:       &eventsEnabled,
	}
}
