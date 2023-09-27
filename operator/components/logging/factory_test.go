package logging

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/testhelper/wftest"
)

func TestFromWavefront(t *testing.T) {
	t.Run("valid wavefront spec config", func(t *testing.T) {
		cr := wftest.CR()
		config := FromWavefront(cr)
		component, _ := NewComponent(ComponentDir, config)

		require.True(t, component.Validate().IsValid())
	})

	t.Run("component config enable should be set to false", func(t *testing.T) {
		cr := wftest.CR()
		cr.Spec.CanExportData = false
		config := FromWavefront(cr)

		require.False(t, config.Enable)
	})
}
