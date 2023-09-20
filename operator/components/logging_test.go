package components

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wavefronthq/observability-for-kubernetes/operator/components/test"
	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/testhelper/wftest"
	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/util"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestNewLoggingComponent(t *testing.T) {
	t.Run("create config hash", func(t *testing.T) {
		config := validLoggingComponentConfig()
		loggingComponent, _ := NewLoggingComponent(config, os.DirFS(DeployDir))
		_ = loggingComponent.Validate()
		require.NotEmpty(t, loggingComponent.Config.ConfigHash)
	})
}

func TestProcessAndValidate(t *testing.T) {
	t.Run("valid component config", func(t *testing.T) {
		config := validLoggingComponentConfig()
		loggingComponent, _ := NewLoggingComponent(config, os.DirFS(DeployDir))
		result := loggingComponent.Validate()
		require.True(t, result.IsValid())
	})

	t.Run("empty component config is not valid", func(t *testing.T) {
		config := LoggingComponentConfig{}
		loggingComponent, err := NewLoggingComponent(config, os.DirFS(DeployDir))
		result := loggingComponent.Validate()
		require.Nil(t, err)
		require.False(t, result.IsValid())
	})

	t.Run("empty cluster name is not valid", func(t *testing.T) {
		config := validLoggingComponentConfig()
		config.ClusterName = ""
		loggingComponent, err := NewLoggingComponent(config, os.DirFS(DeployDir))
		result := loggingComponent.Validate()
		require.Nil(t, err)
		require.False(t, result.IsValid())
		require.Equal(t, "logging: missing cluster name", result.Message())
	})

	t.Run("empty namespace is not valid", func(t *testing.T) {
		config := validLoggingComponentConfig()
		config.Namespace = ""
		loggingComponent, err := NewLoggingComponent(config, os.DirFS(DeployDir))
		result := loggingComponent.Validate()
		require.Nil(t, err)
		require.False(t, result.IsValid())
		require.Equal(t, "logging: missing namespace", result.Message())
	})

	t.Run("empty logging version is not valid", func(t *testing.T) {
		config := validLoggingComponentConfig()
		config.LoggingVersion = ""
		loggingComponent, err := NewLoggingComponent(config, os.DirFS(DeployDir))
		result := loggingComponent.Validate()
		require.Nil(t, err)
		require.False(t, result.IsValid())
		require.Equal(t, "logging: missing log image version", result.Message())
	})

	t.Run("empty image registry is not valid", func(t *testing.T) {
		config := validLoggingComponentConfig()
		config.ImageRegistry = ""
		loggingComponent, err := NewLoggingComponent(config, os.DirFS(DeployDir))
		result := loggingComponent.Validate()
		require.Nil(t, err)
		require.False(t, result.IsValid())
		require.Equal(t, "logging: missing image registry", result.Message())
	})

	t.Run("empty proxy address is not valid", func(t *testing.T) {
		config := validLoggingComponentConfig()
		config.ProxyAddress = ""
		loggingComponent, err := NewLoggingComponent(config, os.DirFS(DeployDir))
		result := loggingComponent.Validate()
		require.Nil(t, err)
		require.False(t, result.IsValid())
		require.Equal(t, "logging: missing proxy address", result.Message())
	})

	t.Run("proxy address without http is not valid", func(t *testing.T) {
		config := validLoggingComponentConfig()
		config.ProxyAddress = wftest.DefaultProxyAddress
		loggingComponent, err := NewLoggingComponent(config, os.DirFS(DeployDir))
		result := loggingComponent.Validate()
		require.Nil(t, err)
		require.False(t, result.IsValid())
		require.Equal(t, "logging: proxy address (wavefront-proxy:2878) must start with http", result.Message())
	})

}

