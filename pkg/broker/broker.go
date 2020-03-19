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
	"github.com/couchbase/service-broker/pkg/log"
	"github.com/couchbase/service-broker/pkg/util"

	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"

	"k8s.io/client-go/kubernetes/scheme"
)

// getHeader returns the header value for a header name.
func getHeader(r *http.Request, name string) ([]string, error) {
	for headerName := range r.Header {
		if strings.EqualFold(headerName, name) {
			return r.Header[headerName], nil
		}
	}

	return nil, fmt.Errorf("no header found for %s", name)
}

// getHeaderSingle returns the header value for a name.
// If the header has more than one value this is an error condition.
func getHeaderSingle(r *http.Request, name string) (string, error) {
	headers, err := getHeader(r, name)
	if err != nil {
		return "", err
	}

	requiredHeaders := 1
	if len(headers) != requiredHeaders {
		return "", fmt.Errorf("multiple headers found for %s", name)
	}

	return headers[0], nil
}

// handleReadiness returns 503 until the configuration is correct.
func handleReadiness(w http.ResponseWriter) error {
	if config.Config() == nil {
		util.HTTPResponse(w, http.StatusServiceUnavailable)
		return fmt.Errorf("service not ready")
	}

	return nil
}

// handleBrokerBearerToken implements RFC-6750.
func handleBrokerBearerToken(w http.ResponseWriter, r *http.Request) error {
	header, err := getHeaderSingle(r, "Authorization")
	if err != nil {
		util.HTTPResponse(w, http.StatusUnauthorized)
		return err
	}

	if header != "Bearer "+config.Token() {
		util.HTTPResponse(w, http.StatusUnauthorized)
		return fmt.Errorf("authorization failed")
	}

	return nil
}

// handleBrokerAPIHeader looks for and verifies the X-Broker-API-Version header.
func handleBrokerAPIHeader(w http.ResponseWriter, r *http.Request) error {
	header, err := getHeaderSingle(r, "X-Broker-API-Version")
	if err != nil {
		util.HTTPResponse(w, http.StatusBadRequest)
		return err
	}

	apiVersion, err := strconv.ParseFloat(header, 64)
	if err != nil {
		util.HTTPResponse(w, http.StatusBadRequest)
		return fmt.Errorf("malformed X-Broker-Api-Version header: %v", err)
	}

	if apiVersion < minBrokerAPIVersion {
		util.HTTPResponse(w, http.StatusPreconditionFailed)
		return fmt.Errorf("unsupported X-Broker-Api-Version header %v, requires at least %.2f", header, minBrokerAPIVersion)
	}

	return nil
}

// handleContentTypeHeader looks for and verifies the Content-Type header.
func handleContentTypeHeader(w http.ResponseWriter, r *http.Request) error {
	// If no content is specified we don't need a type.
	if r.ContentLength == 0 {
		return nil
	}

	header, err := getHeaderSingle(r, "Content-Type")
	if err != nil {
		util.HTTPResponse(w, http.StatusBadRequest)
		return err
	}

	if header != "application/json" {
		util.HTTPResponse(w, http.StatusBadRequest)
		return fmt.Errorf("invalid Content-Type header")
	}

	return nil
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
	router.GET("/v2/service_instances/:instance_id/last_operation", handlePollServiceInstance)
	router.PUT("/v2/service_instances/:instance_id/service_bindings/:binding_id", handleCreateServiceBinding)
	router.DELETE("/v2/service_instances/:instance_id/service_bindings/:binding_id", handleDeleteServiceBinding)

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

	// The configuration is global, and sadly we cannot pass it around as a variable
	// so place a read lock on it for the duration of the request.  Requests must
	// therefore be non-blocking.
	config.Lock()
	defer config.Unlock()

	// Use the wrapped writer so we can capture the status code etc.
	writer := &responseWriter{
		writer: w,
	}

	// Print out request logging information.
	// DO NOT print out headers at info level as that will leak credentials into the log stream.
	glog.Infof(`HTTP req: "%s %v %s" %s `, r.Method, r.URL, r.Proto, r.RemoteAddr)

	for name, values := range r.Header {
		for _, value := range values {
			glog.V(log.LevelDebug).Infof(`HTTP hdr: "%s: %s"`, name, value)
		}
	}

	defer func() {
		glog.Infof(`HTTP rsp: "%d %s" %v`, writer.status, http.StatusText(writer.status), time.Since(start))
	}()

	// Indicate that the service is not ready until configured.
	if err := handleReadiness(writer); err != nil {
		glog.V(log.LevelDebug).Info(err)
		return
	}

	// Ignore security checks for the readiness handler
	if r.URL.Path != "/readyz" {
		// Process headers, API versions, content types.
		if err := handleRequestHeaders(writer, r); err != nil {
			glog.V(log.LevelDebug).Info(err)
			return
		}
	}

	// Route and process the request.
	handler.Handler.ServeHTTP(writer, r)
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
