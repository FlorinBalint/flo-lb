package algos

import (
	"testing"
	"time"
)

const growth = 1.5
const timeToReset = 10 * time.Second
const maxSleep = 5 * time.Second
const initialSleep = 200 * time.Millisecond

type sleeper struct {
	lastSleep time.Duration
}

func (s *sleeper) sleep(dur time.Duration) {
	s.lastSleep = dur
}

func fixedTimer(ti time.Time) getTime {
	return func() time.Time {
		return ti
	}
}

func TestWaitAbit(t *testing.T) {
	tests := []struct {
		name                          string
		lastSleep                     time.Time
		currentSleepTimeBefore        time.Duration
		expectedSleep                 time.Duration
		currentTime                   time.Time
		expectedCurrentSleepTimeAfter time.Duration
	}{
		{
			name:                          "First sleep is the initial sleep",
			lastSleep:                     time.UnixMilli(0),
			currentSleepTimeBefore:        initialSleep,
			expectedSleep:                 initialSleep,
			currentTime:                   time.UnixMilli(1655562339110), // 18th June 2022
			expectedCurrentSleepTimeAfter: 300 * time.Millisecond,        // (1.5 * 200 initial)
		},
		{
			name:                          "Sleep does not become larger than max sleep",
			lastSleep:                     time.UnixMilli(1655562339110), // 18th June 2022
			currentSleepTimeBefore:        4 * time.Second,               // 4 * 1.5 = 6, but max is 5s
			expectedSleep:                 4 * time.Second,
			currentTime:                   time.UnixMilli(1655562339110).Add(100 * time.Millisecond),
			expectedCurrentSleepTimeAfter: maxSleep,
		},
		{
			name:                          "Sleep resets after resetTime",
			lastSleep:                     time.UnixMilli(1655562339110),
			currentSleepTimeBefore:        maxSleep,
			expectedSleep:                 initialSleep,
			currentTime:                   time.UnixMilli(1655562339110).Add(timeToReset + time.Second),
			expectedCurrentSleepTimeAfter: 300 * time.Millisecond, // (1.5 * 200 initial)
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sl := &sleeper{}
			bo := &Backoff{
				time:             fixedTimer(test.currentTime),
				sleep:            sl.sleep,
				lastSleep:        test.lastSleep,
				maxSleep:         maxSleep,
				growth:           growth,
				timeToReset:      timeToReset,
				initialSleep:     initialSleep,
				currentSleepTime: test.currentSleepTimeBefore,
			}

			bo.WaitABit()
			if sl.lastSleep != test.expectedSleep {
				t.Errorf("want sleep(%v), got sleep(%v)", test.expectedSleep, sl.lastSleep)
			}
			if bo.currentSleepTime != test.expectedCurrentSleepTimeAfter {
				t.Errorf("want currentSleepTime %v, got %v", test.expectedCurrentSleepTimeAfter, bo.currentSleepTime)
			}
		})
	}

}
