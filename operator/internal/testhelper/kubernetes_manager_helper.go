package testhelper

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	yaml2 "gopkg.in/yaml.v2"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	yaml "k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type MockKubernetesManager struct {
	deletedYAMLs []string
	appliedYAMLs []string
}

func NewMockKubernetesManager() *MockKubernetesManager {
	return &MockKubernetesManager{}
}

func (skm *MockKubernetesManager) ForAllAppliedYAMLs(do func(appliedYAML client.Object)) {
	var resourceDecoder = yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
	for _, appliedYAML := range skm.appliedYAMLs {
		appliedObject := &unstructured.Unstructured{}
		_, _, err := resourceDecoder.Decode([]byte(appliedYAML), nil, appliedObject)
		if err != nil {
			panic(err)
		}
		do(appliedObject)
	}
}

func (skm *MockKubernetesManager) ApplyResources(resources []client.Object) error {
	resourceYAMLs, err := skm.toResourceYAMLs(resources)
	if err != nil {
		return err
	}
	skm.appliedYAMLs = resourceYAMLs
	return nil
}

func (skm *MockKubernetesManager) toResourceYAMLs(resources []client.Object) ([]string, error) {
	var resourceYAMLs []string
	buf := bytes.NewBuffer(nil)
	for _, resource := range resources {
		buf.Reset()
		err := yaml2.NewEncoder(buf).Encode(resource.(*unstructured.Unstructured).Object)
		if err != nil {
			return nil, fmt.Errorf(
				"error encoding %s %s/%s: %s",
				resource.GetObjectKind().GroupVersionKind().Kind, resource.GetNamespace(), resource.GetName(), err,
			)
		}
		resourceYAMLs = append(resourceYAMLs, buf.String())
	}
	return resourceYAMLs, nil
}

func (skm *MockKubernetesManager) DeleteResources(resources []client.Object) error {
	resourceYAMLs, err := skm.toResourceYAMLs(resources)
	if err != nil {
		return err
	}
	skm.deletedYAMLs = resourceYAMLs
	return nil
}

func (skm MockKubernetesManager) AppliedContains(
	apiVersion,
	kind,
	appKubernetesIOName,
	appKubernetesIOComponent,
	metadataName string,
	otherChecks ...string,
) bool {
	return contains(
		skm.appliedYAMLs,
		apiVersion,
		kind,
		appKubernetesIOName,
		appKubernetesIOComponent,
		metadataName,
		otherChecks...,
	)
}

func (skm MockKubernetesManager) DeletedContains(
	apiVersion,
	kind,
	appKubernetesIOName,
	appKubernetesIOComponent,
	metadataName string,
	otherChecks ...string,
) bool {
	return contains(
		skm.deletedYAMLs,
		apiVersion,
		kind,
		appKubernetesIOName,
		appKubernetesIOComponent,
		metadataName,
		otherChecks...,
	)
}

func (skm MockKubernetesManager) GetAppliedYAML(
	apiVersion,
	kind,
	appKubernetesIOName,
	appKubernetesIOComponent,
	metadataName string,
	otherChecks ...string,
) (*unstructured.Unstructured, error) {
	for _, yamlStr := range skm.appliedYAMLs {
		object, err := unstructuredFromStr(yamlStr)
		if err != nil {
			return nil, err
		}

		if objectMatchesAll(
			object,
			apiVersion,
			kind,
			appKubernetesIOName,
			appKubernetesIOComponent,
			metadataName,
		) {
			for _, other := range otherChecks {
				if !strings.Contains(yamlStr, other) {
					return nil, errors.New("no YAML matched conditions passed")
				}
			}
			return object, err
		}
	}
	return nil, nil
}

func objectMatchesAll(
	object *unstructured.Unstructured,
	apiVersion string,
	kind string,
	appKubernetesIOName string,
	appKubernetesIOComponent string,
	metadataName string,
) bool {
	if object.Object["apiVersion"] != apiVersion {
		return false
	}

	if object.Object["kind"] != kind {
		return false
	}

	if len(appKubernetesIOName) > 0 {
		objectAppK8sIOName, found, err := unstructured.NestedString(object.Object, "metadata", "labels", "app.kubernetes.io/name")
		if objectAppK8sIOName != appKubernetesIOName || !found || err != nil {
			return false
		}
	}

	if len(appKubernetesIOComponent) > 0 {
		objectAppK8sIOComponent, found, err := unstructured.NestedString(object.Object, "metadata", "labels", "app.kubernetes.io/component")
		if objectAppK8sIOComponent != appKubernetesIOComponent || !found || err != nil {
			return false
		}
	}

	objectMetadataName, found, err := unstructured.NestedString(object.Object, "metadata", "name")
	if objectMetadataName != metadataName || !found || err != nil {
		return false
	}
	return true
}

