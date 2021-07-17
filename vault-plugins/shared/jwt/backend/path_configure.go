package backend

import (
	"context"
	"fmt"
	"github.com/flant/negentropy/vault-plugins/shared/jwt/model"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
	"time"
)

func PathConfigure(b *Backend) *framework.Path {
	return &framework.Path{
		Pattern: `jwt/configure`,

		Fields: map[string]*framework.FieldSchema{
			"issuer": {
				Type: framework.TypeString,
				Description: `Issuer URL to be used in the iss claim of the token. 
The issuer is a case sensitive URL using the https scheme that contains scheme, 
host, and optionally, port number and path components, but no query or fragment components.`,
				Default:  "https://auth.negentropy.flant.com/",
				Required: true,
			},
			"multipass_audience": {
				Type:        framework.TypeString,
				Description: "Value of the audience claim.",
				Default:     "limbo",
				Required:    true,
			},
			"rotation_period": {
				Type:        framework.TypeDurationSecond,
				Description: "Force rotate public/private key pair.",
				Default:     "1d",
				Required:    true,
			},
			"preliminary_announce_period": {
				Type:        framework.TypeDurationSecond,
				Description: "Publish the key in advance after specified amount of time.",
				Default:     "1d",
				Required:    true,
			},
		},

		Operations: map[logical.Operation]framework.OperationHandler{
			logical.ReadOperation: &framework.PathOperation{
				Callback: b.handleConfigurationRead,
				Summary:  pathJWTStatusSynopsis,
			},
			logical.UpdateOperation: &framework.PathOperation{
				Callback: b.handleConfigurationUpdate,
				Summary:  pathJWTConfigureSynopsis,
			},
		},

		HelpSynopsis: pathJWTConfigureSynopsis,
	}
}

func (b *Backend) handleConfigurationRead(ctx context.Context, req *logical.Request, _ *framework.FieldData) (*logical.Response, error) {
	tnx := b.memStorage.Txn(false)
	defer tnx.Abort()

	err := b.mustEnabled(tnx)
	if err != nil {
		return logical.ErrorResponse(err.Error()), nil
	}

	conf, err := b.deps.ConfigRepo(tnx).Get()
	if err != nil {
		return nil, err
	}

	d := map[string]interface{}{
		"issuer": conf.Issuer,
		"multipass_audience": conf.OwnAudience,
		"rotation_period":  conf.RotationPeriod.String(),
		"preliminary_announce_period": conf.PreliminaryAnnouncePeriod.String(),
	}


	return &logical.Response{Data: d}, nil
}

func (b *Backend) handleConfigurationUpdate(ctx context.Context, req *logical.Request, fields *framework.FieldData) (*logical.Response, error) {
	tnx := b.memStorage.Txn(true)
	defer tnx.Abort()

	err := b.mustEnabled(tnx)
	if err != nil {
		return logical.ErrorResponse(err.Error()), nil
	}

	fields.Raw = req.Data
	err = fields.Validate()
	if err != nil {
		return nil, err
	}

	if fields.Raw == nil {
		return nil, fmt.Errorf("cannot update configuration because values were not provided")
	}

	c := model.Config{}

	rotate, err := time.ParseDuration(fields.Raw["rotation_period"].(string))
	if err != nil {
		return logical.ErrorResponse("incorrect rotation_period", err), nil
	}

	announce, err := time.ParseDuration(fields.Raw["preliminary_announce_period"].(string))
	if err != nil {
		return logical.ErrorResponse("incorrect preliminary_announce_period", err), nil
	}

	c.RotationPeriod = rotate
	c.PreliminaryAnnouncePeriod = announce
	c.Issuer = fields.Raw["issuer"].(string)
	c.OwnAudience = fields.Raw["multipass_audience"].(string)

	err = b.deps.ConfigRepo(tnx).Put(&c)
	if err != nil {
		return nil, err
	}

	resp := &logical.Response{
		Data: req.Data,
	}

	if err := tnx.Commit(); err != nil {
		return nil, err
	}

	return resp, nil
}

const (
	pathJWTStatusSynopsis = `
Read JWT issuing status and configuration.
`
	pathJWTConfigureSynopsis = `
Configure JWT options.
`
)
