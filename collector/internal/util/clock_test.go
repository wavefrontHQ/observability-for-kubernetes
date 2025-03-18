package util

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const (
	timeAndTz = "15:04 MST"
)

type clockForTesting struct {

	// The current time.
	Current time.Time
}

func (c *clockForTesting) Now() time.Time {
	return c.Current
}

// After immediately advances current time by d and sends that currnet time
// on the returned channel.
func (c *clockForTesting) After(d time.Duration) <-chan time.Time {
	c.Current = c.Current.Add(d)
	result := make(chan time.Time, 1)
	result <- c.Current
	close(result)
	return result
}

func TestEverySeven(t *testing.T) {
	startTime := time.Date(2024, 10, 28, 16, 30, 0, 0, time.UTC)
	fakeClock := &clockForTesting{Current: startTime}
	everySeven, stop := RegularInterval(fakeClock, startTime, 7*time.Minute)
	assert.Equal(t, "16:30 UTC", (<-everySeven).Format(timeAndTz))
	assert.Equal(t, "16:37 UTC", (<-everySeven).Format(timeAndTz))

	// Pretend that we wait a long time before reading the channel.
	// All the hours should still show up.
	fakeClock.Current = time.Date(2024, 10, 28, 17, 0, 0, 0, time.UTC)

	assert.Equal(t, "16:44 UTC", (<-everySeven).Format(timeAndTz))
	assert.Equal(t, "16:51 UTC", (<-everySeven).Format(timeAndTz))

	// Make sure time is still 17:00
	assert.Equal(t, "17:00 UTC", fakeClock.Now().Format(timeAndTz))

	assert.Equal(t, "16:58 UTC", (<-everySeven).Format(timeAndTz))
	assert.Equal(t, "17:05 UTC", (<-everySeven).Format(timeAndTz))
	stop()
	stop()
	for range everySeven {
	}
}

func TestSystemClock(t *testing.T) {
	clock := SystemClock{}
	assert.NotZero(t, clock.Now())
	assert.NotZero(t, <-clock.After(1*time.Nanosecond))
}
