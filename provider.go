package example

import (
	"errors"

	"koding/db/mongodb/modelhelper"
	"koding/kites/kloud/provider"
	"koding/kites/kloud/stack"
	"koding/kites/kloud/stackplan"

	"golang.org/x/net/context"
)

// Cred
type Cred struct {
	AccessKey string `json:"accessKey"` // access key to access example API endpoint
	SecretKey string `json:"secretKey"` // secret key to access example API endpoint

	// BootstrapID
	BootstrapID string `json:"bootstrapID"`
}

var _ stack.Validator = (*Cred)(nil)

// Valid
func (c *Cred) Valid() error {
	if c.AccessKey == "" {
		return errors.New("access key is empty")
	}

	if c.SecretKey == "" {
		return errors.New("secret key is empty")
	}

	return nil
}

// Provider
type Provider struct {
	*provider.BaseProvider
}

var _ stack.Provider = (*Provider)(nil)

// Stack
func (p *Provider) Stack(ctx context.Context) (stack.Stack, error) {
	bs, err := p.BaseStack(ctx)
	if err != nil {
		return nil, err
	}

	s := &Stack{
		BaseStack: bs,
		p: &stackplan.Planner{
			Provider:     "example",
			ResourceType: "instance",
		},
	}

	bs.BuildResources = s.BuildResources
	bs.WaitResources = s.WaitResources
	bs.UpdateResources = s.UpdateResources

	return s, nil
}

// Machine
func (p *Provider) Machine(ctx context.Context, id string) (stack.Machine, error) {
	bm, err := p.BaseMachine(ctx, id)
	if err != nil {
		return nil, err
	}

	var mt Meta
	if err := modelhelper.BsonDecode(bm.Meta, &mt); err != nil {
		return nil, err
	}

	if err := mt.Valid(); err != nil {
		return nil, err
	}

	var cred Cred
	if err := p.FetchCredData(bm, &cred); err != nil {
		return nil, err
	}

	return &Machine{
		BaseMachine: bm,
		Meta:        &mt,
		Cred:        &cred,
	}, nil
}

// Cred
func (p *Provider) Cred() interface{} {
	return &Cred{}
}
