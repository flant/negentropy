package paths

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"

	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/usecase"
	backentutils "github.com/flant/negentropy/vault-plugins/shared/backent-utils"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/uuid"
)

type clientBackend struct {
	*flantFlowExtension
}

func clientPaths(e *flantFlowExtension) []*framework.Path {
	bb := &clientBackend{
		flantFlowExtension: e,
	}
	return bb.paths()
}

func (b clientBackend) paths() []*framework.Path {
	return []*framework.Path{
		// Creation
		{
			Pattern: "client",
			Fields: map[string]*framework.FieldSchema{
				"identifier": {
					Type:        framework.TypeNameString,
					Description: "Identifier for humans and machines",
					Required:    true,
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.CreateOperation: &framework.PathOperation{
					Callback: b.checkConfigured(b.handleCreate(false)),
					Summary:  "Create client.",
				},
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.checkConfigured(b.handleCreate(false)),
					Summary:  "Create client.",
				},
			},
		},
		// Creation with known uuid in advance
		{
			Pattern: "client/privileged",
			Fields: map[string]*framework.FieldSchema{
				"uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a client",
					Required:    true,
				},
				"identifier": {
					Type:        framework.TypeNameString,
					Description: "Identifier for humans and machines",
					Required:    true,
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.CreateOperation: &framework.PathOperation{
					Callback: b.checkConfigured(b.handleCreate(true)),
					Summary:  "Create client with preexistent ID.",
				},
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.checkConfigured(b.handleCreate(true)),
					Summary:  "Create client with preexistent ID.",
				},
			},
		},
		// List
		{
			Pattern: "client/?",
			Fields: map[string]*framework.FieldSchema{
				"show_archived": {
					Type:        framework.TypeBool,
					Description: "Option to list archived clients",
					Required:    false,
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.ReadOperation: &framework.PathOperation{
					Callback: b.checkConfigured(b.handleList),
					Summary:  "Lists all client IDs.",
				},
			},
		},
		// Read, update, delete by uuid
		{
			Pattern: "client/" + uuid.Pattern("uuid") + "$",
			Fields: map[string]*framework.FieldSchema{
				"uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a client",
					Required:    true,
				},
				"identifier": {
					Type:        framework.TypeNameString,
					Description: "Identifier for humans and machines",
					Required:    true,
				},
				"resource_version": {
					Type:        framework.TypeString,
					Description: "Resource version",
					Required:    true,
				},
			},
			ExistenceCheck: b.handleExistence,
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.checkConfigured(b.handleUpdate),
					Summary:  "Update the client by ID.",
				},
				logical.ReadOperation: &framework.PathOperation{
					Callback: b.checkConfigured(b.handleRead),
					Summary:  "Retrieve the client by ID.",
				},
				logical.DeleteOperation: &framework.PathOperation{
					Callback: b.checkConfigured(b.handleDelete),
					Summary:  "Deletes the client by ID.",
				},
			},
		},
		// Restore
		{
			Pattern: "client/" + uuid.Pattern("uuid") + "/restore" + "$",
			Fields: map[string]*framework.FieldSchema{
				"uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a client",
					Required:    true,
				},
				"full_restore": {
					Type:        framework.TypeBool,
					Description: "Option to restore full client data",
					Required:    false,
				},
			},
			ExistenceCheck: b.handleExistence,
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.checkConfigured(b.handleRestore),
					Summary:  "Restore the client by ID.",
				},
			},
		},
	}
}

func (b *clientBackend) handleExistence(_ context.Context, req *logical.Request, data *framework.FieldData) (bool, error) {
	id := data.Get("uuid").(string)
	b.Logger().Debug("checking client existence", "path", req.Path, "id", id, "op", req.Operation)

	if !uuid.IsValid(id) {
		return false, fmt.Errorf("id must be valid UUIDv4")
	}

	tx := b.storage.Txn(false)

	c, err := usecase.Clients(tx, b.getLiveConfig()).GetByID(id)
	if err != nil {
		return false, err
	}
	return c != nil, nil
}

