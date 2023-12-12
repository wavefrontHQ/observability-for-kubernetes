package collector

import (
	"testing"

	"github.com/stretchr/testify/require"
	wf "github.com/wavefronthq/observability-for-kubernetes/operator/api/v1alpha1"
	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/testhelper/wftest"
)

func TestFromWavefront(t *testing.T) {

	t.Run("valid wavefront spec config for metrics enabled", func(t *testing.T) {
		cr := wftest.CR(func(w *wf.Wavefront) {
			w.Spec.DataCollection.Metrics.Enable = true
			w.Spec.CanExportData = true
		})
		config := FromWavefront(cr)

		require.True(t, config.Enable)
	})

	t.Run("valid wavefront spec config for metrics enabled and CanExportData not enabled", func(t *testing.T) {
		cr := wftest.CR(func(w *wf.Wavefront) {
			w.Spec.DataCollection.Metrics.Enable = true
			w.Spec.CanExportData = false
		})
		config := FromWavefront(cr)

		require.False(t, config.Enable)
		require.True(t, config.ShouldValidate)
	})

	t.Run("valid wavefront spec config for insights enabled", func(t *testing.T) {
		cr := wftest.CR(func(w *wf.Wavefront) {
			w.Spec.DataCollection.Metrics.Enable = false
			w.Spec.CanExportData = false
			w.Spec.Experimental.Insights.Enable = true
			w.Spec.Experimental.Insights.IngestionUrl = "https://example.com"
		})
		config := FromWavefront(cr)

		require.True(t, config.Enable)
	})

	t.Run("component config enable should be set to false when metrics disabled", func(t *testing.T) {
		cr := wftest.CR(func(w *wf.Wavefront) {
			w.Spec.DataCollection.Metrics.Enable = false
		})
		config := FromWavefront(cr)

		require.False(t, config.Enable)
	})

	t.Run("component config enable should be set to false when insights disabled", func(t *testing.T) {
		cr := wftest.CR(func(w *wf.Wavefront) {
			w.Spec.Experimental.Insights.Enable = false
		})
		config := FromWavefront(cr)

		require.False(t, config.Enable)
	})

	t.Run("defaults node collector resources if empty", func(t *testing.T) {
		cr := wftest.CR(func(w *wf.Wavefront) {
			w.Spec.DataCollection.Metrics.NodeCollector.Resources = wf.Resources{}
		})
		config := FromWavefront(cr)

		require.False(t, config.NodeCollectorResources.IsEmpty())
	})

	t.Run("if does not default node collector resources if they are not empty", func(t *testing.T) {
		resources := wf.Resources{
			Requests: wf.Resource{
				CPU:              "100m",
				Memory:           "10Mi",
				EphemeralStorage: "20Mi",
			},
			Limits: wf.Resource{
				CPU:              "900m",
				Memory:           "512Mi",
				EphemeralStorage: "512Mi",
			},
		}
		cr := wftest.CR(func(w *wf.Wavefront) {
			w.Spec.DataCollection.Metrics.NodeCollector.Resources = resources

		})
		config := FromWavefront(cr)

		require.Equal(t, resources, config.NodeCollectorResources)
	})

	t.Run("defaults cluster collector resources if empty", func(t *testing.T) {
		cr := wftest.CR(func(w *wf.Wavefront) {
			w.Spec.DataCollection.Metrics.ClusterCollector.Resources = wf.Resources{}
		})
		config := FromWavefront(cr)

		require.False(t, config.ClusterCollectorResources.IsEmpty())
	})

	t.Run("if does not default cluster collector resources if they are not empty", func(t *testing.T) {
		resources := wf.Resources{
			Requests: wf.Resource{
				CPU:              "100m",
				Memory:           "10Mi",
				EphemeralStorage: "20Mi",
			},
			Limits: wf.Resource{
				CPU:              "900m",
				Memory:           "512Mi",
				EphemeralStorage: "512Mi",
			},
		}
		cr := wftest.CR(func(w *wf.Wavefront) {
			w.Spec.DataCollection.Metrics.ClusterCollector.Resources = resources

		})
		config := FromWavefront(cr)

		require.Equal(t, resources, config.ClusterCollectorResources)
	})
}
