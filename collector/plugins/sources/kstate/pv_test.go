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

func setupBasicPV() *v1.PersistentVolume {
	return &v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "pv1",
			Labels: nil,
		},
		Spec: v1.PersistentVolumeSpec{
			Capacity:                      nil,
			PersistentVolumeSource:        v1.PersistentVolumeSource{},
			AccessModes:                   nil,
			ClaimRef:                      nil,
			PersistentVolumeReclaimPolicy: "",
			StorageClassName:              "",
			MountOptions:                  nil,
			VolumeMode:                    nil,
			NodeAffinity:                  nil,
		},
		Status: v1.PersistentVolumeStatus{
			Phase:   "",
			Message: "",
			Reason:  "",
		},
	}
}

func TestPointsForPV(t *testing.T) {
	t.Run("test for basic PV", func(t *testing.T) {
		testPV := setupBasicPV()
		actualWFPoints := pointsForPV(testPV, configuration.Transforms{Prefix: "kubernetes."})
		assert.Equal(t, 3, len(actualWFPoints))

		expectedMetricNames := []string{
			"kubernetes.pv.capacity_bytes",
			"kubernetes.pv.status.phase",
			"kubernetes.pv.info",
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