func TestResources(t *testing.T) {
	t.Run("default configuration", func(t *testing.T) {
		loggingComponent, _ := NewLoggingComponent(validLoggingComponentConfig(), os.DirFS(DeployDir))
		toApply, toDelete, err := loggingComponent.Resources()

		require.Nil(t, err)
		require.NotEmpty(t, toApply)
		require.Empty(t, toDelete)

		// daemonSet
		ds, err := test.GetAppliedDaemonSet(util.LoggingName, toApply)
		require.Nil(t, err)

		require.Equal(t, loggingComponent.Config.ConfigHash, ds.Spec.Template.GetObjectMeta().GetAnnotations()["configHash"])
		require.Equal(t, util.LoggingName, ds.Spec.Template.GetLabels()["name"])
		require.Equal(t, "wavefront", ds.Spec.Template.GetLabels()["app.kubernetes.io/name"])
		require.Equal(t, "logging", ds.Spec.Template.GetLabels()["app.kubernetes.io/component"])
		require.Equal(t, wftest.DefaultNamespace, ds.Namespace)
		require.Equal(t, "1", ds.Spec.Template.GetAnnotations()["proxy-available-replicas"])
		require.NotEmpty(t, ds.Spec.Template.GetObjectMeta().GetAnnotations()["configHash"])
		require.Equal(t, wftest.DefaultImageRegistry+"/kubernetes-operator-fluentbit:"+loggingComponent.Config.LoggingVersion, ds.Spec.Template.Spec.Containers[0].Image)
		require.Equal(t, loggingComponent.Config.ClusterName, ds.Spec.Template.Spec.Containers[0].Env[1].Value)

		// configMap
		configMap, err := test.GetAppliedConfigMap("wavefront-logging-config", toApply)
		require.Nil(t, err)

		require.Equal(t, "wavefront", configMap.GetLabels()["app.kubernetes.io/name"])
		require.Equal(t, "logging", configMap.GetLabels()["app.kubernetes.io/component"])
		require.Equal(t, wftest.DefaultNamespace, configMap.Namespace)

		fluentBitConfig := fluentBitConfiguration(err, toApply)
		require.Nil(t, err)
		require.Contains(t, fluentBitConfig, fmt.Sprintf("Proxy             %s", loggingComponent.Config.ProxyAddress))
	})

	t.Run("k8s resources are set correctly", func(t *testing.T) {
		config := validLoggingComponentConfig()
		config.Resources.Requests.CPU = "200m"
		config.Resources.Requests.Memory = "10Mi"
		config.Resources.Limits.Memory = "256Mi"
		loggingComponent, _ := NewLoggingComponent(config, os.DirFS(DeployDir))
		toApply, _, err := loggingComponent.Resources()

		require.Nil(t, err)
		require.NotEmpty(t, toApply)
		ds, err := test.GetAppliedDaemonSet("wavefront-logging", toApply)
		require.Nil(t, err)
		require.Equal(t, "10Mi", ds.Spec.Template.Spec.Containers[0].Resources.Requests.Memory().String())
		require.Equal(t, "200m", ds.Spec.Template.Spec.Containers[0].Resources.Requests.Cpu().String())
		require.Equal(t, "256Mi", ds.Spec.Template.Spec.Containers[0].Resources.Limits.Memory().String())
	})

	t.Run("tag allow list is set correctly", func(t *testing.T) {
		config := validLoggingComponentConfig()
		config.TagAllowList = map[string][]string{"namespace_name": {"kube-sys", "wavefront"}, "pod_name": {"pet-clinic"}}
		loggingComponent, _ := NewLoggingComponent(config, os.DirFS(DeployDir))
		toApply, _, err := loggingComponent.Resources()

		require.Nil(t, err)
		require.NotEmpty(t, toApply)

		fluentBitConfig := fluentBitConfiguration(err, toApply)
		require.Nil(t, err)
		require.Contains(t, fluentBitConfig, "Regex  namespace_name ^kube-sys$|^wavefront$")
		require.Contains(t, fluentBitConfig, "Regex  pod_name ^pet-clinic$")
	})

	t.Run("tags are set correctly", func(t *testing.T) {
		config := validLoggingComponentConfig()
		config.Tags = map[string]string{"key1": "value1", "key2": "value2"}
		loggingComponent, _ := NewLoggingComponent(config, os.DirFS(DeployDir))
		toApply, _, err := loggingComponent.Resources()

		require.Nil(t, err)
		require.NotEmpty(t, toApply)

		fluentBitConfig := fluentBitConfiguration(err, toApply)
		require.Nil(t, err)
		require.Contains(t, fluentBitConfig, "Record          key1 value1")
		require.Contains(t, fluentBitConfig, "Record          key2 value2")
	})

	t.Run("external wavefront proxy url with http specified in URL is set correctly", func(t *testing.T) {
		config := validLoggingComponentConfig()
		config.ProxyAddress = "http://my-proxy:8888"
		loggingComponent, _ := NewLoggingComponent(config, os.DirFS(DeployDir))
		toApply, _, err := loggingComponent.Resources()

		require.Nil(t, err)
		require.NotEmpty(t, toApply)

		fluentBitConfig := fluentBitConfiguration(err, toApply)
		require.Nil(t, err)
		require.Contains(t, fluentBitConfig, "Proxy             http://my-proxy:8888")
	})

	//TODO - Component Refactor - move over most of the component level tests from wavefront_controller_test#TestReconcileLogging
}

func fluentBitConfiguration(err error, toApply []client.Object) string {
	configMap, err := test.GetAppliedConfigMap("wavefront-logging-config", toApply)
	fluentBitConfig := configMap.Data["fluent-bit.conf"]
	return fluentBitConfig
}

func validLoggingComponentConfig() LoggingComponentConfig {
	return LoggingComponentConfig{
		ClusterName:            wftest.DefaultClusterName,
		Namespace:              wftest.DefaultNamespace,
		LoggingVersion:         "2.1.2",
		ImageRegistry:          wftest.DefaultImageRegistry,
		ProxyAddress:           fmt.Sprintf("http://%s", wftest.DefaultProxyAddress),
		ProxyAvailableReplicas: 1,
	}
}
