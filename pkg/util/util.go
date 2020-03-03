package util

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/couchbase/service-broker/pkg/api"
	"github.com/couchbase/service-broker/pkg/apis/broker.couchbase.com/v1"
	"github.com/couchbase/service-broker/pkg/errors"

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
func JSONRequest(w http.ResponseWriter, r *http.Request, data interface{}) error {
	// Parse the creation request.
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("unable to read body: %v", err)
	}

	glog.V(1).Infof("JSON req: %s", string(body))
	if err := json.Unmarshal(body, data); err != nil {
		return errors.NewParameterError("unable to unmarshal body: %v", err)
	}

	return nil
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

// translateError translates from an internal error type to a HTTP status code and an API error type.
func translateError(err error) (int, api.APIError) {
	switch {
	case errors.IsConfigurationError(err):
		return http.StatusBadRequest, api.ErrorConfigurationError
	case errors.IsQueryError(err):
		return http.StatusBadRequest, api.ErrorQueryError
	case errors.IsParameterError(err):
		return http.StatusBadRequest, api.ErrorParameterError
	case errors.IsValidationError(err):
		return http.StatusBadRequest, api.ErrorValidationError
	case errors.IsAsyncRequiredError(err):
		return http.StatusUnprocessableEntity, api.ErrorAsyncRequired
	case errors.IsResourceConflictError(err):
		return http.StatusConflict, api.ErrorResourceConflict
	case errors.IsResourceNotFoundError(err):
		return http.StatusNotFound, api.ErrorResourceNotFound
	case errors.IsResourceGoneError(err):
		return http.StatusGone, api.ErrorResourceGone
	default:
		return http.StatusInternalServerError, api.ErrorInternalServerError
	}
}

// JSONError is a helper method to return an error back to the client.
func JSONError(w http.ResponseWriter, err error) {
	status, apiError := translateError(err)
	e := &api.Error{
		Error:       apiError,
		Description: err.Error(),
	}
	JSONResponse(w, status, e)
}

// JSONErrorUsable is a helper method to return an error back to the client,
// it also communicates the instance is usable for example when an update goes
// wrong.
func JSONErrorUsable(w http.ResponseWriter, err error) {
	status, apiError := translateError(err)
	usable := true
	e := &api.Error{
		Error:          apiError,
		Description:    err.Error(),
		InstanceUsable: &usable,
	}
	JSONResponse(w, status, e)
}

// AsyncRequired is called when the handler only supports async requests.
func AsyncRequired(r *http.Request) error {
	// Parse any query parameters.
	query, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		return errors.NewQueryError("malformed query data: %v", err)
	}

	if acceptsIncomplete, ok := query["accepts_incomplete"]; !ok || acceptsIncomplete[0] != "true" {
		return errors.NewAsyncRequiredError("client must support asynchronous instance creation")
	}

	return nil
}

// getServicePlan returns the service plan for the given plan and service offering IDs.
func getServicePlan(config *v1.CouchbaseServiceBrokerConfig, serviceID, planID string) (*v1.ServicePlan, error) {
	if config.Spec.Catalog == nil {
		return nil, errors.NewConfigurationError("service catalog not defined")
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
		return nil, errors.NewParameterError("service plan %s not defined for service offering %s", planID, serviceID)
	}
	return nil, errors.NewParameterError("service offering '%s' not defined", serviceID)
}

// ValidateServicePlan checks the paramters are valid for the configuration.
func ValidateServicePlan(config *v1.CouchbaseServiceBrokerConfig, serviceID, planID string) error {
	if _, err := getServicePlan(config, serviceID, planID); err != nil {
		return err
	}
	return nil
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
		return err
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
			return errors.NewParameterError("schema unmarshal failed: %v", err)
		}

		var parameters interface{}
		if err := json.Unmarshal(data, &parameters); err != nil {
			return errors.NewParameterError("parameters unmarshal failed: %v", err)
		}

		if err := validate.AgainstSchema(schema, parameters, strfmt.NewFormats()); err != nil {
			return errors.NewValidationError("schema validation failed: %v", err)
		}
	}
	return nil
}
