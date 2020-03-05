package util

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"testing"
	"time"
)

const (
	// retryPeriod is how often to poll a WaitFunc.
	retryPeriod = 10 * time.Millisecond
)

// WaitFunc is a callback that stops a wait when true.
type WaitFunc func() bool

// ServerRunning is a wait function that checks the server is accepting TCP traffic,
// and responding with a good status to the readiness check endpoint.
var ServerRunning = func() bool {
	request, err := http.NewRequest(http.MethodGet, "https://localhost:8443/readyz", nil)
	if err != nil {
		return false
	}

	request.Header.Set("X-Broker-API-Version", "2.13")
	request.Header.Set("Authorization", "Bearer "+Token)

	certPool := x509.NewCertPool()
	if ok := certPool.AppendCertsFromPEM([]byte(CA)); !ok {
		return false
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
		return false
	}

	if response.StatusCode != http.StatusOK {
		return false
	}

	return true
}

// WaitFor waits until a condition is true.
func WaitFor(f WaitFunc, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	tick := time.NewTicker(retryPeriod)
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

// WaitFor waits until a condition is true.
func MustWaitFor(t *testing.T, f WaitFunc, timeout time.Duration) {
	if err := WaitFor(f, timeout); err != nil {
		t.Fatal(err)
	}
}
