kloud-provider-example
----------------------

This repository is an example of custom kloud provider.

A usual flow for saving a team's stack from UI looks like:

```
create new credentials -> authenticate request -> save template -> bootstrap request
```

The authenticate and bootstrap request are served by a kloud provider. After the stack is successfully saved, then a user typically initializes a stack and builds it:

```
initialize machines -> plan request -> build stack -> apply request
```

A kloud provider is any type that implements `stack.Provider` interface:

```go
// Provider is used to manage architecture for the specific
// cloud provider and to control particular virtual machines
// within it.
//
// Kloud comes with the following built-in providers:
//
//   - aws
//   - vagrant
//
type Provider interface {
	// Stack returns a provider that implements team methods.
	//
	// Team methods are used to manage architecture for
	// the given cloud-provider - they can bootstrap,
	// modify or destroy resources, which can be
	// backed by Terraform-specific provider.
	//
	// The default helper *provider.BaseStack uses
	// Terraform for heavy lifting. Provider-specific
	// implementations are used to augment the user
	// stacks (Terraform templates) with default
	// resources created during bootstrap.
	Stack(context.Context) (Stack, error)

	// Machine returns a value that implements the Machine interface.
	//
	// The Machine interface is used to control a single vm
	// for the specific cloud-provider.
	Machine(ctx context.Context, id string) (Machine, error)

	// Cred returns new value for provider-specific credential.
	// The Cred is called when building credentials
	// for apply and bootstrap requests, so each provider
	// has access to type-friendly credential values.
	//
	// Examples:
	//
	//   - aws.Cred
	//   - vagrant.Cred
	//
	Cred() interface{}
}
```

Usually a provider composes a `*provider.BaseProvider` helper that implements most of the common functionality that a kloud provider offers. During startup kloud is responsible for creating a `*provider.BaseProvider` value, which is set up for use with database and other services like terraformer, which is a regular Terraform binary served as an API kite.

All built-in and external providers are registered in a global `provider.All` map, in a similar manner that a database driver is registered for use with `database/sql` package. In order to use your kloud provider implementation with kloud, clone its sources into `go/src/koding/kites/kloud/provider` directory within  repo


```
koding $ git clone https://github.com/koding/kloud-provider-example go/src/koding/kites/kloud/provider/example
```

And ensure your provider is added to `provider.All` map. Usually it is enough to add it within [an init method](example.go#L6):

```go
func init() {
	provider.All["example"] = func(bp *provider.BaseProvider) stack.Provider {
		return &Provider{
			BaseProvider: bp,
		}
	}
}
```

When building Koding services with [build.sh](https://github.com/koding/koding/blob/master/go/build.sh), the script will generate empty imports for each sub-package under the `koding/kites/kloud/provider` directory, so it is enough to just clone a repository with your provider and build Koding services - your provider will be automatically bundled.

A typical provider is responsible for managing team stack and managing a single machine. In order to manage a stack, custom provider is expected to provide a type that implements a `stack.Stack` interface:

```go
// Stack is a provider-specific handler that implements team methods.
type Stack interface {
	// Apply is responsible for building team's stack template and sending it
	// to Terraformer kite in order to apply requested changes.
	//
	// If modifying an infrastructures with Terraform is successful, stack
	// is expected to update jMachine documents for each instance
	// within the stack, if neccessary. It may update fields like ipAddress
	// or queryString.
	//
	// If existing resources are requested to be deleted, after they are
	// successfully destroyed by Terraform the provider is responsible
	// for removing any data it created for the stack.
	Apply(context.Context) (interface{}, error)

	// Authenticate is responsible for veryfying whether user-provided
	// credentials are valid for the given provider.
	Authenticate(context.Context) (interface{}, error)

	// Bootstrap is responsible for creating resources that are
	// shared among all stacks created using particular credentials.
	//
	// Usually provider creates those resources with internal
	// Terraform template and updates credential data
	// with all the necessarry IDs that are required to reference
	// them during Apply operation.
	//
	// If Destroy of the request is set to true, Bootstrap
	// must attempt to destroy existing resources with Terraform.
	Bootstrap(context.Context) (interface{}, error)

	// Plan is like Apply - it builds stack template and sends it
	// to a Terraformer kite, but it does not modify the infrastructure.
	//
	// It is used when creating jMachine documents for a given stack
	// - kloud is asked what kind of and how many machines will this stack
	// template create, so it uses Terraformer's plan method to
	// read the resulting output attributes after executing the template.
	Plan(context.Context) (interface{}, error)
}
```

When creating a new instance resources, that is meant to be connected to a Koding web terminal, the stack implementation must inject a klient binary into each such resource. Typically this is done by generating a kite.key with `*userdata.KeyCreator` helper and wrapping user's provisionning script into kloud's cloud-init, that is responsible for installing a klient binary and registering it with Kontrol service. For example see [(\*Stack).BuildResources](stack.go#L196) implementation.

Any stack implementation may use helpers, which include common methods for parsing and editing Terraform template or creating and querying klient kite:

	- `*stackplan.Builder`, which fetches stack resources from database, like jMachines, jCredentials or jUsers
    - `*stackplan.Planner`, which checks connectivity to the freshly provisionned klient instances
    - `*provider.BaseStack`, which contains common functionality for team stacks

After a team stack is created, new instances are connected to Koding. Each instance is represented by `stack.Machine` interface:

```go
// Machine represents an instance built by external cloud provider,
// that is connected to Koding via klient interface.
type Machine interface {
	// Start starts the machine.
	//
	// If it is already started, the method is a nop.
	Start(context.Context) error

	// Stop stops the machine.
	// If it is already stopped, the method is a nop.
	Stop(context.Context) error

	// Info describes the machine with *InfoResponse.
	Info(context.Context) (*InfoResponse, error)

	// State gives the state of the machine.
	State() machinestate.State

	// ProviderName gives the name of the provider,
	// which is responsible for managing this machine.
	ProviderName() string
}
```

In addition to common properties, like `ipAddress` or `queryString`, each machine can contain provider-specific metadata. The metadata is usually described by a Meta struct (like `*aws.Meta` or `*vagrant.Meta`). The provider is expected to read or update the metadata from jMachine.meta field of the database model. The `*provider.BaseMachine` is a model for the shared machine document.

TODO

- https://github.com/koding/koding/issues/8352 - improve custom provider API and make it possible for a stack to have multiple providers
