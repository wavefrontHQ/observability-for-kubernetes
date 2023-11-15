package components

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"
	"github.com/wavefronthq/observability-for-kubernetes/operator/components/patch"
	componenttest "github.com/wavefronthq/observability-for-kubernetes/operator/components/test"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const fakeDaemonset = `
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: fake-daemonset
  namespace: observability-system
spec:
  selector:
    matchLabels:
      name: fake-daemonset
  template:
    metadata:
      labels:
        name: fake-daemonset
    spec:
      containers:
      - image: some:image
        buildResources:
          requests:
            cpu: 1m
            memory: 1Ki
          limits:
            cpu: 1m
            memory: 1Ki
`

const fakeDeployment = `
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    name: fake-deployment
  name: fake-deployment
  namespace: observability-system
spec:
  replicas: 1
  selector:
    matchLabels:
      name: fake-deployment
  template:
    metadata:
      labels:
        name: fake-deployment
    spec:
      containers:
        - image: some:image
          buildResources: {}
`

func TestK8sResourceBuilder(t *testing.T) {
	t.Run("resource overrides", func(t *testing.T) {
		t.Run("resource overrides are applied", func(t *testing.T) {
			builder := NewK8sResourceBuilder(patch.ByName{"fake-daemonset": func(resource *unstructured.Unstructured) {
				resource.SetAnnotations(map[string]string{"foo": "1"})
			}})
			fakeFS := fstest.MapFS{
				"fake-daemonset.yaml": &fstest.MapFile{
					Data: []byte(fakeDaemonset),
				},
			}

			toApply, _, err := builder.Build(fakeFS, "some-component", true, "manager-uuid", nil, nil)

			require.NoError(t, err)

			ds, err := componenttest.GetDaemonSet("fake-daemonset", toApply)
			require.NoError(t, err)

			require.Equal(t, "1", ds.Annotations["foo"])
		})

		t.Run("component defaults are applied", func(t *testing.T) {
			builder := NewK8sResourceBuilder(nil)
			fakeFS := fstest.MapFS{
				"fake-daemonset.yaml": &fstest.MapFile{
					Data: []byte(fakeDaemonset),
				},
			}

			toApply, _, err := builder.Build(fakeFS, "some-component", true, "manager-uuid", map[string]patch.Patch{"fake-daemonset": func(resource *unstructured.Unstructured) {
				resource.SetAnnotations(map[string]string{"foo": "2"})
			}}, nil)

			require.NoError(t, err)

			ds, err := componenttest.GetDaemonSet("fake-daemonset", toApply)
			require.NoError(t, err)

			require.Equal(t, "2", ds.Annotations["foo"])
		})

		t.Run("resource overrides are applied on top of component defaults", func(t *testing.T) {
			builder := NewK8sResourceBuilder(patch.ByName{"fake-daemonset": func(resource *unstructured.Unstructured) {
				resource.SetAnnotations(map[string]string{"foo": "1"})
			}})
			fakeFS := fstest.MapFS{
				"fake-daemonset.yaml": &fstest.MapFile{
					Data: []byte(fakeDaemonset),
				},
			}

			toApply, _, err := builder.Build(fakeFS, "some-component", true, "manager-uuid", patch.ByName{"fake-daemonset": func(resource *unstructured.Unstructured) {
				resource.SetAnnotations(map[string]string{"foo": "2"})
			}}, nil)

			require.NoError(t, err)
			require.Len(t, toApply, 1)

			ds, err := componenttest.GetDaemonSet("fake-daemonset", toApply)
			require.NoError(t, err)

			require.Equal(t, "1", ds.Annotations["foo"])
		})

		t.Run("component defaults can override all resource patches", func(t *testing.T) {
			builder := NewK8sResourceBuilder(nil)
			fakeFS := fstest.MapFS{
				"fake-daemonset.yaml": &fstest.MapFile{
					Data: []byte(fakeDaemonset),
				},
			}

			toApply, _, err := builder.Build(fakeFS, "some-component", true, "manager-uuid", patch.ByName{
				"fake-daemonset": func(r *unstructured.Unstructured) {
					r.SetLabels(map[string]string{"app.kubernetes.io/name": "overridden"})
				},
			}, nil)

			require.NoError(t, err)

			ds, err := componenttest.GetDaemonSet("fake-daemonset", toApply)
			require.NoError(t, err)

			require.Equal(t, "overridden", ds.GetLabels()["app.kubernetes.io/name"], "app.kubernetes.io/name label")
		})
	})

	t.Run("all buildResources", func(t *testing.T) {
		t.Run("have an owner reference", func(t *testing.T) {
			builder := NewK8sResourceBuilder(nil)
			fakeFS := fstest.MapFS{
				"fake-daemonset.yaml": &fstest.MapFile{
					Data: []byte(fakeDaemonset),
				},
			}

			toApply, _, err := builder.Build(fakeFS, "some-component", true, "manager-uuid", nil, nil)

			require.NoError(t, err)

			ds, err := componenttest.GetDaemonSet("fake-daemonset", toApply)
			require.NoError(t, err)

			require.Len(t, ds.OwnerReferences, 1, "owner references")
			expectedOwnerReference := v1.OwnerReference{
				APIVersion: "apps/v1",
				Kind:       "Deployment",
				Name:       "wavefront-controller-manager",
				UID:        "manager-uuid",
			}
			require.Equal(t, expectedOwnerReference, ds.OwnerReferences[0], "owner reference")
		})

		t.Run("have app.kubernetes.io/* labels", func(t *testing.T) {
			builder := NewK8sResourceBuilder(nil)
			fakeFS := fstest.MapFS{
				"fake-daemonset.yaml": &fstest.MapFile{
					Data: []byte(fakeDaemonset),
				},
			}

			toApply, _, err := builder.Build(fakeFS, "some-component", true, "manager-uuid", nil, nil)

			require.NoError(t, err)

			ds, err := componenttest.GetDaemonSet("fake-daemonset", toApply)
			require.NoError(t, err)

			require.Equal(t, "wavefront", ds.GetLabels()["app.kubernetes.io/name"], "app.kubernetes.io/name label")
			require.Equal(t, "some-component", ds.GetLabels()["app.kubernetes.io/component"], "app.kubernetes.io/component label")
			require.Equal(t, "wavefront", ds.Spec.Template.Labels["app.kubernetes.io/name"], "app.kubernetes.io/name label")
			require.Equal(t, "some-component", ds.Spec.Template.Labels["app.kubernetes.io/component"], "app.kubernetes.io/component label")
		})
	})
}
