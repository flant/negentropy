package flant_gitops

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/werf/vault-plugin-secrets-trdl/pkg/docker"
)

const (
	fieldNameGitRepoUrl                                 = "git_repo_url"
	fieldNameGitBranch                                  = "git_branch_name"
	fieldNameGitPollPeriod                              = "git_poll_period"
	fieldNameRequiredNumberOfVerifiedSignaturesOnCommit = "required_number_of_verified_signatures_on_commit"
	fieldNameInitialLastSuccessfulCommit                = "initial_last_successful_commit"
	fieldNameDockerImage                                = "docker_image"
	fieldNameCommand                                    = "command"

	storageKeyConfiguration             = "configuration"
	storageKeyLastSuccessfulCommit      = "last_successful_commit"
	storageKeyPrefixTrustedGPGPublicKey = "trusted_gpg_public_key-"

	pathPatternConfigure = "^configure/?$"
)

func configurePaths(b *backend) []*framework.Path {
	return []*framework.Path{
		{
			Pattern: pathPatternConfigure,
			Fields: map[string]*framework.FieldSchema{
				fieldNameGitRepoUrl: {
					Type:     framework.TypeString,
					Required: true,
				},
				fieldNameGitBranch: {
					Type:     framework.TypeString,
					Required: true,
				},
				fieldNameGitPollPeriod: {
					Type:    framework.TypeDurationSecond,
					Default: "5m",
				},
				fieldNameRequiredNumberOfVerifiedSignaturesOnCommit: {
					Type:     framework.TypeInt,
					Required: true,
				},
				fieldNameInitialLastSuccessfulCommit: {
					Type: framework.TypeString,
				},
				fieldNameDockerImage: {
					Type:     framework.TypeString,
					Required: true,
				},
				fieldNameCommand: {
					Type:     framework.TypeString,
					Required: true,
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.pathConfigure,
				},
			},
		},
		{
			Pattern: "configure/trusted_gpg_public_key",
			Fields: map[string]*framework.FieldSchema{
				"name": {
					Type:     framework.TypeNameString,
					Required: true,
				},
				"public_key": {
					Type:     framework.TypeString,
					Required: true,
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.CreateOperation: &framework.PathOperation{
					Callback: b.pathTrustedGPGPublicKeyCreate,
				},
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.pathTrustedGPGPublicKeyCreate,
				},
			},
		},
		{
			Pattern: "configure/trusted_gpg_public_key/?",
			Fields:  map[string]*framework.FieldSchema{},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.ListOperation: &framework.PathOperation{
					Callback: b.pathTrustedGPGPublicKeyList,
				},
			},
		},
		{
			Pattern: "configure/trusted_gpg_public_key/" + framework.GenericNameRegex("name") + "$",
			Fields: map[string]*framework.FieldSchema{
				"name": {
					Type:     framework.TypeNameString,
					Required: true,
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.ReadOperation: &framework.PathOperation{
					Callback: b.pathTrustedGPGPublicKeyRead,
				},
				logical.DeleteOperation: &framework.PathOperation{
					Callback: b.pathTrustedGPGPublicKeyDelete,
				},
			},
		},
	}
}

func (b *backend) pathConfigure(ctx context.Context, req *logical.Request, fields *framework.FieldData) (*logical.Response, error) {
	hclog.L().Debug("Start configuring ...")

	fields.Raw = req.Data
	if err := fields.Validate(); err != nil {
		return logical.ErrorResponse(err.Error()), nil
	}

	for fieldName, schema := range fields.Schema {
		if schema.Required && req.Get(fieldName) == nil {
			return logical.ErrorResponse(fmt.Sprintf("required field %q must be set", fieldName)), nil
		}

		hclog.L().Debug(fmt.Sprintf("Configuring field %s value: %q", fieldName, req.Get(fieldName)))

		switch fieldName {
		case fieldNameDockerImage:
			fieldValue := req.Get(fieldName).(string)

			if err := docker.ValidateImageNameWithDigest(fieldValue); err != nil {
				return logical.ErrorResponse(fmt.Sprintf(`%q field validation failed: %s'`, fieldNameDockerImage, err)), nil
			}
		default:
			continue
		}
	}

	entry, err := logical.StorageEntryJSON(storageKeyConfiguration, fields.Raw)
	if err != nil {
		return nil, err
	}

	if err := req.Storage.Put(ctx, entry); err != nil {
		return nil, err
	}

	return nil, nil
}

func (b *backend) pathTrustedGPGPublicKeyList(ctx context.Context, req *logical.Request, _ *framework.FieldData) (*logical.Response, error) {
	list, err := req.Storage.List(ctx, storageKeyPrefixTrustedGPGPublicKey)
	if err != nil {
		return nil, err
	}

	return &logical.Response{
		Data: map[string]interface{}{
			"names": list,
		},
	}, nil
}

func (b *backend) pathTrustedGPGPublicKeyRead(ctx context.Context, req *logical.Request, fields *framework.FieldData) (*logical.Response, error) {
	name := fields.Get("name").(string)

	e, err := req.Storage.Get(ctx, storageKeyPrefixTrustedGPGPublicKey+name)
	if err != nil {
		return nil, err
	}

	if e == nil {
		return logical.ErrorResponse(fmt.Sprintf("key %q not found in storage", name)), nil
	}

	return &logical.Response{
		Data: map[string]interface{}{
			name: string(e.Value),
		},
	}, nil
}

func (b *backend) pathTrustedGPGPublicKeyCreate(ctx context.Context, req *logical.Request, fields *framework.FieldData) (*logical.Response, error) {
	name := fields.Get("name").(string)
	key := fields.Get("public_key").(string)

	err := req.Storage.Put(ctx, &logical.StorageEntry{
		Key:   storageKeyPrefixTrustedGPGPublicKey + name,
		Value: []byte(key),
	})
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func (b *backend) pathTrustedGPGPublicKeyDelete(ctx context.Context, req *logical.Request, fields *framework.FieldData) (*logical.Response, error) {
	name := fields.Get("name").(string)
	if err := req.Storage.Delete(ctx, storageKeyPrefixTrustedGPGPublicKey+name); err != nil {
		return nil, err
	}

	return nil, nil
}
