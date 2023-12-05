package validation

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wavefronthq/observability-for-kubernetes/operator/api/common"
	rc "github.com/wavefronthq/observability-for-kubernetes/operator/api/resourcecustomizations/v1alpha1"
)

func TestValidateRC(t *testing.T) {
	t.Run("resources", func(t *testing.T) {
		t.Run("resources cannot specify only requests", func(t *testing.T) {
			crSet := defaultCRSet()
			crSet.ResourceCustomizations.Spec.ByName = map[string]rc.WorkloadCustomization{
				"some-resource": {
					Resources: common.ContainerResources{
						Requests: common.ContainerResource{
							CPU:              "100m",
							Memory:           "1Gi",
							EphemeralStorage: "10Gi",
						},
					},
				},
			}
			result := ValidateRC(&crSet.ResourceCustomizations)

			require.False(t, result.IsValid(), "result should not be valid")
			require.Contains(t, result.Message(), "invalid spec.byName.some-resource.resources.limits.memory must be set")
		})

		t.Run("ignores empty resources", func(t *testing.T) {
			crSet := defaultCRSet()
			crSet.ResourceCustomizations.Spec.ByName = map[string]rc.WorkloadCustomization{
				"some-resource": {},
			}
			result := ValidateRC(&crSet.ResourceCustomizations)

			require.True(t, result.IsValid(), "result should be valid")
		})

		t.Run("resource limits must be bigger than requests", func(t *testing.T) {
			crSet := defaultCRSet()
			crSet.ResourceCustomizations.Spec.ByName = map[string]rc.WorkloadCustomization{
				"some-resource": {
					Resources: common.ContainerResources{
						Requests: common.ContainerResource{
							CPU:              "1",
							Memory:           "10Gi",
							EphemeralStorage: "100Gi",
						},
						Limits: common.ContainerResource{
							CPU:              "100m",
							Memory:           "1Gi",
							EphemeralStorage: "10Gi",
						},
					},
				},
			}
			result := ValidateRC(&crSet.ResourceCustomizations)

			require.False(t, result.IsValid(), "result should not be valid")
			require.Contains(t, result.Message(), "invalid spec.byName.some-resource.resources.requests.cpu: 1 must be less than or equal to cpu limit")
		})
	})
}
