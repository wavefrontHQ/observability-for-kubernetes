package factory

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/configuration"
)

func TestSinkFactoryBuild(t *testing.T) {
	factory := NewSinkFactory()

	t.Run("build with wavefront sink configuration", func(t *testing.T) {
		cfg := configuration.SinkConfig{
			ProxyAddress: "wavefront-proxy:2878",
			TestMode:     true,
			Transforms: configuration.Transforms{
				Prefix: "test.",
			},
		}
		sink, err := factory.Build(cfg)
		require.NoError(t, err)
		require.NotNil(t, sink)
		require.Equal(t, "wavefront_sink", sink.Name())
	})

	t.Run("build with k8s event sink configuration", func(t *testing.T) {
		cfg := configuration.SinkConfig{
			Type:                      configuration.K8sEventsSinkType,
			EventsExternalEndpointURL: "http://example.com",
		}
		sink, err := factory.Build(cfg)
		require.NoError(t, err)
		require.NotNil(t, sink)
		require.Equal(t, "k8s_events_sink", sink.Name())
	})

}
func TestSinkFactoryBuildAll(t *testing.T) {
	factory := NewSinkFactory()

	t.Run("build with wavefront only sink", func(t *testing.T) {
		sinkConfigs := make([]*configuration.SinkConfig, 0)
		cfg := &configuration.SinkConfig{
			ProxyAddress: "wavefront-proxy:2878",
			TestMode:     true,
			Transforms: configuration.Transforms{
				Prefix: "test.",
			},
		}
		sinkConfigs = append(sinkConfigs, cfg)
		sinks := factory.BuildAll(sinkConfigs)
		require.Equal(t, "wavefront_sink", sinks[0].Name())
	})

	t.Run("build with multiple sinks", func(t *testing.T) {
		sinkConfigs := make([]*configuration.SinkConfig, 0)
		cfg := &configuration.SinkConfig{
			ProxyAddress: "wavefront-proxy:2878",
			TestMode:     true,
			Transforms: configuration.Transforms{
				Prefix: "test.",
			},
		}
		sinkConfigs = append(sinkConfigs, cfg)

		cfg = &configuration.SinkConfig{
			Type:                      configuration.K8sEventsSinkType,
			EventsExternalEndpointURL: "http://example.com",
		}
		sinkConfigs = append(sinkConfigs, cfg)

		sinks := factory.BuildAll(sinkConfigs)
		require.Equal(t, 2, len(sinks))
		require.Equal(t, "wavefront_sink", sinks[0].Name())
		require.Equal(t, "k8s_events_sink", sinks[1].Name())
	})
}
