package jwt

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

func PathConfigure(b *TokenController) *framework.Path {
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
			"own_audience": {
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
				Callback: protectNonEnabled(b.handleConfigurationRead),
				Summary:  pathJWTStatusSynopsis,
			},
			logical.UpdateOperation: &framework.PathOperation{
				Callback: protectNonEnabled(b.handleConfigurationUpdate),
				Summary:  pathJWTConfigureSynopsis,
			},
		},

		HelpSynopsis: pathJWTConfigureSynopsis,
	}
}

func getConfig(ctx context.Context, storage logical.Storage) (map[string]interface{}, error) {
	entry, err := storage.Get(ctx, "jwt/configuration")
	if err != nil {
		return nil, err
	}

	data := make(map[string]interface{})
	if entry != nil {
		if err := json.Unmarshal(entry.Value, &data); err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("possible bug: no configuration found in storage")
	}

	return data, nil
}

func (b *TokenController) handleConfigurationRead(ctx context.Context, req *logical.Request, _ *framework.FieldData) (*logical.Response, error) {
	data, err := getConfig(ctx, req.Storage)
	if err != nil {
		return nil, err
	}

	return &logical.Response{Data: data}, nil
}

func (b *TokenController) handleConfigurationUpdate(ctx context.Context, req *logical.Request, _ *framework.FieldData) (*logical.Response, error) {
	entry, err := logical.StorageEntryJSON("jwt/configuration", req.Data)
	if err != nil {
		return nil, err
	}

	if err := req.Storage.Put(ctx, entry); err != nil {
		return nil, err
	}

	resp := &logical.Response{
		Data: req.Data,
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
