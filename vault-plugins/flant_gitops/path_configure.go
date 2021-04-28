package flant_gitops

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

const (
	storageEntryConfigurationKey = "configuration"

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
)

func pathConfigure(b *backend) *framework.Path {
	return &framework.Path{
		Pattern: "configure$",
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
				Type: framework.TypeCommaStringSlice,
			},
			fieldRequiredNumberOfVerifiedSignaturesOnCommitName: {
				Type: framework.TypeInt,
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
				return logical.ErrorResponse(fmt.Sprintf("field %q must be set in the extended form \"REPO:TAG@SHA256\" (e.g. \"ubuntu:18.04@sha256:538529c9d229fb55f50e6746b119e899775205d62c0fc1b7e679b30d02ecb6e8\")", fieldName)), nil
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
