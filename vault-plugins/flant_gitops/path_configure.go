package flant_gitops

import (
	"context"
	"fmt"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/werf/vault-plugin-secrets-trdl/pkg/docker"
	"github.com/werf/vault-plugin-secrets-trdl/pkg/util"
)

const (
	fieldNameGitRepoUrl                                 = "git_repo_url"
	fieldNameGitBranch                                  = "git_branch_name"
	fieldNameGitPollPeriod                              = "git_poll_period"
	fieldNameRequiredNumberOfVerifiedSignaturesOnCommit = "required_number_of_verified_signatures_on_commit"
	fieldNameInitialLastSuccessfulCommit                = "initial_last_successful_commit"
	fieldNameDockerImage                                = "docker_image"
	fieldNameCommand                                    = "command"
	fieldNameGitCredentialUsername                      = "username"
	fieldNameGitCredentialPassword                      = "password"

	storageKeyLastSuccessfulCommit = "last_successful_commit"
)

func configurePaths(b *backend) []*framework.Path {
	return []*framework.Path{
		{
			Pattern: "^configure/?$",
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
				logical.CreateOperation: &framework.PathOperation{
					Callback: b.pathConfigureCreateOrUpdate,
				},
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.pathConfigureCreateOrUpdate,
				},
				logical.ReadOperation: &framework.PathOperation{
					Callback: b.pathConfigureRead,
				},
			},
		},
		{
			Pattern: "^configure/git_credential/?$",
			Fields: map[string]*framework.FieldSchema{
				fieldNameGitCredentialUsername: {
					Type:        framework.TypeString,
					Description: "Git username",
					Required:    true,
				},
				fieldNameGitCredentialPassword: {
					Type:        framework.TypeString,
					Description: "Git password",
					Required:    true,
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.CreateOperation: &framework.PathOperation{
					Callback: b.pathConfigureGitCredential,
				},
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.pathConfigureGitCredential,
				},
			},
		},
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
				"name": {
					Type:        framework.TypeNameString,
					Description: "TODO",
					Required:    true,
				},
				"path": {
					Type:        framework.TypeString,
					Description: "TODO",
					Required:    true,
				},
				"method": {
					Type:        framework.TypeString,
					Description: "TODO",
					Required:    true,
				},
				"options": {
					Type:        framework.TypeString,
					Description: "TODO",
				},
				"wrap_ttl": {
					Type:        framework.TypeString,
					Description: "TODO",
					Default:     "1m",
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.CreateOperation: &framework.PathOperation{
					Callback: b.pathConfigureVaultRequestCreate,
					Summary:  "TODO",
				},
				logical.ReadOperation: &framework.PathOperation{
					Callback: b.pathConfigureVaultRequestRead,
					Summary:  "TODO",
				},
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.pathConfigureVaultRequestUpdate,
					Summary:  "TODO",
				},
				logical.DeleteOperation: &framework.PathOperation{
					Callback: b.pathConfigureVaultRequestDelete,
					Summary:  "TODO",
				},
			},
		},
	}
}

func (b *backend) pathConfigureCreateOrUpdate(ctx context.Context, req *logical.Request, fields *framework.FieldData) (*logical.Response, error) {
	b.Logger().Debug("Start configuring ...")

	resp, err := util.ValidateRequestFields(req, fields)
	if resp != nil || err != nil {
		return resp, err
	}

	if err := docker.ValidateImageNameWithDigest(req.Get(fieldNameDockerImage).(string)); err != nil {
		return logical.ErrorResponse(fmt.Sprintf(`%q field is invalid: %s'`, fieldNameDockerImage, err)), nil
	}

	if err := putConfiguration(ctx, req.Storage, fields.Raw); err != nil {
		return nil, err
	}

	return nil, nil
}

func (b *backend) pathConfigureRead(ctx context.Context, req *logical.Request, _ *framework.FieldData) (*logical.Response, error) {
	config, err := getConfiguration(ctx, req.Storage)
	if err != nil {
		return nil, err
	} else if config == nil {
		return logical.ErrorResponse("configuration not set"), nil
	}

	return &logical.Response{
		Data: map[string]interface{}{
			fieldNameGitRepoUrl:    config.GitRepoUrl,
			fieldNameGitBranch:     config.GitBranchName,
			fieldNameGitPollPeriod: config.GitPollPeriod,
			fieldNameRequiredNumberOfVerifiedSignaturesOnCommit: config.RequiredNumberOfVerifiedSignaturesOnCommit,
			fieldNameInitialLastSuccessfulCommit:                config.InitialLastSuccessfulCommit,
			fieldNameDockerImage:                                config.DockerImage,
			fieldNameCommand:                                    config.Command,
		},
	}, nil
}

func (b *backend) pathConfigureGitCredential(ctx context.Context, req *logical.Request, fields *framework.FieldData) (*logical.Response, error) {
	resp, err := util.ValidateRequestFields(req, fields)
	if resp != nil || err != nil {
		return resp, err
	}

	if err := putGitCredential(ctx, req.Storage, fields.Raw); err != nil {
		return nil, err
	}

	return nil, nil
}

func (b *backend) pathConfigureVaultRequestList(ctx context.Context, req *logical.Request, fields *framework.FieldData) (*logical.Response, error) {
	apiClient, err := b.AccessVaultController.APIClient()
	if err != nil {
		return logical.ErrorResponse(err.Error()), nil
	}

	// FIXME: just for test, remove this
	b.Logger().Debug("Vault client token: %s", apiClient.Token())

	// TODO

	return nil, nil
}

func (b *backend) pathConfigureVaultRequestCreate(ctx context.Context, req *logical.Request, fields *framework.FieldData) (*logical.Response, error) {
	// TODO
	return nil, nil
}

func (b *backend) pathConfigureVaultRequestRead(ctx context.Context, req *logical.Request, fields *framework.FieldData) (*logical.Response, error) {
	// TODO
	return nil, nil
}

func (b *backend) pathConfigureVaultRequestUpdate(ctx context.Context, req *logical.Request, fields *framework.FieldData) (*logical.Response, error) {
	// TODO
	return nil, nil
}

func (b *backend) pathConfigureVaultRequestDelete(ctx context.Context, req *logical.Request, fields *framework.FieldData) (*logical.Response, error) {
	// TODO
	return nil, nil
}
