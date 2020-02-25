// Package broker implements the Open Broker API for the Couchbase Operator.
package broker

import (
	"crypto/tls"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/couchbase/service-broker/pkg/apis"
	"github.com/couchbase/service-broker/pkg/client"
	"github.com/couchbase/service-broker/pkg/config"
	"github.com/couchbase/service-broker/pkg/registry"
	"github.com/couchbase/service-broker/pkg/version"

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
		w.WriteHeader(http.StatusServiceUnavailable)
		return fmt.Errorf("service not ready")
	}
	return nil
}

// handleBrokerBearerToken implements RFC-6750.
func handleBrokerBearerToken(w http.ResponseWriter, r *http.Request) error {
	for name := range r.Header {
		if strings.EqualFold(name, "Authorization") {
			if len(r.Header[name]) != 1 {
				w.WriteHeader(http.StatusBadRequest)
				return fmt.Errorf("multiple Authorization headers given")
			}
			if r.Header[name][0] != "Bearer "+string(config.Token()) {
				w.WriteHeader(http.StatusUnauthorized)
				return fmt.Errorf("authorization failed")
			}
			return nil
		}
	}
	w.WriteHeader(http.StatusUnauthorized)
	return fmt.Errorf("no Authorization header")
}

// handleBrokerAPIHeader looks for a verifies the X-Broker-API-Version header
// is supported.
func handleBrokerAPIHeader(w http.ResponseWriter, r *http.Request) error {
	for name := range r.Header {
		if strings.EqualFold(name, "X-Broker-API-Version") {
			if len(r.Header[name]) != 1 {
				w.WriteHeader(http.StatusBadRequest)
				return fmt.Errorf("multiple X-Broker-Api-Version headers given")
			}
			apiVersion, err := strconv.ParseFloat(r.Header[name][0], 64)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return fmt.Errorf("malformed X-Broker-Api-Version header: %v", err)
			}
			if apiVersion < minBrokerAPIVersion {
				w.WriteHeader(http.StatusPreconditionFailed)
				return fmt.Errorf("unsupported X-Broker-Api-Version header %v, requires at least %.2f", r.Header[name][0], minBrokerAPIVersion)
			}
			return nil
		}
	}
	w.WriteHeader(http.StatusBadRequest)
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
			w.WriteHeader(http.StatusBadRequest)
			return fmt.Errorf("invalid Content-Type header")
		}
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
	router.GET("/v2/service_instances/:instance_id/last_operation", handleReadServiceInstanceStatus)
	router.PUT("/v2/service_instances/:instance_id/service_bindings/:binding_id", handleCreateServiceBinding)
	router.GET("/v2/service_instances/:instance_id/service_bindings/:binding_id", handleReadServiceBinding)
	router.PATCH("/v2/service_instances/:instance_id/service_bindings/:binding_id", handleUpdateServiceBinding)
	router.DELETE("/v2/service_instances/:instance_id/service_bindings/:binding_id", handleDeleteServiceBinding)
	router.GET("/v2/service_instances/:instance_id/service_bindings/:binding_id/last_operation", handleReadServiceBindingStatus)
	return &openServiceBrokerHandler{Handler: router}
}

// ServeHTTP performs generic test on all API endpoints.
func (handler *openServiceBrokerHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Indicate that the service is not ready until configured.
	if err := handleReadiness(w, r); err != nil {
		glog.Error(err)
		return
	}

	// Ignore security checks for the readiness handler
	if r.URL.Path != "/readyz" {
		// Process headers, API versions, content types.
		if err := handleRequestHeaders(w, r); err != nil {
			glog.Error(err)
			return
		}
	}

	// Print out logging information.
	userAgent := "-"
	for name := range r.Header {
		if strings.EqualFold(name, "User-Agent") {
			userAgent = r.Header[name][0]
			break
		}
	}

	glog.Infof(`%s "%s %s %s" %s`, r.RemoteAddr, r.Method, r.URL.Path, r.Proto, userAgent)

	// Route and process the request.
	handler.Handler.ServeHTTP(w, r)
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

// Run is the main entry point of the service broker.
func Run() {
	// Start the server.
	glog.Infof("%s %s", version.Application, version.Version)

	// Runtime configuration.
	var tokenPath string
	var tlsCertificatePath string
	var tlsPrivateKeyPath string

	flag.StringVar(&tokenPath, "token", "/var/run/secrets/couchbase.com/service-broker/token", "Bearer token for API authentication")
	flag.StringVar(&tlsCertificatePath, "tls-certificate", "/var/run/secrets/couchbase.com/service-broker/tls-certificate", "Path to the server TLS certificate")
	flag.StringVar(&tlsPrivateKeyPath, "tls-private-key", "/var/run/secrets/couchbase.com/service-broker/tls-private-key", "Path to the server TLS key")
	flag.Parse()

	// Parse implicit configuration.
	namespace, ok := os.LookupEnv("NAMESPACE")
	if !ok {
		glog.Fatal(fmt.Errorf("NAMESPACE environment variable must be set"))
		os.Exit(1)
	}

	// Load up explicit configuration.
	token, err := ioutil.ReadFile(tokenPath)
	if err != nil {
		glog.Fatal(err)
		os.Exit(1)
	}
	cert, err := tls.LoadX509KeyPair(tlsCertificatePath, tlsPrivateKeyPath)
	if err != nil {
		glog.Fatal(err)
		os.Exit(1)
	}

	// Initialize the clients.
	clients, err := client.New()
	if err != nil {
		glog.Fatal(err)
		os.Exit(1)
	}

	// Start the server.
	if err := ConfigureServer(clients, namespace, string(token)); err != nil {
		glog.Fatal(err)
		os.Exit(1)
	}
	if err := RunServer(cert); err != nil {
		glog.Fatal(err)
		os.Exit(1)
	}
}
