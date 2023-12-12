package logging

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	wf "github.com/wavefronthq/observability-for-kubernetes/operator/api/v1alpha1"
	"github.com/wavefronthq/observability-for-kubernetes/operator/components"
	"github.com/wavefronthq/observability-for-kubernetes/operator/components/test"
	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/testhelper/wftest"
	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/util"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var ComponentDir = os.DirFS(filepath.Join("..", DeployDir))

func TestNewLoggingComponent(t *testing.T) {
	t.Run("create config hash", func(t *testing.T) {
		config := validLoggingComponentConfig()
		t.Log(os.Getwd())
		loggingComponent, _ := NewComponent(ComponentDir, config)
		_ = loggingComponent.Validate()
		require.NotEmpty(t, loggingComponent.config.ConfigHash)
	})
}

func TestValidate(t *testing.T) {
	t.Run("valid component config", func(t *testing.T) {
		config := validLoggingComponentConfig()
		loggingComponent, _ := NewComponent(ComponentDir, config)
		result := loggingComponent.Validate()
		require.True(t, result.IsValid())
	})

	t.Run("empty disabled component config is valid", func(t *testing.T) {
		config := ComponentConfig{Enable: false}
		loggingComponent, err := NewComponent(ComponentDir, config)
		result := loggingComponent.Validate()
		require.NoError(t, err)
		require.True(t, result.IsValid())
	})

	t.Run("empty enabled component config is not valid", func(t *testing.T) {
		config := ComponentConfig{ShouldValidate: true}
		loggingComponent, err := NewComponent(ComponentDir, config)
		result := loggingComponent.Validate()
		require.NoError(t, err)
		require.False(t, result.IsValid())
	})

	t.Run("empty cluster name is not valid", func(t *testing.T) {
		config := validLoggingComponentConfig()
		config.ClusterName = ""
		loggingComponent, err := NewComponent(ComponentDir, config)
		result := loggingComponent.Validate()
		require.NoError(t, err)
		require.False(t, result.IsValid())
		require.Equal(t, "logging: missing cluster name", result.Message())
	})

	t.Run("empty namespace is not valid", func(t *testing.T) {
		config := validLoggingComponentConfig()
		config.Namespace = ""
		loggingComponent, err := NewComponent(ComponentDir, config)
		result := loggingComponent.Validate()
		require.NoError(t, err)
		require.False(t, result.IsValid())
		require.Equal(t, "logging: missing namespace", result.Message())
	})

	t.Run("empty logging version is not valid", func(t *testing.T) {
		config := validLoggingComponentConfig()
		config.LoggingVersion = ""
		loggingComponent, err := NewComponent(ComponentDir, config)
		result := loggingComponent.Validate()
		require.NoError(t, err)
		require.False(t, result.IsValid())
		require.Equal(t, "logging: missing log image version", result.Message())
	})

	t.Run("empty image registry is not valid", func(t *testing.T) {
		config := validLoggingComponentConfig()
		config.ImageRegistry = ""
		loggingComponent, err := NewComponent(ComponentDir, config)
		result := loggingComponent.Validate()
		require.NoError(t, err)
		require.False(t, result.IsValid())
		require.Equal(t, "logging: missing image registry", result.Message())
	})

	t.Run("empty proxy address is not valid", func(t *testing.T) {
		config := validLoggingComponentConfig()
		config.ProxyAddress = ""
		loggingComponent, err := NewComponent(ComponentDir, config)
		result := loggingComponent.Validate()
		require.NoError(t, err)
		require.False(t, result.IsValid())
		require.Equal(t, "logging: missing proxy address", result.Message())
	})

	t.Run("proxy address without http is not valid", func(t *testing.T) {
		config := validLoggingComponentConfig()
		config.ProxyAddress = wftest.DefaultProxyAddress
		loggingComponent, err := NewComponent(ComponentDir, config)
		result := loggingComponent.Validate()
		require.NoError(t, err)
		require.False(t, result.IsValid())
		require.Equal(t, "logging: proxy address (wavefront-proxy:2878) must start with http", result.Message())
	})

}

