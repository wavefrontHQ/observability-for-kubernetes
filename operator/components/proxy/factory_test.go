package proxy

import (
	"testing"

	"github.com/stretchr/testify/require"
	wf "github.com/wavefronthq/observability-for-kubernetes/operator/api/v1alpha1"
	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/testhelper/wftest"
)

func TestFromWavefront(t *testing.T) {

	t.Run("valid wavefront spec config for proxy enabled", func(t *testing.T) {
		cr := wftest.CR(func(w *wf.Wavefront) {
			w.Spec.DataExport.WavefrontProxy.Enable = true
		})
		config := FromWavefront(cr)
		component, _ := NewComponent(ComponentDir, config)

		require.True(t, config.Enable)
		require.Equal(t, "", component.Validate().Message())
		require.True(t, component.Validate().IsValid())
	})

	t.Run("component config enable should be set to false", func(t *testing.T) {
		cr := wftest.CR(func(w *wf.Wavefront) {
			w.Spec.DataExport.WavefrontProxy.Enable = false
		})
		config := FromWavefront(cr)

		require.False(t, config.Enable)
	})
}
