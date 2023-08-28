package kubernetes_manager_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kubernetes_manager "github.com/wavefronthq/observability-for-kubernetes/operator/internal/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func fakeService() client.Object {
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-name",
			Namespace: "fake-namespace",
			Labels: map[string]string{
				"app.kubernetes.io/name": "fake-app-kubernetes-name",
			},
		},
		Spec: corev1.ServiceSpec{
			Type: "ClusterIP",
			Ports: []corev1.ServicePort{{
				Name:     "fake-port-name",
				Protocol: "TCP",
				Port:     1111,
			}},
			Selector: map[string]string{
				"app.kubernetes.io/name": "fake-app-kubernetes-name",
			},
		},
	}
}

func fakeServiceUpdated() client.Object {
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-name",
			Namespace: "fake-namespace",
			Labels: map[string]string{
				"app.kubernetes.io/name": "fake-app-kubernetes-name",
			},
		},
		Spec: corev1.ServiceSpec{
			Type: "ClusterIP",
			Ports: []corev1.ServicePort{{
				Name:     "fake-port-name",
				Protocol: "TCP",
				Port:     1112,
			}},
			Selector: map[string]string{
				"app.kubernetes.io/name": "fake-app-kubernetes-name",
			},
		},
	}
}

func otherFakeService() client.Object {
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "other-fake-name",
			Namespace: "fake-namespace",
			Labels: map[string]string{
				"app.kubernetes.io/name": "other-fake-app-kubernetes-name",
			},
		},
		Spec: corev1.ServiceSpec{
			Type: "ClusterIP",
			Ports: []corev1.ServicePort{{
				Name:     "fake-port-name",
				Protocol: "TCP",
				Port:     1111,
			}},
			Selector: map[string]string{
				"app.kubernetes.io/name": "other-fake-app-kubernetes-name",
			},
		},
	}
}

func missingCRD() client.Object {
	return &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "security.openshift.io/v1",
		"kind":       "SecurityContextConstraints",
		"metadata": map[string]interface{}{
			"name":      "wavefront-proxy-scc",
			"namespace": "system",
			"labels": map[string]string{
				"app.kubernetes.io/name":      "wavefront",
				"app.kubernetes.io/component": "proxy",
			},
			"annotations": map[string]string{
				"wavefront.com/conditionally-provision": "false",
			},
		},
		"allowHostDirVolumePlugin": false,
		"allowHostIPC":             false,
		"allowHostNetwork":         false,
		"allowHostPID":             false,
		"allowHostPorts":           false,
		"allowPrivilegedContainer": false,
		"readOnlyRootFilesystem":   true,
		"runAsUser": map[string]interface{}{
			"type": "RunAsAny",
		},
		"seLinuxContext": map[string]interface{}{
			"type": "MustRunAs",
		},
		"users": []string{"system:serviceaccount:observability-system:wavefront-proxy"},
	}}
}

func TestKubernetesManager(t *testing.T) {
	t.Run("applying", func(t *testing.T) {
		t.Run("creates kubernetes objects", func(t *testing.T) {
			objClient := fake.NewClientBuilder().Build()
			km := kubernetes_manager.NewKubernetesManager(objClient)

			require.NoError(t, km.ApplyResources([]client.Object{fakeService()}))

			require.NoError(t, objClient.Get(
				context.Background(),
				util.ObjKey("fake-namespace", "fake-name"),
				&corev1.Service{},
			))
		})

		t.Run("patches kubernetes objects", func(t *testing.T) {
			objClient := fake.NewClientBuilder().Build()
			km := kubernetes_manager.NewKubernetesManager(objClient)

			err := km.ApplyResources([]client.Object{fakeService(), fakeServiceUpdated()})
			require.NoError(t, err)

			var service corev1.Service
			require.NoError(t, objClient.Get(
				context.Background(),
				util.ObjKey("fake-namespace", "fake-name"),
				&service,
			))

			require.Equal(t, int32(1112), service.Spec.Ports[0].Port)
		})

		t.Run("reports client errors", func(t *testing.T) {
			km := kubernetes_manager.NewKubernetesManager(&errClient{errors.New("some error")})

			err := km.ApplyResources([]client.Object{fakeService()})
			require.Error(t, err)
		})
	})

	t.Run("deleting", func(t *testing.T) {
		t.Run("deletes objects that exist", func(t *testing.T) {
			objClient := fake.NewClientBuilder().Build()
			km := kubernetes_manager.NewKubernetesManager(objClient)

			_ = km.ApplyResources([]client.Object{fakeService()})

			require.NoError(t, km.DeleteResources([]client.Object{fakeService()}))

			require.Error(t, objClient.Get(
				context.Background(),
				util.ObjKey("fake-namespace", "fake-name"),
				&corev1.Service{},
			))
		})

		t.Run("reports client errors", func(t *testing.T) {
			km := kubernetes_manager.NewKubernetesManager(&errClient{errors.New("some error")})

			require.Error(t, km.DeleteResources([]client.Object{fakeService()}))
		})

		t.Run("does not return an error for objects that do not exist", func(t *testing.T) {
			km := kubernetes_manager.NewKubernetesManager(fake.NewClientBuilder().Build())

			require.NoError(t, km.DeleteResources([]client.Object{fakeService()}))
		})

		t.Run("does not return an error for custom objects that are not defined", func(t *testing.T) {
			km := kubernetes_manager.NewKubernetesManager(fake.NewClientBuilder().Build())

			require.NoError(t, km.DeleteResources([]client.Object{missingCRD()}))
		})

		t.Run("deletes all resources", func(t *testing.T) {
			objClient := fake.NewClientBuilder().Build()
			km := kubernetes_manager.NewKubernetesManager(objClient)

			_ = km.ApplyResources([]client.Object{fakeService(), otherFakeService()})

			require.NoError(t, km.DeleteResources([]client.Object{fakeService(), otherFakeService()}))

			require.Error(t, objClient.Get(
				context.Background(),
				util.ObjKey("fake-namespace", "fake-name"),
				&corev1.Service{},
			))

			require.Error(t, objClient.Get(
				context.Background(),
				util.ObjKey("fake-namespace", "other-fake-name"),
				&corev1.Service{},
			))
		})
	})
}

type errClient struct {
	err error
}

func (c *errClient) Get(_ context.Context, _ client.ObjectKey, _ client.Object) error {
	return c.err
}

func (c *errClient) Create(_ context.Context, _ client.Object, _ ...client.CreateOption) error {
	return c.err
}

func (c *errClient) Patch(_ context.Context, _ client.Object, _ client.Patch, _ ...client.PatchOption) error {
	return c.err
}

func (c *errClient) Delete(_ context.Context, _ client.Object, _ ...client.DeleteOption) error {
	return c.err
}
