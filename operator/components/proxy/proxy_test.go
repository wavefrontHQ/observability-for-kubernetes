package proxy

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	wf "github.com/wavefronthq/observability-for-kubernetes/operator/api/v1alpha1"
	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/testhelper/wftest"
)

var ComponentDir = os.DirFS(filepath.Join("..", DeployDir))

func TestNewProxyComponent(t *testing.T) {
	t.Run("create config hash", func(t *testing.T) {
		config := validComponentConfig()
		t.Log(os.Getwd())

		component, err := NewComponent(ComponentDir, config)

		require.NoError(t, err)
		require.NotNil(t, component)
		require.NotEmpty(t, component.config.ConfigHash)
	})
}

func TestValidate(t *testing.T) {
	t.Run("valid component config", func(t *testing.T) {
		config := validComponentConfig()
		component, _ := NewComponent(ComponentDir, config)
		result := component.Validate()
		require.True(t, result.IsValid())
	})

	t.Run("empty disabled component config is valid", func(t *testing.T) {
		config := ComponentConfig{Enable: false}
		component, err := NewComponent(ComponentDir, config)
		result := component.Validate()
		require.NoError(t, err)
		require.True(t, result.IsValid())
	})

	t.Run("empty enabled component config is not valid", func(t *testing.T) {
		config := ComponentConfig{Enable: true}
		component, err := NewComponent(ComponentDir, config)
		result := component.Validate()
		require.NoError(t, err)
		require.False(t, result.IsValid())
	})

	t.Run("empty controller manager uid is not valid", func(t *testing.T) {
		config := validComponentConfig()
		config.ControllerManagerUID = ""
		component, err := NewComponent(ComponentDir, config)
		result := component.Validate()
		require.NoError(t, err)
		require.False(t, result.IsValid())
		require.Equal(t, "proxy: missing controller manager uid", result.Message())
	})

	t.Run("empty namespace is not valid", func(t *testing.T) {
		config := validComponentConfig()
		config.Namespace = ""
		component, err := NewComponent(ComponentDir, config)
		result := component.Validate()
		require.NoError(t, err)
		require.False(t, result.IsValid())
		require.Equal(t, "proxy: missing namespace", result.Message())
	})

	t.Run("empty cluster name is not valid", func(t *testing.T) {
		config := validComponentConfig()
		config.ClusterName = ""
		component, err := NewComponent(ComponentDir, config)
		result := component.Validate()
		require.NoError(t, err)
		require.False(t, result.IsValid())
		require.Equal(t, "proxy: missing cluster name", result.Message())
	})

	t.Run("empty cluster uuid is not valid", func(t *testing.T) {
		config := validComponentConfig()
		config.ClusterUUID = ""
		component, err := NewComponent(ComponentDir, config)
		result := component.Validate()
		require.NoError(t, err)
		require.False(t, result.IsValid())
		require.Equal(t, "proxy: missing cluster uuid", result.Message())
	})

	t.Run("empty image registry is not valid", func(t *testing.T) {
		config := validComponentConfig()
		config.ImageRegistry = ""
		component, err := NewComponent(ComponentDir, config)
		result := component.Validate()
		require.NoError(t, err)
		require.False(t, result.IsValid())
		require.Equal(t, "proxy: missing image registry", result.Message())
	})

	t.Run("empty wavefront token secret is not valid", func(t *testing.T) {
		config := validComponentConfig()
		config.WavefrontTokenSecret = ""
		component, err := NewComponent(ComponentDir, config)
		result := component.Validate()
		require.NoError(t, err)
		require.False(t, result.IsValid())
		require.Equal(t, "proxy: missing wavefront token secret", result.Message())
	})

	t.Run("empty wavefront url is not valid", func(t *testing.T) {
		config := validComponentConfig()
		config.WavefrontUrl = ""
		component, err := NewComponent(ComponentDir, config)
		result := component.Validate()
		require.NoError(t, err)
		require.False(t, result.IsValid())
		require.Equal(t, "proxy: missing wavefront url", result.Message())
	})

	t.Run("empty resources is not valid", func(t *testing.T) {
		config := validComponentConfig()
		config.Resources = wf.Resources{}
		component, err := NewComponent(ComponentDir, config)
		result := component.Validate()
		require.NoError(t, err)
		require.False(t, result.IsValid())
		require.Equal(t, "proxy: [invalid wavefront-proxy.resources.limits.memory must be set, invalid wavefront-proxy.resources.limits.cpu must be set]", result.Message())
	})

	t.Run("empty metric port is not valid", func(t *testing.T) {
		config := validComponentConfig()
		config.MetricPort = 0
		component, err := NewComponent(ComponentDir, config)
		result := component.Validate()
		require.NoError(t, err)
		require.False(t, result.IsValid())
		require.Equal(t, "proxy: missing metric port", result.Message())
	})

	t.Run("empty proxy version is not valid", func(t *testing.T) {
		config := validComponentConfig()
		config.ProxyVersion = ""
		component, err := NewComponent(ComponentDir, config)
		result := component.Validate()
		require.NoError(t, err)
		require.False(t, result.IsValid())
		require.Equal(t, "proxy: missing proxy version", result.Message())
	})

	//t.Run("empty secret hash is not valid", func(t *testing.T) {
	//	config := validComponentConfig()
	//	config.SecretHash = ""
	//	component, err := NewComponent(ComponentDir, config)
	//	result := component.Validate()
	//	require.NoError(t, err)
	//	require.False(t, result.IsValid())
	//	require.Equal(t, "proxy: missing secret hash", result.Message())
	//})
}

func TestResources(t *testing.T) {
	//t.Run("default configuration", func(t *testing.T) {
	//	component, _ := NewComponent(ComponentDir, validComponentConfig())
	//	toApply, toDelete, err := component.Resources()
	//
	//	require.NoError(t, err)
	//	require.NotEmpty(t, toApply)
	//	require.Empty(t, toDelete)
	//
	//	// check all resources for component labels
	//	test.RequireCommonLabels(t, toApply, "wavefront", "proxy", util.Namespace)
	//
	//	// cluster name configmap
	//	configmap, err := test.GetAppliedConfigMap("pl-cloud-config", toApply)
	//	require.NoError(t, err)
	//	require.Equal(t, component.config.ClusterName, configmap.Data["PL_CLUSTER_NAME"])
	//
	//	secret, err := test.GetAppliedSecret("pl-cluster-secrets", toApply)
	//	require.NoError(t, err)
	//	require.Equal(t, component.config.ClusterName, secret.StringData["cluster-name"])
	//	require.Equal(t, component.config.ClusterUUID, secret.StringData["cluster-id"])
	//})
}

func validComponentConfig() ComponentConfig {
	return ComponentConfig{
		Enable:               true,
		ControllerManagerUID: "controller-manager-uid",
		Namespace:            wftest.DefaultNamespace,
		ClusterName:          wftest.DefaultClusterName,
		ClusterUUID:          "cluster-uuid",
		ImageRegistry:        wftest.DefaultImageRegistry,
		WavefrontTokenSecret: "secret",
		WavefrontUrl:         "https://example.wavefront.com",
		Resources: wf.Resources{
			Limits: wf.Resource{
				CPU:    "100Mi",
				Memory: "1Gi",
			},
		},
		MetricPort:   2878,
		ProxyVersion: "2.0.0",
		Replicas:     1,
	}
}