func unstructuredFromStr(yamlStr string) (*unstructured.Unstructured, error) {
	object := &unstructured.Unstructured{}
	var resourceDecoder = yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
	_, _, err := resourceDecoder.Decode([]byte(yamlStr), nil, object)
	return object, err
}

func (skm MockKubernetesManager) GetAppliedServiceAccount(appKubernetesIOComponent, metadataName string) (corev1.ServiceAccount, error) {
	yamlUnstructured, err := skm.GetAppliedYAML(
		"v1",
		"ServiceAccount",
		"wavefront",
		appKubernetesIOComponent,
		metadataName,
	)
	if err != nil {
		return corev1.ServiceAccount{}, err
	}

	var serviceAccount corev1.ServiceAccount
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(yamlUnstructured.Object, &serviceAccount)
	if err != nil {
		return corev1.ServiceAccount{}, err
	}

	return serviceAccount, nil
}

func (skm MockKubernetesManager) GetAppliedConfigMap(appKubernetesIOComponent, metadataName string) (corev1.ConfigMap, error) {
	yamlUnstructured, err := skm.GetAppliedYAML(
		"v1",
		"ConfigMap",
		"wavefront",
		appKubernetesIOComponent,
		metadataName,
	)
	if err != nil {
		return corev1.ConfigMap{}, err
	}

	var configMap corev1.ConfigMap
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(yamlUnstructured.Object, &configMap)
	if err != nil {
		return corev1.ConfigMap{}, err
	}

	return configMap, nil
}

func (skm MockKubernetesManager) GetAppliedDaemonSet(appKubernetesIOComponent, metadataName string) (appsv1.DaemonSet, error) {
	yamlUnstructured, err := skm.GetAppliedYAML(
		"apps/v1",
		"DaemonSet",
		"wavefront",
		appKubernetesIOComponent,
		metadataName,
	)
	if err != nil {
		return appsv1.DaemonSet{}, err
	}

	var daemonSet appsv1.DaemonSet
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(yamlUnstructured.Object, &daemonSet)
	if err != nil {
		return appsv1.DaemonSet{}, err
	}

	return daemonSet, nil
}

func (skm MockKubernetesManager) GetAppliedDeployment(appKubernetesIOComponent, metadataName string) (appsv1.Deployment, error) {
	yamlUnstructured, err := skm.GetAppliedYAML(
		"apps/v1",
		"Deployment",
		"wavefront",
		appKubernetesIOComponent,
		metadataName,
	)
	if err != nil {
		return appsv1.Deployment{}, err
	}

	var deployment appsv1.Deployment
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(yamlUnstructured.Object, &deployment)
	if err != nil {
		return appsv1.Deployment{}, err
	}

	return deployment, nil
}

func (skm MockKubernetesManager) GetAppliedService(appKubernetesIOComponent, metadataName string) (corev1.Service, error) {
	yamlUnstructured, err := skm.GetAppliedYAML(
		"v1",
		"Service",
		"wavefront",
		appKubernetesIOComponent,
		metadataName,
	)
	if err != nil {
		return corev1.Service{}, err
	}

	var service corev1.Service
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(yamlUnstructured.Object, &service)
	if err != nil {
		return corev1.Service{}, err
	}

	return service, nil
}

func (skm MockKubernetesManager) PixieComponentContains(apiVersion, kind, metadataName string, checks ...string) bool {
	return contains(
		skm.appliedYAMLs,
		apiVersion,
		kind,
		"wavefront",
		"pixie",
		metadataName,
		checks...,
	)
}

func (skm MockKubernetesManager) AutotracingComponentContains(apiVersion, kind, metadataName string, checks ...string) bool {
	return contains(
		skm.appliedYAMLs,
		apiVersion,
		kind,
		"wavefront",
		"autotracing",
		metadataName,
		checks...,
	)
}

