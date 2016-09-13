package example

import (
	"errors"
	"fmt"
	"time"

	"koding/db/mongodb/modelhelper"
	"koding/kites/kloud/provider"
	"koding/kites/kloud/stack"
	"koding/kites/kloud/stackplan"
	"koding/kites/kloud/terraformer"
	"koding/kites/kloud/userdata"
	tf "koding/kites/terraformer"

	multierror "github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform/terraform"
	uuid "github.com/satori/go.uuid"
	"golang.org/x/net/context"
	"gopkg.in/mgo.v2/bson"
)

// Stack
type Stack struct {
	*provider.BaseStack

	p *stackplan.Planner

	ids     stackplan.KiteMap
	klients map[string]*stackplan.DialState
	ident   string
	cred    *Cred
}

var _ stack.Stack = (*Stack)(nil)

// Authenticate
func (s *Stack) Authenticate(context.Context) (interface{}, error) {
	var arg stack.AuthenticateRequest
	if err := s.Req.Args.One().Unmarshal(&arg); err != nil {
		return nil, err
	}

	if err := arg.Valid(); err != nil {
		return nil, err
	}

	if err := s.Builder.BuildCredentials(s.Req.Method, s.Req.Username, arg.GroupName, arg.Identifiers); err != nil {
		return nil, err
	}

	s.Log.Debug("Fetched terraform data: koding=%+v, template=%+v", s.Builder.Koding, s.Builder.Template)

	resp := make(stack.AuthenticateResponse)

	for _, cred := range s.Builder.Credentials {
		// Validate credential.
		resp[cred.Identifier] = &stack.AuthenticateResult{
			Verified: true,
		}

		// Update credential status.
		modelhelper.SetCredentialVerified(cred.Identifier, true)
	}

	return resp, nil
}

// Bootstrap
func (s *Stack) Bootstrap(context.Context) (interface{}, error) {
	var arg stack.BootstrapRequest
	if err := s.Req.Args.One().Unmarshal(&arg); err != nil {
		return nil, err
	}

	// Fetch credentials associated with the stack.
	if err := s.Builder.BuildCredentials(s.Req.Method, s.Req.Username, arg.GroupName, arg.Identifiers); err != nil {
		return nil, err
	}

	var ident string
	for _, cred := range s.Builder.Credentials {
		if cred.Provider != "example" {
			continue
		}

		ident = cred.Identifier
		s.cred = cred.Meta.(*Cred)
	}

	if s.cred == nil {
		return nil, errors.New("no credential found")
	}

	// Prepare bootstrap template.
	bootstrapTemplate := "Example bootstrap template"

	// Apply the template.
	c, err := terraformer.Connect(s.Session.Terraformer)
	if err != nil {
		return nil, err
	}
	defer c.Close()

	state, err := c.Apply(&tf.TerraformRequest{
		Content:   bootstrapTemplate,
		ContentID: fmt.Sprintf("example-%s-%s", arg.GroupName, ident),
		TraceID:   s.TraceID,
	})
	if err != nil {
		return nil, err
	}

	// Update bootstrap data for the credential.
	if err := s.Builder.Object.Decode(state.RootModule().Outputs, s.cred); err != nil {
		return nil, err
	}

	datas := map[string]interface{}{
		ident: s.cred,
	}

	if err := s.Builder.CredStore.Put(s.Req.Username, datas); err != nil {
		return nil, err
	}

	return true, nil
}

