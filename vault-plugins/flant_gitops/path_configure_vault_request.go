package flant_gitops

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/fatih/structs"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

const (
	fieldNameVaultRequestName    = "name"
	fieldNameVaultRequestPath    = "path"
	fieldNameVaultRequestMethod  = "method"
	fieldNameVaultRequestOptions = "options"
	fieldNameVaultRequestWrapTTL = "wrap_ttl"

	storageKeyVaultRequestPrefix = "vault_request/"

	vaultRequestWrapTTLMinSec = 60
)

type vaultRequests []*vaultRequest

type vaultRequest struct {
	Name    string                 `structs:"name" json:"name"`
	Path    string                 `structs:"path" json:"path"`
	Method  string                 `structs:"method" json:"method"`
	Options map[string]interface{} `structs:"options" json:"options"`
	WrapTTL time.Duration          `structs:"wrap_ttl" json:"wrap_ttl"`
}

func configureVaultRequestPaths(b *backend) []*framework.Path {
	return []*framework.Path{
		{
			Pattern: "^configure/vault_request/?$",
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.ListOperation: &framework.PathOperation{
					Callback: b.pathConfigureVaultRequestList,
					Summary:  "List currently configured flant_gitops backend vault requests",
				},
			},

			HelpSynopsis:    configureVaultRequestHelpSyn,
			HelpDescription: configureVaultRequestHelpDesc,
		},
		{
			Pattern: "^configure/vault_request/" + framework.GenericNameRegex(fieldNameVaultRequestName) + "/?$",
			Fields: map[string]*framework.FieldSchema{
				fieldNameVaultRequestName: {
					Type:        framework.TypeNameString,
					Description: "On the name of this request depends the name of the $VAULT_REQUEST_TOKEN_$name variable, which is passed to the Docker container. Required.",
				},
				fieldNameVaultRequestPath: {
					Type:        framework.TypeString,
					Description: `URL path of request, which should start with "/". Required for CREATE, UPDATE.`,
				},
				fieldNameVaultRequestMethod: {
					Type:          framework.TypeString,
					Default:       "GET",
					AllowedValues: []interface{}{"GET", "POST", "LIST", "PUT", "DELETE"},
					Description:   "HTTP method of the request",
				},
				fieldNameVaultRequestOptions: {
					Type:        framework.TypeMap,
					Description: fmt.Sprintf("Optional JSON data to be send with the request. Should be passed in JSON data under the key %q", fieldNameVaultRequestOptions),
				},
				fieldNameVaultRequestWrapTTL: {
					Type:        framework.TypeDurationSecond,
					Default:     "1m",
					Description: fmt.Sprintf("Vault response wrap TTL specified as golang duration string (https://golang.org/pkg/time/#ParseDuration). Minimum: %ds", vaultRequestWrapTTLMinSec),
				},
			},

			Operations: map[logical.Operation]framework.OperationHandler{
				logical.CreateOperation: &framework.PathOperation{
					Callback: b.pathConfigureVaultRequestCreateOrUpdate,
					Summary:  "Append new vault request configuration",
				},
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.pathConfigureVaultRequestCreateOrUpdate,
					Summary:  "Update existing vault request configuration by the name",
				},
				logical.ReadOperation: &framework.PathOperation{
					Callback: b.pathConfigureVaultRequestRead,
					Summary:  "Read existing vault request configuration by the name",
				},
				logical.DeleteOperation: &framework.PathOperation{
					Callback: b.pathConfigureVaultRequestDelete,
					Summary:  "Delete existing vault request configuration by the name",
				},
			},

			HelpSynopsis:    configureVaultRequestHelpSyn,
			HelpDescription: configureVaultRequestHelpDesc,
		},
	}
}

func (b *backend) pathConfigureVaultRequestCreateOrUpdate(ctx context.Context, req *logical.Request, fields *framework.FieldData) (*logical.Response, error) {
	vaultRequestName := fields.Get(fieldNameVaultRequestName).(string)

	b.Logger().Debug(fmt.Sprintf("%q Vault request configuration started...", vaultRequestName))

	vaultRequest := vaultRequest{
		Name:    vaultRequestName,
		Path:    fields.Get(fieldNameVaultRequestPath).(string),
		Method:  fields.Get(fieldNameVaultRequestMethod).(string),
		Options: fields.Get(fieldNameVaultRequestOptions).(map[string]interface{}),
		WrapTTL: time.Duration(fields.Get(fieldNameVaultRequestWrapTTL).(int)) * time.Second,
	}

	if !strings.HasPrefix(vaultRequest.Path, "/") {
		return logical.ErrorResponse(`%q field value must begin with "/", got: %s`, fieldNameVaultRequestPath, vaultRequest.Path), nil
	}

	switch vaultRequest.Method {
	case "GET", "POST", "LIST", "PUT", "DELETE":
	default:
		return logical.ErrorResponse("%q field value must be one of GET, POST, LIST, PUT or DELETE, got: %s", fieldNameVaultRequestMethod, vaultRequest.Method), nil
	}

	if vaultRequest.WrapTTL.Seconds() < vaultRequestWrapTTLMinSec {
		return logical.ErrorResponse("%q field value must be no less than %ds, got: %.0fs", fieldNameVaultRequestWrapTTL, vaultRequestWrapTTLMinSec, vaultRequest.WrapTTL.Seconds()), nil
	}

	{
		cfgData, cfgErr := json.MarshalIndent(vaultRequest, "", "  ")
		b.Logger().Debug(fmt.Sprintf("Got Vault request configuration (err=%v):\n%s", cfgErr, string(cfgData)))
	}

	if err := putVaultRequest(ctx, req.Storage, vaultRequest); err != nil {
		return logical.ErrorResponse("Unable to put %q Vault request into storage: %s", vaultRequest.Name, err), nil
	}

	return nil, nil
}