func (skm MockKubernetesManager) CollectorServiceAccountContains(checks ...string) bool {
	return contains(
		skm.appliedYAMLs,
		"v1",
		"ServiceAccount",
		"wavefront",
		"collector",
		"wavefront-collector",
		checks...,
	)
}

func (skm MockKubernetesManager) ConfigMapContains(configMapName string, checks ...string) bool {
	return contains(
		skm.appliedYAMLs,
		"v1",
		"ConfigMap",
		"wavefront",
		"",
		configMapName,
		checks...,
	)
}

func (skm MockKubernetesManager) CollectorConfigMapContains(checks ...string) bool {
	return skm.ConfigMapContains("default-wavefront-collector-config", checks...)
}

func (skm MockKubernetesManager) ProxyPreprocessorRulesConfigMapContains(checks ...string) bool {
	return contains(
		skm.appliedYAMLs,
		"v1",
		"ConfigMap",
		"wavefront",
		"proxy",
		"operator-proxy-preprocessor-rules-config",
		checks...,
	)
}

func (skm MockKubernetesManager) NodeCollectorDaemonSetContains(checks ...string) bool {
	return contains(
		skm.appliedYAMLs,
		"apps/v1",
		"DaemonSet",
		"wavefront",
		"node-collector",
		"wavefront-node-collector",
		checks...,
	)
}

func (skm MockKubernetesManager) LoggingDaemonSetContains(checks ...string) bool {
	return contains(
		skm.appliedYAMLs,
		"apps/v1",
		"DaemonSet",
		"wavefront",
		"logging",
		"wavefront-logging",
		checks...,
	)
}

func (skm MockKubernetesManager) LoggingConfigMapContains(checks ...string) bool {
	return contains(
		skm.appliedYAMLs,
		"v1",
		"ConfigMap",
		"wavefront",
		"logging",
		"wavefront-logging-config",
		checks...,
	)
}

func (skm MockKubernetesManager) ClusterCollectorDeploymentContains(checks ...string) bool {
	return contains(
		skm.appliedYAMLs,
		"apps/v1",
		"Deployment",
		"wavefront",
		"cluster-collector",
		"wavefront-cluster-collector",
		checks...,
	)
}

func (skm MockKubernetesManager) ProxyServiceContains(checks ...string) bool {
	return contains(
		skm.appliedYAMLs,
		"v1",
		"Service",
		"wavefront",
		"proxy",
		"wavefront-proxy",
		checks...,
	)
}

func (skm MockKubernetesManager) ProxyDeploymentContains(checks ...string) bool {
	return contains(
		skm.appliedYAMLs,
		"apps/v1",
		"Deployment",
		"wavefront",
		"proxy",
		"wavefront-proxy",
		checks...,
	)
}

func (skm MockKubernetesManager) GetUnstructuredCollectorServiceAccount() (*unstructured.Unstructured, error) {
	return skm.GetAppliedYAML(
		"v1",
		"ServiceAccount",
		"wavefront",
		"collector",
		"wavefront-collector",
	)
}

func (skm MockKubernetesManager) GetUnstructuredCollectorConfigMap(configMapName string) (*unstructured.Unstructured, error) {
	return skm.GetAppliedYAML(
		"v1",
		"ConfigMap",
		"wavefront",
		"collector",
		configMapName,
	)
}

func (skm MockKubernetesManager) GetUnstructuredNodeCollectorDaemonSet() (*unstructured.Unstructured, error) {
	return skm.GetAppliedYAML(
		"apps/v1",
		"DaemonSet",
		"wavefront",
		"collector",
		"wavefront-node-collector",
	)
}

func (skm MockKubernetesManager) GetUnstructuredClusterCollectorDeployment() (*unstructured.Unstructured, error) {
	return skm.GetAppliedYAML(
		"apps/v1",
		"Deployment",
		"wavefront",
		"collector",
		"wavefront-cluster-collector",
	)
}

func (skm MockKubernetesManager) GetUnstructuredProxyService() (*unstructured.Unstructured, error) {
	return skm.GetAppliedYAML(
		"v1",
		"Service",
		"wavefront",
		"proxy",
		"wavefront-proxy",
	)
}

func (skm MockKubernetesManager) GetUnstructuredProxyDeployment() (*unstructured.Unstructured, error) {
	return skm.GetAppliedYAML(
		"apps/v1",
		"Deployment",
		"wavefront",
		"proxy",
		"wavefront-proxy",
	)
}

