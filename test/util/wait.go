package util

import (
	"context"
	"net"
	"fmt"
	"time"
)

// WaitFunc is a callback that stops a wait when true.
type WaitFunc func() bool

// ServerRunning is a wait function that checks the server is accepting TCP traffic.
var ServerRunning = func() bool {
	if _, err := net.Dial("tcp", "localhost:8443"); err != nil {
		return false
	}
	return true
}

// WaitFor waits until a condition is true.
func WaitFor(f WaitFunc, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	tick := time.NewTicker(time.Second)
	defer tick.Stop()

	for !f() {
		select {
		case <-tick.C:
		case <-ctx.Done():
			return fmt.Errorf("failed to wait for condition")
		}
	}

	return nil
}
