package kstate

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestBuildTags(t *testing.T) {
	t.Run("creates new tags map and copies in key, namespace, and source tags", func(t *testing.T) {
		expectedTags := map[string]string{
			"fake-key":       "fake-name",
			"namespace_name": "fake-namespace",
			"fake-src-key-1": "fake-src-val-1",
		}

		actualTags := buildTags("fake-key", "fake-name", "fake-namespace", map[string]string{
			"fake-src-key-1": "fake-src-val-1",
		})

		require.Equal(t, expectedTags, actualTags)
	})
}
