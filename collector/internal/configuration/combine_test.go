package configuration_test

import (
	"testing"
	"time"

	fuzz "github.com/google/gofuzz"
	"github.com/r3labs/diff/v3"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/configuration"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/util"
)

const MaxRuns = 100

var seed = time.Now().UnixNano()

func TestCombine(t *testing.T) {
	t.Run("combining any config with an empty config always results in the same config (identity)", func(t *testing.T) {
		f := makeFuzzer(seed)
		for i := 0; i < MaxRuns; i++ {
			var config configuration.Config
			f.Fuzz(&config)

			requireConfigsEqual(t, &config, configuration.Combine(&config, configuration.Empty))
			requireConfigsEqual(t, &config, configuration.Combine(configuration.Empty, &config))
		}
	})

	t.Run("configs combination can be grouped in any order (associativity)", func(t *testing.T) {
		f := makeFuzzer(seed)
		var a configuration.Config
		var b configuration.Config
		var c configuration.Config
		for i := 0; i < MaxRuns; i++ {
			f.Fuzz(&a)
			f.Fuzz(&b)
			f.Fuzz(&c)

			requireConfigsEqual(t,
				configuration.Combine(&a, configuration.Combine(&b, &c)),
				configuration.Combine(configuration.Combine(&a, &b), &c),
			)
		}
	})

	t.Run("configs can be combined in any order (commutativity)", func(t *testing.T) {
		f := makeFuzzer(seed)
		var a configuration.Config
		var b configuration.Config
		for i := 0; i < MaxRuns; i++ {
			f.Fuzz(&a)
			f.Fuzz(&b)

			requireConfigsEqual(t, configuration.Combine(&a, &b), configuration.Combine(&b, &a))
		}
	})

	t.Run("the same config combined with itself produces the same config (idempotence)", func(t *testing.T) {
		f := makeFuzzer(seed)
		var a configuration.Config
		for i := 0; i < MaxRuns; i++ {
			f.Fuzz(&a)

			requireConfigsEqual(t, &a, configuration.Combine(&a, &a))
		}
	})
}

func requireConfigsEqual(t *testing.T, a, b *configuration.Config) {
	t.Helper()
	if !configuration.Equal(a, b) {
		changes, _ := diff.Diff(a, b)
		for _, change := range changes {
			t.Logf("%+#v", change)
		}
		t.Fatal("configurations are not equal")
	}
}

func makeFuzzer(seed int64) *fuzz.Fuzzer {
	f := fuzz.NewWithSeed(seed).NilChance(0).Funcs(
		func(e *util.WorkloadCache, c fuzz.Continue) {},
	)
	return f
}
