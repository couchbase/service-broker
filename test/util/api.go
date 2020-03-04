package util

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/couchbase/service-broker/pkg/api"
)

// MustBasicRequest creates a HTTP request object for the requested method
// on a path.
// It applies no additional headers.
func MustBasicRequest(t *testing.T, method, path string) *http.Request {
	request, err := http.NewRequest(method, "https://localhost:8443"+path, nil)
	if err != nil {
		t.Fatal(err)
	}
	return request
}

// DefaultRequest creates a HTTP request object for the requested method
// on a path.
// It applies known good configuration to provide connectivity with the broker
// for the common case.
func DefaultRequest(method, path string) (*http.Request, error) {
	request, err := http.NewRequest(method, "https://localhost:8443"+path, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("X-Broker-API-Version", "2.13")
	request.Header.Set("Authorization", "Bearer "+Token)
	return request, nil
}

// MustDefaultRequest creates a HTTP request object for the requested method
// on a path.
// It applies known good configuration to provide connectivity with the broker
// for the common case.
func MustDefaultRequest(t *testing.T, method, path string) *http.Request {
	request, err := DefaultRequest(method, path)
	if err != nil {
		t.Fatal(err)
	}
	return request
}

// DefaultRequestWithBody creates a HTTP request object for the requested method
// on a path.
// It applies known good configuration to provide connectivity with the broker
// for the common case.
func DefaultRequestWithBody(method, path string, body io.Reader) (*http.Request, error) {
	request, err := http.NewRequest(method, "https://localhost:8443"+path, body)
	if err != nil {
		return nil, err
	}
	request.Header.Set("X-Broker-API-Version", "2.13")
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+Token)
	return request, nil
}

// MustDefaultRequestWithBody creates a HTTP request object for the requested method
// on a path.
// It applies known good configuration to provide connectivity with the broker
// for the common case.
func MustDefaultRequestWithBody(t *testing.T, method, path string, body io.Reader) *http.Request {
	request, err := DefaultRequestWithBody(method, path, body)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("X-Broker-API-Version", "2.13")
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+Token)
	return request
}

// DefaultClient creates a HTTP client for use against the service broker.
// It applies known good configuration to provide connectivity with the broker
// for the common case.
func DefaultClient() (*http.Client, error) {
	certPool := x509.NewCertPool()
	if ok := certPool.AppendCertsFromPEM([]byte(CA)); !ok {
		return nil, fmt.Errorf("failed to import CA certificate")
	}

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: certPool,
			},
		},
	}

	return client, nil
}

// MustDefaultClient creates a HTTP client for use against the service broker.
// It applies known good configuration to provide connectivity with the broker
// for the common case.
func MustDefaultClient(t *testing.T) *http.Client {
	client, err := DefaultClient()
	if err != nil {
		t.Fatal(err)
	}
	return client
}

// DoRequest performs a requests against the broker API with the provided client.
// This call will cause test failure if the network transport fails.
func DoRequest(client *http.Client, request *http.Request) (*http.Response, error) {
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	return response, nil
}

// MustDoRequest performs a requests against the broker API with the provided client.
// This call will cause test failure if the network transport fails.
func MustDoRequest(t *testing.T, client *http.Client, request *http.Request) *http.Response {
	response, err := DoRequest(client, request)
	if err != nil {
		t.Fatal(err)
	}
	return response
}

// MustNotDoRequest performs a requests against the broker API with the provided client.
// This call will cause test failure if the network transport succeeds.
func MustNotDoRequest(t *testing.T, client *http.Client, request *http.Request) *http.Response {
	response, err := DoRequest(client, request)
	if err == nil {
		t.Fatal(err)
	}
	return response
}

// VerifyStatusCode verifies the HTTP status code is as expected.
// This call will cause test failure if the HTTP status code does not match.
func VerifyStatusCode(response *http.Response, statusCode int) error {
	if response.StatusCode != statusCode {
		return fmt.Errorf("unexpected status code %d, expected %d", response.StatusCode, statusCode)
	}
	return nil
}

// MustVerifyStatusCode verifies the HTTP status code is as expected.
// This call will cause test failure if the HTTP status code does not match.
func MustVerifyStatusCode(t *testing.T, response *http.Response, statusCode int) {
	if err := VerifyStatusCode(response, statusCode); err != nil {
		t.Fatal(fmt.Errorf("unexpected status code %d, expected %d", response.StatusCode, statusCode))
	}
}

// MatchHeader checks if the header exists with the specified value.
func MatchHeader(response *http.Response, name, value string) error {
	for headerName := range response.Header {
		if strings.EqualFold(headerName, name) {
			for _, headerValue := range response.Header[headerName] {
				if strings.EqualFold(headerValue, value) {
					return nil
				}
			}
			return fmt.Errorf("expected header %s does not contain value %s", name, value)
		}
	}
	return fmt.Errorf("expected header %s does not exist", name)
}

