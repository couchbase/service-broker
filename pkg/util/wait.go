package util

import (
	"context"
	"fmt"
	"testing"
	"time"
)

const (
	// retryPeriod is how often to poll a WaitFunc.
	retryPeriod = 10 * time.Millisecond
)

// WaitFunc is a callback that stops a wait when nil.
type WaitFunc func() error

// WaitFor waits until a condition is nil.
func WaitFor(f WaitFunc, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	tick := time.NewTicker(retryPeriod)
	defer tick.Stop()

	for err := f(); err != nil; err = f() {
		select {
		case <-tick.C:
		case <-ctx.Done():
			return fmt.Errorf("failed to wait for condition: %v", err)
		}
	}

	return nil
}

// MustWaitFor waits until a condition is nil.
func MustWaitFor(t *testing.T, f WaitFunc, timeout time.Duration) {
	if err := WaitFor(f, timeout); err != nil {
		t.Fatal(err)
	}
}
