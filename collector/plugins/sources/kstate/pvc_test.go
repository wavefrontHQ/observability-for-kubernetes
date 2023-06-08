package kstate

import (
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/configuration"
	"log"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func setupBasicPVC() *v1.PersistentVolumeClaim {
	storageClassName := "storage-class-1"
	return &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "pvc1",
			Labels: nil,
		},
		Spec: v1.PersistentVolumeClaimSpec{
			AccessModes: []v1.PersistentVolumeAccessMode{
				v1.ReadWriteOnce,
			},
			VolumeName:       "pvc-volume-1",
			StorageClassName: &storageClassName,
		},
		Status: v1.PersistentVolumeClaimStatus{
			Phase:       "",
			AccessModes: nil,
			Capacity:    nil,
			Conditions: []v1.PersistentVolumeClaimCondition{
				{Type: v1.PersistentVolumeClaimResizing, Status: v1.ConditionTrue},
			},
			AllocatedResources: nil,
			ResizeStatus:       nil,
		},
	}
}

func TestPointsForPVC(t *testing.T) {
	t.Run("test for basic PVC", func(t *testing.T) {
		testPVC := setupBasicPVC()
		actualWFPoints := pointsForPVC(testPVC, configuration.Transforms{Prefix: "kubernetes."})
		assert.Equal(t, 5, len(actualWFPoints))

		expectedMetricNames := []string{
			"kubernetes.pvc.info",
			"kubernetes.pvc.access_mode",
			"kubernetes.pvc.status.phase",
			"kubernetes.pvc.status.condition",
			"kubernetes.pvc.request.storage_bytes",
		}

		var actualMetricNames []string

		for _, point := range actualWFPoints {
			log.Printf("Point name: %s\n", point.Name())
			actualMetricNames = append(actualMetricNames, point.Name())
		}

		sort.Strings(expectedMetricNames)
		sort.Strings(actualMetricNames)
		assert.Equal(t, expectedMetricNames, actualMetricNames)
	})
}
