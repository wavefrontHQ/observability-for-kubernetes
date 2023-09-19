package components

import (
	"os"
	"testing"

	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/util"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

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
	t.Run("default configuration", func(t *testing.T) {
		loggingComponent := NewLoggingComponent(validLoggingComponentConfig(), os.DirFS(DeployDir))
		result := loggingComponent.PreprocessAndValidate()
		require.True(t, result.IsValid())

		toApply, toDelete, err := loggingComponent.Resources()
		require.Nil(t, err)
		require.NotEmpty(t, toApply)
		ds, err := GetAppliedDaemonSet("logging", toApply)
		require.Nil(t, err)
		require.Equal(t, loggingComponent.Config.ConfigHash, ds.Spec.Template.GetObjectMeta().GetAnnotations()["configHash"])
		require.Equal(t, ds.Name, util.LoggingName)
		require.Equal(t, ds.Spec.Template.GetLabels()["name"], util.LoggingName)
		require.Equal(t, ds.Spec.Template.GetLabels()["app.kubernetes.io/name"], "wavefront")
		require.Equal(t, ds.Spec.Template.GetLabels()["app.kubernetes.io/component"], "logging")

		require.Equal(t, ds.Namespace, wftest.DefaultNamespace)
		require.Equal(t, ds.Spec.Template.GetAnnotations()["proxy-available-replicas"], "1")
		require.NotEmpty(t, ds.Spec.Template.GetObjectMeta().GetAnnotations()["configHash"])
		require.Equal(t, ds.Spec.Template.Spec.Containers[0].Image, wftest.DefaultImageRegistry+"/kubernetes-operator-fluentbit:"+loggingComponent.Config.LoggingVersion)
		require.Equal(t, ds.Spec.Template.Spec.Containers[0].Env[1].Value, loggingComponent.Config.ClusterName)

		require.Empty(t, toDelete)
	})

}

func validLoggingComponentConfig() LoggingComponentConfig {
	return LoggingComponentConfig{
		ClusterName:            wftest.DefaultClusterName,
		Namespace:              wftest.DefaultNamespace,
		LoggingVersion:         "2.1.2",
		ImageRegistry:          wftest.DefaultImageRegistry,
		ProxyAddress:           wftest.DefaultProxyAddress,
		ProxyAvailableReplicas: 1,
	}
}

func GetAppliedDaemonSet(metadataName string, toApply []client.Object) (appsv1.DaemonSet, error) {

	var found client.Object
	for _, clientObject := range toApply {
		if clientObject.GetObjectKind().GroupVersionKind().Kind == "DaemonSet" {
			found = clientObject
		}
	}

	var daemonSet appsv1.DaemonSet
	unstructuredObj, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(found)

	err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObj, &daemonSet)
	if err != nil {
		return appsv1.DaemonSet{}, err
	}

	return daemonSet, nil
}
