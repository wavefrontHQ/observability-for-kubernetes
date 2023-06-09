package kstate

import (
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/configuration"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/wf"
	"k8s.io/apimachinery/pkg/api/resource"
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
				v1.ReadOnlyMany,
			},
			VolumeName:       "pvc-volume-1",
			StorageClassName: &storageClassName,
			Resources: v1.ResourceRequirements{
				Requests: map[v1.ResourceName]resource.Quantity{
					v1.ResourceStorage: *resource.NewQuantity(7, resource.BinarySI),
				},
			},
		},
		Status: v1.PersistentVolumeClaimStatus{
			Phase:    v1.ClaimBound,
			Capacity: nil,
			Conditions: []v1.PersistentVolumeClaimCondition{
				{Type: v1.PersistentVolumeClaimResizing, Status: v1.ConditionTrue},
				{Type: v1.PersistentVolumeClaimFileSystemResizePending, Status: v1.ConditionFalse},
			},
			AllocatedResources: nil,
			ResizeStatus:       nil,
		},
	}
}

func getBasicMetricInput() (claim *v1.PersistentVolumeClaim, transforms configuration.Transforms, timestamp int64, tags map[string]string) {
	return setupBasicPVC(),
		configuration.Transforms{Prefix: "kubernetes.", Source: "test-source"},
		0,
		map[string]string{
			"tag1": "value1",
			"tag2": "value2",
			"tag3": "value3",
		}
}

func TestPointsForPVC(t *testing.T) {
	t.Run("test for basic PVC", func(t *testing.T) {
		testPVC := setupBasicPVC()
		actualWFPoints := pointsForPVC(testPVC, configuration.Transforms{Prefix: "kubernetes."})
		assert.Equal(t, 7, len(actualWFPoints))

		expectedMetricNames := []string{
			"kubernetes.pvc.info",
			"kubernetes.pvc.access_mode",
			"kubernetes.pvc.access_mode",
			"kubernetes.pvc.status.phase",
			"kubernetes.pvc.status.condition",
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

	t.Run("metric tags and values", func(t *testing.T) {
		t.Run("buildPVCRequestStorage has base tags and resource storage from requests", func(t *testing.T) {
			actualMetric := buildPVCRequestStorage(getBasicMetricInput())
			expectedMetric := []wf.Metric{
				metricPoint(
					"kubernetes.",
					"pvc.request.storage_bytes",
					7.0,
					0.0,
					"test-source",
					map[string]string{
						"tag1": "value1",
						"tag2": "value2",
						"tag3": "value3",
					},
				),
			}
			assert.Equal(t, expectedMetric, actualMetric)
		})

		t.Run("buildPVCInfo has base tags, volume name, storage class name by default", func(t *testing.T) {
			actualMetric := buildPVCInfo(getBasicMetricInput())
			expectedMetric := metricPoint(
				"kubernetes.",
				"pvc.info",
				1.0,
				0.0,
				"test-source",
				map[string]string{
					"tag1":               "value1",
					"tag2":               "value2",
					"tag3":               "value3",
					"volume_name":        "pvc-volume-1",
					"storage_class_name": "storage-class-1",
				},
			)
			assert.Equal(t, expectedMetric, actualMetric)
		})

		t.Run("buildPVCInfo gets storage class from beta annotation first", func(t *testing.T) {
			claim, transforms, value, tags := getBasicMetricInput()
			claim.Annotations = map[string]string{
				v1.BetaStorageClassAnnotation: "test-beta-storage-class-name",
			}

			actualMetric := buildPVCInfo(claim, transforms, value, tags)
			expectedMetric := metricPoint(
				"kubernetes.",
				"pvc.info",
				1.0,
				0.0,
				"test-source",
				map[string]string{
					"tag1":               "value1",
					"tag2":               "value2",
					"tag3":               "value3",
					"volume_name":        "pvc-volume-1",
					"storage_class_name": "test-beta-storage-class-name",
				},
			)
			assert.Equal(t, expectedMetric, actualMetric)
		})

		t.Run("buildPVCPhaseMetric has phase and converted value", func(t *testing.T) {
			actualMetric := buildPVCPhaseMetric(getBasicMetricInput())
			expectedMetric := metricPoint(
				"kubernetes.",
				"pvc.status.phase",
				2.0,
				0.0,
				"test-source",
				map[string]string{
					"tag1":  "value1",
					"tag2":  "value2",
					"tag3":  "value3",
					"phase": "Bound",
				},
			)
			assert.Equal(t, expectedMetric, actualMetric)
		})

		t.Run("buildPVCConditions has a metric with status and condition for each condition", func(t *testing.T) {
			actualMetrics := buildPVCConditions(getBasicMetricInput())
			resizingMetric := metricPoint(
				"kubernetes.",
				"pvc.status.condition",
				1.0,
				0.0,
				"test-source",
				map[string]string{
					"tag1":      "value1",
					"tag2":      "value2",
					"tag3":      "value3",
					"status":    string(v1.ConditionTrue),
					"condition": string(v1.PersistentVolumeClaimResizing),
				},
			)

			fsResizePending := metricPoint(
				"kubernetes.",
				"pvc.status.condition",
				0.0,
				0.0,
				"test-source",
				map[string]string{
					"tag1":      "value1",
					"tag2":      "value2",
					"tag3":      "value3",
					"status":    string(v1.ConditionFalse),
					"condition": string(v1.PersistentVolumeClaimFileSystemResizePending),
				},
			)

			assert.Contains(t, actualMetrics, resizingMetric)
			assert.Contains(t, actualMetrics, fsResizePending)
		})

		t.Run("buildPVCAccessModes has a metric with access mode tag for each access mode", func(t *testing.T) {
			actualMetrics := buildPVCAccessModes(getBasicMetricInput())
			rwoMetric := metricPoint(
				"kubernetes.",
				"pvc.access_mode",
				1.0,
				0.0,
				"test-source",
				map[string]string{
					"tag1":        "value1",
					"tag2":        "value2",
					"tag3":        "value3",
					"access_mode": string(v1.ReadWriteOnce),
				},
			)

			romMetric := metricPoint(
				"kubernetes.",
				"pvc.access_mode",
				1.0,
				0.0,
				"test-source",
				map[string]string{
					"tag1":        "value1",
					"tag2":        "value2",
					"tag3":        "value3",
					"access_mode": string(v1.ReadOnlyMany),
				},
			)

			assert.Contains(t, actualMetrics, rwoMetric)
			assert.Contains(t, actualMetrics, romMetric)
		})
	})
}