func TestResources(t *testing.T) {
	t.Run("default configuration", func(t *testing.T) {
		loggingComponent, _ := NewComponent(ComponentDir, validLoggingComponentConfig())
		toApply, toDelete, err := loggingComponent.Resources(components.NewK8sResourceBuilder(nil))

		require.NoError(t, err)
		require.Equal(t, 3, len(toApply))
		require.Empty(t, toDelete)

		// check all resources for component labels
		test.RequireCommonLabels(t, toApply, "wavefront", "logging", loggingComponent.config.Namespace)

		ds, err := test.GetDaemonSet(util.LoggingName, toApply)
		require.NoError(t, err)

		require.Equal(t, loggingComponent.config.ConfigHash, ds.Spec.Template.GetObjectMeta().GetAnnotations()["configHash"])
		require.Equal(t, util.LoggingName, ds.Spec.Template.GetLabels()["name"])
		require.Equal(t, "wavefront", ds.Spec.Template.GetLabels()["app.kubernetes.io/name"])
		require.Equal(t, "logging", ds.Spec.Template.GetLabels()["app.kubernetes.io/component"])
		require.Equal(t, "1", ds.Spec.Template.GetAnnotations()["proxy-available-replicas"])
		require.NotEmpty(t, ds.Spec.Template.GetObjectMeta().GetAnnotations()["configHash"])
		require.Equal(t, wftest.DefaultImageRegistry+"/kubernetes-operator-fluentbit:"+loggingComponent.config.LoggingVersion, ds.Spec.Template.Spec.Containers[0].Image)
		require.Equal(t, loggingComponent.config.ClusterName, ds.Spec.Template.Spec.Containers[0].Env[1].Value)

		configMap, err := test.GetConfigMap("wavefront-logging-config", toApply)
		require.NoError(t, err)
		require.NotEmpty(t, configMap)

		fluentBitConfig := fluentBitConfiguration(toApply)
		require.NoError(t, err)
		require.Contains(t, fluentBitConfig, fmt.Sprintf("Proxy             %s", loggingComponent.config.ProxyAddress))

		serviceAccount, err := test.GetServiceAccount(util.LoggingName, toApply)
		require.NoError(t, err)
		require.NotEmpty(t, serviceAccount)
	})

	t.Run("k8s resources are set correctly", func(t *testing.T) {
		config := validLoggingComponentConfig()
		config.Resources.Requests.CPU = "200m"
		config.Resources.Requests.Memory = "10Mi"
		config.Resources.Limits.Memory = "256Mi"
		loggingComponent, _ := NewComponent(ComponentDir, config)
		toApply, _, err := loggingComponent.Resources(components.NewK8sResourceBuilder(nil))

		require.NoError(t, err)
		require.NotEmpty(t, toApply)
		ds, err := test.GetDaemonSet("wavefront-logging", toApply)
		require.NoError(t, err)
		require.Equal(t, "10Mi", ds.Spec.Template.Spec.Containers[0].Resources.Requests.Memory().String())
		require.Equal(t, "200m", ds.Spec.Template.Spec.Containers[0].Resources.Requests.Cpu().String())
		require.Equal(t, "256Mi", ds.Spec.Template.Spec.Containers[0].Resources.Limits.Memory().String())
	})

	t.Run("tag allow list is set correctly", func(t *testing.T) {
		config := validLoggingComponentConfig()
		config.TagAllowList = map[string][]string{"namespace_name": {"kube-sys", "wavefront"}, "pod_name": {"pet-clinic"}}
		loggingComponent, _ := NewComponent(ComponentDir, config)
		toApply, _, err := loggingComponent.Resources(components.NewK8sResourceBuilder(nil))

		require.NoError(t, err)
		require.NotEmpty(t, toApply)

		fluentBitConfig := fluentBitConfiguration(toApply)
		require.NoError(t, err)
		require.Contains(t, fluentBitConfig, "Regex  namespace_name ^kube-sys$|^wavefront$")
		require.Contains(t, fluentBitConfig, "Regex  pod_name ^pet-clinic$")
	})

	t.Run("tags are set correctly", func(t *testing.T) {
		config := validLoggingComponentConfig()
		config.Tags = map[string]string{"key1": "value1", "key2": "value2"}
		loggingComponent, _ := NewComponent(ComponentDir, config)
		toApply, _, err := loggingComponent.Resources(components.NewK8sResourceBuilder(nil))

		require.NoError(t, err)
		require.NotEmpty(t, toApply)

		fluentBitConfig := fluentBitConfiguration(toApply)
		require.NoError(t, err)
		require.Contains(t, fluentBitConfig, "Record          key1 value1")
		require.Contains(t, fluentBitConfig, "Record          key2 value2")
	})

	t.Run("external wavefront proxy url with http specified in URL is set correctly", func(t *testing.T) {
		config := validLoggingComponentConfig()
		config.ProxyAddress = "http://my-proxy:8888"
		loggingComponent, _ := NewComponent(ComponentDir, config)
		toApply, _, err := loggingComponent.Resources(components.NewK8sResourceBuilder(nil))

		require.NoError(t, err)
		require.NotEmpty(t, toApply)

		fluentBitConfig := fluentBitConfiguration(toApply)
		require.NoError(t, err)
		require.Contains(t, fluentBitConfig, "Proxy             http://my-proxy:8888")
	})
}

func fluentBitConfiguration(toApply []client.Object) string {
	configMap, _ := test.GetConfigMap("wavefront-logging-config", toApply)
	fluentBitConfig := configMap.Data["fluent-bit.conf"]
	return fluentBitConfig
}

func validLoggingComponentConfig() ComponentConfig {
	return ComponentConfig{
		Enable:                 true,
		ShouldValidate:         true,
		ClusterName:            wftest.DefaultClusterName,
		ControllerManagerUID:   "asdfgh",
		Namespace:              wftest.DefaultNamespace,
		LoggingVersion:         "2.1.2",
		ImageRegistry:          wftest.DefaultImageRegistry,
		ProxyAddress:           fmt.Sprintf("http://%s", wftest.DefaultProxyAddress),
		ProxyAvailableReplicas: 1,
		Resources: wf.Resources{
			Requests: wf.Resource{
				CPU:    "100m",
				Memory: "50Mi",
			},
			Limits: wf.Resource{
				CPU:    "100m",
				Memory: "50Mi",
			},
		},
	}
}
