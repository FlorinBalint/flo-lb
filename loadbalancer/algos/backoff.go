package algos

import (
	"log"
	"time"
)

type getTime func() time.Time

var defaultTime getTime = func() time.Time {
	return time.Now()
}

type sleep func(time.Duration)

var defaultSleeper sleep = func(duration time.Duration) {
	time.Sleep(duration)
}

type Backoff struct {
	time             getTime
	sleep            sleep
	lastSleep        time.Time
	initialSleep     time.Duration
	maxSleep         time.Duration
	growth           float64
	timeToReset      time.Duration
	currentSleepTime time.Duration
}

func NewBackoff(initialSleep, maxSleep, timeToReset time.Duration, growth float64) *Backoff {
	log.Printf("Backoff initialSleep: %v, maxSleep:%v, timeToReset: %v, growth:%v", initialSleep, maxSleep, timeToReset, growth)
	return &Backoff{
		time:             defaultTime,
		sleep:            defaultSleeper,
		lastSleep:        time.Unix(0, 0),
		initialSleep:     initialSleep,
		timeToReset:      timeToReset,
		growth:           growth,
		currentSleepTime: initialSleep,
		maxSleep:         maxSleep,
	}
}

func (b *Backoff) WaitABit() {
	currentTime := b.time()
	timeToSleep := b.currentSleepTime
	resetTime := b.lastSleep.Add(b.timeToReset)
	if currentTime.After(resetTime) {
		timeToSleep = b.initialSleep
	}

	log.Printf("Sleeping a bit, %v", timeToSleep)
	b.sleep(timeToSleep)
	b.lastSleep = currentTime
	dur := int64(float64(timeToSleep) * b.growth)
	nextSleepTime := time.Duration(dur)
	if b.maxSleep < nextSleepTime {
		nextSleepTime = b.maxSleep
	}
	b.currentSleepTime = nextSleepTime
}
