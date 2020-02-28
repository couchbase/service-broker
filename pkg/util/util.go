package util

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/couchbase/service-broker/pkg/api"
	"github.com/couchbase/service-broker/pkg/apis/broker.couchbase.com/v1"

	"github.com/go-openapi/spec"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/validate"
	"github.com/golang/glog"

	"k8s.io/apimachinery/pkg/runtime"
)

// HTTPResponse is the canonical writer for HTTP responses.
func HTTPResponse(w http.ResponseWriter, status int) {
	w.WriteHeader(status)
}

// JSONRequest reads the JSON body into the give structure and raises the
// appropriate errors on error.
func JSONRequest(w http.ResponseWriter, r *http.Request, data interface{}) bool {
	// Parse the creation request.
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		JSONError(w, http.StatusBadRequest, fmt.Errorf("unable to read body: %v", err))
		return false
	}

	glog.V(1).Infof("JSON req: %s", string(body))
	if err := json.Unmarshal(body, data); err != nil {
		JSONError(w, http.StatusBadRequest, fmt.Errorf("unable to unmarshal body: %v", err))
		return false
	}

	return true
}

// JSONResponse sends generic JSON data back to the client and replies
// with a HTTP status code.
func JSONResponse(w http.ResponseWriter, status int, data interface{}) {
	resp, err := json.Marshal(data)
	if err != nil {
		glog.Errorf("failed to marshal body: %v", err)
		HTTPResponse(w, http.StatusInternalServerError)
	}

	glog.V(1).Infof("JSON rsp: %s", string(resp))
	w.Header().Set("Content-Type", "application/json")
	HTTPResponse(w, status)
	if _, err := w.Write(resp); err != nil {
		glog.Errorf("error writing response: %v", err)
	}
}

// JSONError is a helper method to return an error back to the client.
func JSONError(w http.ResponseWriter, status int, err error) {
	e := &api.Error{
		Description: err.Error(),
	}
	JSONResponse(w, status, e)
}

// JSONErrorUsable is a helper method to return an error back to the client,
// it also communicates the instance is usable for example when an update goes
// wrong.
func JSONErrorUsable(w http.ResponseWriter, status int, err error) {
	usable := true
	e := &api.Error{
		Description:    err.Error(),
		InstanceUsable: &usable,
	}
	JSONResponse(w, status, e)
}

// AsyncOnlyResponse is called when the handler only supports async requests.
// It will respond with the correct status code and error message, and return
// an error for logging.  Clients must terminate processing when false is returned.
func AsyncOnlyResponse(w http.ResponseWriter, r *http.Request) bool {
	// Parse any query parameters.
	query, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		JSONError(w, http.StatusBadRequest, fmt.Errorf("malformed query data: %v", err))
		return false
	}

	if acceptsIncomplete, ok := query["accepts_incomplete"]; !ok || acceptsIncomplete[0] != "true" {
		// Respond to the client.
		response := &api.Error{
			Error:       "AsyncRequired",
			Description: "client must support asynchronous instance creation",
		}
		JSONResponse(w, http.StatusUnprocessableEntity, response)
		return false
	}

	return true
}

// getServicePlan returns the service plan for the given plan and service offering IDs.
func getServicePlan(config *v1.CouchbaseServiceBrokerConfig, serviceID, planID string) (*v1.ServicePlan, error) {
	if config.Spec.Catalog == nil {
		return nil, fmt.Errorf("illegal configuration: empty catalog")
	}
	for serviceIndex, service := range config.Spec.Catalog.Services {
		if service.ID != serviceID {
			continue
		}
		for planIndex, plan := range service.Plans {
			if plan.ID != planID {
				continue
			}
			return &config.Spec.Catalog.Services[serviceIndex].Plans[planIndex], nil
		}
		return nil, fmt.Errorf("service plan %s not found in service offering %s", planID, serviceID)
	}
	return nil, fmt.Errorf("service offering '%s' not found", serviceID)
}

// schemaType is the type of schema we are referring to, either for a service instance
// or a service binding.
type schemaType string

// schemaOperation is type of schema operation we are referring to, either a create
// or an update.
type schemaOperation string

const (
	SchemaTypeServiceInstance schemaType = "serviceInstance"
	SchemaTypeServiceBinding  schemaType = "serviceBinding"

	SchemaOperationCreate schemaOperation = "create"
	SchemaOperationUpdate schemaOperation = "update"
)

// getSchema returns the schema associated with an operation on a resource type.  If none
// is associated with the plan for the operation it will return nil.
func getSchema(config *v1.CouchbaseServiceBrokerConfig, serviceID, planID string, t schemaType, o schemaOperation) (*v1.InputParamtersSchema, error) {
	plan, err := getServicePlan(config, serviceID, planID)
	if err != nil {
		return nil, err
	}

	if plan.Schemas == nil {
		return nil, nil
	}

	switch t {
	case SchemaTypeServiceInstance:
		if plan.Schemas.ServiceInstance == nil {
			return nil, nil
		}
		switch o {
		case SchemaOperationCreate:
			return plan.Schemas.ServiceInstance.Create, nil
		case SchemaOperationUpdate:
			return plan.Schemas.ServiceInstance.Update, nil
		default:
			return nil, fmt.Errorf("unexpected schema operation: %v", o)
		}
	case SchemaTypeServiceBinding:
		if plan.Schemas.ServiceBinding == nil {
			return nil, nil
		}
		switch o {
		case SchemaOperationCreate:
			return plan.Schemas.ServiceBinding.Create, nil
		default:
			return nil, fmt.Errorf("unexpected schema operation: %v", o)
		}
	default:
		return nil, fmt.Errorf("unexpected schema type: %v", t)
	}
}

// ValidateParameters validates any supplied parameters against an JSON schema if it exists.
func ValidateParameters(config *v1.CouchbaseServiceBrokerConfig, serviceID, planID string, t schemaType, o schemaOperation, parametersRaw *runtime.RawExtension) error {
	schemaRaw, err := getSchema(config, serviceID, planID, t, o)
	if err != nil {
		return fmt.Errorf("unable to lookup schema: %v", err)
	}
	if schemaRaw != nil {
		// Default to an empty object, that way we can detect when required
		// fields are missing.
		data := []byte("{}")
		if parametersRaw != nil {
			data = parametersRaw.Raw
		}

		schema := &spec.Schema{}
		if err := json.Unmarshal(schemaRaw.Parameters.Raw, schema); err != nil {
			return fmt.Errorf("schema unmarshal failed: %v", err)
		}

		var parameters interface{}
		if err := json.Unmarshal(data, &parameters); err != nil {
			return fmt.Errorf("parameters unmarshal failed: %v", err)
		}

		if err := validate.AgainstSchema(schema, parameters, strfmt.NewFormats()); err != nil {
			return fmt.Errorf("schema validation failed: %v", err)
		}
	}
	return nil
}
