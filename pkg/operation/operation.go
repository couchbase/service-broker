package operation

import (
	"fmt"

	"github.com/couchbase/service-broker/pkg/registry"

	"github.com/google/uuid"
)

// Type is the type of operation being performed.
type Type string

const (
	// TypeProvision is used when a resource is being created.
	TypeProvision Type = "provision"

	// TypeUpdate is used when a resource is being updated.
	TypeUpdate Type = "update"

	// TypeDeprovision is used when a resource is being deleted.
	TypeDeprovision Type = "deprovision"
)

// Operation represents an asynchronous operation.
// All state is persisted in the registry entry associated with the instance or binding.
// It is ostensibly an ephemeral cache of status channels so we can poll for completion.
type Operation struct {
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
func Delete(instanceID string, entry *registry.Entry) error {
	delete(operations, instanceID)

	entry.Unset(registry.Operation)
	entry.Unset(registry.OperationID)

	if err := entry.Commit(); err != nil {
		return err
	}

	return nil
}

// New creates a new aysnchronous operation for an instance ID.
func New(t Type, instanceID string, entry *registry.Entry) (*Operation, error) {
	id := uuid.New().String()

	// Persist operation information to the registry.
	entry.Set(registry.Operation, string(t))
	entry.Set(registry.OperationID, id)

	if err := entry.Commit(); err != nil {
		return nil, err
	}

	operation := &Operation{
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
