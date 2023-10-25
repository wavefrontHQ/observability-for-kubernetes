package components

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"
	wf "github.com/wavefronthq/observability-for-kubernetes/operator/api/v1alpha1"
	componenttest "github.com/wavefronthq/observability-for-kubernetes/operator/components/test"
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
        resources:
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
          resources: {}
`

func TestK8sResourceBuilder(t *testing.T) {
	t.Run("container resources", func(t *testing.T) {
		t.Run("overrides override YAML", func(t *testing.T) {
			builder := NewK8sResourceBuilder(map[string]wf.Resources{"fake-daemonset": {
				Requests: wf.Resource{
					CPU:    "100m",
					Memory: "100Mi",
				},
				Limits: wf.Resource{
					CPU:    "1",
					Memory: "1Gi",
				},
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

			require.Equal(t, "100m", ds.Spec.Template.Spec.Containers[0].Resources.Requests.Cpu().String())
			require.Equal(t, "100Mi", ds.Spec.Template.Spec.Containers[0].Resources.Requests.Memory().String())
			require.Equal(t, "1", ds.Spec.Template.Spec.Containers[0].Resources.Limits.Cpu().String())
			require.Equal(t, "1Gi", ds.Spec.Template.Spec.Containers[0].Resources.Limits.Memory().String())
		})

		t.Run("defaults override YAML", func(t *testing.T) {
			builder := NewK8sResourceBuilder(nil)
			fakeFS := fstest.MapFS{
				"fake-daemonset.yaml": &fstest.MapFile{
					Data: []byte(fakeDaemonset),
				},
			}

			toApply, _, err := builder.Build(fakeFS, "some-component", true, "manager-uuid", map[string]wf.Resources{"fake-daemonset": {
				Requests: wf.Resource{
					CPU:    "100m",
					Memory: "100Mi",
				},
				Limits: wf.Resource{
					CPU:    "1",
					Memory: "1Gi",
				},
			}}, nil)

			require.NoError(t, err)

			ds, err := componenttest.GetDaemonSet("fake-daemonset", toApply)
			require.NoError(t, err)

			require.Equal(t, "100m", ds.Spec.Template.Spec.Containers[0].Resources.Requests.Cpu().String())
			require.Equal(t, "100Mi", ds.Spec.Template.Spec.Containers[0].Resources.Requests.Memory().String())
			require.Equal(t, "1", ds.Spec.Template.Spec.Containers[0].Resources.Limits.Cpu().String())
			require.Equal(t, "1Gi", ds.Spec.Template.Spec.Containers[0].Resources.Limits.Memory().String())
		})

		t.Run("overrides override defaults", func(t *testing.T) {
			builder := NewK8sResourceBuilder(map[string]wf.Resources{"fake-daemonset": {
				Requests: wf.Resource{
					CPU:    "50m",
					Memory: "50Mi",
				},
				Limits: wf.Resource{
					CPU:    "500m",
					Memory: "500Mi",
				},
			}})
			fakeFS := fstest.MapFS{
				"fake-daemonset.yaml": &fstest.MapFile{
					Data: []byte(fakeDaemonset),
				},
			}

			toApply, _, err := builder.Build(fakeFS, "some-component", true, "manager-uuid", map[string]wf.Resources{"fake-daemonset": {
				Requests: wf.Resource{
					CPU:    "100m",
					Memory: "100Mi",
				},
				Limits: wf.Resource{
					CPU:    "1",
					Memory: "1Gi",
				},
			}}, nil)

			require.NoError(t, err)
			require.Len(t, toApply, 1)

			ds, err := componenttest.GetDaemonSet("fake-daemonset", toApply)
			require.NoError(t, err)

			require.Equal(t, "50m", ds.Spec.Template.Spec.Containers[0].Resources.Requests.Cpu().String())
			require.Equal(t, "50Mi", ds.Spec.Template.Spec.Containers[0].Resources.Requests.Memory().String())
			require.Equal(t, "500m", ds.Spec.Template.Spec.Containers[0].Resources.Limits.Cpu().String())
			require.Equal(t, "500Mi", ds.Spec.Template.Spec.Containers[0].Resources.Limits.Memory().String())
		})

		t.Run("merges overrides with defaults", func(t *testing.T) {
			builder := NewK8sResourceBuilder(map[string]wf.Resources{"fake-daemonset": {
				Requests: wf.Resource{
					CPU:    "50m",
					Memory: "50Mi",
				},
				Limits: wf.Resource{
					CPU:    "500m",
					Memory: "500Mi",
				},
			}})
			fakeFS := fstest.MapFS{
				"fake-daemonset.yaml": &fstest.MapFile{
					Data: []byte(fakeDaemonset),
				},
				"fake-deployment.yaml": &fstest.MapFile{
					Data: []byte(fakeDeployment),
				},
			}

			toApply, _, err := builder.Build(fakeFS, "some-component", true, "manager-uuid", map[string]wf.Resources{
				"fake-daemonset": {
					Requests: wf.Resource{
						CPU:    "100m",
						Memory: "100Mi",
					},
					Limits: wf.Resource{
						CPU:    "1",
						Memory: "1Gi",
					},
				},
				"fake-deployment": {
					Requests: wf.Resource{
						CPU:    "100m",
						Memory: "100Mi",
					},
					Limits: wf.Resource{
						CPU:    "1",
						Memory: "1Gi",
					},
				},
			}, nil)

			require.NoError(t, err)

			ds, err := componenttest.GetDaemonSet("fake-daemonset", toApply)
			require.NoError(t, err)

			require.Equal(t, "50m", ds.Spec.Template.Spec.Containers[0].Resources.Requests.Cpu().String())
			require.Equal(t, "50Mi", ds.Spec.Template.Spec.Containers[0].Resources.Requests.Memory().String())
			require.Equal(t, "500m", ds.Spec.Template.Spec.Containers[0].Resources.Limits.Cpu().String())
			require.Equal(t, "500Mi", ds.Spec.Template.Spec.Containers[0].Resources.Limits.Memory().String())

			deploy, err := componenttest.GetDeployment("fake-deployment", toApply)
			require.NoError(t, err)

			require.Equal(t, "100m", deploy.Spec.Template.Spec.Containers[0].Resources.Requests.Cpu().String())
			require.Equal(t, "100Mi", deploy.Spec.Template.Spec.Containers[0].Resources.Requests.Memory().String())
			require.Equal(t, "1", deploy.Spec.Template.Spec.Containers[0].Resources.Limits.Cpu().String())
			require.Equal(t, "1Gi", deploy.Spec.Template.Spec.Containers[0].Resources.Limits.Memory().String())
		})

		t.Run("merges overrides with defaults when only memory is specified", func(t *testing.T) {
			builder := NewK8sResourceBuilder(map[string]wf.Resources{"fake-daemonset": {
				Requests: wf.Resource{
					Memory: "50Mi",
				},
				Limits: wf.Resource{
					Memory: "500Mi",
				},
			}})
			fakeFS := fstest.MapFS{
				"fake-daemonset.yaml": &fstest.MapFile{
					Data: []byte(fakeDaemonset),
				},
			}

			toApply, _, err := builder.Build(fakeFS, "some-component", true, "manager-uuid", map[string]wf.Resources{
				"fake-daemonset": {
					Requests: wf.Resource{
						CPU:              "100m",
						Memory:           "100Mi",
						EphemeralStorage: "100Mi",
					},
					Limits: wf.Resource{
						CPU:              "1",
						Memory:           "1Gi",
						EphemeralStorage: "100Mi",
					},
				},
			}, nil)

			require.NoError(t, err)

			ds, err := componenttest.GetDaemonSet("fake-daemonset", toApply)
			require.NoError(t, err)

			require.Equal(t, "100m", ds.Spec.Template.Spec.Containers[0].Resources.Requests.Cpu().String())
			require.Equal(t, "50Mi", ds.Spec.Template.Spec.Containers[0].Resources.Requests.Memory().String())
			require.Equal(t, "100Mi", ds.Spec.Template.Spec.Containers[0].Resources.Requests.StorageEphemeral().String())

			require.Equal(t, "1", ds.Spec.Template.Spec.Containers[0].Resources.Limits.Cpu().String())
			require.Equal(t, "500Mi", ds.Spec.Template.Spec.Containers[0].Resources.Limits.Memory().String())
			require.Equal(t, "100Mi", ds.Spec.Template.Spec.Containers[0].Resources.Limits.StorageEphemeral().String())
		})

		t.Run("merges overrides with defaults when only CPU is specified", func(t *testing.T) {
			builder := NewK8sResourceBuilder(map[string]wf.Resources{"fake-daemonset": {
				Requests: wf.Resource{
					CPU: "50m",
				},
				Limits: wf.Resource{
					CPU: "500m",
				},
			}})
			fakeFS := fstest.MapFS{
				"fake-daemonset.yaml": &fstest.MapFile{
					Data: []byte(fakeDaemonset),
				},
			}

			toApply, _, err := builder.Build(fakeFS, "some-component", true, "manager-uuid", map[string]wf.Resources{
				"fake-daemonset": {
					Requests: wf.Resource{
						CPU:              "100m",
						Memory:           "100Mi",
						EphemeralStorage: "100Mi",
					},
					Limits: wf.Resource{
						CPU:              "1",
						Memory:           "1Gi",
						EphemeralStorage: "100Mi",
					},
				},
			}, nil)

			require.NoError(t, err)

			ds, err := componenttest.GetDaemonSet("fake-daemonset", toApply)
			require.NoError(t, err)

			require.Equal(t, "50m", ds.Spec.Template.Spec.Containers[0].Resources.Requests.Cpu().String())
			require.Equal(t, "100Mi", ds.Spec.Template.Spec.Containers[0].Resources.Requests.Memory().String())
			require.Equal(t, "100Mi", ds.Spec.Template.Spec.Containers[0].Resources.Requests.StorageEphemeral().String())

			require.Equal(t, "500m", ds.Spec.Template.Spec.Containers[0].Resources.Limits.Cpu().String())
			require.Equal(t, "1Gi", ds.Spec.Template.Spec.Containers[0].Resources.Limits.Memory().String())
			require.Equal(t, "100Mi", ds.Spec.Template.Spec.Containers[0].Resources.Limits.StorageEphemeral().String())
		})

		t.Run("merges overrides with defaults when only EphemeralStorage is specified", func(t *testing.T) {
			builder := NewK8sResourceBuilder(map[string]wf.Resources{"fake-daemonset": {
				Requests: wf.Resource{
					EphemeralStorage: "50Mi",
				},
				Limits: wf.Resource{
					EphemeralStorage: "500Mi",
				},
			}})
			fakeFS := fstest.MapFS{
				"fake-daemonset.yaml": &fstest.MapFile{
					Data: []byte(fakeDaemonset),
				},
			}

			toApply, _, err := builder.Build(fakeFS, "some-component", true, "manager-uuid", map[string]wf.Resources{
				"fake-daemonset": {
					Requests: wf.Resource{
						CPU:              "100m",
						Memory:           "100Mi",
						EphemeralStorage: "100Mi",
					},
					Limits: wf.Resource{
						CPU:              "1",
						Memory:           "1Gi",
						EphemeralStorage: "100Mi",
					},
				},
			}, nil)

			require.NoError(t, err)

			ds, err := componenttest.GetDaemonSet("fake-daemonset", toApply)
			require.NoError(t, err)

			require.Equal(t, "100m", ds.Spec.Template.Spec.Containers[0].Resources.Requests.Cpu().String())
			require.Equal(t, "100Mi", ds.Spec.Template.Spec.Containers[0].Resources.Requests.Memory().String())
			require.Equal(t, "50Mi", ds.Spec.Template.Spec.Containers[0].Resources.Requests.StorageEphemeral().String())

			require.Equal(t, "1", ds.Spec.Template.Spec.Containers[0].Resources.Limits.Cpu().String())
			require.Equal(t, "1Gi", ds.Spec.Template.Spec.Containers[0].Resources.Limits.Memory().String())
			require.Equal(t, "500Mi", ds.Spec.Template.Spec.Containers[0].Resources.Limits.StorageEphemeral().String())
		})
	})
}
