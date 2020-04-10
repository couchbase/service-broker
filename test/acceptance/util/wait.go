package util

import (
	"testing"
	"time"

	"github.com/couchbase/service-broker/pkg/util"
)

// WaitFor waits until a condition is nil.
func WaitFor(f util.WaitFunc, timeout time.Duration) error {
	return util.WaitFor(f, timeout)
}

// MustWaitFor waits until a condition is nil.
func MustWaitFor(t *testing.T, f util.WaitFunc, timeout time.Duration) {
	util.MustWaitFor(t, f, timeout)
}
