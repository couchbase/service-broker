package test

import (
	"net/http"
	"testing"

	"github.com/couchbase/service-broker/test/util"
)

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

// TestConnectNotFound tests that illegal paths return a 404.
func TestConnectNotFound(t *testing.T) {
	request := util.MustDefaultRequest(t, http.MethodGet, "/batman")
	client := util.MustDefaultClient(t)
	response := util.MustDoRequest(t, client, request)
	util.MustVerifyStatusCode(t, response, http.StatusNotFound)
}

// TestConnectNoBearerToken tests that the Authorization is required by the broker.
// https://github.com/openservicebrokerapi/servicebroker/blob/master/spec.md#platform-to-service-broker-authentication
func TestConnectNoBearerToken(t *testing.T) {
	request := util.MustDefaultRequest(t, http.MethodGet, "/v2/catalog")
	request.Header.Del("Authorization")
	client := util.MustDefaultClient(t)
	response := util.MustDoRequest(t, client, request)
	util.MustVerifyStatusCode(t, response, http.StatusUnauthorized)
}

// TestConnectNoBearerTokenPrecedence tests that Authorization takes precedence
// over broker API versioning.
// https://github.com/openservicebrokerapi/servicebroker/blob/master/spec.md#platform-to-service-broker-authentication
func TestConnectNoBearerTokenPrecedence(t *testing.T) {
	request := util.MustDefaultRequest(t, http.MethodGet, "/v2/catalog")
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
