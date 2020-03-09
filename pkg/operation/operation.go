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

// Start begins an asynchronous operation on the registry entry.
func Start(entry *registry.Entry, t Type) error {
	if op, ok := entry.Get(registry.Operation); ok {
		return fmt.Errorf("%s operation already exists for instance", op)
	}

	id := uuid.New().String()

	entry.Set(registry.Operation, string(t))
	entry.Set(registry.OperationID, id)

	if err := entry.Commit(); err != nil {
		return err
	}

	return nil
}

// Complete sets the asynchronous operation completion on the registry entry.
func Complete(entry *registry.Entry, err error) error {
	if op, ok := entry.Get(registry.Operation); !ok {
		return fmt.Errorf("%s operation does not exist for instance", op)
	}

	errString := ""
	if err != nil {
		errString = err.Error()
	}

	entry.Set(registry.OperationStatus, errString)

	if err := entry.Commit(); err != nil {
		return err
	}

	return err
}

// End ends an asynchronous operation on the registry entry.
func End(entry *registry.Entry) error {
	if op, ok := entry.Get(registry.Operation); !ok {
		return fmt.Errorf("%s operation does not exist for instance", op)
	}

	entry.Unset(registry.Operation)
	entry.Unset(registry.OperationID)
	entry.Unset(registry.OperationStatus)

	if err := entry.Commit(); err != nil {
		return err
	}

	return nil
}
