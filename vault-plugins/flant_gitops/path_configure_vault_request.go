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
	fieldNameVaultRequestName    = "name"
	fieldNameVaultRequestPath    = "path"
	fieldNameVaultRequestMethod  = "method"
	fieldNameVaultRequestOptions = "options"
	fieldNameVaultRequestWrapTTL = "wrap_ttl"

	wrapTTLMinSec = 60
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
			Pattern: "^configure/vault_request/" + framework.GenericNameRegex(fieldNameVaultRequestName) + "/?$",
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
					Description: "GET, POST, LIST, PUT or DELETE",
					Required:    true,
				},
				fieldNameVaultRequestOptions: {
					Type:        framework.TypeString,
					Description: "Optional JSON encoded string with data for the request",
				},
				fieldNameVaultRequestWrapTTL: {
					Type:        framework.TypeString,
					Description: fmt.Sprintf("Vault response wrap TTL specified as golang duration string (https://golang.org/pkg/time/#ParseDuration). Minimum: %ds", wrapTTLMinSec),
					Default:     "1m",
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.CreateOperation: &framework.PathOperation{
					Callback: b.pathConfigureVaultRequestCreateOrUpdate,
				},
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.pathConfigureVaultRequestCreateOrUpdate,
				},
			},
		},
		{
			Pattern: "^configure/vault_request/" + framework.GenericNameRegex(fieldNameVaultRequestName) + "/?$",
			Fields: map[string]*framework.FieldSchema{
				fieldNameVaultRequestName: {
					Type:        framework.TypeNameString,
					Description: "Name this request so that request result will be available in the $VAULT_REQUEST_TOKEN_$name variable",
					Required:    true,
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.ReadOperation: &framework.PathOperation{
					Callback: b.pathConfigureVaultRequestRead,
				},
				logical.DeleteOperation: &framework.PathOperation{
					Callback: b.pathConfigureVaultRequestDelete,
				},
			},
		},
	}
}

func (b *backend) pathConfigureVaultRequestCreateOrUpdate(ctx context.Context, req *logical.Request, fields *framework.FieldData) (*logical.Response, error) {
	b.Logger().Debug("Start creating vault request configuration ...")

	resp, err := util.ValidateRequestFields(req, fields)
	if resp != nil || err != nil {
		return resp, err
	}

	name := req.Get(fieldNameVaultRequestName).(string)
	path := req.Get(fieldNameVaultRequestPath).(string)

	method := req.Get(fieldNameVaultRequestMethod).(string)
	switch method {
	case "GET", "POST", "LIST", "PUT", "DELETE":
	default:
		return logical.ErrorResponse("invalid option %q given: expected GET, POST, LIST, PUT or DELETE", fieldNameVaultRequestMethod), nil
	}

	var options string
	if v := req.Get(fieldNameVaultRequestOptions); v != nil {
		options = v.(string)

		var data interface{}
		if err := json.Unmarshal([]byte(options), &data); err != nil {
			return logical.ErrorResponse("invalid option %q given: expected json: %s", fieldNameVaultRequestOptions, err), nil
		}
	}

	var wrapTTL time.Duration
	if wrapTTLRaw := req.Get(fieldNameVaultRequestWrapTTL); wrapTTLRaw != nil {
		wrapTTL, err := time.ParseDuration(wrapTTLRaw.(string))
		if err != nil {
			return logical.ErrorResponse("invalid option %q given, expected golang time duration: %s", fieldNameVaultRequestWrapTTL, err), nil
		}

		if wrapTTL.Seconds() < wrapTTLMinSec {
			return logical.ErrorResponse("Can't set %q for Vault request as %.0f, minimum value of %qs required", fieldNameVaultRequestWrapTTL, wrapTTL.Seconds(), wrapTTLMinSec), nil
		}
	}

	vaultRequest := vaultRequest{
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

	if err := putVaultRequest(ctx, req.Storage, vaultRequest); err != nil {
		return logical.ErrorResponse("Can't put Vault request %q into storage: %s", name, err), nil
	}

	return nil, nil
}

func (b *backend) pathConfigureVaultRequestRead(ctx context.Context, req *logical.Request, fields *framework.FieldData) (*logical.Response, error) {
	b.Logger().Debug("Start reading vault request configuration ...")

	// TODO: return only error instead?
	resp, err := util.ValidateRequestFields(req, fields)
	if resp != nil || err != nil {
		return resp, err
	}

	vaultRequestName := req.Get(fieldNameVaultRequestName).(string)

	vaultRequest, err := getVaultRequest(ctx, req.Storage, vaultRequestName)
	if err != nil {
		return logical.ErrorResponse("Unable to Read Vault request %q configuration: %s", vaultRequestName, err), nil
	}
	if vaultRequest == nil {
		return logical.ErrorResponse("Vault request %q not found", vaultRequestName), nil
	}

	return &logical.Response{
		Data: map[string]interface{}{
			fieldNameVaultRequestName:    vaultRequest.Name,
			fieldNameVaultRequestPath:    vaultRequest.Path,
			fieldNameVaultRequestMethod:  vaultRequest.Method,
			fieldNameVaultRequestOptions: vaultRequest.Options,
			fieldNameVaultRequestWrapTTL: vaultRequest.WrapTTL.String(),
		},
	}, nil
}

func (b *backend) pathConfigureVaultRequestList(ctx context.Context, req *logical.Request, _ *framework.FieldData) (*logical.Response, error) {
	b.Logger().Debug("Start listing vault request configuration ...")

	var keys []string
	keysInfo := map[string]interface{}{}

	allVaultRequests, err := listVaultRequests(ctx, req.Storage)
	if err != nil {
		return logical.ErrorResponse("Unable to List all Vault requests configurations: %s", err), nil
	}
	if len(allVaultRequests) == 0 {
		return logical.ListResponseWithInfo(keys, keysInfo), nil
	}

	for _, vaultRequest := range allVaultRequests {
		keys = append(keys, vaultRequest.Name)
		keysInfo[vaultRequest.Name] = map[string]interface{}{
			fieldNameVaultRequestName:    vaultRequest.Name,
			fieldNameVaultRequestPath:    vaultRequest.Path,
			fieldNameVaultRequestMethod:  vaultRequest.Method,
			fieldNameVaultRequestOptions: vaultRequest.Options,
			fieldNameVaultRequestWrapTTL: vaultRequest.WrapTTL.String(),
		}
	}

	return logical.ListResponseWithInfo(keys, keysInfo), nil
}

func (b *backend) pathConfigureVaultRequestDelete(ctx context.Context, req *logical.Request, fields *framework.FieldData) (*logical.Response, error) {
	b.Logger().Debug("Start deleting vault request configuration ...")

	// TODO: return only error instead?
	resp, err := util.ValidateRequestFields(req, fields)
	if resp != nil || err != nil {
		return resp, err
	}

	vaultRequestName := req.Get(fieldNameVaultRequestName).(string)

	if err := deleteVaultRequest(ctx, req.Storage, vaultRequestName); err != nil {
		return logical.ErrorResponse("Unable to Delete Vault request %q: %s", vaultRequestName, err), nil
	}

	return nil, nil
}
