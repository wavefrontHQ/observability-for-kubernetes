package version

import (
	"fmt"
	"strconv"
	"strings"

	gm "github.com/rcrowley/go-metrics"
)

// These variables are set by the build process
var (
	Version string
	Commit  string
)

func Float64() float64 {
	parts := strings.Split(Version, ".")
	if len(parts) != 3 {
		return 0.0
	}
	friendly := fmt.Sprintf("%s.%s%s", parts[0], parts[1], parts[2])
	f, _ := strconv.ParseFloat(friendly, 2)
	return f
}

func RegisterMetric() {
	gm.GetOrRegisterGaugeFloat64("version", gm.DefaultRegistry).Update(Float64())
}
