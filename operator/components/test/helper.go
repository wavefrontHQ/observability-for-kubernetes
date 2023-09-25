package test

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetAppliedDaemonSet(metadataName string, toApply []client.Object) (appsv1.DaemonSet, error) {
	var daemonSet appsv1.DaemonSet
	var found client.Object

	for _, clientObject := range toApply {
		if clientObject.GetObjectKind().GroupVersionKind().Kind == "DaemonSet" && clientObject.GetName() == metadataName {
			found = clientObject
		}
	}

	if found == nil {
		return daemonSet, fmt.Errorf("DaemonSet with name:%s, not found", metadataName)
	}

	unstructuredObj, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(found)

	err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObj, &daemonSet)
	if err != nil {
		return daemonSet, err
	}

	return daemonSet, nil
}

func GetAppliedDeployment(metadataName string, toApply []client.Object) (appsv1.Deployment, error) {
	var deployment appsv1.Deployment
	var found client.Object

	for _, clientObject := range toApply {
		if clientObject.GetObjectKind().GroupVersionKind().Kind == "Deployment" && clientObject.GetName() == metadataName {
			found = clientObject
		}
	}

	if found == nil {
		return deployment, fmt.Errorf("Deployment with name:%s, not found", metadataName)
	}

	unstructuredObj, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(found)

	err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObj, &deployment)
	if err != nil {
		return deployment, err
	}

	return deployment, nil
}

func GetAppliedStatefulSet(metadataName string, toApply []client.Object) (appsv1.StatefulSet, error) {
	var statefulSet appsv1.StatefulSet
	var found client.Object

	for _, clientObject := range toApply {
		if clientObject.GetObjectKind().GroupVersionKind().Kind == "StatefulSet" && clientObject.GetName() == metadataName {
			found = clientObject
		}
	}

	if found == nil {
		return statefulSet, fmt.Errorf("StatefulSet with name:%s, not found", metadataName)
	}

	unstructuredObj, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(found)

	err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObj, &statefulSet)
	if err != nil {
		return statefulSet, err
	}

	return statefulSet, nil
}

func GetAppliedConfigMap(metadataName string, toApply []client.Object) (v1.ConfigMap, error) {
	var configMap v1.ConfigMap
	var found client.Object

	for _, clientObject := range toApply {
		if clientObject.GetObjectKind().GroupVersionKind().Kind == "ConfigMap" && clientObject.GetName() == metadataName {
			found = clientObject
		}
	}

	if found == nil {
		return configMap, fmt.Errorf("ConfigMap with name:%s, not found", metadataName)
	}

	unstructuredObj, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(found)

	err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObj, &configMap)
	if err != nil {
		return configMap, err
	}

	return configMap, nil
}

func GetAppliedSecret(metadataName string, toApply []client.Object) (v1.Secret, error) {
	var secret v1.Secret
	var found client.Object

	for _, clientObject := range toApply {
		if clientObject.GetObjectKind().GroupVersionKind().Kind == "Secret" && clientObject.GetName() == metadataName {
			found = clientObject
		}
	}

	if found == nil {
		return secret, fmt.Errorf("Secret with name:%s, not found", metadataName)
	}

	unstructuredObj, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(found)

	err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObj, &secret)
	if err != nil {
		return secret, err
	}

	return secret, nil
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
