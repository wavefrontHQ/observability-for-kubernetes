package autotracing

import (
	"testing"

	"github.com/stretchr/testify/require"
	wf "github.com/wavefronthq/observability-for-kubernetes/operator/api/v1alpha1"
	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/testhelper/wftest"
)

func TestFromWavefront(t *testing.T) {
	t.Run("valid config for autotracing enabled", func(t *testing.T) {
		cr := validAutoTracingEnabledCR()
		config := FromWavefront(cr)

		require.True(t, config.Enable)
		require.True(t, config.ShouldValidate)
		require.Equal(t, cr.Spec.ControllerManagerUID, config.ControllerManagerUID)
		require.Equal(t, cr.Spec.Namespace, config.Namespace)
	})

	t.Run("valid config for autotracing enabled but proxy and PEMs not running", func(t *testing.T) {
		cr := validAutoTracingEnabledCR(func(w *wf.Wavefront) {
			w.Spec.CanExportData = false
			w.Spec.Experimental.Autotracing.CanExportAutotracingScripts = false
		})
		config := FromWavefront(cr)

		require.False(t, config.Enable)
		require.Equal(t, cr.Spec.ControllerManagerUID, config.ControllerManagerUID)
		require.Equal(t, cr.Spec.Namespace, config.Namespace)
	})

	t.Run("should validate even when not yet ready to export", func(t *testing.T) {
		cr := validAutoTracingEnabledCR(func(w *wf.Wavefront) {
			w.Spec.CanExportData = false
		})
		config := FromWavefront(cr)

		require.False(t, config.Enable)
		require.True(t, config.ShouldValidate)
	})

	t.Run("valid config for autotracing enabled with proxy running and PEMs not running", func(t *testing.T) {
		cr := wftest.NothingEnabledCR(func(w *wf.Wavefront) {
			w.Spec.Experimental.Autotracing.CanExportAutotracingScripts = false
		})
		config := FromWavefront(cr)

		require.False(t, config.Enable)
		require.Equal(t, cr.Spec.ControllerManagerUID, config.ControllerManagerUID)
		require.Equal(t, cr.Spec.Namespace, config.Namespace)
	})

	t.Run("valid config for autotracing not enabled", func(t *testing.T) {
		cr := validAutoTracingEnabledCR(func(w *wf.Wavefront) {
			w.Spec.Experimental.Autotracing.Enable = false
		})
		config := FromWavefront(cr)

		require.False(t, config.Enable)
		require.Equal(t, cr.Spec.ControllerManagerUID, config.ControllerManagerUID)
		require.Equal(t, cr.Spec.Namespace, config.Namespace)
	})
}

func validAutoTracingEnabledCR(options ...wftest.CROption) *wf.Wavefront {
	defaults := func(w *wf.Wavefront) {
		w.Spec.CanExportData = true
		w.Spec.Experimental.Autotracing.Enable = true
		w.Spec.Experimental.Autotracing.CanExportAutotracingScripts = true
	}
	return wftest.NothingEnabledCR(append([]wftest.CROption{defaults}, options...)...)
}
