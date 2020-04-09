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