func (skm MockKubernetesManager) GetUnstructuredLoggingDaemonset() (*unstructured.Unstructured, error) {
	return skm.GetAppliedYAML(
		"apps/v1",
		"DaemonSet",
		"wavefront",
		"logging",
		"wavefront-logging",
	)
}

func (skm MockKubernetesManager) GetCollectorServiceAccount() (corev1.ServiceAccount, error) {
	yamlUnstructured, err := skm.GetUnstructuredCollectorServiceAccount()
	if err != nil {
		return corev1.ServiceAccount{}, err
	}

	var serviceAccount corev1.ServiceAccount
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(yamlUnstructured.Object, &serviceAccount)
	if err != nil {
		return corev1.ServiceAccount{}, err
	}

	return serviceAccount, nil
}

func (skm MockKubernetesManager) GetCollectorConfigMap(configMapName string) (corev1.ConfigMap, error) {
	yamlUnstructured, err := skm.GetUnstructuredCollectorConfigMap(configMapName)
	if err != nil {
		return corev1.ConfigMap{}, err
	}

	var configMap corev1.ConfigMap
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(yamlUnstructured.Object, &configMap)
	if err != nil {
		return corev1.ConfigMap{}, err
	}

	return configMap, nil
}

func (skm MockKubernetesManager) GetNodeCollectorDaemonSet() (appsv1.DaemonSet, error) {
	yamlUnstructured, err := skm.GetUnstructuredNodeCollectorDaemonSet()
	if err != nil {
		return appsv1.DaemonSet{}, err
	}

	var daemonSet appsv1.DaemonSet
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(yamlUnstructured.Object, &daemonSet)
	if err != nil {
		return appsv1.DaemonSet{}, err
	}

	return daemonSet, nil
}

func (skm MockKubernetesManager) GetClusterCollectorDeployment() (appsv1.Deployment, error) {
	yamlUnstructured, err := skm.GetUnstructuredClusterCollectorDeployment()
	if err != nil {
		return appsv1.Deployment{}, err
	}

	var deployment appsv1.Deployment
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(yamlUnstructured.Object, &deployment)
	if err != nil {
		return appsv1.Deployment{}, err
	}

	return deployment, nil
}

func (skm MockKubernetesManager) GetProxyService() (corev1.Service, error) {
	yamlUnstructured, err := skm.GetUnstructuredProxyService()
	if err != nil {
		return corev1.Service{}, err
	}

	var service corev1.Service
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(yamlUnstructured.Object, &service)
	if err != nil {
		return corev1.Service{}, err
	}

	return service, nil
}

func (skm MockKubernetesManager) GetProxyDeployment() (appsv1.Deployment, error) {
	yamlUnstructured, err := skm.GetUnstructuredProxyDeployment()
	if err != nil {
		return appsv1.Deployment{}, err
	}

	var deployment appsv1.Deployment
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(yamlUnstructured.Object, &deployment)
	if err != nil {
		return appsv1.Deployment{}, err
	}

	return deployment, nil
}

func (skm MockKubernetesManager) GetProxyPreprocessorRulesConfigMap() (corev1.ConfigMap, error) {
	yamlUnstructured, err := skm.GetAppliedYAML(
		"v1",
		"ConfigMap",
		"wavefront",
		"proxy",
		"operator-proxy-preprocessor-rules-config",
	)

	if err != nil {
		return corev1.ConfigMap{}, err
	}

	var configMap corev1.ConfigMap
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(yamlUnstructured.Object, &configMap)
	if err != nil {
		return corev1.ConfigMap{}, err
	}

	return configMap, nil
}

func contains(
	yamls []string,
	apiVersion,
	kind,
	appKubernetesIOName,
	appKubernetesIOComponent,
	metadataName string,
	otherChecks ...string,
) bool {
	for _, yamlStr := range yamls {
		object, err := unstructuredFromStr(yamlStr)
		if err != nil {
			panic(err)
		}

		if objectMatchesAll(
			object,
			apiVersion,
			kind,
			appKubernetesIOName,
			appKubernetesIOComponent,
			metadataName,
		) {
			for _, other := range otherChecks {
				if !strings.Contains(yamlStr, other) {
					return false
				}
			}
			return true
		}
	}

	return false
}
