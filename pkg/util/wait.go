// Copyright 2020-2021 Couchbase, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file  except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the  License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package util

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

const (
	// retryPeriod is how often to poll a WaitFunc.
	retryPeriod = 10 * time.Millisecond
)

// ErrTimeout is raised when a wait doesn't terminate in time.
var ErrTimeout = errors.New("process timed out")

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
			return fmt.Errorf("%w: failed to wait for condition: %v", ErrTimeout, err)
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
