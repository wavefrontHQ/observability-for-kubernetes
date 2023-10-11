package test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetDaemonSet(metadataName string, objects []client.Object) (appsv1.DaemonSet, error) {
	var daemonSet appsv1.DaemonSet
	var found client.Object

	unstructuredObj, err := findUnstructured("DaemonSet", metadataName, objects, found)
	if err != nil {
		return daemonSet, err
	}

	err = runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObj, &daemonSet)
	if err != nil {
		return daemonSet, err
	}

	return daemonSet, nil
}

func GetDeployment(metadataName string, objects []client.Object) (appsv1.Deployment, error) {
	var deployment appsv1.Deployment
	var found client.Object

	unstructuredObj, err := findUnstructured("Deployment", metadataName, objects, found)
	if err != nil {
		return deployment, err
	}

	err = runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObj, &deployment)
	if err != nil {
		return deployment, err
	}

	return deployment, nil
}

func GetJob(metadataName string, objects []client.Object) (batchv1.Job, error) {
	var job batchv1.Job
	var found client.Object

	unstructuredObj, err := findUnstructured("Job", metadataName, objects, found)
	if err != nil {
		return job, err
	}

	err = runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObj, &job)
	if err != nil {
		return job, err
	}

	return job, nil
}

func GetPersistentVolumeClaim(metadataName string, objects []client.Object) (v1.PersistentVolumeClaim, error) {
	var pvc v1.PersistentVolumeClaim
	var found client.Object

	unstructuredObj, err := findUnstructured("PersistentVolumeClaim", metadataName, objects, found)
	if err != nil {
		return pvc, err
	}

	err = runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObj, &pvc)
	if err != nil {
		return pvc, err
	}

	return pvc, nil
}

func GetService(metadataName string, objects []client.Object) (v1.Service, error) {
	var service v1.Service
	var found client.Object

	unstructuredObj, err := findUnstructured("Service", metadataName, objects, found)
	if err != nil {
		return service, err
	}

	err = runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObj, &service)
	if err != nil {
		return service, err
	}

	return service, nil
}

func GetServiceAccount(metadataName string, objects []client.Object) (v1.ServiceAccount, error) {
	var serviceAccount v1.ServiceAccount
	var found client.Object

	unstructuredObj, err := findUnstructured("ServiceAccount", metadataName, objects, found)
	if err != nil {
		return serviceAccount, err
	}

	err = runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObj, &serviceAccount)
	if err != nil {
		return serviceAccount, err
	}

	return serviceAccount, nil
}

func GetStatefulSet(metadataName string, objects []client.Object) (appsv1.StatefulSet, error) {
	var statefulSet appsv1.StatefulSet
	var found client.Object

	unstructuredObj, err := findUnstructured("StatefulSet", metadataName, objects, found)
	if err != nil {
		return statefulSet, err
	}

	err = runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObj, &statefulSet)
	if err != nil {
		return statefulSet, err
	}

	return statefulSet, nil
}

func GetConfigMap(metadataName string, objects []client.Object) (v1.ConfigMap, error) {
	var configMap v1.ConfigMap
	var found client.Object

	unstructuredObj, err := findUnstructured("ConfigMap", metadataName, objects, found)
	if err != nil {
		return configMap, err
	}

	err = runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObj, &configMap)
	if err != nil {
		return configMap, err
	}

	return configMap, nil
}

func GetSecret(metadataName string, objects []client.Object) (v1.Secret, error) {
	var secret v1.Secret
	var found client.Object

	unstructuredObj, err := findUnstructured("Secret", metadataName, objects, found)
	if err != nil {
		return secret, err
	}

	err = runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObj, &secret)
	if err != nil {
		return secret, err
	}

	return secret, nil
}

func findUnstructured(kind, metadataName string, objects []client.Object, found client.Object) (map[string]interface{}, error) {
	for _, clientObject := range objects {
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

func GetContainer(name string, containers []v1.Container) v1.Container {
	for _, container := range containers {
		if container.Name == name {
			return container
		}
	}
	panic(fmt.Sprintf("container %s not found", name))
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

func RequireEnv(t *testing.T, expectedValue string, name string, container v1.Container) {
	require.Equalf(t, expectedValue, GetENVValue(name, container.Env), `expected %s to be set to "%s" on %s container`, name, expectedValue, container.Name)
}
