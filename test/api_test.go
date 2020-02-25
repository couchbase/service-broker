package test

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/couchbase/service-broker/test/util"
)

// TestReadiness tests a TLS readiness probe succeeds with no other headers.
func TestReadiness(t *testing.T) {
	request := util.MustBasicRequest(t, http.MethodGet, "/readyz")
	client := util.MustDefaultClient(t)
	response := util.MustDoRequest(t, client, request)
	util.MustVerifyStatusCode(t, response, http.StatusOK)
}

// TestReadinessUnconfigured tests removal of the service broker configuration
// results in the server becoming unavailable.
func TestReadinessUnconfigured(t *testing.T) {
	util.MustDeleteServiceBrokerConfig(t, clients)
	request := util.MustBasicRequest(t, http.MethodGet, "/readyz")
	client := util.MustDefaultClient(t)
	response := util.MustDoRequest(t, client, request)
	util.MustVerifyStatusCode(t, response, http.StatusServiceUnavailable)
	util.MustCreateServiceBrokerConfig(t, clients, util.DefaultBrokerConfig)
}

// TestConnectNoTLS tests that the client fails when connecting without using
// TLS transport.
func TestConnectNoTLS(t *testing.T) {
	request := util.MustDefaultRequest(t, http.MethodGet, "/v2/catalog")
	client := util.MustDefaultClient(t)
	client.Transport = nil
	util.MustNotDoRequest(t, client, request)
}

// TestConnectNoAPIVersion tests that the X-Broker-API-Version header is required
// by the broker.
// https://github.com/openservicebrokerapi/servicebroker/blob/master/spec.md#api-version-header
func TestConnectNoAPIVersion(t *testing.T) {
	request := util.MustDefaultRequest(t, http.MethodGet, "/v2/catalog")
	request.Header.Del("X-Broker-API-Version")
	client := util.MustDefaultClient(t)
	response := util.MustDoRequest(t, client, request)
	util.MustVerifyStatusCode(t, response, http.StatusBadRequest)
}

// TestConnectAPIVersionTooOld tests that X-Broker-API-Version headers too small
// are rejeted by the broker with a 400.  Currently >= 2.13 is the minimum supported.
// https://github.com/openservicebrokerapi/servicebroker/blob/master/spec.md#api-version-header
func TestConnectAPIVersionTooOld(t *testing.T) {
	request := util.MustDefaultRequest(t, http.MethodGet, "/v2/catalog")
	request.Header.Set("X-Broker-API-Version", "2.12")
	client := util.MustDefaultClient(t)
	response := util.MustDoRequest(t, client, request)
	util.MustVerifyStatusCode(t, response, http.StatusPreconditionFailed)
}

// TestConnectPathNotFound tests that illegal paths return a 404.
func TestConnectPathNotFound(t *testing.T) {
	request := util.MustDefaultRequest(t, http.MethodGet, "/batman")
	client := util.MustDefaultClient(t)
	response := util.MustDoRequest(t, client, request)
	util.MustVerifyStatusCode(t, response, http.StatusNotFound)
}

// TestConnectMethodNotFound tests that illegal paths return a 405.
func TestConnectMethodNotFound(t *testing.T) {
	request := util.MustDefaultRequest(t, http.MethodPost, "/v2/catalog")
	client := util.MustDefaultClient(t)
	response := util.MustDoRequest(t, client, request)
	util.MustVerifyStatusCode(t, response, http.StatusMethodNotAllowed)
}

// TestConnectNoAuthorization tests that the Authorization is required by the broker.
// https://github.com/openservicebrokerapi/servicebroker/blob/master/spec.md#platform-to-service-broker-authentication
func TestConnectNoAuthorization(t *testing.T) {
	request := util.MustDefaultRequest(t, http.MethodGet, "/v2/catalog")
	request.Header.Del("Authorization")
	client := util.MustDefaultClient(t)
	response := util.MustDoRequest(t, client, request)
	util.MustVerifyStatusCode(t, response, http.StatusUnauthorized)
}

// TestConnectAuthorizationPrecedence tests that Authorization takes precedence
// over everything else.
// https://github.com/openservicebrokerapi/servicebroker/blob/master/spec.md#platform-to-service-broker-authentication
func TestConnectAuthorizationPrecedence(t *testing.T) {
	request := util.MustDefaultRequest(t, http.MethodGet, "/batman")
	request.Header.Del("Authorization")
	request.Header.Del("X-Broker-API-Version")
	client := util.MustDefaultClient(t)
	response := util.MustDoRequest(t, client, request)
	util.MustVerifyStatusCode(t, response, http.StatusUnauthorized)
}

// TestConnect tests basic connection to the service broker.
func TestConnect(t *testing.T) {
	request := util.MustDefaultRequest(t, http.MethodGet, "/v2/catalog")
	client := util.MustDefaultClient(t)
	response := util.MustDoRequest(t, client, request)
	util.MustVerifyStatusCode(t, response, http.StatusOK)
}

// TestConnectWithBody tests that the server accepts a content of type application/json.
// Note we use a GET against /v2/catalog, the server will ignore payloads when not required
// however content type checking occurrs regardless.
// https://github.com/openservicebrokerapi/servicebroker/blob/master/spec.md#content-type
func TestConnectWithBody(t *testing.T) {
	request := util.MustDefaultRequestWithBody(t, http.MethodGet, "/v2/catalog", bytes.NewBufferString("{}"))
	client := util.MustDefaultClient(t)
	response := util.MustDoRequest(t, client, request)
	util.MustVerifyStatusCode(t, response, http.StatusOK)
}

// TestConnectInvalidContentType types that the server rejects content that isn't of type application/json.
// Note we use a GET against /v2/catalog, the server will ignore payloads when not required
// however content type checking occurrs regardless.
// https://github.com/openservicebrokerapi/servicebroker/blob/master/spec.md#content-type
func TestConnectInvalidContentType(t *testing.T) {
	request := util.MustDefaultRequestWithBody(t, http.MethodGet, "/v2/catalog", bytes.NewBufferString("{}"))
	request.Header.Set("Content-Type", "text/plain")
	client := util.MustDefaultClient(t)
	response := util.MustDoRequest(t, client, request)
	util.MustVerifyStatusCode(t, response, http.StatusBadRequest)
}
