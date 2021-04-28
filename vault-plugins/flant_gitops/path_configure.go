package flant_gitops

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

const (
	fieldGitRepoUrlName                                 = "git_repo_url"
	fieldGitBranchName                                  = "git_branch_name"
	fieldPeriodicityName                                = "periodicity"
	fieldTrustedGpgPublicKeysName                       = "trusted_gpg_public_keys"
	fieldRequiredNumberOfVerifiedSignaturesOnCommitName = "required_number_of_verified_signatures_on_commit"
	fieldLastSuccessfulCommitName                       = "last_successful_commit"
	fieldBuildDockerImageName                           = "build_docker_image"
	fieldBuildCommandName                               = "build_command"
	fieldBuildTimeoutName                               = "build_timeout"
	fieldBuildHistoryLimitName                          = "build_history_limit"

	storageEntryConfigurationKey        = "configuration"
	storageEntryLastSuccessfulCommitKey = fieldLastSuccessfulCommitName

	configurePathPattern = "configure$"
)

func pathConfigure(b *backend) *framework.Path {
	return &framework.Path{
		Pattern: configurePathPattern,
		Fields: map[string]*framework.FieldSchema{
			fieldGitRepoUrlName: {
				Type:     framework.TypeString,
				Required: true,
			},
			fieldGitBranchName: {
				Type:     framework.TypeString,
				Required: true,
			},
			fieldPeriodicityName: {
				Type: framework.TypeDurationSecond,
			},
			fieldTrustedGpgPublicKeysName: {
				Type:     framework.TypeCommaStringSlice,
				Required: true,
			},
			fieldRequiredNumberOfVerifiedSignaturesOnCommitName: {
				Type:     framework.TypeInt,
				Required: true,
			},
			fieldLastSuccessfulCommitName: {
				Type: framework.TypeString,
			},
			fieldBuildDockerImageName: {
				Type:     framework.TypeString,
				Required: true,
			},
			fieldBuildCommandName: {
				Type:     framework.TypeString,
				Required: true,
			},
			fieldBuildTimeoutName: {
				Type: framework.TypeDurationSecond,
			},
			fieldBuildHistoryLimitName: {
				Type: framework.TypeInt,
			},
		},
		Operations: map[logical.Operation]framework.OperationHandler{
			logical.UpdateOperation: &framework.PathOperation{
				Callback: b.pathConfigure,
			},
		},
	}
}

func (b *backend) pathConfigure(ctx context.Context, req *logical.Request, fields *framework.FieldData) (*logical.Response, error) {
	fields.Raw = req.Data
	if err := fields.Validate(); err != nil {
		return logical.ErrorResponse(err.Error()), nil
	}

	for fieldName, schema := range fields.Schema {
		if schema.Required && req.Get(fieldName) == nil {
			return logical.ErrorResponse(fmt.Sprintf("required field %q must be set", fieldName)), nil
		}

		switch fieldName {
		case fieldBuildDockerImageName:
			if !strings.ContainsRune(req.Get(fieldName).(string), '@') {
				return logical.ErrorResponse(fmt.Sprintf("field %q must be set in the extended form \"REPO[:TAG]@SHA256\" (e.g. \"ubuntu:18.04@sha256:538529c9d229fb55f50e6746b119e899775205d62c0fc1b7e679b30d02ecb6e8\")", fieldName)), nil
			}
		}
	}

	entry, err := logical.StorageEntryJSON(storageEntryConfigurationKey, fields.Raw)
	if err != nil {
		return nil, err
	}

	if err := req.Storage.Put(ctx, entry); err != nil {
		return nil, err
	}

	return nil, nil
}

func (b *backend) getConfiguration(ctx context.Context, req *logical.Request) (*framework.FieldData, error) {
	entry, err := req.Storage.Get(ctx, storageEntryConfigurationKey)
	if err != nil {
		return nil, err
	}

	if entry == nil {
		return nil, fmt.Errorf("no configuration found in storage")
	}

	data := make(map[string]interface{})
	if err := json.Unmarshal(entry.Value, &data); err != nil {
		return nil, err
	}

	fields := &framework.FieldData{}
	fields.Raw = data
	fields.Schema = b.getConfigureFieldSchemaMap()

	return fields, nil
}

func (b *backend) getConfigureFieldSchemaMap() map[string]*framework.FieldSchema {
	for _, p := range b.Paths {
		if p.Pattern == configurePathPattern {
			return p.Fields
		}
	}

	panic("runtime error")
}
