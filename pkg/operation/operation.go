package operation

import (
	"fmt"

	"github.com/google/uuid"
)

// Type is the type of operation being performed.
type Type string

const (
	// TypeServiceInstanceCreate is used when a service instance is being created.
	TypeServiceInstanceCreate Type = "serviceInstanceCreate"

	// TypeServiceInstanceUpdate is used when a service instance is being updated.
	TypeServiceInstanceUpdate Type = "serviceInstanceUpdate"

	// TypeServiceInstanceDelete is used when a service instance is being deleted.
	TypeServiceInstanceDelete Type = "serviceInstanceDelete"
)

// Operation represents an asynchronous operation.
type Operation struct {
	// Type is the type of operation being performed.
	Type Type

	// ID is a unique identifier for the operation.
	ID string

	// ServiceID is the identity of the service related to the operation.
	ServiceID string

	// PlanID is the identity of the plan related to the operation.
	PlanID string

	// Status is used to asynchronously poll for completion and read
	// the operation's error status.
	Status chan error
}

// operations is the global cache of operations.
var operations = map[string]*Operation{}

// Get returns the operation associated with an instance ID.
func Get(instanceID string) (op *Operation, ok bool) {
	op, ok = operations[instanceID]
	return
}

// Delete deletes the operation associated with an instance ID.
func Delete(instanceID string) {
	delete(operations, instanceID)
}

// New creates a new aysnchronous operation for an instance ID.
func New(t Type, instanceID, serviceID, planID string) (*Operation, error) {
	operation := &Operation{
		Type:      t,
		ID:        uuid.New().String(),
		ServiceID: serviceID,
		PlanID:    planID,
		Status:    make(chan error),
	}

	if _, ok := operations[instanceID]; ok {
		return nil, fmt.Errorf("operation already exists for instance")
	}

	operations[instanceID] = operation

	return operation, nil
}

// Reset is only to be used for testing to restore pristine state between test cases.
func Reset() {
	operations = map[string]*Operation{}
}

// Runnable defines an asynchronous operation that is compatable with this package.
type Runnable interface {
	Run() error
}

// Run executes the provided asynchronous operation and returns the status code via
// the operation channel.
func (o *Operation) Run(r Runnable) {
	o.Status <- r.Run()
}
