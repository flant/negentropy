package flant_gitops

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/fatih/structs"
	"github.com/go-git/go-git/v5/plumbing/transport"
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
	fieldNameCommands                                   = "commands"

	storageKeyConfiguration = "configuration"
)

type configuration struct {
	GitRepoUrl                                 string        `structs:"git_repo_url" json:"git_repo_url"`
	GitBranch                                  string        `structs:"git_branch_name" json:"git_branch_name"`
	GitPollPeriod                              time.Duration `structs:"git_poll_period" json:"git_poll_period"`
	RequiredNumberOfVerifiedSignaturesOnCommit int           `structs:"required_number_of_verified_signatures_on_commit" json:"required_number_of_verified_signatures_on_commit"`
	InitialLastSuccessfulCommit                string        `structs:"initial_last_successful_commit" json:"initial_last_successful_commit"`
	DockerImage                                string        `structs:"docker_image" json:"docker_image"`
	Commands                                   []string      `structs:"commands" json:"commands"`
}

func configurePaths(b *backend) []*framework.Path {
	return []*framework.Path{
		{
			Pattern: "^configure/?$",
			Fields: map[string]*framework.FieldSchema{
				fieldNameGitRepoUrl: {
					Type:        framework.TypeString,
					Description: "Git repo URL. Required for CREATE, UPDATE.",
				},
				fieldNameGitBranch: {
					Type:        framework.TypeString,
					Default:     "main",
					Description: "Git repo branch",
				},
				fieldNameGitPollPeriod: {
					Type:        framework.TypeDurationSecond,
					Default:     "5m",
					Description: "Period between polls of Git repo",
				},
				fieldNameRequiredNumberOfVerifiedSignaturesOnCommit: {
					Type:        framework.TypeInt,
					Default:     0,
					Description: "Verify that the commit has enough verified signatures",
				},
				fieldNameInitialLastSuccessfulCommit: {
					Type:        framework.TypeString,
					Description: "Last successful commit",
				},
				fieldNameDockerImage: {
					Type:        framework.TypeString,
					Description: "Docker image name for the container in which the commands will be executed. Required for CREATE, UPDATE.",
				},
				fieldNameCommands: {
					Type:        framework.TypeCommaStringSlice,
					Description: "Comma-separated list of commands to execute in Docker container. Can also be passed as a list of strings in JSON payload",
				},
			},

			Operations: map[logical.Operation]framework.OperationHandler{
				logical.CreateOperation: &framework.PathOperation{
					Callback: b.pathConfigureCreateOrUpdate,
					Summary:  "Create new flant_gitops backend configuration.",
				},
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.pathConfigureCreateOrUpdate,
					Summary:  "Update the current flant_gitops backend configuration.",
				},
				logical.ReadOperation: &framework.PathOperation{
					Callback: b.pathConfigureRead,
					Summary:  "Read the current flant_gitops backend configuration.",
				},
				logical.DeleteOperation: &framework.PathOperation{
					Callback: b.pathConfigureDelete,
					Summary:  "Delete the current flant_gitops backend configuration.",
				},
			},

			HelpSynopsis:    configureHelpSyn,
			HelpDescription: configureHelpDesc,
		},
	}
}

func (b *backend) pathConfigureCreateOrUpdate(ctx context.Context, req *logical.Request, fields *framework.FieldData) (*logical.Response, error) {
	b.Logger().Debug("Configuration started...")

	config := configuration{
		GitRepoUrl:    fields.Get(fieldNameGitRepoUrl).(string),
		GitBranch:     fields.Get(fieldNameGitBranch).(string),
		GitPollPeriod: time.Duration(fields.Get(fieldNameGitPollPeriod).(int)) * time.Second,
		RequiredNumberOfVerifiedSignaturesOnCommit: fields.Get(fieldNameRequiredNumberOfVerifiedSignaturesOnCommit).(int),
		InitialLastSuccessfulCommit:                fields.Get(fieldNameInitialLastSuccessfulCommit).(string),
		DockerImage:                                fields.Get(fieldNameDockerImage).(string),
		Commands:                                   fields.Get(fieldNameCommands).([]string),
	}

	if err := docker.ValidateImageNameWithDigest(config.DockerImage); err != nil {
		return logical.ErrorResponse("%q field is invalid: %s", fieldNameDockerImage, err), nil
	}

	if config.GitRepoUrl == "" {
		return logical.ErrorResponse("%q field value should not be empty", fieldNameGitRepoUrl), nil
	}
	if _, err := transport.NewEndpoint(config.GitRepoUrl); err != nil {
		return logical.ErrorResponse("%q field is invalid: %s", fieldNameGitRepoUrl, err), nil
	}

	{
		cfgData, cfgErr := json.MarshalIndent(config, "", "  ")
		b.Logger().Debug(fmt.Sprintf("Got configuration (err=%v):\n%s", cfgErr, string(cfgData)))
	}

	if err := putConfiguration(ctx, req.Storage, config); err != nil {
		return nil, err
	}

	return nil, nil
}

func (b *backend) pathConfigureRead(ctx context.Context, req *logical.Request, _ *framework.FieldData) (*logical.Response, error) {
	b.Logger().Debug("Reading configuration...")

	config, err := getConfiguration(ctx, req.Storage)
	if err != nil {
		return logical.ErrorResponse("Unable to get configuration: %s", err), nil
	}
	if config == nil {
		return nil, nil
	}

	return &logical.Response{Data: configurationStructToMap(config)}, nil
}

func (b *backend) pathConfigureDelete(ctx context.Context, req *logical.Request, _ *framework.FieldData) (*logical.Response, error) {
	b.Logger().Debug("Deleting configuration...")

	if err := deleteConfiguration(ctx, req.Storage); err != nil {
		return logical.ErrorResponse("Unable to delete configuration: %s", err), nil
	}

	return nil, nil
}

func putConfiguration(ctx context.Context, storage logical.Storage, config configuration) error {
	storageEntry, err := logical.StorageEntryJSON(storageKeyConfiguration, config)
	if err != nil {
		return err
	}

	if err := storage.Put(ctx, storageEntry); err != nil {
		return err
	}

	return err
}

func getConfiguration(ctx context.Context, storage logical.Storage) (*configuration, error) {
	storageEntry, err := storage.Get(ctx, storageKeyConfiguration)
	if err != nil {
		return nil, err
	}
	if storageEntry == nil {
		return nil, nil
	}

	var config *configuration
	if err := storageEntry.DecodeJSON(&config); err != nil {
		return nil, err
	}

	return config, nil
}

func deleteConfiguration(ctx context.Context, storage logical.Storage) error {
	return storage.Delete(ctx, storageKeyConfiguration)
}

func configurationStructToMap(config *configuration) map[string]interface{} {
	data := structs.Map(config)
	data[fieldNameGitPollPeriod] = config.GitPollPeriod.Seconds()

	return data
}

const (
	configureHelpSyn = `
Main configuration of the flant_gitops backend.
`
	configureHelpDesc = `
The flant_gitops periodic function performs periodic run of configured command
when a new commit arrives into the configured git repository.

This is main configuration for the flant_gitops plugin. Plugin will not
function when configuration is not set.
`
)
