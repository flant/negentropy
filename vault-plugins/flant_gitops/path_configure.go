package flant_gitops

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/go-git/go-git/v5/plumbing/transport"
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

	fieldNameGitCredentialUsername = "username"
	fieldNameGitCredentialPassword = "password"

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

	gitRepoUrl := req.Get(fieldNameGitRepoUrl).(string)
	if _, err := transport.NewEndpoint(gitRepoUrl); err != nil {
		return logical.ErrorResponse(fmt.Sprintf("invalid %s given: %s", fieldNameGitRepoUrl, err)), nil
	}

	gitBranch := req.Get(fieldNameGitBranch).(string)

	var gitPollPeriod string
	gitPollPeriodI := req.Get(fieldNameGitPollPeriod)
	if gitPollPeriodI == nil {
		gitPollPeriod = fields.Schema[fieldNameGitPollPeriod].Default.(string)
	} else {
		gitPollPeriod = gitPollPeriodI.(string)

		if _, err := time.ParseDuration(gitPollPeriod); err != nil {
			return logical.ErrorResponse(fmt.Sprintf("invalid %s given, expected golang time duration: %s", fieldNameGitPollPeriod, err)), nil
		}
	}

	var initialLastSuccessfulCommit string
	if v := req.Get(fieldNameInitialLastSuccessfulCommit); v != nil {
		initialLastSuccessfulCommit = v.(string)
	}

	var requiredNumberOfVerifiedSignaturesOnCommit int
	{
		valueStr := req.Get(fieldNameRequiredNumberOfVerifiedSignaturesOnCommit).(string)
		valueUint, err := strconv.ParseUint(valueStr, 10, 64)
		if err != nil {
			return logical.ErrorResponse(fmt.Sprintf("invalid %s given, expected number: %s", fieldNameGitPollPeriod, err)), nil
		}
		requiredNumberOfVerifiedSignaturesOnCommit = int(valueUint)
	}

	dockerImage := req.Get(fieldNameDockerImage).(string)
	if err := docker.ValidateImageNameWithDigest(dockerImage); err != nil {
		return logical.ErrorResponse(fmt.Sprintf("invalid %s given, expected docker image name with digest: %s", fieldNameDockerImage, err)), nil
	}

	command := req.Get(fieldNameCommand).(string)

	cfg := &configuration{
		GitRepoUrl:    gitRepoUrl,
		GitBranch:     gitBranch,
		GitPollPeriod: gitPollPeriod,
		RequiredNumberOfVerifiedSignaturesOnCommit: requiredNumberOfVerifiedSignaturesOnCommit,
		InitialLastSuccessfulCommit:                initialLastSuccessfulCommit,
		DockerImage:                                dockerImage,
		Command:                                    command,
	}

	{
		cfgData, cfgErr := json.MarshalIndent(cfg, "", "  ")
		b.Logger().Debug(fmt.Sprintf("Got configuration (err=%v):\n%s", cfgErr, string(cfgData)))
	}

	if err := putConfiguration(ctx, req.Storage, cfg); err != nil {
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
			fieldNameGitBranch:     config.GitBranch,
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
