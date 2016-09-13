package example

import (
	"errors"

	"koding/kites/kloud/machinestate"
	"koding/kites/kloud/provider"
	"koding/kites/kloud/stack"

	"golang.org/x/net/context"
)

// Meta
type Meta struct {
	AlwaysOn  bool   `bson:"alwaysOn"`
	ExampleID string `bson:"exampleID"`
}

var _ stack.Validator = (*Meta)(nil)

// Valid
func (m *Meta) Valid() error {
	if m.ExampleID == "" {
		return errors.New("example ID is empty")
	}

	return nil
}

// Machine
type Machine struct {
	*provider.BaseMachine

	Meta *Meta `bson:"-"`
	Cred *Cred `bson:"-"`
}

var _ stack.Machine = (*Machine)(nil)

// Start
func (m *Machine) Start(ctx context.Context) error {
	return nil
}

// Stop
func (m *Machine) Stop(ctx context.Context) error {
	return nil
}

// Info
func (m *Machine) Info(ctx context.Context) (*stack.InfoResponse, error) {
	return &stack.InfoResponse{
		State: machinestate.Running,
	}, nil
}
