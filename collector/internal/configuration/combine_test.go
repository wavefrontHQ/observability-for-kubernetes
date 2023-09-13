package configuration_test

import (
	"math"
	"testing"
	"time"

	fuzz "github.com/google/gofuzz"
	"github.com/stretchr/testify/require"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/configuration"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/util"
)

const MaxRuns = 1_000_000

func TestCombine(t *testing.T) {
	f := fuzz.New().NilChance(0).Funcs(
		func(e *util.WorkloadCache, c fuzz.Continue) {},
		func(e *configuration.Config, c fuzz.Continue) {
			c.Fuzz(e)
			e.FlushInterval = time.Duration(c.Int63n(math.MaxInt64))
			if len(e.Sinks) == 0 {
				e.Sinks = nil
			}
		},
	)

	t.Run("combining any config with an empty config always results in the same config (identity)", func(t *testing.T) {
		var empty configuration.Config
		var config configuration.Config
		for i := 0; i < MaxRuns; i++ {
			f.Fuzz(&config)

			require.Equal(t, config, *configuration.Combine(&empty, &config), "left identity")
			require.Equal(t, config, *configuration.Combine(&config, &empty), "right identity")
		}
	})

	t.Run("configs combination can be grouped in any order (associativity)", func(t *testing.T) {
		var a configuration.Config
		var b configuration.Config
		var c configuration.Config
		for i := 0; i < MaxRuns; i++ {
			f.Fuzz(&a)
			f.Fuzz(&b)
			f.Fuzz(&c)

			require.Equal(t,
				*configuration.Combine(&a, configuration.Combine(&b, &c)),
				*configuration.Combine(configuration.Combine(&a, &b), &c),
			)
		}
	})

	t.Run("configs can be combined in any order (commutativity)", func(t *testing.T) {
		var a configuration.Config
		var b configuration.Config
		for i := 0; i < MaxRuns; i++ {
			f.Fuzz(&a)
			f.Fuzz(&b)

			require.Equal(t,
				*configuration.Combine(&a, &b),
				*configuration.Combine(&b, &a),
			)
		}
	})
}
