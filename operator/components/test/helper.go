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

func GetAppliedConfigMap(metadataName string, toApply []client.Object) (v1.ConfigMap, error) {
	var configMap v1.ConfigMap
	var found client.Object

	for _, clientObject := range toApply {
		if clientObject.GetObjectKind().GroupVersionKind().Kind == "ConfigMap" && clientObject.GetName() == metadataName {
			found = clientObject
		}
	}

	if found == nil {
		return configMap, fmt.Errorf("DaemonSet with name:%s, not found", metadataName)
	}

	unstructuredObj, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(found)

	err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObj, &configMap)
	if err != nil {
		return configMap, err
	}

	return configMap, nil
}
