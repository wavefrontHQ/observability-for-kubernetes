package validation

import (
	"fmt"

	"github.com/wavefronthq/observability-for-kubernetes/operator/api/common"
	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/result"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/errors"
)

func ValidateContainerResources(resources *common.ContainerResources, resourceName string) result.Result {
	var errs []error
	if len(resources.Limits.Memory) == 0 {
		errs = append(errs, fmt.Errorf("invalid %s.resources.limits.memory must be set", resourceName))
	}
	if len(resources.Limits.CPU) == 0 {
		errs = append(errs, fmt.Errorf("invalid %s.resources.limits.cpu must be set", resourceName))
	}
	if len(errs) > 0 {
		return result.NewError(errors.NewAggregate(errs))
	}

	errs = append(errs, validateResources(resources, resourceName+".resources")...)

	err := errors.NewAggregate(errs)
	if err != nil {
		return result.NewError(errors.NewAggregate(errs))
	}
	return result.Valid
}

func validateResources(resources *common.ContainerResources, resourcePath string) []error {
	var errs []error
	errs = append(errs, validateContainerResource(resources.Requests, resourcePath+".requests")...)
	errs = append(errs, validateContainerResource(resources.Limits, resourcePath+".limits")...)
	if len(errs) > 0 {
		return errs
	}
	errs = validateContainerResourcesQuantities(resources, resourcePath, errs)
	return errs
}

func validateContainerResourcesQuantities(resources *common.ContainerResources, resourcePath string, errs []error) []error {
	if compareQuantities(resources.Requests.CPU, resources.Limits.CPU) > 0 {
		errs = append(errs, fmt.Errorf("invalid %s.requests.cpu: %s must be less than or equal to cpu limit", resourcePath, resources.Requests.CPU))
	}
	if compareQuantities(resources.Requests.Memory, resources.Limits.Memory) > 0 {
		errs = append(errs, fmt.Errorf("invalid %s.requests.memory: %s must be less than or equal to memory limit", resourcePath, resources.Requests.Memory))
	}
	if compareQuantities(resources.Requests.EphemeralStorage, resources.Limits.EphemeralStorage) > 0 {
		errs = append(errs, fmt.Errorf("invalid %s.requests.ephemeral-storage: %s must be less than or equal to ephemeral-storage limit", resourcePath, resources.Requests.EphemeralStorage))
	}
	return errs
}

func validateContainerResource(containerResource common.ContainerResource, resourcePath string) []error {
	var errs []error
	if err := validateResourceQuantity(containerResource.CPU, resourcePath+".cpu"); err != nil {
		errs = append(errs, err)
	}
	if err := validateResourceQuantity(containerResource.Memory, resourcePath+".memory"); err != nil {
		errs = append(errs, err)
	}
	if err := validateResourceQuantity(containerResource.EphemeralStorage, resourcePath+".ephemeral-storage"); err != nil {
		errs = append(errs, err)
	}
	return errs
}

func validateResourceQuantity(quantity, resourcePath string) error {
	if len(quantity) > 0 {
		if _, err := resource.ParseQuantity(quantity); err != nil {
			return fmt.Errorf("invalid %s: '%s'", resourcePath, quantity)
		}
	}
	return nil
}

func compareQuantities(request string, limit string) int {
	requestQuantity, _ := resource.ParseQuantity(request)
	limitQuanity, _ := resource.ParseQuantity(limit)
	return requestQuantity.Cmp(limitQuanity)
}
