package util

import (
	gometrics "github.com/rcrowley/go-metrics"
	"sync"
	"time"
)

// Interval starts a particular function at regular time intervals. If Interval is to
// start a function every 7 seconds, and the function takes 2 seconds to run, it will pause
// 5 seconds between runs. If Interval is to start the function every 7 seconds, and the function
// takes 10 seconds to run, that is called an overrun. Interval will log an overrun and wait
// 4 seconds for the start of the following interval to run the function again. If the function
// takes 20 seconds to run on 7 second intervals, Interval will log 2 overruns and wait 1
// second for the start of the following interval to run the function again.
type Interval struct {
	clock        Clock
	interval     time.Duration
	runTimes     <-chan time.Time
	f            func()
	stop         func()
	overrunCount gometrics.Counter
	wg           sync.WaitGroup
}

// NewInterval returns a new Interval. interval is the amount of time between
// function starts e.g 7 seconds, f is the function to run periodically,
// overrunCount is where overruns are logged.
func NewInterval(interval time.Duration, f func(), overrunCount gometrics.Counter) *Interval {
	return newInterval(SystemClock{}, interval, f, overrunCount)
}

// StopAndWait stops the periodic execution of f. In case f is currently running,
// StopAndWait waits for f to finish before returning.
func (in *Interval) StopAndWait() {
	in.stop()
	in.wg.Wait()
}

func (in *Interval) loop() {
	for nextRunTime := range in.runTimes {
		now := in.clock.Now()

		// If nextRunTime is more than 1/30 of an interval earlier than now,
		// we had an overrun. The 1/30 is arbitrary
		if now.Sub(nextRunTime) > in.interval/30 {
			in.overrunCount.Inc(1)
			continue
		}
		in.f()
	}
	in.wg.Done()
}

func newInterval(
	clock Clock, interval time.Duration, f func(), overrunCount gometrics.Counter) *Interval {
	now := clock.Now()
	runTimes, stop := RegularInterval(clock, now.Add(interval), interval)
	result := &Interval{
		clock:        clock,
		interval:     interval,
		runTimes:     runTimes,
		f:            f,
		stop:         stop,
		overrunCount: overrunCount,
	}
	result.wg.Add(1)
	go result.loop()
	return result
}
