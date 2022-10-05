package git_repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/fatih/structs"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

const (
	FieldNameGitRepoUrl                                 = "git_repo_url"
	FieldNameGitBranch                                  = "git_branch_name"
	FieldNameGitPollPeriod                              = "git_poll_period"
	FieldNameRequiredNumberOfVerifiedSignaturesOnCommit = "required_number_of_verified_signatures_on_commit"
	FieldNameInitialLastSuccessfulCommit                = "initial_last_successful_commit"

	StorageKeyConfiguration = "git_repository_configuration"
)

type Configuration struct {
	GitRepoUrl                                 string        `structs:"git_repo_url" json:"git_repo_url"`
	GitBranch                                  string        `structs:"git_branch_name" json:"git_branch_name"`
	GitPollPeriod                              time.Duration `structs:"git_poll_period" json:"git_poll_period"`
	RequiredNumberOfVerifiedSignaturesOnCommit int           `structs:"required_number_of_verified_signatures_on_commit" json:"required_number_of_verified_signatures_on_commit"`
	InitialLastSuccessfulCommit                string        `structs:"initial_last_successful_commit" json:"initial_last_successful_commit"`
}

type backend struct {
	// just for logger provider
	baseBackend *framework.Backend
}

func (b *backend) Logger() hclog.Logger {
	return b.baseBackend.Logger()
}

func ConfigurePaths(baseBackend *framework.Backend) []*framework.Path {
	b := backend{
		baseBackend: baseBackend,
	}

	return []*framework.Path{
		{
			Pattern: "^configure/git_repository/?$",
			Fields: map[string]*framework.FieldSchema{
				FieldNameGitRepoUrl: {
					Type:        framework.TypeString,
					Description: "Git repo URL. Required for CREATE, UPDATE.",
				},
				FieldNameGitBranch: {
					Type:        framework.TypeString,
					Default:     "main",
					Description: "Git repo branch",
				},
				FieldNameGitPollPeriod: {
					Type:        framework.TypeDurationSecond,
					Default:     "5m",
					Description: "Period between polls of Git repo",
				},
				FieldNameRequiredNumberOfVerifiedSignaturesOnCommit: {
					Type:        framework.TypeInt,
					Default:     0,
					Description: "Verify that the commit has enough verified signatures",
				},
				FieldNameInitialLastSuccessfulCommit: {
					Type:        framework.TypeString,
					Description: "Last successful commit",
				},
			},

			Operations: map[logical.Operation]framework.OperationHandler{
				logical.CreateOperation: &framework.PathOperation{
					Callback: b.pathConfigureCreateOrUpdate,
					Summary:  "Create new flant_gitops git_repository configuration.",
				},
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.pathConfigureCreateOrUpdate,
					Summary:  "Update the current flant_gitops git_repository configuration.",
				},
				logical.ReadOperation: &framework.PathOperation{
					Callback: b.pathConfigureRead,
					Summary:  "Read the current flant_gitops git_repository configuration.",
				},
				logical.DeleteOperation: &framework.PathOperation{
					Callback: b.pathConfigureDelete,
					Summary:  "Delete the current flant_gitops git_repository configuration.",
				},
			},

			HelpSynopsis:    configureHelpSyn,
			HelpDescription: configureHelpDesc,
		},
	}
}

func (b *backend) pathConfigureCreateOrUpdate(ctx context.Context, req *logical.Request, fields *framework.FieldData) (*logical.Response, error) {
	b.Logger().Debug("Git repository configuration started...")

	config := Configuration{
		GitRepoUrl:    fields.Get(FieldNameGitRepoUrl).(string),
		GitBranch:     fields.Get(FieldNameGitBranch).(string),
		GitPollPeriod: time.Duration(fields.Get(FieldNameGitPollPeriod).(int)) * time.Second,
		RequiredNumberOfVerifiedSignaturesOnCommit: fields.Get(FieldNameRequiredNumberOfVerifiedSignaturesOnCommit).(int),
		InitialLastSuccessfulCommit:                fields.Get(FieldNameInitialLastSuccessfulCommit).(string),
	}

	if config.GitRepoUrl == "" {
		return logical.ErrorResponse("%q field value should not be empty", FieldNameGitRepoUrl), nil
	}
	if _, err := transport.NewEndpoint(config.GitRepoUrl); err != nil {
		return logical.ErrorResponse("%q field is invalid: %s", FieldNameGitRepoUrl, err), nil
	}

	{
		cfgData, cfgErr := json.MarshalIndent(config, "", "  ")
		b.Logger().Debug(fmt.Sprintf("Got Configuration (err=%v):\n%s", cfgErr, string(cfgData)))
	}

	if err := putConfiguration(ctx, req.Storage, config); err != nil {
		return nil, err
	}

	return nil, nil
}

func (b *backend) pathConfigureRead(ctx context.Context, req *logical.Request, _ *framework.FieldData) (*logical.Response, error) {
	b.Logger().Debug("Reading git repository configuration...")

	config, err := getConfiguration(ctx, req.Storage)
	if err != nil {
		return logical.ErrorResponse("Unable to get Configuration: %s", err), nil
	}
	if config == nil {
		return nil, nil
	}

	return &logical.Response{Data: configurationStructToMap(config)}, nil
}

func (b *backend) pathConfigureDelete(ctx context.Context, req *logical.Request, _ *framework.FieldData) (*logical.Response, error) {
	b.Logger().Debug("Deleting Configuration...")

	if err := deleteConfiguration(ctx, req.Storage); err != nil {
		return logical.ErrorResponse("Unable to delete Configuration: %s", err), nil
	}

	return nil, nil
}

func putConfiguration(ctx context.Context, storage logical.Storage, config Configuration) error {
	storageEntry, err := logical.StorageEntryJSON(StorageKeyConfiguration, config)
	if err != nil {
		return err
	}

	if err := storage.Put(ctx, storageEntry); err != nil {
		return err
	}

	return err
}

func getConfiguration(ctx context.Context, storage logical.Storage) (*Configuration, error) {
	storageEntry, err := storage.Get(ctx, StorageKeyConfiguration)
	if err != nil {
		return nil, err
	}
	if storageEntry == nil {
		return nil, nil
	}

	var config *Configuration
	if err := storageEntry.DecodeJSON(&config); err != nil {
		return nil, err
	}

	return config, nil
}

func deleteConfiguration(ctx context.Context, storage logical.Storage) error {
	return storage.Delete(ctx, StorageKeyConfiguration)
}

func configurationStructToMap(config *Configuration) map[string]interface{} {
	data := structs.Map(config)
	data[FieldNameGitPollPeriod] = config.GitPollPeriod.Seconds()

	return data
}

const (
	configureHelpSyn = `
Git repository configuration of the flant_gitops backend.
`
	configureHelpDesc = `
The flant_gitops periodic function performs periodic run of configured command
when a new commit arrives into the configured git repository.

This is git repository configuration for the flant_gitops plugin. Plugin will not
function when Configuration is not set.
`
)
