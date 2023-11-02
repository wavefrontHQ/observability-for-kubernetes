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
		})
		config := FromWavefront(cr)
		component, _ := NewComponent(ComponentDir, config)

		require.True(t, config.Enable)
		require.Equal(t, "", component.Validate().Message())
		require.True(t, component.Validate().IsValid())
		require.Equal(t, HubSources, config.StirlingSources)
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
		})
		config := FromWavefront(cr)
		component, _ := NewComponent(ComponentDir, config)

		require.True(t, config.Enable)
		require.Equal(t, "", component.Validate().Message())
		require.True(t, component.Validate().IsValid())
		require.Equal(t, AutoTracingSources, config.StirlingSources)
	})

	t.Run("wavefront spec with autotracing and hub enabled", func(t *testing.T) {
		cr := wftest.CR(func(w *wf.Wavefront) {
			w.Spec.Experimental.Autotracing.Enable = true
			w.Spec.ClusterName = "test-clusterName"
			w.Spec.Experimental.Hub.Enable = true
			w.Spec.Experimental.Hub.Pixie.Enable = true
		})
		config := FromWavefront(cr)
		component, _ := NewComponent(ComponentDir, config)

		require.True(t, config.Enable)
		require.Equal(t, "", component.Validate().Message())
		require.True(t, component.Validate().IsValid())
		require.Equal(t, HubSources, config.StirlingSources)
		require.Equal(t, 0, config.MaxHTTPBodyBytes)
	})

	t.Run("sizing", func(t *testing.T) {
		for _, clusterSize := range wf.ClusterSizes {
			t.Run(clusterSize, func(t *testing.T) {
				cr := wftest.NothingEnabledCR(func(w *wf.Wavefront) {
					w.Spec.ClusterSize = clusterSize
				})

				config := FromWavefront(cr)

				require.Equal(t, PEMResources[clusterSize], config.PEMResources)
				require.Equal(t, TableStoreLimits[clusterSize], config.TableStoreLimits)
				require.Equal(t, KelvinResources[clusterSize], config.KelvinResources)
				require.Equal(t, QueryBrokerResources[clusterSize], config.QueryBrokerResources)
				require.Equal(t, NATSResources[clusterSize], config.NATSResources)
				require.Equal(t, MetadataResources[clusterSize], config.MetadataResources)
				require.Equal(t, CertProvisionerJobResources[clusterSize], config.CertProvisionerJobResources)
			})
		}

		t.Run("config.TableStoreLimits matches table_store_limits when it is configured", func(t *testing.T) {
			cr := wftest.NothingEnabledCR(func(w *wf.Wavefront) {
				w.Spec.ClusterSize = wf.ClusterSizeSmall
				w.Spec.Experimental.Pixie.TableStoreLimits = wf.TableStoreLimits{TotalMiB: 9, HttpEventsPercent: 10}
			})

			config := FromWavefront(cr)

			require.Equal(t, cr.Spec.Experimental.Pixie.TableStoreLimits, config.TableStoreLimits)
		})
	})
}
