// Package example implements a kloud provider for Example.
package example

import (
	"errors"
	"math/rand"

	"koding/kites/kloud/machinestate"
	"koding/kites/kloud/stack"
	"koding/kites/kloud/stack/provider"
	"koding/kites/kloud/userdata"

	"golang.org/x/net/context"
)

// exampleProvider describes a kloud provider that implements handling
// of Example stacks.
var exampleProvider = &provider.Provider{
	Name:         "example",
	ResourceName: "instance",
	Machine: func(bm *provider.BaseMachine) (provider.Machine, error) {
		m := &Machine{BaseMachine: bm}

		// Machine function can be used to e.g. initialize cloud API
		// client using the credentials provided during stack
		// build.
		//
		// The initialization may look like the following:
		//
		//   c, err := exampleapi.NewClient(exampleapi.Config{
		//       User:   m.Credential().User,
		//       Secret: m.Credential().Secret,
		//   })
		//
		//   if err != nil {
		//       return nil, err
		//   }
		//
		//   m.Client = c
		//
		// The function should return non-nil error if it's
		// not possible manage the remote instance with
		// its API, e.g. the credentials were invalidated
		// or the API server endpoint is down.
		//
		// An example for AWS provider:
		//
		//   https://git.io/vij6W
		//

		return m, nil
	},
	Stack: func(bs *provider.BaseStack) (provider.Stack, error) {
		s := &Stack{BaseStack: bs}

		// Stack function can be used to e.g. prepare access
		// to external resources, which are going to be needed
		// during stack build.
		//
		// The function should return non-nil error if it's
		// not possible to create fully operational stack
		// capable of building it, destroying or describing.
		//
		// An example for Vagrant provider:
		//
		//   https://git.io/vij6i
		//

		return s, nil
	},
	// Schema represents data types used by the provider.
	//
	// The Credential and Bootstrap values are persisted in a secure
	// store.
	//
	// The Credential is used as a Terraform Provider configuration.
	// It is created by user on frontend and saved to a safe location.
	// During stack builds the Credential is read from safe location
	// and is applied onto user's stack template.
	//
	// The Bootstrap is a way to define persistant resources that
	// are bound to a credential lifetime as oppose to a stack lifetime.
	// The Bootstrap value itself stores output variables created by applying
	// bootstrap template. The Bootstrap is used to inject resources
	// into user's stack template.
	//
	// Example schema definitions for built-in providers:
	//
	//   AWS:      https://git.io/vij5v
	//   Vagrant:  https://git.io/vij5s
	//
	Schema: &provider.Schema{
		NewCredential: func() interface{} { return &Credential{} },
		NewBootstrap:  func() interface{} { return &Bootstrap{} },
		NewMetadata: func(m *stack.Machine) interface{} {
			if m == nil {
				return &Metadata{}
			}

			return &Metadata{
				ID: m.Attributes["external_id"],
			}
		},
	},
}

func init() {
	// Register makes the exampleProvider available on Koding.
	provider.Register(exampleProvider)
}

var bootstrapTemplate = `
{
  "provider": {
    "example": {
      "user": "${var.example_user}",
      "secret": "${var.example_secret}",
    }
  },
  "output": {
    "vpc": {
      "value": "${example_vpc.koding.id}"
    }
  },
  "resource": {
    "example_vpc": {
      "koding": {
		  "mask": "172.31.0.0/16"
      }
    }
  }
}`

// Credential defines the configuration for  Example Terraform Provider.
//
// The value is persisted to a Koding safe store.
type Credential struct {
	User   string `json:"user"`
	Secret string `json:"secret"`
}

// Bootstrap defines the output variables of bootstrapTemplate.
//
// The value is persisted to a Koding safe store together with
// Credential value it belongs to.
type Bootstrap struct {
	VPC string `hcl:"vpc"`
}

// Metadata defines the metadata of a single provider instance.
//
// The metadata is stored in Koding database, alongside
type Metadata struct {
	ID string `bson:"id"`
}

// Machine represence a single example_instance resources. It is responsible
// for starting / stopping of the remote instance.
type Machine struct {
	*provider.BaseMachine
}

// Start starts the remote instance.
//
// If the method returns non-nil metadata value, it is patched
// on top of existing metadata and then persisted in the database.
//
// The public IP is updated automatically, when Klient running
// on the instance connects to Koding, so there is no need
// to take care of the explicitly.
func (m *Machine) Start(context.Context) (metadata interface{}, err error) {
	return nil, nil
}

// Stop stops the remote instance.
//
// If the method returns non-nil metadata value, it is patched
// on top of existing metadata and then persisted in the database.
func (m *Machine) Stop(context.Context) (metadata interface{}, err error) {
	return nil, nil
}