func (b *backend) pathConfigureVaultRequestRead(ctx context.Context, req *logical.Request, fields *framework.FieldData) (*logical.Response, error) {
	vaultRequestName := fields.Get(fieldNameVaultRequestName).(string)

	b.Logger().Debug(fmt.Sprintf("Getting %q Vault request configuration...", vaultRequestName))

	vaultRequest, err := getVaultRequest(ctx, req.Storage, vaultRequestName)
	if err != nil {
		return logical.ErrorResponse("Unable to get %q Vault request configuration: %s", vaultRequestName, err), nil
	}
	if vaultRequest == nil {
		return nil, nil
	}

	return &logical.Response{Data: vaultRequestStructToMap(vaultRequest)}, nil
}

func (b *backend) pathConfigureVaultRequestList(ctx context.Context, req *logical.Request, _ *framework.FieldData) (*logical.Response, error) {
	b.Logger().Debug("Listing all Vault requests configurations...")

	var keys []string
	keysInfo := map[string]interface{}{}

	allVaultRequests, err := listVaultRequests(ctx, req.Storage)
	if err != nil {
		return logical.ErrorResponse("Unable to get all Vault requests configurations: %s", err), nil
	}
	if len(allVaultRequests) == 0 {
		return nil, nil
	}

	for _, vaultRequest := range allVaultRequests {
		keys = append(keys, vaultRequest.Name)
		keysInfo[vaultRequest.Name] = vaultRequestStructToMap(vaultRequest)
	}

	return logical.ListResponseWithInfo(keys, keysInfo), nil
}

func (b *backend) pathConfigureVaultRequestDelete(ctx context.Context, req *logical.Request, fields *framework.FieldData) (*logical.Response, error) {
	vaultRequestName := fields.Get(fieldNameVaultRequestName).(string)

	b.Logger().Debug(fmt.Sprintf("Deleting %q Vault request configuration...", vaultRequestName))

	if err := deleteVaultRequest(ctx, req.Storage, vaultRequestName); err != nil {
		return logical.ErrorResponse("Unable to delete %q Vault request: %s", vaultRequestName, err), nil
	}

	return nil, nil
}

func putVaultRequest(ctx context.Context, storage logical.Storage, vaultRequest vaultRequest) error {
	storageEntry, err := logical.StorageEntryJSON(getAbsStoragePathToVaultRequest(vaultRequest.Name), vaultRequest)
	if err != nil {
		return err
	}

	if err := storage.Put(ctx, storageEntry); err != nil {
		return err
	}

	return nil
}

func getVaultRequest(ctx context.Context, storage logical.Storage, vaultRequestName string) (*vaultRequest, error) {
	storageEntry, err := storage.Get(ctx, getAbsStoragePathToVaultRequest(vaultRequestName))
	if err != nil {
		return nil, err
	}
	if storageEntry == nil {
		return nil, nil
	}

	var request *vaultRequest
	if err := storageEntry.DecodeJSON(&request); err != nil {
		return nil, err
	}

	return request, nil
}

func listVaultRequests(ctx context.Context, storage logical.Storage) (vaultRequests, error) {
	requestNames, err := storage.List(ctx, storageKeyVaultRequestPrefix)
	if err != nil {
		return nil, err
	}
	if len(requestNames) == 0 {
		return nil, nil
	}

	var requests vaultRequests
	for _, requestName := range requestNames {
		request, err := getVaultRequest(ctx, storage, requestName)
		if err != nil {
			return nil, err
		}
		if request == nil {
			continue
		}
		requests = append(requests, request)
	}

	return requests, nil
}

func deleteVaultRequest(ctx context.Context, storage logical.Storage, vaultRequestName string) error {
	return storage.Delete(ctx, getAbsStoragePathToVaultRequest(vaultRequestName))
}

func getAbsStoragePathToVaultRequest(vaultRequestName string) string {
	return path.Join(storageKeyVaultRequestPrefix, vaultRequestName)
}

func vaultRequestStructToMap(vaultRequest *vaultRequest) map[string]interface{} {
	data := structs.Map(vaultRequest)
	data[fieldNameVaultRequestWrapTTL] = vaultRequest.WrapTTL.Seconds()

	return data
}

const (
	configureVaultRequestHelpSyn = `
Supplement configuration of the flant_gitops plugin to perform optional requests
`
	configureVaultRequestHelpDesc = `
The flant_gitops periodic function will perform configured vault requests before running
periodic command. All provided requests will be executed in the wrapped form, meaning that 
the result of each request will be available by the token in the container written into
the special variable $VAULT_REQUEST_TOKEN_<REQUEST_NAME>. These tokens can then be unwrapped
from inside the container using provided vault connection settings.
`
)
