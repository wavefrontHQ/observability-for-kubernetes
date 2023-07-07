package configuration

import (
	"bytes"
	"errors"
	"io"
	"os"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/util"
)

func TestNew(t *testing.T) {
	t.Run("has defaults defaults", func(t *testing.T) {
		cfg, _ := New(func(config *Config) error {
			config.Sources = &SourceConfig{SummaryConfig: &SummarySourceConfig{}}
			config.Sinks = []*SinkConfig{{}}
			return nil
		})

		require.Equal(t, 30*time.Second, cfg.FlushInterval, "default cfg.FlushInterval")
		require.Equal(t, 60*time.Second, cfg.DefaultCollectionInterval, "default cfg.DefaultCollectionInterval")
		require.Equal(t, 20*time.Second, cfg.SinkExportDataTimeout, "default cfg.SinkExportDataTimeout")
		require.Equal(t, "k8s-cluster", cfg.ClusterName, "default cfg.ClusterName")
		require.Equal(t, 5*time.Minute, cfg.DiscoveryConfig.DiscoveryInterval, "default cfg.DiscoveryInterval")
	})

	t.Run("allows custom initialization", func(t *testing.T) {
		cfg, _ := New(func(config *Config) error {
			config.Sources = &SourceConfig{SummaryConfig: &SummarySourceConfig{}}
			config.Sinks = []*SinkConfig{{}}
			config.FlushInterval = 6 * time.Second
			return nil
		})

		require.Equal(t, 6*time.Second, cfg.FlushInterval, "overriding cfg.FlushInterval")
	})

	t.Run("reports custom initialization errors", func(t *testing.T) {
		expectedErr := errors.New("custom initialization error")
		_, actualErr := New(func(config *Config) error {
			return expectedErr
		})

		require.ErrorContains(t, actualErr, expectedErr.Error())
	})

	t.Run("does not allow FlushInterval to be less than 5 seconds", func(t *testing.T) {
		_, actualErr := New(func(config *Config) error {
			config.Sources = &SourceConfig{SummaryConfig: &SummarySourceConfig{}}
			config.Sinks = []*SinkConfig{{}}
			config.FlushInterval = 1 * time.Second
			return nil
		})

		require.ErrorContains(t, actualErr, "metric resolution should not be less than 5 seconds")
	})

	t.Run("does not allow Sources to be nil", func(t *testing.T) {
		_, actualErr := New(func(config *Config) error {
			return nil
		})

		require.ErrorContains(t, actualErr, "missing sources")
	})

	t.Run("does not allow Sources.SummaryConfig to be nil", func(t *testing.T) {
		_, actualErr := New(func(config *Config) error {
			config.Sources = &SourceConfig{}
			return nil
		})

		require.ErrorContains(t, actualErr, "kubernetes_source is missing")
	})

	t.Run("does not allow Sinks to be empty", func(t *testing.T) {
		_, actualErr := New(func(config *Config) error {
			config.Sources = &SourceConfig{SummaryConfig: &SummarySourceConfig{}}
			return nil
		})

		require.ErrorContains(t, actualErr, "missing sink")
	})

	t.Run("sinks have no InternalStatsPrefix when Sources.StatsConfig is not configured", func(t *testing.T) {
		cfg, _ := New(func(config *Config) error {
			config.Sources = &SourceConfig{SummaryConfig: &SummarySourceConfig{}}
			config.Sinks = []*SinkConfig{{}}
			return nil
		})

		require.Empty(t, cfg.Sinks[0].InternalStatsPrefix)
	})

	t.Run("sinks have a 'kubernetes.' InternalStatsPrefix when Sources.StatsConfig is configured", func(t *testing.T) {
		cfg, _ := New(func(config *Config) error {
			config.Sources = &SourceConfig{
				SummaryConfig: &SummarySourceConfig{},
				StatsConfig:   &StatsSourceConfig{},
			}
			config.Sinks = []*SinkConfig{{}}
			return nil
		})

		require.Equal(t, "kubernetes.", cfg.Sinks[0].InternalStatsPrefix)
	})

	t.Run("sinks have InternalStatsPrefix when Sources.StatsConfig.Prefix is configured", func(t *testing.T) {
		cfg, _ := New(func(config *Config) error {
			config.Sources = &SourceConfig{
				SummaryConfig: &SummarySourceConfig{},
				StatsConfig:   &StatsSourceConfig{Transforms: Transforms{Prefix: "custom."}},
			}
			config.Sinks = []*SinkConfig{{}}
			return nil
		})

		require.Equal(t, "custom.", cfg.Sinks[0].InternalStatsPrefix)
	})

	t.Run("sinks have the correct ClusterName", func(t *testing.T) {
		cfg, _ := New(func(config *Config) error {
			config.ClusterName = "some-cluster"
			config.Sources = &SourceConfig{SummaryConfig: &SummarySourceConfig{}}
			config.Sinks = []*SinkConfig{{}}
			return nil
		})

		require.Equal(t, "some-cluster", cfg.Sinks[0].ClusterName)
	})

	t.Run("overrides unset EnableEvents on sinks with the global EnableEvents", func(t *testing.T) {
		cfg, _ := New(func(config *Config) error {
			config.Sources = &SourceConfig{SummaryConfig: &SummarySourceConfig{}}
			config.EnableEvents = true
			config.Sinks = []*SinkConfig{{}}
			return nil
		})

		require.Equal(t, true, *cfg.Sinks[0].EnableEvents)
	})

	t.Run("does not override EnableEvents on sinks when they are set", func(t *testing.T) {
		enabled := true
		cfg, _ := New(func(config *Config) error {
			config.Sources = &SourceConfig{SummaryConfig: &SummarySourceConfig{}}
			config.EnableEvents = false
			config.Sinks = []*SinkConfig{{EnableEvents: &enabled}}
			return nil
		})

		require.Equal(t, true, *cfg.Sinks[0].EnableEvents)
	})
}