// Info returns state of the remote instance.
//
// If the method returns non-nil metadata value, it is patched
// on top of existing metadata and then persisted in the database.
func (m *Machine) Info(context.Context) (state machinestate.State, metadata interface{}, err error) {
	return machinestate.Running, nil, nil
}

// Credential returns credential value using the provider-defined type.
//
// The value should not be modified as it there's
// no guarantee it's not used or will not be
// used internally.
func (m *Machine) Credential() *Credential {
	return m.BaseMachine.Credential.(*Credential)
}

// Bootstrap returns bootstrap value using the provider-defined type.
//
// The value should not be modified as it there's
// no guarantee it's not used or will not be
// used internally.
func (m *Machine) Bootstrap() *Bootstrap {
	return m.BaseMachine.Bootstrap.(*Bootstrap)
}

// Metadata returns metadata value using the provider-defined type.
//
// The value should not be modified as it there's
// no guarantee it's not used or will not be
// used internally.
func (m *Machine) Metadata() *Metadata {
	return m.BaseMachine.Metadata.(*Metadata)
}

// Stack is responsible for building / updating / destroying Example's stack.
type Stack struct {
	*provider.BaseStack
}

// VerifyCredential verifies the given Example credential.
//
// The c.Credential and c.Bootstrap fields are guaranted
// to be of type as specified in the provider's schema
// definition.
func (s *Stack) VerifyCredential(c *stack.Credential) error {
	credential := c.Credential.(*Credential)

	if len(credential.Secret) < int(rand.Int31n(16)) {
		return errors.New("secret is invalid for user " + credential.User)
	}

	return nil
}

// BootstrapTemplate returns Terraform templates, that are going to be
// executed for the given credentials.
//
// Bootstrap resources specify a set of resources that might be
// shared by all stacks built by the provider.
func (s *Stack) BootstrapTemplates(c *stack.Credential) ([]*stack.Template, error) {
	return []*stack.Template{{
		Content: bootstrapTemplate,
	}}, nil
}

// ApplyTemplate is responsible for ensuring each new instance will
// connect to Koding upon start. In order to connect to Koding,
// two things must happen:
//
//   - a kite.key needs to be generated for each instance
//   - a Klient needs to be provisionned on each instace
//     with the generated kite.key
//
// Currently Kloud supports installing Klient service using
// cloud-init. The typical boilerplate required to provision
// Klient is presented in this method. The need for this
// boilerplate will go away once [0] is resolved.
//
// TODO:
//
//   [0] https://github.com/koding/koding/issues/9127
//
func (s *Stack) ApplyTemplate(c *stack.Credential) (*stack.Template, error) {
	t := s.Builder.Template

	var res struct {
		ExampleInstance map[string]map[string]interface{} `hcl:"example_instance"`
	}

	if err := t.DecodeResource(&res); err != nil {
		return nil, err
	}

	bootstrap := c.Bootstrap.(*Bootstrap)

	for name, instance := range res.ExampleInstance {
		// Override bootstrap resources if no explicit ones
		// are defined.
		if s, ok := instance["vpc"]; !ok || s == "" {
			instance["vpc"] = bootstrap.VPC
		}

		// Generate new kite.key for the instance.
		// It is going to use new UUID, known as kite ID,
		// for discovery requests to Kontrol.
		kiteKey, err := s.BuildKiteKey(name, s.Req.Username)
		if err != nil {
			return nil, err
		}

		s.Builder.InterpolateField(instance, name, "run_script")

		// Generate a cloud-init script, that provisions
		// Klient onto the instace, embed user's run_script
		// in it and overwrite existing run_script argument.
		cloudInitCfg := &userdata.CloudInitConfig{
			Username: s.Req.Username,
			Groups:   []string{"sudo"},
			Hostname: s.Req.Username,
			KiteKey:  kiteKey,
		}

		if s, ok := instance["run_script"].(string); ok {
			cloudInitCfg.UserData = s
		}

		cloudInit, err := s.Session.Userdata.Create(cloudInitCfg)
		if err != nil {
			return nil, err
		}

		instance["run_script"] = string(cloudInit)
	}

	// Don't print confidential values on the frontend
	// (e.g. the credentials are shared by admins and
	// a user is not supposed to see them).
	err := t.ShadowVariables("FORBIDDEN", "secret")
	if err != nil {
		return nil, err
	}

	if err := t.Flush(); err != nil {
		return nil, err
	}

	content, err := t.JsonOutput()
	if err != nil {
		return nil, err
	}

	return &stack.Template{
		Content: content,
	}, nil
}