// Get does a GET API call and expects a certain response and to be able to
// unmarshal the data into the provided structure.  All communication from the
// broker should be in JSON, so encode this check.
func Get(path string, statusCode int, body interface{}) error {
	request, err := DefaultRequest(http.MethodGet, path)
	if err != nil {
		return err
	}
	client, err := DefaultClient()
	if err != nil {
		return err
	}
	response, err := DoRequest(client, request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if err := VerifyStatusCode(response, statusCode); err != nil {
		return err
	}
	if err := MatchHeader(response, "Content-Type", "application/json"); err != nil {
		return err
	}
	raw, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(raw, body); err != nil {
		return err
	}
	return nil
}

// MustGet does a GET API call and expects a certain response and to be able to
// unmarshal the data into the provided structure.
func MustGet(t *testing.T, path string, statusCode int, body interface{}) {
	if err := Get(path, statusCode, body); err != nil {
		t.Fatal(err)
	}
}

// GetWithError does a GET API call and expects a certain response and JSON
// formatted error with a specific error code.
func GetWithError(path string, statusCode int, apiError api.APIError) error {
	request, err := DefaultRequest(http.MethodGet, path)
	if err != nil {
		return err
	}
	client, err := DefaultClient()
	if err != nil {
		return err
	}
	response, err := DoRequest(client, request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if err := VerifyStatusCode(response, statusCode); err != nil {
		return err
	}
	if err := MatchHeader(response, "Content-Type", "application/json"); err != nil {
		return err
	}
	raw, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}
	e := &api.Error{}
	if err := json.Unmarshal(raw, e); err != nil {
		return err
	}
	if e.Error != apiError {
		return fmt.Errorf("expected error %s does not match %s", apiError, e.Error)
	}
	return nil
}

// MustGetWithError does a GET API call and expects a certain response and JSON
// formatted error with a specific error code.
func MustGetWithError(t *testing.T, path string, statusCode int, apiError api.APIError) {
	if err := GetWithError(path, statusCode, apiError); err != nil {
		t.Fatal(err)
	}
}

// Put does a PUT API call and expects a certain response.
func Put(path string, body interface{}, statusCode int) error {
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	buffer := bytes.NewBuffer(raw)
	request, err := DefaultRequestWithBody(http.MethodPut, path, buffer)
	if err != nil {
		return err
	}
	client, err := DefaultClient()
	if err != nil {
		return err
	}
	response, err := DoRequest(client, request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if err := VerifyStatusCode(response, statusCode); err != nil {
		return err
	}
	return nil
}

// MustPut does a PUT API call and expects a certain response.
func MustPut(t *testing.T, path string, body interface{}, statusCode int) {
	if err := Put(path, body, statusCode); err != nil {
		t.Fatal(err)
	}
}

// PutWithResponse does a PUT API call and expects a certain response.  It also
// expects a JSON formatted response object.
func PutWithResponse(path string, body interface{}, statusCode int, rsp interface{}) error {
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	buffer := bytes.NewBuffer(raw)
	request, err := DefaultRequestWithBody(http.MethodPut, path, buffer)
	if err != nil {
		return err
	}
	client, err := DefaultClient()
	if err != nil {
		return err
	}
	response, err := DoRequest(client, request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if err := VerifyStatusCode(response, statusCode); err != nil {
		return err
	}
	if err := MatchHeader(response, "Content-Type", "application/json"); err != nil {
		return err
	}
	raw, err = ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(raw, rsp); err != nil {
		return err
	}
	return nil
}

// MustPutWithResponse does a PUT API call and expects a certain response.  It also
// expects a JSON formatted response object.
func MustPutWithResponse(t *testing.T, path string, body interface{}, statusCode int, rsp interface{}) {
	if err := PutWithResponse(path, body, statusCode, rsp); err != nil {
		t.Fatal(err)
	}
}

// PutWithError does a PUT API call and expects a certain response.
func PutWithError(path string, body interface{}, statusCode int, apiError api.APIError) error {
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	buffer := bytes.NewBuffer(raw)
	request, err := DefaultRequestWithBody(http.MethodPut, path, buffer)
	if err != nil {
		return err
	}
	client, err := DefaultClient()
	if err != nil {
		return err
	}
	response, err := DoRequest(client, request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if err := VerifyStatusCode(response, statusCode); err != nil {
		return err
	}
	if err := MatchHeader(response, "Content-Type", "application/json"); err != nil {
		return err
	}
	if raw, err = ioutil.ReadAll(response.Body); err != nil {
		return err
	}
	e := &api.Error{}
	if err := json.Unmarshal(raw, e); err != nil {
		return err
	}
	if e.Error != apiError {
		return fmt.Errorf("expected error %s does not match %s", apiError, e.Error)
	}
	return nil
}

// MustPutWithError does a PUT API call and expects a certain response.
func MustPutWithError(t *testing.T, path string, body interface{}, statusCode int, apiError api.APIError) {
	if err := PutWithError(path, body, statusCode, apiError); err != nil {
		t.Fatal(err)
	}
}

// PollServiceInstanceQuery creates a query string for use with the service instance polling
// API.  It is generated from the original service instance creation request and the response
// containing the operation ID.
func PollServiceInstanceQuery(req *api.CreateServiceInstanceRequest, rsp *api.CreateServiceInstanceResponse) string {
	return fmt.Sprintf("service_id=%s&plan_id=%s&operation=%s", req.ServiceID, req.PlanID, rsp.Operation)
}
