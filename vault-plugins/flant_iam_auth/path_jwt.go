package jwtauth

import (
	"context"
	"encoding/json"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

const (
	pathJWTEnableSynopsis = `
Enable JWT issuing.
`
	pathJWTEnableDesc = `
After enabling JWT issuing, users will be able to issue tokens with predefined claims and verify them using JWKS.
`
)

func pathJWTEnable(b *jwtAuthBackend) *framework.Path {
	return &framework.Path{
		Pattern: `jwt/enable`,

		Operations: map[logical.Operation]framework.OperationHandler{
			logical.UpdateOperation: &framework.PathOperation{
				Callback: b.handleJWTEnableCreate,
				Summary:  pathJWTEnableSynopsis,
			},
		},

		HelpSynopsis:    pathJWTEnableSynopsis,
		HelpDescription: pathJWTEnableDesc,
	}
}

const pathJWTDisableSynopsis = `
Disable JWT issuing.
`

func pathJWTDisable(b *jwtAuthBackend) *framework.Path {
	return &framework.Path{
		Pattern: `jwt/disable`,

		Operations: map[logical.Operation]framework.OperationHandler{
			logical.UpdateOperation: &framework.PathOperation{
				Callback: b.handleJWTDisableCreate,
				Summary:  pathJWTDisableSynopsis,
			},
		},

		HelpSynopsis: pathJWTDisableSynopsis,
	}
}

func switchJWT(ctx context.Context, req *logical.Request, data interface{}) (*logical.Response, error) {
	entry, err := logical.StorageEntryJSON("jwt/enable", data)
	if err != nil {
		return nil, err
	}
	if err := req.Storage.Put(ctx, entry); err != nil {
		return nil, err
	}

	resp := &logical.Response{
		Data: map[string]interface{}{"enabled": data},
	}

	return resp, nil
}

func (b *jwtAuthBackend) handleJWTEnableCreate(ctx context.Context, req *logical.Request, _ *framework.FieldData) (*logical.Response, error) {
	return switchJWT(ctx, req, true)
}

func (b *jwtAuthBackend) handleJWTDisableCreate(ctx context.Context, req *logical.Request, _ *framework.FieldData) (*logical.Response, error) {
	return switchJWT(ctx, req, false)
}

const (
	pathJWTStatusSynopsis = `
Read JWT issuing status and configuration.
`
	pathJWTConfigureSynopsis = `
Configure JWT options.
`
)

var jwtConfigFields = map[string]*framework.FieldSchema{
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
}

func pathJWTConfigure(b *jwtAuthBackend) *framework.Path {
	return &framework.Path{
		Pattern: `jwt/configure`,

		Fields: jwtConfigFields,

		Operations: map[logical.Operation]framework.OperationHandler{
			logical.ReadOperation: &framework.PathOperation{
				Callback: b.handleJWTStatusRead,
				Summary:  pathJWTStatusSynopsis,
			},
			logical.UpdateOperation: &framework.PathOperation{
				Callback: b.handleJWTConfigurationUpdate,
				Summary:  pathJWTConfigureSynopsis,
			},
		},

		HelpSynopsis: pathJWTConfigureSynopsis,
	}
}

func (b *jwtAuthBackend) handleJWTStatusRead(ctx context.Context, req *logical.Request, _ *framework.FieldData) (*logical.Response, error) {
	entry, err := req.Storage.Get(ctx, "jwt/enable")
	if err != nil {
		return nil, err
	}

	var status bool
	if entry != nil {
		err = entry.DecodeJSON(&status)
		if err != nil {
			return nil, err
		}
	}

	entry, err = req.Storage.Get(ctx, "jwt/configuration")
	if err != nil {
		return nil, err
	}

	var config []byte
	if entry != nil {
		config = entry.Value
	} else {
		rawConfig := make(map[string]interface{})
		for key, field := range jwtConfigFields {
			rawConfig[key] = field.Default
		}

		config, err = json.Marshal(rawConfig)
		if err != nil {
			return nil, err
		}
	}

	data := map[string]interface{}{
		"enabled":       status,
		"configuration": string(config),
	}

	return &logical.Response{Data: data}, nil
}

func (b *jwtAuthBackend) handleJWTConfigurationUpdate(ctx context.Context, req *logical.Request, _ *framework.FieldData) (*logical.Response, error) {
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
