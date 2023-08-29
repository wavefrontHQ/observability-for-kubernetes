package testhelper

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
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
	applied      []client.Object
}

func NewMockKubernetesManager() *MockKubernetesManager {
	return &MockKubernetesManager{}
}

func (skm *MockKubernetesManager) ForAllApplied(do func(appliedYAML client.Object)) {
	for _, applied := range skm.applied {
		do(applied)
	}
}

func (skm *MockKubernetesManager) ApplyResources(resources []client.Object) error {
	resourceYAMLs, err := skm.toResourceYAMLs(resources)
	if err != nil {
		return err
	}
	skm.appliedYAMLs = resourceYAMLs
	skm.applied = resources
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
		"",
		"",
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

func (skm *MockKubernetesManager) RequireApplied(t *testing.T, expectApplied int, apiVersion, kind, name string) []client.Object {
	t.Helper()
	var actualApplied []client.Object
	skm.ForAllApplied(func(applied client.Object) {
		gvk := applied.GetObjectKind().GroupVersionKind()
		appliedKind := gvk.Kind
		appliedApiVersion := gvk.GroupVersion().String()
		if apiVersion != appliedApiVersion || kind != appliedKind {
			return
		}
		if applied.GetName() != name {
			return
		}
		actualApplied = append(actualApplied, applied)
	})
	require.Equal(t, expectApplied, len(actualApplied), "expected %s[name=%s] to be applied %d times but was applied %d", kind, name, expectApplied, len(actualApplied))
	return actualApplied
}

func (skm *MockKubernetesManager) RequireConfigMapContains(t *testing.T, name, key string, checks ...string) {
	t.Helper()
	applied := skm.RequireApplied(t, 1, "v1", "ConfigMap", name)[0]
	var configMap corev1.ConfigMap
	switch obj := applied.(type) {
	case *unstructured.Unstructured:
		require.NoError(t, runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &configMap), "expected unstructured to be configmap")
	case *corev1.ConfigMap:
		configMap = *obj
	default:
		t.Fatalf("%s is not a ConfigMap", name)
	}
	for _, check := range checks {
		require.Containsf(t, configMap.Data[key], check, "ConfigMap[name=%s].data.%s does not contain expected value", name, key)
	}
}

func (skm *MockKubernetesManager) RequireConfigMapNotContains(t *testing.T, name, key string, checks ...string) {
	t.Helper()
	applied := skm.RequireApplied(t, 1, "v1", "ConfigMap", name)[0]
	var configMap corev1.ConfigMap
	switch obj := applied.(type) {
	case *unstructured.Unstructured:
		require.NoError(t, runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &configMap), "expected unstructured to be configmap")
	case *corev1.ConfigMap:
		configMap = *obj
	default:
		t.Fatalf("%s is not a ConfigMap", name)
	}
	for _, check := range checks {
		require.NotContainsf(t, configMap.Data[key], check, "ConfigMap[name=%s].data.%s does contains unexpected value", name, key)
	}
}

func (skm *MockKubernetesManager) RequireCollectorConfigMapContains(t *testing.T, checks ...string) {
	t.Helper()
	skm.RequireConfigMapContains(t, "default-wavefront-collector-config", "config.yaml", checks...)
}

func (skm *MockKubernetesManager) RequireCollectorConfigMapNotContains(t *testing.T, checks ...string) {
	t.Helper()
	skm.RequireConfigMapNotContains(t, "default-wavefront-collector-config", "config.yaml", checks...)
}

func (skm *MockKubernetesManager) RequireConfigMapNotApplied(t *testing.T, name string) {
	t.Helper()
	skm.RequireApplied(t, 0, "v1", "ConfigMap", name)
}

func (skm *MockKubernetesManager) RequireCollectorConfigMapNotApplied(t *testing.T) {
	t.Helper()
	skm.RequireConfigMapNotApplied(t, "default-wavefront-collector-config")
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

func (skm MockKubernetesManager) GetUnstructuredCollectorConfigMap() (*unstructured.Unstructured, error) {
	return skm.GetAppliedYAML(
		"v1",
		"ConfigMap",
		"wavefront",
		"collector",
		"default-wavefront-collector-config",
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

func (skm MockKubernetesManager) GetCollectorConfigMap() (corev1.ConfigMap, error) {
	yamlUnstructured, err := skm.GetUnstructuredCollectorConfigMap()
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
