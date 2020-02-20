package operation

import (
	"github.com/google/uuid"
)

type OperationKind string

const (
	OperationKindServiceInstanceCreate OperationKind = "serviceInstanceCreate"
	OperationKindServiceInstanceUpdate OperationKind = "serviceInstanceUpdate"
	OperationKindServiceInstanceDelete OperationKind = "serviceInstanceDelete"
)

type Operation struct {
	Kind   OperationKind
	ID     string
	Status chan error
}

var operations = map[string]*Operation{}

func Get(instanceID string) (op *Operation, ok bool) {
	op, ok = operations[instanceID]
	return
}

func Delete(instanceID string) {
	delete(operations, instanceID)
}

func New(kind OperationKind, instanceID string) *Operation {
	operation := &Operation{
		Kind:   kind,
		ID:     uuid.New().String(),
		Status: make(chan error),
	}
	operations[instanceID] = operation
	return operation
}

type Runnable interface {
	Run() error
}

func (o *Operation) Run(r Runnable) {
	o.Status <- r.Run()
}
