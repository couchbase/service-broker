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

package broker

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/couchbase/service-broker/pkg/api"
	v1 "github.com/couchbase/service-broker/pkg/apis/servicebroker/v1alpha1"
	"github.com/couchbase/service-broker/pkg/errors"
	"github.com/couchbase/service-broker/pkg/log"

	"github.com/go-openapi/jsonpointer"
	"github.com/go-openapi/spec"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/validate"
	"github.com/golang/glog"

	"k8s.io/apimachinery/pkg/runtime"
)

// httpResponse is the canonical writer for HTTP responses.
func httpResponse(w http.ResponseWriter, status int) {
	w.WriteHeader(status)
}

// jsonRequest reads the JSON body into the give structure and raises the
// appropriate errors on error.
func jsonRequest(r *http.Request, data interface{}) error {
	// Parse the creation request.
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("unable to read body: %w", err)
	}

	glog.V(log.LevelDebug).Infof("JSON req: %s", string(body))

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
		glog.Infof("failed to marshal body: %v", err)
		httpResponse(w, http.StatusInternalServerError)
	}

	glog.V(log.LevelDebug).Infof("JSON rsp: %s", string(resp))

	w.Header().Set("Content-Type", "application/json")

	httpResponse(w, status)

	if _, err := w.Write(resp); err != nil {
		glog.Infof("error writing response: %v", err)
	}
}

// translateError translates from an internal error type to a HTTP status code and an API error type.
func translateError(err error) (int, api.ErrorType) {
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

// jsonError is a helper method to return an error back to the client.
func jsonError(w http.ResponseWriter, err error) {
	status, apiError := translateError(err)
	e := &api.Error{
		Error:       apiError,
		Description: err.Error(),
	}
	JSONResponse(w, status, e)
}

// jsonErrorUsable is a helper method to return an error back to the client,
// it also communicates the instance is usable for example when an update goes
// wrong.
func jsonErrorUsable(w http.ResponseWriter, err error) {
	status, apiError := translateError(err)
	usable := true
	e := &api.Error{
		Error:          apiError,
		Description:    err.Error(),
		InstanceUsable: &usable,
	}
	JSONResponse(w, status, e)
}

// maygetSingleParameter gets a named parameter from the request URL.  Returns false
// if it doesn't exist and an error if there is any abiguity.
func maygetSingleParameter(r *http.Request, name string) (string, bool, error) {
	query, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		return "", false, errors.NewQueryError("malformed query data: %v", err)
	}

	values, ok := query[name]
	if !ok {
		return "", false, nil
	}

	requiredParameters := 1
	if len(values) != requiredParameters {
		return "", true, errors.NewQueryError("query parameter %s not unique", name)
	}

	return values[0], true, nil
}

// getSingleParameter gets a named parameter from the request URL.  Returns an
// error if it doesn't exist or there is any abiguity.
func getSingleParameter(r *http.Request, name string) (string, error) {
	value, exists, err := maygetSingleParameter(r, name)
	if err != nil {
		return "", err
	}

	if !exists {
		return "", errors.NewQueryError("query parameter %s not found", name)
	}

	return value, nil
}

// asyncRequired is called when the handler only supports async requests.
// Don't use getSingleParameter as we need to selectively return the correct
// status codes.
func asyncRequired(r *http.Request) error {
	query, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		return errors.NewQueryError("malformed query data: %v", err)
	}

	if acceptsIncomplete, ok := query["accepts_incomplete"]; !ok || acceptsIncomplete[0] != "true" {
		return errors.NewAsyncRequiredError("client must support asynchronous instance creation")
	}

	return nil
}

// getServiceOffering returns the service offering for a given service offering ID.
func getServiceOffering(config *v1.ServiceBrokerConfig, serviceID string) (*v1.ServiceOffering, error) {
	for index, service := range config.Spec.Catalog.Services {
		if service.ID == serviceID {
			return &config.Spec.Catalog.Services[index], nil
		}
	}

	return nil, errors.NewParameterError("service offering '%s' not defined", serviceID)
}

// getServicePlan returns the service plan for the given plan and service offering IDs.
func getServicePlan(config *v1.ServiceBrokerConfig, serviceID, planID string) (*v1.ServicePlan, error) {
	service, err := getServiceOffering(config, serviceID)
	if err != nil {
		return nil, err
	}

	for index, plan := range service.Plans {
		if plan.ID == planID {
			return &service.Plans[index], nil
		}
	}

	return nil, errors.NewParameterError("service plan %s not defined for service offering %s", planID, serviceID)
}

