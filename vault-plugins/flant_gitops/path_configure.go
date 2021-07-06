package flant_gitops

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/go-hclog"
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

	storageKeyConfiguration        = "configuration"
	storageKeyLastSuccessfulCommit = "last_successful_commit"

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
			Pattern: "configure/git_credential/?",
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
	}
}

func (b *backend) pathConfigureCreateOrUpdate(ctx context.Context, req *logical.Request, fields *framework.FieldData) (*logical.Response, error) {
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

func (b *backend) pathConfigureRead(ctx context.Context, req *logical.Request, _ *framework.FieldData) (*logical.Response, error) {
	v, err := req.Storage.Get(ctx, storageKeyConfiguration)
	if err != nil {
		return nil, fmt.Errorf("unable to get storage entry %q: %s", storageKeyConfiguration, err)
	}

	if v == nil {
		return logical.ErrorResponse("configuration not found"), nil
	}

	var res map[string]interface{}
	if err := json.Unmarshal(v.Value, &res); err != nil {
		return nil, fmt.Errorf("unable to unmarshal storage entry %q: %s", storageKeyConfiguration, err)
	}

	return &logical.Response{Data: res}, nil
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
