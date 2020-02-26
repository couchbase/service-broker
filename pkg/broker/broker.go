// Package broker implements the Open Broker API for the Couchbase Operator.
package broker

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/couchbase/service-broker/pkg/apis"
	"github.com/couchbase/service-broker/pkg/client"
	"github.com/couchbase/service-broker/pkg/config"
	"github.com/couchbase/service-broker/pkg/registry"
	"github.com/couchbase/service-broker/pkg/util"

	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"

	"k8s.io/client-go/kubernetes/scheme"
)

var (
	instanceRegistry *registry.Registry
)

// handleReadiness returns 503 until the configuration is correct.
func handleReadiness(w http.ResponseWriter, r *http.Request) error {
	if !config.Ready() {
		util.HTTPResponse(w, http.StatusServiceUnavailable)
		return fmt.Errorf("service not ready")
	}
	return nil
}

// handleBrokerBearerToken implements RFC-6750.
func handleBrokerBearerToken(w http.ResponseWriter, r *http.Request) error {
	for name := range r.Header {
		if strings.EqualFold(name, "Authorization") {
			if len(r.Header[name]) != 1 {
				util.HTTPResponse(w, http.StatusBadRequest)
				return fmt.Errorf("multiple Authorization headers given")
			}
			if r.Header[name][0] != "Bearer "+string(config.Token()) {
				util.HTTPResponse(w, http.StatusUnauthorized)
				return fmt.Errorf("authorization failed")
			}
			return nil
		}
	}
	util.HTTPResponse(w, http.StatusUnauthorized)
	return fmt.Errorf("no Authorization header")
}

// handleBrokerAPIHeader looks for a verifies the X-Broker-API-Version header
// is supported.
func handleBrokerAPIHeader(w http.ResponseWriter, r *http.Request) error {
	for name := range r.Header {
		if strings.EqualFold(name, "X-Broker-API-Version") {
			if len(r.Header[name]) != 1 {
				util.HTTPResponse(w, http.StatusBadRequest)
				return fmt.Errorf("multiple X-Broker-Api-Version headers given")
			}
			apiVersion, err := strconv.ParseFloat(r.Header[name][0], 64)
			if err != nil {
				util.HTTPResponse(w, http.StatusBadRequest)
				return fmt.Errorf("malformed X-Broker-Api-Version header: %v", err)
			}
			if apiVersion < minBrokerAPIVersion {
				util.HTTPResponse(w, http.StatusPreconditionFailed)
				return fmt.Errorf("unsupported X-Broker-Api-Version header %v, requires at least %.2f", r.Header[name][0], minBrokerAPIVersion)
			}
			return nil
		}
	}
	util.HTTPResponse(w, http.StatusBadRequest)
	return fmt.Errorf("no X-Broker-Api-Version header")
}

// handleContentTypeHeader looks for a verifies the Content-Type header is supported.
// If not specified we just return the standard JSON anyway.
func handleContentTypeHeader(w http.ResponseWriter, r *http.Request) error {
	// If no content is specified we don't need a type.
	if r.ContentLength == 0 {
		return nil
	}
	for name := range r.Header {
		if strings.EqualFold(name, "Content-Type") {
			for _, contentType := range r.Header[name] {
				if strings.EqualFold(contentType, "application/json") {
					return nil
				}
			}
			util.HTTPResponse(w, http.StatusBadRequest)
			return fmt.Errorf("invalid Content-Type header")
		}
	}
	util.HTTPResponse(w, http.StatusBadRequest)
	return fmt.Errorf("no Content-Type header")
}

// handleRequestHeaders checks that required headers are sent and are
// valid, and that content encodings are correct.
func handleRequestHeaders(w http.ResponseWriter, r *http.Request) error {
	if err := handleBrokerBearerToken(w, r); err != nil {
		return err
	}
	if err := handleBrokerAPIHeader(w, r); err != nil {
		return err
	}
	if err := handleContentTypeHeader(w, r); err != nil {
		return err
	}
	return nil
}

// OpenServiceBrokerHandler wraps up a standard router but performs Open Service Broker
// specific checks before performing the routing, such as making sure the correct API
// version is being used and the cnntent type is correct.
type openServiceBrokerHandler struct {
	http.Handler
}

