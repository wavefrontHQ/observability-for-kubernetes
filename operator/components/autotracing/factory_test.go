package autotracing

import (
	"testing"

	"github.com/stretchr/testify/require"
	wf "github.com/wavefronthq/observability-for-kubernetes/operator/api/v1alpha1"
	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/testhelper/wftest"
)

func TestFromWavefront(t *testing.T) {

	t.Run("valid config for autotracing enabled", func(t *testing.T) {
		cr := wftest.NothingEnabledCR(func(w *wf.Wavefront) {
			w.Spec.CanExportData = true
			w.Spec.Experimental.Autotracing.Enable = true
		})
		config := FromWavefront(cr)

		require.True(t, config.Enable)
		require.Equal(t, cr.Spec.ControllerManagerUID, config.ControllerManagerUID)
		require.Equal(t, cr.Spec.Namespace, config.Namespace)
	})

	t.Run("valid config for autotracing enabled but proxy not running", func(t *testing.T) {
		cr := wftest.NothingEnabledCR(func(w *wf.Wavefront) {
			w.Spec.CanExportData = false
			w.Spec.Experimental.Autotracing.Enable = true
		})
		config := FromWavefront(cr)

		require.False(t, config.Enable)
		require.Equal(t, cr.Spec.ControllerManagerUID, config.ControllerManagerUID)
		require.Equal(t, cr.Spec.Namespace, config.Namespace)
	})

	t.Run("valid config for autotracing not enabled", func(t *testing.T) {
		cr := wftest.NothingEnabledCR(func(w *wf.Wavefront) {
			w.Spec.CanExportData = false
			w.Spec.Experimental.Autotracing.Enable = false
		})
		config := FromWavefront(cr)

		require.False(t, config.Enable)
		require.Equal(t, cr.Spec.ControllerManagerUID, config.ControllerManagerUID)
		require.Equal(t, cr.Spec.Namespace, config.Namespace)
	})
}
