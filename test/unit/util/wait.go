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
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/couchbase/service-broker/pkg/util"
)

// ServerRunning is a wait function that checks the server is accepting TCP traffic,
// and responding with a good status to the readiness check endpoint.
func ServerRunning() error {
	request, err := http.NewRequest(http.MethodGet, "https://localhost:8443/readyz", nil)
	if err != nil {
		return err
	}

	request.Header.Set("X-Broker-API-Version", "2.13")
	request.Header.Set("Authorization", "Bearer "+Token)

	certPool := x509.NewCertPool()
	if ok := certPool.AppendCertsFromPEM([]byte(CA)); !ok {
		return fmt.Errorf("unable to append CA certificate")
	}

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: certPool,
			},
		},
	}

	response, err := client.Do(request)
	if err != nil {
		return err
	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code %v", response.StatusCode)
	}

	return nil
}

// WaitFor waits until a condition is nil.
func WaitFor(f util.WaitFunc, timeout time.Duration) error {
	return util.WaitFor(f, timeout)
}

// MustWaitFor waits until a condition is nil.
func MustWaitFor(t *testing.T, f util.WaitFunc, timeout time.Duration) {
	util.MustWaitFor(t, f, timeout)
}