// validateServicePlan checks the parameters are valid for the configuration.
func validateServicePlan(config *v1.ServiceBrokerConfig, serviceID, planID string) error {
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
	schemaTypeServiceInstance schemaType = "serviceInstance"
	schemaTypeServiceBinding  schemaType = "serviceBinding"

	schemaOperationCreate schemaOperation = "create"
	schemaOperationUpdate schemaOperation = "update"
)

// getSchema returns the schema associated with an operation on a resource type.  If none
// is associated with the plan for the operation it will return nil.
func getSchema(config *v1.ServiceBrokerConfig, serviceID, planID string, t schemaType, o schemaOperation) (*v1.InputParamtersSchema, error) {
	plan, err := getServicePlan(config, serviceID, planID)
	if err != nil {
		return nil, err
	}

	if plan.Schemas == nil {
		return nil, nil
	}

	switch t {
	case schemaTypeServiceInstance:
		if plan.Schemas.ServiceInstance == nil {
			return nil, nil
		}

		switch o {
		case schemaOperationCreate:
			return plan.Schemas.ServiceInstance.Create, nil
		case schemaOperationUpdate:
			return plan.Schemas.ServiceInstance.Update, nil
		default:
			return nil, fmt.Errorf("%w: unexpected schema operation: %v", ErrUnexpected, o)
		}
	case schemaTypeServiceBinding:
		if plan.Schemas.ServiceBinding == nil {
			return nil, nil
		}

		switch o {
		case schemaOperationCreate:
			return plan.Schemas.ServiceBinding.Create, nil
		default:
			return nil, fmt.Errorf("%w: unexpected schema operation: %v", ErrUnexpected, o)
		}
	default:
		return nil, fmt.Errorf("%w: unexpected schema type: %v", ErrUnexpected, t)
	}
}

// validateParameters validates any supplied parameters against an JSON schema if it exists.
func validateParameters(config *v1.ServiceBrokerConfig, serviceID, planID string, t schemaType, o schemaOperation, parametersRaw *runtime.RawExtension) error {
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

// planUpdatable accepts the service ID the original plan ID, and the new one, returning
// an error if the service catalog doesn't allow it.
func planUpdatable(config *v1.ServiceBrokerConfig, serviceID, planID, newPlanID string) error {
	service, err := getServiceOffering(config, serviceID)
	if err != nil {
		return err
	}

	if !service.PlanUpdatable && (planID != newPlanID) {
		return errors.NewParameterError("service plan %s for service %s cannot be updated", planID, serviceID)
	}

	return nil
}

// verifyBindable returns an error if the plan cannot be bound to.
func verifyBindable(config *v1.ServiceBrokerConfig, serviceID, planID string) error {
	service, err := getServiceOffering(config, serviceID)
	if err != nil {
		return err
	}

	plan, err := getServicePlan(config, serviceID, planID)
	if err != nil {
		return err
	}

	bindable := service.Bindable
	if plan.Bindable != nil {
		bindable = *plan.Bindable
	}

	if !bindable {
		return errors.NewConfigurationError("service plan %s for service %s is not bindable", planID, serviceID)
	}

	return nil
}

// getNamespace returns the namespace to provision resources in.  This is the namespace
// the broker lives in by default, however when operating as a kubernetes cluster service
// broker then this information is passed as request context.
func getNamespace(context *runtime.RawExtension, namespace string) (string, error) {
	if context != nil {
		var ctx interface{}

		if err := json.Unmarshal(context.Raw, &ctx); err != nil {
			glog.Infof("unmarshal of client context failed: %v", err)
			return "", err
		}

		pointer, err := jsonpointer.New("/namespace")
		if err != nil {
			glog.Infof("failed to parse JSON pointer: %v", err)
			return "", err
		}

		v, _, err := pointer.Get(ctx)
		if err == nil {
			namespace, ok := v.(string)
			if ok {
				return namespace, nil
			}

			glog.Infof("request context namespace not a string")

			return "", errors.NewParameterError("request context namespace not a string")
		}
	}

	return namespace, nil
}