func TestLoadOrDie(t *testing.T) {
	t.Run("returns nil when the file is empty", func(t *testing.T) {
		require.Nil(t, LoadOrDie(""))
	})

	t.Run("loads config from a file", func(t *testing.T) {
		expectedUUID := "c246955e-21ff-4bc6-9b30-8479ea7f218c"
		_ = os.Setenv(util.ClusterUUIDEnvVar, expectedUUID)
		defer os.Unsetenv(util.ClusterUUIDEnvVar)

		configFile, _ := os.CreateTemp(os.TempDir(), "collector-config*.yaml")
		const testConfig = `
clusterName: new-collector
enableEvents: true

sinks:
- externalEndpointURL: 'https://example.com'
  type: external
- proxyAddress: "foobar"

sources:
  kubernetes_source: {}
`
		_, _ = io.Copy(configFile, bytes.NewBufferString(testConfig))
		configFile.Close()

		cfg := LoadOrDie(configFile.Name())

		require.Equal(t, "new-collector", cfg.ClusterName)
		require.Equal(t, "new-collector", cfg.EventsConfig.ClusterName)
		require.Equal(t, expectedUUID, cfg.EventsConfig.ClusterUUID)
		require.Equal(t, true, *cfg.Sinks[0].EnableEvents)
		require.Equal(t, true, *cfg.Sinks[1].EnableEvents)
	})

	t.Run("dies when the file cannot be parsed", func(t *testing.T) {
		exitCode := 0
		oldExit := log.StandardLogger().ExitFunc
		log.StandardLogger().ExitFunc = func(code int) {
			exitCode = code
		}
		defer func() {
			log.StandardLogger().ExitFunc = oldExit
		}()

		LoadOrDie("does-not-exist.yaml")

		require.Equal(t, 1, exitCode)
	})
}
