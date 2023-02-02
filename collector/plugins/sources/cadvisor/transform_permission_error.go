package cadvisor

import (
	"errors"
	"fmt"

	"github.com/wavefronthq/observability-for-kubernetes/collector/plugins/sources/prometheus"
)

func TransformPermissionError(err error) error {
	var httpErr *prometheus.HTTPError
	if errors.As(err, &httpErr) && httpErr != nil && (httpErr.StatusCode == 401 || httpErr.StatusCode == 403) {
		return fmt.Errorf("missing nodes/metrics permission in the collector's cluster role: %s", err.Error())
	}
	return err
}
