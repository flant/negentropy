package vault

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/fatih/structs"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"

	"github.com/flant/negentropy/vault-plugins/shared/consts"
)

const (
	FieldNameVaults = "vaults"

	StorageKeyConfiguration = "vaults_configuration"
)

type VaultConfiguration struct {
	VaultName string `structs:"name" json:"name"`
	VaultUrl  string `structs:"url" json:"url"`
	CaCert    string `structs:"vault_cacert" json:"vault_cacert"`
}

type Configuration struct {
	Vaults []VaultConfiguration `structs:"vaults" json:"vaults"`
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
			Pattern: "^configure/vaults?$",
			Fields: map[string]*framework.FieldSchema{
				FieldNameVaults: {
					Type: framework.TypeSlice,
					Description: "Vaults list as json. Required for CREATE, UPDATE. Each vault should has name and url, " +
						"also if used https need be passed vault_cacert",
				},
			},

			Operations: map[logical.Operation]framework.OperationHandler{
				logical.CreateOperation: &framework.PathOperation{
					Callback: b.pathConfigureCreateOrUpdate,
					Summary:  "Create new flant_gitops processed vaults configuration.",
				},
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.pathConfigureCreateOrUpdate,
					Summary:  "Update the current flant_gitops processed vaults configuration..",
				},
				logical.ReadOperation: &framework.PathOperation{
					Callback: b.pathConfigureRead,
					Summary:  "Read the current flant_gitops processed vaults configuration.",
				},
			},

			HelpSynopsis:    configureHelpSyn,
			HelpDescription: configureHelpDesc,
		},
	}
}

func (b *backend) pathConfigureCreateOrUpdate(ctx context.Context, req *logical.Request, fields *framework.FieldData) (*logical.Response, error) {
	b.Logger().Debug("Configuration started...")

	rawVaults := fields.Get(FieldNameVaults)
	if rawVaults == nil {
		return nil, fmt.Errorf("%w: not passed value at field %q", consts.ErrInvalidArg, FieldNameVaults)
	}
	data, err := json.Marshal(rawVaults)
	if err != nil {
		return nil, fmt.Errorf("%w: wrong format value at field %q: %s", consts.ErrInvalidArg, FieldNameVaults, err.Error())
	}
	var vaults []VaultConfiguration

	err = json.Unmarshal(data, &vaults)
	if err != nil {
		return nil, fmt.Errorf("%w: wrong format value at field %q: %s", consts.ErrInvalidArg, FieldNameVaults, err.Error())
	}
	for _, v := range vaults {
		if v.VaultUrl == "" || v.VaultName == "" {
			return nil, fmt.Errorf("%w: name and url are required for each vault, passed empty", consts.ErrInvalidArg)
		}
	}

	config := Configuration{
		Vaults: vaults,
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
	b.Logger().Debug("Reading Configuration...")

	config, err := getConfiguration(ctx, req.Storage)
	if err != nil {
		return logical.ErrorResponse("Unable to get Configuration: %s", err), nil
	}
	if config == nil {
		return nil, nil
	}

	return &logical.Response{Data: configurationStructToMap(config)}, nil
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
		return nil, fmt.Errorf("%w: vaults configuration is empty", consts.ErrNotConfigured)
	}

	var config Configuration
	if err := storageEntry.DecodeJSON(&config); err != nil {
		return nil, err
	}

	if config.Vaults == nil {
		return nil, fmt.Errorf("%w: vaults configuration is empty", consts.ErrNotConfigured)
	}

	return &config, nil
}

func configurationStructToMap(config *Configuration) map[string]interface{} {
	data := structs.Map(config)
	return data
}

const (
	configureHelpSyn = `
Processed vaults configuration of the flant_gitops backend.
`
	configureHelpDesc = `
The flant_gitops periodic function performs periodic run of configured command
when a new commit arrives into the configured git repository. If processed vaults are set, 
periodic function try to login specified vaults by issued cert, and pass to command list of
vaults and client tokens.

This is processed vaults configuration for the flant_gitops plugin. Plugin will not
pass vaults with client tokens when configuration is not set.
`
)
