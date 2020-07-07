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

package unit_test

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/couchbase/service-broker/test/unit/util"
)

// TestReadiness tests a TLS readiness probe succeeds with no other headers.
func TestReadiness(t *testing.T) {
	defer mustReset(t)

	request := util.MustBasicRequest(t, http.MethodGet, "/readyz")
	client := util.MustDefaultClient(t)

	response := util.MustDoRequest(t, client, request)
	defer response.Body.Close()

	util.MustVerifyStatusCode(t, response, http.StatusOK)
}

// TestReadinessUnconfigured tests removal of the service broker configuration
// results in the server becoming unavailable.
func TestReadinessUnconfigured(t *testing.T) {
	defer mustReset(t)

	util.MustDeleteServiceBrokerConfig(t, clients)

	request := util.MustBasicRequest(t, http.MethodGet, "/readyz")
	client := util.MustDefaultClient(t)

	response := util.MustDoRequest(t, client, request)
	defer response.Body.Close()

	util.MustVerifyStatusCode(t, response, http.StatusServiceUnavailable)
	util.MustCreateServiceBrokerConfig(t, clients, util.DefaultBrokerConfig)
}

// TestConnect tests basic connection to the service broker.
func TestConnect(t *testing.T) {
	defer mustReset(t)

	request := util.MustDefaultRequest(t, http.MethodGet, "/v2/catalog")
	client := util.MustDefaultClient(t)

	response := util.MustDoRequest(t, client, request)
	defer response.Body.Close()

	util.MustVerifyStatusCode(t, response, http.StatusOK)
}

// TestConnectNoTLS tests that the client fails when connecting without using
// TLS transport.
func TestConnectNoTLS(t *testing.T) {
	defer mustReset(t)

	request := util.MustDefaultRequest(t, http.MethodGet, "/v2/catalog")

	client := util.MustDefaultClient(t)
	client.Transport = nil

	util.MustNotDoRequest(t, client, request)
}

// TestConnectNoAPIVersion tests that the X-Broker-API-Version header is required
// by the broker.
// https://github.com/openservicebrokerapi/servicebroker/blob/master/spec.md#api-version-header
func TestConnectNoAPIVersion(t *testing.T) {
	defer mustReset(t)

	request := util.MustDefaultRequest(t, http.MethodGet, "/v2/catalog")
	request.Header.Del("X-Broker-API-Version")

	client := util.MustDefaultClient(t)

	response := util.MustDoRequest(t, client, request)
	defer response.Body.Close()

	util.MustVerifyStatusCode(t, response, http.StatusBadRequest)
}

// TestConnectMultipleAPIVersion tests we reject requests with multiple X-Broker-API-Version
// headers due to ambiguity.
func TestConnectMultipleAPIVersion(t *testing.T) {
	defer mustReset(t)

	request := util.MustDefaultRequest(t, http.MethodGet, "/v2/catalog")
	request.Header.Add("X-Broker-API-Version", "2.13")

	client := util.MustDefaultClient(t)

	response := util.MustDoRequest(t, client, request)
	defer response.Body.Close()

	util.MustVerifyStatusCode(t, response, http.StatusBadRequest)
}

// TestConnectInvalidAPIVersion tests we reject requests with an invalid X-Broker-API-Version
// header.
func TestConnectInvalidAPIVersion(t *testing.T) {
	defer mustReset(t)

	request := util.MustDefaultRequest(t, http.MethodGet, "/v2/catalog")
	request.Header.Set("X-Broker-API-Version", "dave")

	client := util.MustDefaultClient(t)

	response := util.MustDoRequest(t, client, request)
	defer response.Body.Close()

	util.MustVerifyStatusCode(t, response, http.StatusBadRequest)
}

// TestConnectAPIVersionTooOld tests that X-Broker-API-Version headers too small
// are rejeted by the broker with a 400.  Currently >= 2.13 is the minimum supported.
// https://github.com/openservicebrokerapi/servicebroker/blob/master/spec.md#api-version-header
func TestConnectAPIVersionTooOld(t *testing.T) {
	defer mustReset(t)

	request := util.MustDefaultRequest(t, http.MethodGet, "/v2/catalog")
	request.Header.Set("X-Broker-API-Version", "2.12")

	client := util.MustDefaultClient(t)

	response := util.MustDoRequest(t, client, request)
	defer response.Body.Close()

	util.MustVerifyStatusCode(t, response, http.StatusPreconditionFailed)
}

// TestConnectPathNotFound tests that illegal paths return a 404.
func TestConnectPathNotFound(t *testing.T) {
	defer mustReset(t)

	request := util.MustDefaultRequest(t, http.MethodGet, "/batman")
	client := util.MustDefaultClient(t)

	response := util.MustDoRequest(t, client, request)
	defer response.Body.Close()

	util.MustVerifyStatusCode(t, response, http.StatusNotFound)
}

// TestConnectMethodNotFound tests that illegal paths return a 405.
func TestConnectMethodNotFound(t *testing.T) {
	defer mustReset(t)

	request := util.MustDefaultRequest(t, http.MethodPost, "/v2/catalog")
	client := util.MustDefaultClient(t)

	response := util.MustDoRequest(t, client, request)
	defer response.Body.Close()

	util.MustVerifyStatusCode(t, response, http.StatusMethodNotAllowed)
}

