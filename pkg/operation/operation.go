package operation

import (
	"fmt"

	"github.com/google/uuid"
)

// OperationKind is the type of operation being performed.
type OperationKind string

const (
	// OperationKindServiceInstanceCreate is used when a service instance is being created.
	OperationKindServiceInstanceCreate OperationKind = "serviceInstanceCreate"

	// OperationKindServiceInstanceUpdate is used when a service instance is being updated.
	OperationKindServiceInstanceUpdate OperationKind = "serviceInstanceUpdate"

	// OperationKindServiceInstanceDelete is used when a service instance is being deleted.
	OperationKindServiceInstanceDelete OperationKind = "serviceInstanceDelete"
)

// Operation represents an asyncronous operation.
type Operation struct {
	// Kind is the type of operation being performed.
	Kind OperationKind

	// ID is a unique identifier for the operation.
	ID string

	// Status is used to asynchronously poll for completion and read
	// the operation's error status.
	Status chan error
}

// operations is the global cache of operations.
// TODO: Persist as a configmap?
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
func New(kind OperationKind, instanceID string) (*Operation, error) {
	operation := &Operation{
		Kind:   kind,
		ID:     uuid.New().String(),
		Status: make(chan error),
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
