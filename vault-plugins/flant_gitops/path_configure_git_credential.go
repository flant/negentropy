package flant_gitops

import (
	"context"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

const (
	fieldNameGitCredentialUsername = "username"
	fieldNameGitCredentialPassword = "password"

	storageKeyConfigurationGitCredential = "configuration_git_credential"
)

type gitCredential struct {
	Username string `structs:"username" json:"username"`
	Password string `structs:"password" json:"password"`
}

func configureGitCredentialPaths(b *backend) []*framework.Path {
	return []*framework.Path{
		{
			Pattern: "^configure/git_credential/?$",
			Fields: map[string]*framework.FieldSchema{
				fieldNameGitCredentialUsername: {
					Type:        framework.TypeString,
					Description: "Git username. Required for CREATE, UPDATE.",
				},
				fieldNameGitCredentialPassword: {
					Type:        framework.TypeString,
					Description: "Git password. Required for CREATE, UPDATE.",
				},
			},

			Operations: map[logical.Operation]framework.OperationHandler{
				logical.CreateOperation: &framework.PathOperation{
					Callback: b.pathConfigureGitCredentialCreateOrUpdate,
				},
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.pathConfigureGitCredentialCreateOrUpdate,
				},
				logical.DeleteOperation: &framework.PathOperation{
					Callback: b.pathConfigureGitCredentialDelete,
				},
			},

			HelpSynopsis:    "TODO",
			HelpDescription: "TODO",
		},
	}
}

func (b *backend) pathConfigureGitCredentialCreateOrUpdate(ctx context.Context, req *logical.Request, fields *framework.FieldData) (*logical.Response, error) {
	b.Logger().Debug("Git credentials configuration started...")

	gitCredential := gitCredential{
		Username: fields.Get(fieldNameGitCredentialUsername).(string),
		Password: fields.Get(fieldNameGitCredentialPassword).(string),
	}

	if gitCredential.Username != "" && gitCredential.Password == "" {
		return logical.ErrorResponse("%q field value specified, but %q field value is not", fieldNameGitCredentialUsername, fieldNameGitCredentialPassword), nil
	}
	if gitCredential.Password != "" && gitCredential.Username == "" {
		return logical.ErrorResponse("%q field value specified, but %q field value is not", fieldNameGitCredentialPassword, fieldNameGitCredentialUsername), nil
	}

	if err := putGitCredential(ctx, req.Storage, gitCredential); err != nil {
		return nil, err
	}

	return nil, nil
}

func (b *backend) pathConfigureGitCredentialDelete(ctx context.Context, req *logical.Request, fields *framework.FieldData) (*logical.Response, error) {
	b.Logger().Debug("Git credentials configuration deleting...")

	if err := deleteGitCredential(ctx, req.Storage); err != nil {
		return logical.ErrorResponse("Unable to delete Git credentials configuration: %s", err), nil
	}

	return nil, nil
}

func getGitCredential(ctx context.Context, storage logical.Storage) (*gitCredential, error) {
	storageEntry, err := storage.Get(ctx, storageKeyConfigurationGitCredential)
	if err != nil {
		return nil, err
	}
	if storageEntry == nil {
		return nil, nil
	}

	var config *gitCredential
	if err := storageEntry.DecodeJSON(&config); err != nil {
		return nil, err
	}

	return config, nil
}

func putGitCredential(ctx context.Context, storage logical.Storage, gitCredential gitCredential) error {
	storageEntry, err := logical.StorageEntryJSON(storageKeyConfigurationGitCredential, gitCredential)
	if err != nil {
		return err
	}

	if err := storage.Put(ctx, storageEntry); err != nil {
		return err
	}

	return err
}

func deleteGitCredential(ctx context.Context, storage logical.Storage) error {
	return storage.Delete(ctx, storageKeyConfigurationGitCredential)
}