// TestConnectNoAuthorization tests that the Authorization header is required by the broker.
// https://github.com/openservicebrokerapi/servicebroker/blob/master/spec.md#platform-to-service-broker-authentication
func TestConnectNoAuthorization(t *testing.T) {
	defer mustReset(t)

	request := util.MustDefaultRequest(t, http.MethodGet, "/v2/catalog")
	request.Header.Del("Authorization")

	client := util.MustDefaultClient(t)

	response := util.MustDoRequest(t, client, request)
	defer response.Body.Close()

	util.MustVerifyStatusCode(t, response, http.StatusUnauthorized)
}

// TestConnectMultipleAuthorization tests we reject requests with multiple Authorization
// headers due to ambiguity.
func TestConnectMultipleAuthorization(t *testing.T) {
	defer mustReset(t)

	request := util.MustDefaultRequest(t, http.MethodGet, "/v2/catalog")
	request.Header.Add("Authorization", "She-ra")

	client := util.MustDefaultClient(t)

	response := util.MustDoRequest(t, client, request)
	defer response.Body.Close()

	util.MustVerifyStatusCode(t, response, http.StatusUnauthorized)
}

// TestConnectInvalidAuthorization tests we reject requests with an invalid Authorization header.
func TestConnectInvalidAuthorization(t *testing.T) {
	defer mustReset(t)

	request := util.MustDefaultRequest(t, http.MethodGet, "/v2/catalog")
	request.Header.Set("Authorization", "Bearer She-ra")

	client := util.MustDefaultClient(t)

	response := util.MustDoRequest(t, client, request)
	defer response.Body.Close()

	util.MustVerifyStatusCode(t, response, http.StatusUnauthorized)
}

// TestConnectAuthorizationPrecedence tests that authorization takes precedence
// over everything else.
func TestConnectAuthorizationPrecedence(t *testing.T) {
	defer mustReset(t)

	request := util.MustDefaultRequest(t, http.MethodGet, "/batman")
	request.Header.Del("Authorization")
	request.Header.Del("X-Broker-API-Version")

	client := util.MustDefaultClient(t)

	response := util.MustDoRequest(t, client, request)
	defer response.Body.Close()

	util.MustVerifyStatusCode(t, response, http.StatusUnauthorized)
}

// TestConnectWithBody tests that the server accepts a content of type application/json.
// NOTE: we use a GET against /v2/catalog, the server will ignore payloads when not required
// however content type checking occurrs regardless.
// https://github.com/openservicebrokerapi/servicebroker/blob/master/spec.md#content-type
func TestConnectWithBody(t *testing.T) {
	defer mustReset(t)

	request := util.MustDefaultRequestWithBody(t, http.MethodGet, "/v2/catalog", bytes.NewBufferString("{}"))
	client := util.MustDefaultClient(t)

	response := util.MustDoRequest(t, client, request)
	defer response.Body.Close()

	util.MustVerifyStatusCode(t, response, http.StatusOK)
}

// TestConnectNoContentType that the server rejects content that doesn't have a content type.
// NOTE: we use a GET against /v2/catalog, the server will ignore payloads when not required
// however content type checking occurrs regardless.
func TestConnectNoContentType(t *testing.T) {
	defer mustReset(t)

	request := util.MustDefaultRequestWithBody(t, http.MethodGet, "/v2/catalog", bytes.NewBufferString("{}"))
	request.Header.Del("Content-Type")

	client := util.MustDefaultClient(t)

	response := util.MustDoRequest(t, client, request)
	defer response.Body.Close()

	util.MustVerifyStatusCode(t, response, http.StatusBadRequest)
}

// TestConnectMultipleContentType that the server rejects content that has multiple a content types.
// NOTE: we use a GET against /v2/catalog, the server will ignore payloads when not required
// however content type checking occurrs regardless.
func TestConnectMultipleContentType(t *testing.T) {
	defer mustReset(t)

	request := util.MustDefaultRequestWithBody(t, http.MethodGet, "/v2/catalog", bytes.NewBufferString("{}"))
	request.Header.Add("Content-Type", "text/plain")

	client := util.MustDefaultClient(t)

	response := util.MustDoRequest(t, client, request)
	defer response.Body.Close()

	util.MustVerifyStatusCode(t, response, http.StatusBadRequest)
}

// TestConnectInvalidContentType tests that the server rejects content that isn't of type application/json.
// NOTE: we use a GET against /v2/catalog, the server will ignore payloads when not required
// however content type checking occurrs regardless.
// https://github.com/openservicebrokerapi/servicebroker/blob/master/spec.md#content-type
func TestConnectInvalidContentType(t *testing.T) {
	defer mustReset(t)

	request := util.MustDefaultRequestWithBody(t, http.MethodGet, "/v2/catalog", bytes.NewBufferString("{}"))
	request.Header.Set("Content-Type", "text/plain")

	client := util.MustDefaultClient(t)

	response := util.MustDoRequest(t, client, request)
	defer response.Body.Close()

	util.MustVerifyStatusCode(t, response, http.StatusBadRequest)
}
