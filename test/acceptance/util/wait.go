// Copyright 2020 Couchbase, Inc.
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