// NewOpenServiceBrokerHandler initializes the main router with the Open Service Broker API.
func NewOpenServiceBrokerHandler() http.Handler {
	router := httprouter.New()
	router.GET("/readyz", handleReadyz)
	router.GET("/v2/catalog", handleReadCatalog)
	router.PUT("/v2/service_instances/:instance_id", handleCreateServiceInstance)
	router.GET("/v2/service_instances/:instance_id", handleReadServiceInstance)
	router.PATCH("/v2/service_instances/:instance_id", handleUpdateServiceInstance)
	router.DELETE("/v2/service_instances/:instance_id", handleDeleteServiceInstance)
	router.GET("/v2/service_instances/:instance_id/last_operation", handleReadServiceInstanceStatus)
	router.PUT("/v2/service_instances/:instance_id/service_bindings/:binding_id", handleCreateServiceBinding)
	router.GET("/v2/service_instances/:instance_id/service_bindings/:binding_id", handleReadServiceBinding)
	router.PATCH("/v2/service_instances/:instance_id/service_bindings/:binding_id", handleUpdateServiceBinding)
	router.DELETE("/v2/service_instances/:instance_id/service_bindings/:binding_id", handleDeleteServiceBinding)
	router.GET("/v2/service_instances/:instance_id/service_bindings/:binding_id/last_operation", handleReadServiceBindingStatus)
	return &openServiceBrokerHandler{Handler: router}
}

// responseWriter wraps the standard response writer so we can extract the response data.
type responseWriter struct {
	writer http.ResponseWriter
	status int
}

// Header returns a reference to the response headers.
func (w *responseWriter) Header() http.Header {
	return w.writer.Header()
}

// Write writes out data after the headers have been written.
func (w *responseWriter) Write(body []byte) (int, error) {
	return w.writer.Write(body)
}

// WriteHeader writes out the headers.
func (w *responseWriter) WriteHeader(statusCode int) {
	w.status = statusCode
	w.writer.WriteHeader(statusCode)
}

// ServeHTTP performs generic test on all API endpoints.
func (handler *openServiceBrokerHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Start the profiling timer.
	start := time.Now()

	// Print out request logging information.
	// DO NOT print out headers at info level as that will leak credentials into the log stream.
	userAgent := "-"
	for name := range r.Header {
		if strings.EqualFold(name, "User-Agent") {
			userAgent = r.Header[name][0]
			break
		}
	}
	glog.Infof(`HTTP req: "%s %s %s" %s %s`, r.Method, r.URL.Path, r.Proto, r.RemoteAddr, userAgent)

	// Start using the wrapped writer so we can capture the status code etc.
	writer := &responseWriter{
		writer: w,
	}

	// Indicate that the service is not ready until configured.
	if err := handleReadiness(writer, r); err != nil {
		glog.V(1).Info(err)
		goto ServeHTTPTail
	}

	// Ignore security checks for the readiness handler
	if r.URL.Path != "/readyz" {
		// Process headers, API versions, content types.
		if err := handleRequestHeaders(writer, r); err != nil {
			glog.V(1).Info(err)
			goto ServeHTTPTail
		}
	}

	// Route and process the request.
	handler.Handler.ServeHTTP(writer, r)

ServeHTTPTail:
	// Print out response logging information.
	glog.Infof(`HTTP rsp: "%d %s" %v`, writer.status, http.StatusText(writer.status), time.Since(start))
}

// ConfigureServer is the main entry point for both the container and test
func ConfigureServer(clients client.Clients, namespace, token string) error {
	// Static configuration.
	if err := apis.AddToScheme(scheme.Scheme); err != nil {
		return err
	}

	// Setup globals.
	if err := config.Configure(clients, namespace, token); err != nil {
		return err
	}

	// Setup managers.
	instanceRegistry = registry.New(namespace)

	return nil
}

func RunServer(certificate tls.Certificate) error {
	// Start the server.
	server := &http.Server{
		Addr:    ":8443",
		Handler: NewOpenServiceBrokerHandler(),
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{
				certificate,
			},
		},
	}

	return server.ListenAndServeTLS("", "")
}
