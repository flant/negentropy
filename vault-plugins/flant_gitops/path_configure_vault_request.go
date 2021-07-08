package flant_gitops

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/werf/vault-plugin-secrets-trdl/pkg/util"
)

const (
	storageKeyVaultRequestPrefix = "vault_request/"

	fieldNameVaultRequestName    = "name"
	fieldNameVaultRequestPath    = "path"
	fieldNameVaultRequestMethod  = "method"
	fieldNameVaultRequestOptions = "options"
	fieldNameVaultRequestWrapTTL = "wrap_ttl"
)

func configureVaultRequestPaths(b *backend) []*framework.Path {
	return []*framework.Path{
		{
			Pattern: "^configure/vault_request/?$",
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.ListOperation: &framework.PathOperation{
					Callback: b.pathConfigureVaultRequestList,
				},
			},
		},
		{
			Pattern: "^configure/vault_request/" + framework.GenericNameRegex("name") + "$",
			Fields: map[string]*framework.FieldSchema{
				fieldNameVaultRequestName: {
					Type:        framework.TypeNameString,
					Description: "Name this request so that request result will be available in the $VAULT_REQUEST_TOKEN_$name variable",
					Required:    true,
				},
				fieldNameVaultRequestPath: {
					Type:        framework.TypeString,
					Description: "URL path of request",
					Required:    true,
				},
				fieldNameVaultRequestMethod: {
					Type:        framework.TypeString,
					Description: "GET, POST, LIST or PUT",
					Required:    true,
				},
				fieldNameVaultRequestOptions: {
					Type:        framework.TypeString,
					Description: "JSON encoded string with parameters for a request",
				},
				fieldNameVaultRequestWrapTTL: {
					Type:        framework.TypeString,
					Description: "Vault response wrap TTL specified as golang duration string (https://golang.org/pkg/time/#ParseDuration)",
					Default:     "1m",
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.CreateOperation: &framework.PathOperation{
					Callback: b.pathConfigureVaultRequestCreate,
				},
				logical.ReadOperation: &framework.PathOperation{
					Callback: b.pathConfigureVaultRequestRead,
				},
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.pathConfigureVaultRequestUpdate,
				},
				logical.DeleteOperation: &framework.PathOperation{
					Callback: b.pathConfigureVaultRequestDelete,
				},
			},
		},
	}
}

func (b *backend) pathConfigureVaultRequestList(ctx context.Context, req *logical.Request, fields *framework.FieldData) (*logical.Response, error) {
	b.Logger().Debug("Start listing vault request configuration ...")

	// TODO

	return nil, nil
}

func (b *backend) pathConfigureVaultRequestCreate(ctx context.Context, req *logical.Request, fields *framework.FieldData) (*logical.Response, error) {
	b.Logger().Debug("Start creating vault request configuration ...")

	resp, err := util.ValidateRequestFields(req, fields)
	if resp != nil || err != nil {
		return resp, err
	}

	name := req.Get(fieldNameVaultRequestName).(string)
	path := req.Get(fieldNameVaultRequestPath).(string)

	method := req.Get(fieldNameVaultRequestMethod).(string)
	switch method {
	case "GET", "POST", "LIST", "PUT":
	default:
		return logical.ErrorResponse(fmt.Sprintf("invalid %s given: expected GET, POST, LIST or PUT", fieldNameVaultRequestMethod)), nil
	}

	var options string
	if v := req.Get(fieldNameVaultRequestOptions); v != nil {
		options = v.(string)

		var data interface{}
		if err := json.Unmarshal([]byte(options), data); err != nil {
			return logical.ErrorResponse(fmt.Sprintf("invalid %s given: expected json: %s", fieldNameVaultRequestOptions, err)), nil
		}
	}

	var wrapTTL string
	if v := req.Get(fieldNameVaultRequestWrapTTL); v != nil {
		wrapTTL = v.(string)

		if _, err := time.ParseDuration(wrapTTL); err != nil {
			return logical.ErrorResponse(fmt.Sprintf("invalid %s given, expected golang time duration: %s", wrapTTL, err)), nil
		}
	}

	vaultRequest := &vaultRequest{
		Name:    name,
		Path:    path,
		Method:  method,
		Options: options,
		WrapTTL: wrapTTL,
	}

	{
		cfgData, cfgErr := json.MarshalIndent(vaultRequest, "", "  ")
		b.Logger().Debug(fmt.Sprintf("Got vault request configuration (err=%v):\n%s", cfgErr, string(cfgData)))
	}

	vaultRequestsConfig, err := getVaultRequests(ctx, req.Storage)
	if err != nil {
		return nil, fmt.Errorf("error getting existing vault requests from the storage: %s", err)
	}

	vaultRequestsConfig = append(vaultRequestsConfig, vaultRequest)

	if err := putVaultRequests(ctx, req.Storage, vaultRequestsConfig); err != nil {
		return nil, fmt.Errorf("error putting updated vault requests into the storage: %s", err)
	}

	return nil, nil
}

func (b *backend) pathConfigureVaultRequestUpdate(ctx context.Context, req *logical.Request, fields *framework.FieldData) (*logical.Response, error) {
	b.Logger().Debug("Start updating vault request configuration ...")

	// TODO
	return nil, nil
}

func (b *backend) pathConfigureVaultRequestRead(ctx context.Context, req *logical.Request, fields *framework.FieldData) (*logical.Response, error) {
	b.Logger().Debug("Start reading vault request configuration ...")

	// TODO
	return nil, nil
}

func (b *backend) pathConfigureVaultRequestDelete(ctx context.Context, req *logical.Request, fields *framework.FieldData) (*logical.Response, error) {
	b.Logger().Debug("Start deleting vault request configuration ...")

	// TODO
	return nil, nil
}
