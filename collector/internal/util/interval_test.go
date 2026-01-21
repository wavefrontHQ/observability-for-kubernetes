package util

import (
	gm "github.com/rcrowley/go-metrics"
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
	"time"
)

func TestIntervalNoOverrun(t *testing.T) {
	var overruns counterForTesting
	var calls counterForTesting
	interval := NewInterval(70*time.Millisecond, sleepFunc(50*time.Millisecond, &calls), &overruns)
	time.Sleep(150 * time.Millisecond)
	start := time.Now()
	interval.StopAndWait()

	// We expect StopAndWait to take around 40ms
	assert.Less(t, 30*time.Millisecond, time.Since(start))

	// Runs 70ms, 140ms after interval created
	assert.Equal(t, int64(2), calls.Count())
	assert.Equal(t, int64(0), overruns.Count())
}

func TestIntervalOverrun(t *testing.T) {
	var overruns counterForTesting
	var calls counterForTesting
	interval := NewInterval(70*time.Millisecond, sleepFunc(100*time.Millisecond, &calls), &overruns)
	time.Sleep(300 * time.Millisecond)
	start := time.Now()
	interval.StopAndWait()

	// We expect StopAndWait to take around 10ms
	assert.Less(t, 5*time.Millisecond, time.Since(start))

	// Runs 70ms, 210ms after interval created
	assert.Equal(t, int64(2), calls.Count())
	assert.Equal(t, int64(2), overruns.Count())
}

func sleepFunc(d time.Duration, counter gm.Counter) func() {
	return func() {
		time.Sleep(d)
		counter.Inc(1)
	}
}

type counterForTesting struct {
	mu      sync.Mutex
	counter int64
}

func (c *counterForTesting) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.counter = 0
}

func (c *counterForTesting) Count() int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.counter
}

func (c *counterForTesting) Dec(i int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.counter -= i
}

func (c *counterForTesting) Inc(i int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.counter += i
}

func (c *counterForTesting) Snapshot() gm.Counter {
	//TODO implement me
	panic("implement me")
}
