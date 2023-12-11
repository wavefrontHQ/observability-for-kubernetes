package logging

import (
	"testing"

	"github.com/stretchr/testify/require"
	wf "github.com/wavefronthq/observability-for-kubernetes/operator/api/v1alpha1"
	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/testhelper/wftest"
)

func TestFromWavefront(t *testing.T) {
	t.Run("valid wavefront spec config", func(t *testing.T) {
		cr := wftest.CR(func(w *wf.Wavefront) {
			w.Spec.CanExportData = true
			w.Spec.DataCollection.Logging.Enable = true
		})
		config := FromWavefront(cr)

		require.True(t, config.Enable)
		require.True(t, config.ShouldValidate)
	})

	t.Run("component config enable should be set to false", func(t *testing.T) {
		cr := wftest.CR(func(w *wf.Wavefront) {
			w.Spec.CanExportData = false
		})
		config := FromWavefront(cr)

		require.False(t, config.Enable)
		require.True(t, config.ShouldValidate)
	})
}
