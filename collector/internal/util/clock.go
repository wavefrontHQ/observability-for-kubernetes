package util

import (
	"sync"
	"time"
)

// Interface Clock represents the clock functions this package needs.
// SystemClock implements this interface.
type Clock interface {

	// Now returns the current time in the local time zone.
	Now() time.Time

	// After waits for given duration to elapse and then sends current time on
	// the returned channel.
	After(d time.Duration) <-chan time.Time
}

// RegularInterval returns start, start + interval, start + 2*interval etc.
// on result at those times. For example, if interval is 7*time.Minute,
// RegularInterval returns a time every 7 minutes on result. When caller wants
// to quit reading from result, the caller must call the stop function. After
// calling stop, the caller must read any times in the past from the result channel.
// Once the caller reads all of those, RegularInterval closes the result
// channel.
func RegularInterval(
	clock Clock, start time.Time, interval time.Duration) (result <-chan time.Time, stop func()) {
	resultCh := make(chan time.Time)
	stopCh := make(chan struct{})
	result = resultCh
	stop = sync.OnceFunc(func() {
		close(stopCh)
	})
	go func() {
		nextTime := start
		for {
			timeToWait := nextTime.Sub(clock.Now())
			if timeToWait > 0 {
				select {
				case <-stopCh:
					close(resultCh)
					return
				case <-clock.After(timeToWait):
				}
			}
			resultCh <- nextTime
			nextTime = nextTime.Add(interval)
		}
	}()
	return
}

// SystemClock represents the system clock. SystemClock implements Clock.
type SystemClock struct {
}

// Now calls time.Now()
func (s SystemClock) Now() time.Time {
	return time.Now()
}

// After calls time.After()
func (s SystemClock) After(d time.Duration) <-chan time.Time {
	return time.After(d)
}
