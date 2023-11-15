package collector

import (
	"testing"

	"github.com/stretchr/testify/require"
	wf "github.com/wavefronthq/observability-for-kubernetes/operator/api/wavefront/v1alpha1"
	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/testhelper/wftest"
)

func TestFromWavefront(t *testing.T) {

	t.Run("valid wavefront spec config for metrics enabled", func(t *testing.T) {
		cr := wftest.CR(func(w *wf.Wavefront) {
			w.Spec.DataCollection.Metrics.Enable = true
			w.Spec.CanExportData = true
		})
		config := FromWavefront(cr)
		component, _ := NewComponent(ComponentDir, config)

		require.True(t, config.Enable)
		require.Equal(t, "", component.Validate().Message())
		require.True(t, component.Validate().IsValid())
	})

	t.Run("valid wavefront spec config for insights enabled", func(t *testing.T) {
		cr := wftest.CR(func(w *wf.Wavefront) {
			w.Spec.DataCollection.Metrics.Enable = false
			w.Spec.CanExportData = false
			w.Spec.Experimental.Insights.Enable = true
			w.Spec.Experimental.Insights.IngestionUrl = "https://example.com"
		})
		config := FromWavefront(cr)
		component, _ := NewComponent(ComponentDir, config)

		require.True(t, config.Enable)
		require.Equal(t, "", component.Validate().Message())
		require.True(t, component.Validate().IsValid())
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
}
