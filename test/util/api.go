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
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/couchbase/service-broker/pkg/api"
)

const (
	// pollTimeout is how long to poll for provisioning completion
	// before giving up.
	pollTimeout = 30 * time.Second
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
func MustNotDoRequest(t *testing.T, client *http.Client, request *http.Request) {
	response, err := DoRequest(client, request)
	if err == nil {
		defer response.Body.Close()
		t.Fatal(err)
	}
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

// basicOperation does a generic HTTP call with the given method and path.
// Request and response parameters are serialized to/from JSON.  The response
// status is checked and some basic sanity testing done on the payload.
// The request and response parameters are optional and may be nil.
func basicOperation(method, path string, statusCode int, req interface{}, resp interface{}) error {
	var buffer io.Reader

	if req != nil {
		raw, err := json.Marshal(req)
		if err != nil {
			return err
		}

		buffer = bytes.NewBuffer(raw)
	}

	request, err := DefaultRequestWithBody(method, path, buffer)
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

	if resp != nil {
		if err := MatchHeader(response, "Content-Type", "application/json"); err != nil {
			return err
		}

		raw, err := ioutil.ReadAll(response.Body)
		if err != nil {
			return err
		}

		if err := json.Unmarshal(raw, resp); err != nil {
			return err
		}
	}

	return nil
}

// basicOperationAndError does a generic HTTP call with the given method and path.
// Request parameters are serialized to JSON.  The response is implicitly expected
// to be a service broker error.  The response status and error code are tested for
// expected correctness.
func basicOperationAndError(method, path string, statusCode int, req interface{}, apiError api.ErrorType) error {
	e := &api.Error{}
	if err := basicOperation(method, path, statusCode, req, e); err != nil {
		return err
	}

	if e.Error != apiError {
		return fmt.Errorf("expected error %s does not match %s", apiError, e.Error)
	}

	return nil
}

// Get does a GET API call and expects a certain response.
func Get(path string, statusCode int, response interface{}) error {
	if err := basicOperation(http.MethodGet, path, statusCode, nil, response); err != nil {
		return err
	}

	return nil
}

// MustGet does a GET API call and expects a certain response.
func MustGet(t *testing.T, path string, statusCode int, response interface{}) {
	if err := Get(path, statusCode, response); err != nil {
		t.Fatal(err)
	}
}

// GetAndError does a GET API call and expects a certain response with a valid JSON error.
func GetAndError(path string, statusCode int, apiError api.ErrorType) error {
	if err := basicOperationAndError(http.MethodGet, path, statusCode, nil, apiError); err != nil {
		return err
	}

	return nil
}

// MustGetAndError does a GET API call and expects a certain response with a valid JSON error.
func MustGetAndError(t *testing.T, path string, statusCode int, apiError api.ErrorType) {
	if err := GetAndError(path, statusCode, apiError); err != nil {
		t.Fatal(err)
	}
}

// Put does a PUT API call and expects a certain response.
func Put(path string, statusCode int, request, response interface{}) error {
	if err := basicOperation(http.MethodPut, path, statusCode, request, response); err != nil {
		return err
	}

	return nil
}

// MustPut does a PUT API call and expects a certain response.
func MustPut(t *testing.T, path string, statusCode int, request, response interface{}) {
	if err := Put(path, statusCode, request, response); err != nil {
		t.Fatal(err)
	}
}

// PutAndError does a PUT API call and expects a certain response with a valid JSON error.
func PutAndError(path string, statusCode int, request interface{}, apiError api.ErrorType) error {
	if err := basicOperationAndError(http.MethodPut, path, statusCode, request, apiError); err != nil {
		return err
	}

	return nil
}

// MustPutAndError does a PUT API call and expects a certain response with a valid JSON error.
func MustPutAndError(t *testing.T, path string, statusCode int, request interface{}, apiError api.ErrorType) {
	if err := PutAndError(path, statusCode, request, apiError); err != nil {
		t.Fatal(err)
	}
}

// Delete does a DELETE API call and expects a certain response.
func Delete(path string, statusCode int, response interface{}) error {
	if err := basicOperation(http.MethodDelete, path, statusCode, nil, response); err != nil {
		return err
	}

	return nil
}

// MustDelete does a DELETE API call and expects a certain response.
func MustDelete(t *testing.T, path string, statusCode int, response interface{}) {
	if err := Delete(path, statusCode, response); err != nil {
		t.Fatal(err)
	}
}

// DeleteAndError does a DELETE API call and expects a certain response with a valid JSON error.
func DeleteAndError(path string, statusCode int, apiError api.ErrorType) error {
	if err := basicOperationAndError(http.MethodDelete, path, statusCode, nil, apiError); err != nil {
		return err
	}

	return nil
}

// MustDeleteAndError does a DELETE API call and expects a certain response with a valid JSON error.
func MustDeleteAndError(t *testing.T, path string, statusCode int, apiError api.ErrorType) {
	if err := DeleteAndError(path, statusCode, apiError); err != nil {
		t.Fatal(err)
	}
}

// Patch does a PATCH API call and expects a certain response.
func Patch(path string, statusCode int, request, response interface{}) error {
	if err := basicOperation(http.MethodPatch, path, statusCode, request, response); err != nil {
		return err
	}

	return nil
}

// MustPatch does a PATCH API call and expects a certain response.
func MustPatch(t *testing.T, path string, statusCode int, request, response interface{}) {
	if err := Patch(path, statusCode, request, response); err != nil {
		t.Fatal(err)
	}
}

// PatchAndError does a PATCH API call and expects a certain response with a valid JSON error.
func PatchAndError(path string, statusCode int, request interface{}, apiError api.ErrorType) error {
	if err := basicOperationAndError(http.MethodPatch, path, statusCode, request, apiError); err != nil {
		return err
	}

	return nil
}

// MustPatchAndError does a PATCH API call and expects a certain response with a valid JSON error.
func MustPatchAndError(t *testing.T, path string, statusCode int, request interface{}, apiError api.ErrorType) {
	if err := PatchAndError(path, statusCode, request, apiError); err != nil {
		t.Fatal(err)
	}
}

// PollServiceInstanceQuery creates a query string for use with the service instance polling
// API.  It is generated from the original service instance creation request and the response
// containing the operation ID.
func PollServiceInstanceQuery(req *api.CreateServiceInstanceRequest, rsp *api.CreateServiceInstanceResponse) url.Values {
	values := url.Values{}

	values.Add("service_id", req.ServiceID)
	values.Add("plan_id", req.PlanID)
	values.Add("operation", rsp.Operation)

	return values
}

// DeleteServiceInstanceQuery creates a query string for use with the service instance polling
// API.  It is generated from the original service instance creation request.
func DeleteServiceInstanceQuery(req *api.CreateServiceInstanceRequest) url.Values {
	values := url.Values{}

	values.Add("accepts_incomplete", "true")
	values.Add("service_id", req.ServiceID)
	values.Add("plan_id", req.PlanID)

	return values
}

// ReadServiceInstanceQuery creates a query string for use with the service instance get
// API.  It is generated from the original service instance creation request.
func ReadServiceInstanceQuery(req *api.CreateServiceInstanceRequest) url.Values {
	values := url.Values{}

	values.Add("service_id", req.ServiceID)
	values.Add("plan_id", req.PlanID)

	return values
}

// MustCreateServiceInstance wraps up service instance creation.
func MustCreateServiceInstance(t *testing.T, name string, req *api.CreateServiceInstanceRequest) *api.CreateServiceInstanceResponse {
	rsp := &api.CreateServiceInstanceResponse{}
	MustPut(t, "/v2/service_instances/"+name+"?accepts_incomplete=true", http.StatusAccepted, req, rsp)

	// All create operations are asynchronous and must have an operation string.
	Assert(t, rsp.Operation != "")

	return rsp
}

// MustPollServiceInstanceForCompletion wraps up service instance poll.
func MustPollServiceInstanceForCompletion(t *testing.T, name string, rsp *api.CreateServiceInstanceResponse) {
	callback := func() bool {
		// Polling will usually always return OK with the status embedded in the response.
		poll := &api.PollServiceInstanceResponse{}
		MustGet(t, "/v2/service_instances/"+name+"/last_operation?operation="+rsp.Operation, http.StatusOK, poll)

		// A failed is always an error.
		Assert(t, poll.State != api.PollStateFailed)

		// Polling completes when the the state is success.
		return poll.State == api.PollStateSucceeded
	}
	MustWaitFor(t, callback, pollTimeout)
}

// MustPollServiceInstanceForDeletion wraps up polling for an aysnc deletion.
func MustPollServiceInstanceForDeletion(t *testing.T, name string, rsp *api.CreateServiceInstanceResponse) {
	callback := func() bool {
		// When polling for deletion, it will start as OK (as per MustPollServiceInstanceForCompletion)
		// however will finally respond with Gone.
		apiError := &api.Error{}
		if err := Get("/v2/service_instances/"+name+"/last_operation?operation="+rsp.Operation, http.StatusGone, apiError); err != nil {
			return false
		}

		// Assert that the correct error message is given.
		Assert(t, apiError.Error == api.ErrorResourceGone)

		return true
	}
	MustWaitFor(t, callback, pollTimeout)
}

// MustCreateServiceInstanceSuccessfully wraps up service instance creation and polling.
func MustCreateServiceInstanceSuccessfully(t *testing.T, name string, req *api.CreateServiceInstanceRequest) {
	rsp := MustCreateServiceInstance(t, name, req)
	MustPollServiceInstanceForCompletion(t, name, rsp)
}

// MustUpdateServiceInstance wraps up service instance creation.
func MustUpdateServiceInstance(t *testing.T, name string, req *api.UpdateServiceInstanceRequest) *api.CreateServiceInstanceResponse {
	rsp := &api.CreateServiceInstanceResponse{}
	MustPatch(t, "/v2/service_instances/"+name+"?accepts_incomplete=true", http.StatusAccepted, req, rsp)

	// All create operations are asynchronous and must have an operation string.
	Assert(t, rsp.Operation != "")

	return rsp
}

// MustUpdateServiceInstanceSuccessfully wraps up service instance update and polling.
func MustUpdateServiceInstanceSuccessfully(t *testing.T, name string, req *api.UpdateServiceInstanceRequest) {
	rsp := MustUpdateServiceInstance(t, name, req)
	MustPollServiceInstanceForCompletion(t, name, rsp)
}
