package pixie

import (
	"testing"

	"github.com/stretchr/testify/require"
	wf "github.com/wavefronthq/observability-for-kubernetes/operator/api/v1alpha1"
	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/testhelper/wftest"
)

func TestFromWavefront(t *testing.T) {
	t.Run("valid wavefront spec config for hub enabled", func(t *testing.T) {
		cr := wftest.NothingEnabledCR(func(w *wf.Wavefront) {
			w.Spec.Experimental.Hub.Enable = true
			w.Spec.Experimental.Hub.Pixie.Enable = true
			w.Spec.Experimental.Hub.Pixie.Pem.Resources = wf.Resources{
				Requests: wf.Resource{
					CPU:    "100m",
					Memory: "600Mi",
				},
				Limits: wf.Resource{
					CPU:    "1000m",
					Memory: "600Mi",
				},
			}
			w.Spec.Experimental.Hub.Pixie.Pem.TableStoreLimits = wf.TableStoreLimits{
				TotalMiB:          1,
				HttpEventsPercent: 2,
			}
		})
		config := FromWavefront(cr)
		component, _ := NewComponent(ComponentDir, config)

		require.True(t, config.Enable)
		require.Equal(t, "", component.Validate().Message())
		require.True(t, component.Validate().IsValid())
		require.Equal(t, HubSources, config.StirlingSources)
		require.Equal(t, cr.Spec.Experimental.Hub.Pixie.Pem.Resources, config.PemResources)
		require.Equal(t, cr.Spec.Experimental.Hub.Pixie.Pem.TableStoreLimits, config.TableStoreLimits)
	})

	t.Run("component config enable should be set to false", func(t *testing.T) {
		cr := wftest.CR()
		config := FromWavefront(cr)

		require.False(t, config.Enable)
	})

	t.Run("wavefront spec with autotracing enabled", func(t *testing.T) {
		cr := wftest.CR(func(w *wf.Wavefront) {
			w.Spec.Experimental.Autotracing.Enable = true
			w.Spec.ClusterName = "test-clusterName"
			w.Spec.Experimental.Autotracing.Pem.Resources = wf.Resources{
				Requests: wf.Resource{
					CPU:    "100m",
					Memory: "600Mi",
				},
				Limits: wf.Resource{
					CPU:    "1000m",
					Memory: "600Mi",
				},
			}
			w.Spec.Experimental.Autotracing.Pem.TableStoreLimits = wf.TableStoreLimits{
				TotalMiB:          1,
				HttpEventsPercent: 2,
			}
		})
		config := FromWavefront(cr)
		component, _ := NewComponent(ComponentDir, config)

		require.True(t, config.Enable)
		require.Equal(t, "", component.Validate().Message())
		require.True(t, component.Validate().IsValid())
		require.Equal(t, AutoTracingSources, config.StirlingSources)
		require.Equal(t, cr.Spec.Experimental.Autotracing.Pem.Resources, config.PemResources)
		require.Equal(t, cr.Spec.Experimental.Autotracing.Pem.TableStoreLimits, config.TableStoreLimits)
	})

	t.Run("wavefront spec with autotracing and hub enabled", func(t *testing.T) {
		cr := wftest.CR(func(w *wf.Wavefront) {
			w.Spec.Experimental.Autotracing.Enable = true
			w.Spec.ClusterName = "test-clusterName"
			w.Spec.Experimental.Autotracing.Pem.Resources = wf.Resources{
				Requests: wf.Resource{
					CPU:    "100m",
					Memory: "600Mi",
				},
				Limits: wf.Resource{
					CPU:    "1000m",
					Memory: "600Mi",
				},
			}
			w.Spec.Experimental.Autotracing.Pem.TableStoreLimits = wf.TableStoreLimits{
				TotalMiB:          1,
				HttpEventsPercent: 2,
			}
			w.Spec.Experimental.Hub.Enable = true
			w.Spec.Experimental.Hub.Pixie.Enable = true
			w.Spec.Experimental.Hub.Pixie.Pem.Resources = wf.Resources{
				Requests: wf.Resource{
					CPU:    "999m",
					Memory: "500Mi",
				},
				Limits: wf.Resource{
					CPU:    "99999m",
					Memory: "900Mi",
				},
			}
			w.Spec.Experimental.Hub.Pixie.Pem.TableStoreLimits = wf.TableStoreLimits{
				TotalMiB:          4,
				HttpEventsPercent: 3,
			}
		})
		config := FromWavefront(cr)
		component, _ := NewComponent(ComponentDir, config)

		require.True(t, config.Enable)
		require.Equal(t, "", component.Validate().Message())
		require.True(t, component.Validate().IsValid())
		require.Equal(t, HubSources, config.StirlingSources)
		require.Equal(t, cr.Spec.Experimental.Hub.Pixie.Pem.Resources, config.PemResources)
		require.Equal(t, cr.Spec.Experimental.Hub.Pixie.Pem.TableStoreLimits, config.TableStoreLimits)
		require.Equal(t, 0, config.MaxHTTPBodyBytes)
	})
}
