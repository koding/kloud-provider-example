kloud-provider-example
----------------------

This repository is an example of custom Kloud provider plugin.

Kloud is a backend behind Koding that is responsible for bootstrapping, building and destroying user stacks and managing access to each machine instance within those stacks.

## Requirements

- [go1.6+](https://golang.org/dl/)

## Installation

A new Kloud provider can be added by simply clonning your provider plugin into `kloud/provider` directory and rebuilding Kloud.

```bash
koding $ git clone git@github.com:kloud/kloud-provider-example go/src/koding/kites/kloud/provider/example
koding $ ./go/build.sh
```

## Plugins

A Kloud provider is responsible for composing a single Terraform template from multiple data sources:

- stack template
- credentials
- bootstrapped resources

A single Kloud provider validates credentials provided by a user, bootstraps a stack by creating provider-specific, persistant resources and provisions a Klient service for each instance built within a stack. The Klient service is used to connect remote machine to Koding allowing for webterm sessions in the browser, machine-sharing with other Koding users and starting / stopping the machine itself.

# Example plugin

This repository contains a documented example of a Kloud provider plugin - [example.go](./example.go).

The [Koding repository](https://github.com/koding/koding) contains more examples of Kloud providers:

- [AWS](https://github.com/koding/koding/tree/master/go/src/koding/kites/kloud/provider/aws)
- [Vagrant](https://github.com/koding/koding/tree/master/go/src/koding/kites/kloud/provider/vagrant)
- [Azure](https://github.com/koding/koding/tree/master/go/src/koding/kites/kloud/provider/azure)

# Quick start

Go services in Koding repository use currently project-based GOPATH, that's why ensure you point your GOPATH to the `go` directory inside Koding repository:

```bash
~ $ export GOPATH=~/github.com/koding/koding/go
```

A project structure of your Kloud provider may look like the following:

```
your/
├── your.go        <-- registers *provider.Provider
├── machine.go     <-- defines *Machine struct
├── schema.go      <-- defines *Credential, *Bootstrap and *Metadata structs
└── stack.go       <-- defines *Stack struct
```

- `example.go` registers provider definition (like [this one](./example.go#L18-L105))

The content of this file is:

```go
package your

import (
	"koding/kites/kloud/stack"
	"koding/kites/kloud/stack/provider"
)

var p = &provider.Provider{
	Name:         "your",
	ResourceName: "instance",
	
	// Machine type is defined in machine.go
	Machine: func(bm *provider.BaseMachine) (provider.Machine, error) {
		return &Machine{BaseMachine: bm}, nil
	},
	
	// Stack type is defined in stack.go
	Stack: func(bs *provider.BaseStack) (provider.Stack, error) {
		return &Stack{BaseStack: bs}, nil
	},
	
	// Schema value is defined in schema.go 
	Schema: Schema,
}

func init() {
	provider.Register(p)
}
```

- `schema.go` defines models which are used to persist provider data (like [this one](./example.go#L92-L104))

And its contents:

```go
package your

import "koding/kites/kloud/stack/provider"

var Schema = &provider.Schema{
	NewCredential: func() interface{} { return &Credential{} },
	NewBootstrap:  func() interface{} { return &Bootstrap{} },
	NewMetadata:   func(*stack.Machine) interface{} { &Metadata{} },
}

type Credential struct {
	User string
	Pass string
}

type Bootstrap struct {
	PersistentResourceName string
}

type Metadata struct {
	MachineRegion string
	MachineCNAME  string
}
```

If any of the schema types implement the following interface:

```go
type Validator interface {
	Valid() error
}
```

the `Valid()` method is going to be called after reading the value from Koding database / safe store in order to validate it.

More details on schema can be found [here](./example.go#L71-L91), [here](./example.go#L134-L155) and [here](./example.go#L191-L216).

- `stack.go` defines a Stack struct which implements the `provider.Stack` interface

The stub definition looks like:

```go
package your

import (
	"errors"

	"koding/kites/kloud/stack"
	"koding/kites/kloud/stack/provider"
)

var errNotImplemented = errors.New("not implemented")

type Stack struct {
	*provider.BaseStack
}

func (*Stack) VerifyCredential(*stack.Credential) error {
	return errNotImplemented
}

func (*Stack) BootstrapTemplates(*stack.Credential) ([]*stack.Template, error) {
	return nil, errNotImplemented
}

func (*Stack) ApplyTemplate(*stack.Credential) (*stack.Template, error) {
	return nil, errNotImplemented
}
```

More details on expected behavior of each method can be found [here](./example.go#L223-L236), [here](./example.go#L238-L247) and [here](./example.go#L249-L338).

- and `machine.go`, which defines \*Machine for controlling single remote machine

The stub:

```go
package your

import (
	"errors"

	"koding/kites/kloud/machinestate"
	"koding/kites/kloud/stack/provider"

	"golang.org/x/net/context"
)

var errNotImplemented = errors.New("not implemented")

type Machine struct {
	*provider.BaseMachine
}

func (*Machine) Start(context.Context) (metadata interface{}, err error) {
	return nil, errNotImplemented
}

func (*Machine) Stop(context.Context) (metadata interface{}, err error) {
	return nil, errNotImplemented
}

func (*Machine) Info(context.Context) (state machinestate.State, metadata interface{}, err error) {
	return 0, nil, errNotImplemented
}
```

More details [here](./example.go#L157-L189) and [an example of AWS implementation](https://github.com/koding/koding/blob/358a070fd24700cee4a39dc556ad79164f3b9918/go/src/koding/kites/kloud/provider/aws/machine.go#L42-L68).

## Final notes

The `*provider.BaseMachine` and `*provider.BaseStack` API should be considered not stable, thus a subject to change. Usually it means a rename here and there or new fields that remove the boilerplate even further.

Relavant issues:

- https://github.com/koding/koding/issues/9127
- https://github.com/koding/koding/issues/8903