// Plan
func (s *Stack) Plan(context.Context) (interface{}, error) {
	var arg stack.PlanRequest
	if err := s.Req.Args.One().Unmarshal(&arg); err != nil {
		return nil, err
	}

	if err := arg.Valid(); err != nil {
		return nil, err
	}

	// Fetch and build template.
	stackTemplate, err := modelhelper.GetStackTemplate(arg.StackTemplateID)
	if err != nil {
		return nil, stackplan.ResError(err, "jStackTemplate")
	}

	c, err := terraformer.Connect(s.Session.Terraformer)
	if err != nil {
		return nil, err
	}
	defer c.Close()

	contentID := s.Req.Username + "-" + arg.StackTemplateID

	if err := s.Builder.BuildTemplate(stackTemplate.Template.Content, contentID); err != nil {
		return nil, err
	}

	if err := s.Builder.Template.FillVariables("userInput_"); err != nil {
		return nil, err
	}

	if err := s.Builder.Template.FillVariables("example_"); err != nil {
		return nil, err
	}

	out, err := s.Builder.Template.JsonOutput()
	if err != nil {
		return nil, err
	}

	// Call plan on the template.
	tfReq := &tf.TerraformRequest{
		Content:   out,
		ContentID: contentID,
		TraceID:   s.TraceID,
	}

	plan, err := c.Plan(tfReq)
	if err != nil {
		return nil, err
	}

	// Create machines definition from plan response.
	machines, err := s.p.MachinesFromPlan(plan)
	if err != nil {
		return nil, err
	}

	return &stack.PlanResponse{
		Machines: machines.Slice(),
	}, nil
}

// BuildResources
func (s *Stack) BuildResources() error {
	t := s.Builder.Template

	for _, cred := range s.Builder.Credentials {
		if cred.Provider != "example" {
			continue
		}

		s.cred = cred.Meta.(*Cred)
	}

	if s.cred == nil {
		return errors.New("no credential found")
	}

	var resource struct {
		ExampleInstances map[string]map[string]interface{} `hcl:"example_instance"`
	}

	if err := t.DecodeResource(&resource); err != nil {
		return err
	}

	for name, instance := range resource.ExampleInstances {
		// Inject bootstrap data.
		if id, ok := instance["bootstrapID"]; !ok || id == "" {
			instance["bootstrapID"] = s.cred.BootstrapID
		}

		// Inject cloud-init data.
		cfg := &userdata.CloudInitConfig{
			Username: s.Req.Username,
			Groups:   []string{"sudo"},
			Hostname: s.Req.Username,
			KiteId:   uuid.NewV4().String(),
		}

		// Cache kite ID for connectivity check - WaitResources.
		s.ids[name] = cfg.KiteId

		var err error
		cfg.KiteKey, err = s.Session.Userdata.Keycreator.Create(s.Req.Username, cfg.KiteId)
		if err != nil {
			return err
		}

		if s, ok := instance["example_data"].(string); ok {
			cfg.UserData = s
		}

		s.Builder.InterpolateField(instance, name, "example_data")

		cloudInit, err := s.Session.Userdata.Create(cfg)
		if err != nil {
			return err
		}

		instance["example_data"] = string(cloudInit)

		// Make it possible to use variables within example_data field.
		s.Builder.InterpolateField(instance, name, "example_data")
	}

	// Update terraform template.
	return t.Flush()
}

// WaitResources
func (s *Stack) WaitResources(ctx context.Context) error {
	var err error

	s.klients, err = s.p.DialKlients(ctx, s.ids)

	return err
}

// UpdateResources
func (s *Stack) UpdateResources(state *terraform.State) error {
	machines, err := s.p.MachinesFromState(state, s.klients)
	if err != nil {
		return err
	}

	now := time.Now()

	for label, m := range s.Builder.Machines {
		machine, ok := machines[label]
		if !ok {
			err = multierror.Append(err, fmt.Errorf("machine %q does not exist in terraform state file", label))
			continue
		}

		if machine.Provider != "example" {
			continue
		}

		e := modelhelper.UpdateMachine(m.ObjectId, bson.M{"$set": bson.M{
			"credential":        s.ident,
			"provider":          machine.Provider,
			"queryString":       machine.QueryString,
			"ipAddress":         machine.Attributes["public_ip"],
			"status.modifiedAt": now,
			"status.state":      machine.State.String(),
			"status.reason":     machine.StateReason,
			"meta.exampleID":    machine.Attributes["example_id"],
		}})

		if e != nil {
			err = multierror.Append(err, e)
			continue
		}
	}

	return err
}
