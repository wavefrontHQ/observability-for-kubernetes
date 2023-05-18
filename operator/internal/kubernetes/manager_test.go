package kubernetes_manager_test

import (
	"context"
	"errors"
	"testing"

	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/util"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kubernetes_manager "github.com/wavefronthq/observability-for-kubernetes/operator/internal/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const fakeServiceYAML = `
apiVersion: v1
kind: Service
metadata:
  labels:
    app.kubernetes.io/name: fake-app-kubernetes-name
  name: fake-name
  namespace: fake-namespace
spec:
  ports:
  - name: fake-port-name
    port: 1111
    protocol: TCP
  selector:
    app.kubernetes.io/name: fake-app-kubernetes-name
  type: ClusterIP
`

const fakeServiceUpdatedYAML = `
apiVersion: v1
kind: Service
metadata:
  labels:
    app.kubernetes.io/name: fake-app-kubernetes-name
  name: fake-name
  namespace: fake-namespace
spec:
  ports:
  - name: fake-port-name
    port: 1112
    protocol: TCP
  selector:
    app.kubernetes.io/name: fake-app-kubernetes-name
  type: ClusterIP
`

const otherFakeServiceYAML = `
apiVersion: v1
kind: Service
metadata:
  labels:
    app.kubernetes.io/name: other-fake-app-kubernetes-name
  name: other-fake-name
  namespace: fake-namespace
spec:
  ports:
  - name: fake-port-name
    port: 1111
    protocol: TCP
  selector:
    app.kubernetes.io/name: other-fake-app-kubernetes-name
  type: ClusterIP
`

const missingCRDYAML = `
apiVersion: security.openshift.io/v1
kind: SecurityContextConstraints
metadata:
  name: wavefront-proxy-scc
  namespace: system
  labels:
    app.kubernetes.io/name: wavefront
    app.kubernetes.io/component: proxy
  annotations:
    wavefront.com/conditionally-provision: "false"
allowHostDirVolumePlugin: false
allowHostIPC: false
allowHostNetwork: false
allowHostPID: false
allowHostPorts: false
allowPrivilegedContainer: false
readOnlyRootFilesystem: true
runAsUser:
  type: RunAsAny
seLinuxContext:
  type: MustRunAs
users:
- system:serviceaccount:observability-system:wavefront-proxy
`

func TestKubernetesManager(t *testing.T) {
	t.Run("applying", func(t *testing.T) {
		t.Run("rejects invalid objects", func(t *testing.T) {
			km := kubernetes_manager.NewKubernetesManager(fake.NewClientBuilder().Build())

			err := km.ApplyResources([]string{"invalid: {{object}}"})

			require.ErrorContains(t, err, "yaml: invalid")
		})

		t.Run("creates kubernetes objects", func(t *testing.T) {
			objClient := fake.NewClientBuilder().Build()
			km := kubernetes_manager.NewKubernetesManager(objClient)

			require.NoError(t, km.ApplyResources([]string{fakeServiceYAML}))

			require.NoError(t, objClient.Get(
				context.Background(),
				util.ObjKey("fake-namespace", "fake-name"),
				&corev1.Service{},
			))
		})

		t.Run("patches kubernetes objects", func(t *testing.T) {
			objClient := fake.NewClientBuilder().Build()
			km := kubernetes_manager.NewKubernetesManager(objClient)

			err := km.ApplyResources([]string{fakeServiceYAML, fakeServiceUpdatedYAML})
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

			err := km.ApplyResources([]string{fakeServiceYAML})
			require.Error(t, err)
		})
	})

	t.Run("deleting", func(t *testing.T) {
		t.Run("rejects invalid objects", func(t *testing.T) {
			km := kubernetes_manager.NewKubernetesManager(fake.NewClientBuilder().Build())

			err := km.DeleteResources([]string{"invalid: {{object}}"})

			require.ErrorContains(t, err, "yaml: invalid")
		})

		t.Run("deletes objects that exist", func(t *testing.T) {
			objClient := fake.NewClientBuilder().Build()
			km := kubernetes_manager.NewKubernetesManager(objClient)

			_ = km.ApplyResources([]string{fakeServiceYAML})

			require.NoError(t, km.DeleteResources([]string{fakeServiceYAML}))

			require.Error(t, objClient.Get(
				context.Background(),
				util.ObjKey("fake-namespace", "fake-name"),
				&corev1.Service{},
			))
		})

		t.Run("reports client errors", func(t *testing.T) {
			km := kubernetes_manager.NewKubernetesManager(&errClient{errors.New("some error")})

			require.Error(t, km.DeleteResources([]string{fakeServiceYAML}))
		})

		t.Run("does not return an error for objects that do not exist", func(t *testing.T) {
			km := kubernetes_manager.NewKubernetesManager(fake.NewClientBuilder().Build())

			require.NoError(t, km.DeleteResources([]string{fakeServiceYAML}))
		})

		t.Run("does not return an error for custom objects that are not defined", func(t *testing.T) {
			km := kubernetes_manager.NewKubernetesManager(fake.NewClientBuilder().Build())

			require.NoError(t, km.DeleteResources([]string{missingCRDYAML}))
		})

		t.Run("deletes all resources", func(t *testing.T) {
			objClient := fake.NewClientBuilder().Build()
			km := kubernetes_manager.NewKubernetesManager(objClient)

			_ = km.ApplyResources([]string{fakeServiceYAML, otherFakeServiceYAML})

			require.NoError(t, km.DeleteResources([]string{fakeServiceYAML, otherFakeServiceYAML}))

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
