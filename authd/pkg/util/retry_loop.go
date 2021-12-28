package util

import (
	"context"
	"errors"
	"time"

	"github.com/flant/negentropy/authd/pkg/util/exponential"
)

const (
	RefreshDelayInitial = time.Second
	RefreshDelayFactor  = 2.0
	RefreshDelayMax     = 5 * time.Minute
	RefreshWaitDelay    = time.Minute // delays between polling for refresh time.
)

var StopRetriesErr = errors.New("RetryLoop is stopped by handler")

type PostponedRetryLoop struct {
	StartAfter time.Time
	Handler    func(context.Context) error
	Backoff    *exponential.Backoff

	retries int

	cancel context.CancelFunc
	done   chan struct{}
}

func (r *PostponedRetryLoop) RunLoop(ctx context.Context) error {
	if r.Handler == nil {
		return nil
	}
	if r.StartAfter.IsZero() {
		r.StartAfter = time.Now()
	}
	if r.Backoff == nil {
		r.Backoff = exponential.NewBackoff(RefreshDelayInitial, RefreshDelayMax, RefreshDelayFactor)
	}

	var retryLoopCtx context.Context
	retryLoopCtx, r.cancel = context.WithCancel(ctx)
	r.done = make(chan struct{})

	return r.runRetryLoop(retryLoopCtx)
}

func (r *PostponedRetryLoop) Stop() {
	if r.cancel != nil {
		r.cancel()
	}
	<-r.done
}

func (r *PostponedRetryLoop) runRetryLoop(ctx context.Context) error {
	defer func() {
		close(r.done)
	}()

	// Wait until refresh time.
	err := r.waitForRefreshTime(ctx)
	if err != nil {
		return err
	}

	// Do retries until success or cancel or explicit stop.
	for {
		err = r.handle(ctx)
		if err == nil || errors.Is(err, context.Canceled) || errors.Is(err, StopRetriesErr) {
			return err
		}
		// Logging error should be done in Handler.
	}
}

func (r *PostponedRetryLoop) waitForRefreshTime(ctx context.Context) error {
	now := time.Now()
	// Return immediately if it is already time to refresh.
	if now.After(r.StartAfter) {
		return nil
	}

	// Sleep if delay is less then hour (a maximum Duration discretization).
	delta := r.StartAfter.Sub(now)
	if delta.Seconds() <= time.Hour.Seconds() {
		time.Sleep(delta)
		return nil
	}

	// Wait for refresh time.
	tk := time.NewTicker(RefreshWaitDelay)
	defer tk.Stop()
	for {
		select {
		case <-tk.C:
			now := time.Now()
			if now.After(r.StartAfter) {
				return nil
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (r *PostponedRetryLoop) handle(ctx context.Context) error {
	// Check if refresher is stopped.
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Call Handler and do sleep if a general error is occurred.
	err := r.Handler(ctx)
	if err == nil || errors.Is(err, context.Canceled) || errors.Is(err, StopRetriesErr) {
		return err
	}
	delay := r.Backoff.Delay(r.retries)
	sleepErr := stoppableSleep(ctx, delay)
	// Return Canceled error if sleep is interrupted.
	if sleepErr != nil {
		return sleepErr
	}
	r.retries++
	return err
}

func stoppableSleep(ctx context.Context, delay time.Duration) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	t := time.NewTicker(delay)
	defer t.Stop()
	select {
	case <-t.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
