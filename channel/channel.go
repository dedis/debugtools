package channel

import (
	"context"
	"fmt"
	"runtime"
	"time"
)

type TimedChannel[T any] struct {
	c chan T
	t time.Duration
}

// NewTimedChannel creates a new cryChan of the given type and size
func NewTimedChannel[T any](bufSize int, timeout time.Duration) TimedChannel[T] {
	Logger = Logger.With().Int("size", bufSize).Logger()

	return TimedChannel[T]{
		c: make(chan T, bufSize),
		t: timeout,
	}
}

// Push adds an element in the channel, or logs if it's blocked for too long
func (c *TimedChannel[T]) Push(e T) {
	start := time.Now()
	select {
	case c.c <- e:
	case <-time.After(c.t):
		// prints the first 16 bytes of the trace, which should contain at least
		// the goroutine id.
		trace := make([]byte, 16)
		runtime.Stack(trace, false)
		Logger.Warn().Str("obj", fmt.Sprintf("%+v", e)).
			Str("trace", string(trace)).Msg("channel blocking")
		c.c <- e
		Logger.Warn().Str("obj", fmt.Sprintf("%+v", e)).
			Str("elapsed", time.Since(start).String()).
			Str("trace", string(trace)).Msg("channel unblocked")
	}
}

// Pop removes an element from the channel
func (c *TimedChannel[T]) Pop(ctx context.Context) (t T, err error) {
	select {
	case el := <-c.c:
		return el, nil
	case <-ctx.Done():
		return t, ctx.Err()
	}
}

// Len gives the current number of elements in the channel
func (c *TimedChannel[T]) Len() int {
	return len(c.c)
}
