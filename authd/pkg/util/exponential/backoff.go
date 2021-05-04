package exponential

import (
	"math"
	"math/rand"
	"time"
)

// Backoff5Min goes from 1 second to 5 minutes with each delay is twice bigger.
var Backoff5Min = NewBackoff(time.Second, 5*time.Minute, 2.0)

const NoiseDelayMs = 1000 // Each delay has random additional milliseconds.

type Backoff struct {
	initial time.Duration
	max     time.Duration
	factor  float64

	// Cached value
	maxRetries *int
}

func NewBackoff(initial time.Duration, max time.Duration, factor float64) *Backoff {
	return &Backoff{
		initial: initial,
		max:     max,
		factor:  factor,
	}
}

// Delay returns delay distributed from initial to max.
//
// Example:
//   Retry 0: 5s
//   Retry 1: 6s
//   Retry 2: 7.8s
//   Retry 3: 9.8s
//   Retry 4: 13s
//   Retry 5: 21s
//   Retry 6: 32s
//   Retry 7: 32s
func (b Backoff) Delay(retries int) time.Duration {
	return b.calculateDelayWithMax(NoiseDelayMs, retries)
}

// calculateDelayWithMax returns delay distributed from initialDelay to maxDelay based on retries number.
//
// Delay for retry number 0 is an initialDelay.
//
// Calculation of exponential delays starts from retry number 1.
//
// maxRetries is precalculated and after maxRetries rounds of calculations, maxDelay is returned.
func (b Backoff) calculateDelayWithMax(randomMs int, retries int) time.Duration {
	if b.maxRetries == nil {
		// Count of exponential calculations before return max delay to prevent overflow with big numbers.
		m := int(math.Log(b.max.Seconds()) / math.Log(b.factor))
		b.maxRetries = &m
	}

	var delayNs int64
	switch {
	case retries == 0:
		return b.initial
	case retries <= *b.maxRetries:
		// Calculate exponential delta for delay.
		delayNs = int64(float64(time.Second) * math.Pow(b.factor, float64(retries-1)))
	default:
		// No calculation, return maxDelay.
		delayNs = b.max.Nanoseconds()
	}

	// Random addition to delay.
	noiseDelayNs := rand.Int63n(int64(randomMs)) * int64(time.Millisecond)

	delay := b.initial + time.Duration(delayNs) + time.Duration(noiseDelayNs)
	delay = delay.Truncate(100 * time.Millisecond)
	if delay.Nanoseconds() > b.max.Nanoseconds() {
		return b.max
	}
	return delay
}