func (b *clientBackend) handleCreate(expectID bool) framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		b.Logger().Debug("create client", "path", req.Path)
		id, err := backentutils.GetCreationID(expectID, data)
		if err != nil {
			return backentutils.ResponseErr(req, err)
		}
		client := &model.Client{

			UUID:       id,
			Identifier: data.Get("identifier").(string),
		}

		tx := b.storage.Txn(true)
		defer tx.Abort()

		if client, err = usecase.Clients(tx, b.getLiveConfig()).Create(client); err != nil {
			err = fmt.Errorf("cannot create client:%w", err)
			b.Logger().Error(err.Error())
			return backentutils.ResponseErr(req, err)
		}
		if err := io.CommitWithLog(tx, b.Logger()); err != nil {
			return nil, err
		}

		resp := &logical.Response{Data: map[string]interface{}{"client": client}}
		return logical.RespondWithStatusCode(resp, req, http.StatusCreated)
	}
}

func (b *clientBackend) handleUpdate(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	b.Logger().Debug("update client", "path", req.Path)
	id := data.Get("uuid").(string)
	tx := b.storage.Txn(true)
	defer tx.Abort()

	client := &model.Client{
		UUID:       id,
		Identifier: data.Get("identifier").(string),
		Version:    data.Get("resource_version").(string),
	}

	client, err := usecase.Clients(tx, b.getLiveConfig()).Update(client)
	if err != nil {
		return backentutils.ResponseErr(req, err)
	}
	if err := io.CommitWithLog(tx, b.Logger()); err != nil {
		return nil, err
	}

	resp := &logical.Response{Data: map[string]interface{}{"client": client}}
	return logical.RespondWithStatusCode(resp, req, http.StatusOK)
}

func (b *clientBackend) handleDelete(_ context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	b.Logger().Debug("delete client", "path", req.Path)
	tx := b.storage.Txn(true)
	defer tx.Abort()

	id := data.Get("uuid").(string)

	err := usecase.Clients(tx, b.getLiveConfig()).Delete(id)
	if err != nil {
		return backentutils.ResponseErr(req, err)
	}
	if err := io.CommitWithLog(tx, b.Logger()); err != nil {
		return nil, err
	}

	return logical.RespondWithStatusCode(nil, req, http.StatusNoContent)
}

func (b *clientBackend) handleRead(_ context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	b.Logger().Debug("read client", "path", req.Path)
	id := data.Get("uuid").(string)

	tx := b.storage.Txn(false)

	client, err := usecase.Clients(tx, b.getLiveConfig()).GetByID(id)
	if err != nil {
		return backentutils.ResponseErr(req, err)
	}

	resp := &logical.Response{Data: map[string]interface{}{
		"client":       client,
		"full_restore": false, // TODO check if full restore available
	}}
	return logical.RespondWithStatusCode(resp, req, http.StatusOK)
}

func (b *clientBackend) handleList(_ context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	b.Logger().Debug("listing clients", "path", req.Path)
	var showArchived bool
	rawShowArchived, ok := data.GetOk("show_archived")
	if ok {
		showArchived = rawShowArchived.(bool)
	}

	tx := b.storage.Txn(false)
	clients, err := usecase.Clients(tx, b.getLiveConfig()).List(showArchived)
	if err != nil {
		return nil, err
	}

	resp := &logical.Response{
		Data: map[string]interface{}{
			"clients": clients,
		},
	}
	return logical.RespondWithStatusCode(resp, req, http.StatusOK)
}

func (b *clientBackend) handleRestore(_ context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	b.Logger().Debug("restore client", "path", req.Path)
	tx := b.storage.Txn(true)
	defer tx.Abort()

	id := data.Get("uuid").(string)
	var fullRestore bool
	rawFullRestore, ok := data.GetOk("full_restore")
	if ok {
		fullRestore = rawFullRestore.(bool)
	}

	client, err := usecase.Clients(tx, b.getLiveConfig()).Restore(id, fullRestore)
	if err != nil {
		return backentutils.ResponseErr(req, err)
	}

	if err := io.CommitWithLog(tx, b.Logger()); err != nil {
		return nil, err
	}

	resp := &logical.Response{Data: map[string]interface{}{
		"client": client,
	}}
	return logical.RespondWithStatusCode(resp, req, http.StatusOK)
}
