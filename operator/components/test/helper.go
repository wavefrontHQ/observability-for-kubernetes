package test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetAppliedDaemonSet(metadataName string, toApply []client.Object) (appsv1.DaemonSet, error) {
	var daemonSet appsv1.DaemonSet
	var found client.Object

	unstructuredObj, err := findUnstructured("DaemonSet", metadataName, toApply, found)
	if err != nil {
		return daemonSet, err
	}

	err = runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObj, &daemonSet)
	if err != nil {
		return daemonSet, err
	}

	return daemonSet, nil
}

func GetAppliedDeployment(metadataName string, toApply []client.Object) (appsv1.Deployment, error) {
	var deployment appsv1.Deployment
	var found client.Object

	unstructuredObj, err := findUnstructured("Deployment", metadataName, toApply, found)
	if err != nil {
		return deployment, err
	}

	err = runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObj, &deployment)
	if err != nil {
		return deployment, err
	}

	return deployment, nil
}

func GetAppliedService(metadataName string, toApply []client.Object) (v1.Service, error) {
	var service v1.Service
	var found client.Object

	unstructuredObj, err := findUnstructured("Service", metadataName, toApply, found)
	if err != nil {
		return service, err
	}

	err = runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObj, &service)
	if err != nil {
		return service, err
	}

	return service, nil
}

func GetAppliedServiceAccount(metadataName string, toApply []client.Object) (v1.ServiceAccount, error) {
	var serviceAccount v1.ServiceAccount
	var found client.Object

	unstructuredObj, err := findUnstructured("ServiceAccount", metadataName, toApply, found)
	if err != nil {
		return serviceAccount, err
	}

	err = runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObj, &serviceAccount)
	if err != nil {
		return serviceAccount, err
	}

	return serviceAccount, nil
}

func GetAppliedStatefulSet(metadataName string, toApply []client.Object) (appsv1.StatefulSet, error) {
	var statefulSet appsv1.StatefulSet
	var found client.Object

	unstructuredObj, err := findUnstructured("StatefulSet", metadataName, toApply, found)
	if err != nil {
		return statefulSet, err
	}

	err = runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObj, &statefulSet)
	if err != nil {
		return statefulSet, err
	}

	return statefulSet, nil
}

func GetAppliedConfigMap(metadataName string, toApply []client.Object) (v1.ConfigMap, error) {
	var configMap v1.ConfigMap
	var found client.Object

	unstructuredObj, err := findUnstructured("ConfigMap", metadataName, toApply, found)
	if err != nil {
		return configMap, err
	}

	err = runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObj, &configMap)
	if err != nil {
		return configMap, err
	}

	return configMap, nil
}

func GetAppliedSecret(metadataName string, toApply []client.Object) (v1.Secret, error) {
	var secret v1.Secret
	var found client.Object

	unstructuredObj, err := findUnstructured("Secret", metadataName, toApply, found)
	if err != nil {
		return secret, err
	}

	err = runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObj, &secret)
	if err != nil {
		return secret, err
	}

	return secret, nil
}

func findUnstructured(kind, metadataName string, toApply []client.Object, found client.Object) (map[string]interface{}, error) {
	for _, clientObject := range toApply {
		if clientObject.GetObjectKind().GroupVersionKind().Kind == kind && clientObject.GetName() == metadataName {
			found = clientObject
		}
	}

	if found == nil {
		return nil, fmt.Errorf("%s with name:%s, not found", kind, metadataName)
	}

	unstructuredObj, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(found)
	return unstructuredObj, nil
}

func GetENVValue(name string, vars []v1.EnvVar) string {
	for _, envVar := range vars {
		if envVar.Name == name {
			return envVar.Value
		}
	}
	return ""
}
func ENVVarExists(name string, vars []v1.EnvVar) bool {
	for _, envVar := range vars {
		if envVar.Name == name {
			return true
		}
	}
	return false
}

func RequireCommonLabels(t *testing.T, objects []client.Object, appName, componentName, ns string) {
	for _, clientObject := range objects {
		require.Equal(t, componentName, clientObject.GetLabels()["app.kubernetes.io/component"])
		require.Equal(t, appName, clientObject.GetLabels()["app.kubernetes.io/name"])
		require.Equal(t, ns, clientObject.GetNamespace())

		kind := clientObject.GetObjectKind().GroupVersionKind().Kind
		if kind == "DaemonSet" || kind == "Deployment" || kind == "StatefulSet" {
			unstructuredObject := clientObject.(*unstructured.Unstructured)

			templateLabels, _, _ := unstructured.NestedStringMap(unstructuredObject.Object, "spec", "template", "metadata", "labels")
			require.Equal(t, componentName, templateLabels["app.kubernetes.io/component"])
			require.Equal(t, appName, templateLabels["app.kubernetes.io/name"])
		}
	}
}
