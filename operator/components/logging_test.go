package components

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/testhelper/wftest"
)

func TestProcessAndValidate(t *testing.T) {
	t.Run("component config is not valid", func(t *testing.T) {
		config := LoggingComponentConfig{}
		loggingComponent := NewLoggingComponent(config, os.DirFS(DeployDir))
		result := loggingComponent.PreprocessAndValidate()
		require.False(t, result.IsValid())
	})

	t.Run("create config hash", func(t *testing.T) {
		config := validLoggingComponentConfig()
		loggingComponent := NewLoggingComponent(config, os.DirFS(DeployDir))
		_ = loggingComponent.PreprocessAndValidate()
		require.NotEmpty(t, loggingComponent.Config.ConfigHash)
	})

	t.Run("component config is valid", func(t *testing.T) {
		config := validLoggingComponentConfig()
		loggingComponent := NewLoggingComponent(config, os.DirFS(DeployDir))
		result := loggingComponent.PreprocessAndValidate()
		require.True(t, result.IsValid())
	})
}

func TestResources(t *testing.T) {
	t.Run("component config is not valid", func(t *testing.T) {
		config := LoggingComponentConfig{}
		loggingComponent := NewLoggingComponent(config, os.DirFS(DeployDir))
		toApply, toDelete, err := loggingComponent.Resources()
		require.Nil(t, err)
		require.NotEmpty(t, toApply)
		require.NotEmpty(t, toDelete)
	})

}

func validLoggingComponentConfig() LoggingComponentConfig {
	return LoggingComponentConfig{
		ClusterName:    wftest.DefaultClusterName,
		Namespace:      wftest.DefaultNamespace,
		LoggingVersion: "2.1.2",
		ImageRegistry:  wftest.DefaultImageRegistry,
		ProxyAddress:   wftest.DefaultProxyAddress,
	}
}
