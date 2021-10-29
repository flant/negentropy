package backend

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"

	"github.com/flant/negentropy/vault-plugins/flant_flow/model"
	"github.com/flant/negentropy/vault-plugins/flant_flow/usecase"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/uuid"
)

type teamBackend struct {
	logical.Backend
	storage *io.MemoryStore
}

func teamPaths(b logical.Backend, storage *io.MemoryStore) []*framework.Path {
	bb := &teamBackend{
		Backend: b,
		storage: storage,
	}
	return bb.paths()
}

func (b teamBackend) paths() []*framework.Path {
	return []*framework.Path{
		// Creation
		{
			Pattern: "team",
			Fields: map[string]*framework.FieldSchema{
				"identifier": {
					Type:        framework.TypeNameString,
					Description: "Identifier team",
					Required:    true,
				},
				"team_type": {
					Type:          framework.TypeNameString,
					Description:   "Type of team",
					Required:      true,
					AllowedValues: model.AllowedTeamTypes,
				},
				"parent_team_uuid": {
					Type:        framework.TypeString,
					Description: "ID of parent team",
					Required:    false,
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.CreateOperation: &framework.PathOperation{
					Callback: b.handleCreate(false),
					Summary:  "Create team.",
				},
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.handleCreate(false),
					Summary:  "Create team.",
				},
			},
		},
		// Creation with known uuid in advance
		{
			Pattern: "team/privileged",
			Fields: map[string]*framework.FieldSchema{
				"uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a team",
					Required:    true,
				},
				"identifier": {
					Type:        framework.TypeNameString,
					Description: "Identifier for humans and machines",
					Required:    true,
				},
				"team_type": {
					Type:          framework.TypeNameString,
					Description:   "Type of team",
					Required:      true,
					AllowedValues: model.AllowedTeamTypes,
				},
				"parent_team_uuid": {
					Type:        framework.TypeString,
					Description: "ID of parent team",
					Required:    false,
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.CreateOperation: &framework.PathOperation{
					Callback: b.handleCreate(true),
					Summary:  "Create team with preexistent ID.",
				},
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.handleCreate(true),
					Summary:  "Create team with preexistent ID.",
				},
			},
		},
		// List
		{
			Pattern: "team/?",
			Fields: map[string]*framework.FieldSchema{
				"show_archived": {
					Type:        framework.TypeBool,
					Description: "Option to list archived teams",
					Required:    false,
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.ReadOperation: &framework.PathOperation{
					Callback: b.handleList(),
					Summary:  "Lists all teams IDs.",
				},
			},
		},
		// Read, update, delete by uuid
		{
			Pattern: "team/" + uuid.Pattern("uuid") + "$",
			Fields: map[string]*framework.FieldSchema{
				"uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a team",
					Required:    true,
				},
				"identifier": {
					Type:        framework.TypeNameString,
					Description: "Identifier team",
					Required:    false,
				},
				"parent_team_uuid": {
					Type:        framework.TypeString,
					Description: "ID of parent team",
					Required:    false,
				},
				"resource_version": {
					Type:        framework.TypeString,
					Description: "Resource version",
					Required:    true,
				},
			},
			ExistenceCheck: b.handleExistence(),
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.handleUpdate(),
					Summary:  "Update the team by ID.",
				},
				logical.ReadOperation: &framework.PathOperation{
					Callback: b.handleRead(),
					Summary:  "Retrieve the team by ID.",
				},
				logical.DeleteOperation: &framework.PathOperation{
					Callback: b.handleDelete(),
					Summary:  "Deletes the team by ID.",
				},
			},
		},
		// Restore
		{
			Pattern: "team/" + uuid.Pattern("uuid") + "/restore" + "$",
			Fields: map[string]*framework.FieldSchema{
				"uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a team",
					Required:    true,
				},
				"full_restore": {
					Type:        framework.TypeBool,
					Description: "Option to restore full team data",
					Required:    false,
				},
			},
			ExistenceCheck: b.handleExistence(),
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.handleRestore(),
					Summary:  "Restore the team by ID.",
				},
			},
		},
	}
}

func (b *teamBackend) handleExistence() framework.ExistenceFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (bool, error) {
		id := data.Get("uuid").(string)
		b.Logger().Debug("checking team existence", "path", req.Path, "id", id, "op", req.Operation)

		if !uuid.IsValid(id) {
			return false, fmt.Errorf("id must be valid UUIDv4")
		}

		tx := b.storage.Txn(false)

		t, err := usecase.Teams(tx).GetByID(id)
		if err != nil {
			return false, err
		}
		return t != nil, nil
	}
}

func (b *teamBackend) handleCreate(expectID bool) framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		b.Logger().Debug("create team", "path", req.Path)
		id := getCreationID(expectID, data)
		team := &model.Team{
			UUID:           id,
			Identifier:     data.Get("identifier").(string),
			TeamType:       data.Get("team_type").(string),
			ParentTeamUUID: data.Get("parent_team_uuid").(string),
		}

		tx := b.storage.Txn(true)
		defer tx.Abort()

		if err := usecase.Teams(tx).Create(team); err != nil {
			msg := "cannot create team"
			b.Logger().Debug(msg, "err", err.Error())
			return logical.ErrorResponse(msg), nil
		}
		if err := commit(tx, b.Logger()); err != nil {
			return nil, err
		}

		resp := &logical.Response{Data: map[string]interface{}{"team": team}}
		return logical.RespondWithStatusCode(resp, req, http.StatusCreated)
	}
}

func (b *teamBackend) handleUpdate() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		b.Logger().Debug("update team", "path", req.Path)
		id := data.Get("uuid").(string)
		tx := b.storage.Txn(true)
		defer tx.Abort()

		team := &model.Team{
			UUID:           id,
			Identifier:     data.Get("identifier").(string),
			ParentTeamUUID: data.Get("parent_team_uuid").(string),
			Version:        data.Get("resource_version").(string),
		}

		err := usecase.Teams(tx).Update(team)
		if err != nil {
			return responseErr(req, err)
		}
		if err := commit(tx, b.Logger()); err != nil {
			return nil, err
		}

		resp := &logical.Response{Data: map[string]interface{}{"team": team}}
		return logical.RespondWithStatusCode(resp, req, http.StatusOK)
	}
}

func (b *teamBackend) handleDelete() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		b.Logger().Debug("delete team", "path", req.Path)
		tx := b.storage.Txn(true)
		defer tx.Abort()

		id := data.Get("uuid").(string)

		err := usecase.Teams(tx).Delete(id)
		if err != nil {
			return responseErr(req, err)
		}
		if err := commit(tx, b.Logger()); err != nil {
			return nil, err
		}

		return logical.RespondWithStatusCode(nil, req, http.StatusNoContent)
	}
}

func (b *teamBackend) handleRead() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		b.Logger().Debug("read team", "path", req.Path)
		id := data.Get("uuid").(string)

		tx := b.storage.Txn(false)

		team, err := usecase.Teams(tx).GetByID(id)
		if err != nil {
			return responseErr(req, err)
		}

		resp := &logical.Response{Data: map[string]interface{}{
			"team":         team,
			"full_restore": false, // TODO check if full restore available
		}}
		return logical.RespondWithStatusCode(resp, req, http.StatusOK)
	}
}

func (b *teamBackend) handleList() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		b.Logger().Debug("listing teams", "path", req.Path)
		var showArchived bool
		rawShowArchived, ok := data.GetOk("show_archived")
		if ok {
			showArchived = rawShowArchived.(bool)
		}

		tx := b.storage.Txn(false)
		teams, err := usecase.Teams(tx).List(showArchived)
		if err != nil {
			return nil, err
		}

		resp := &logical.Response{
			Data: map[string]interface{}{
				"teams": teams,
			},
		}
		return logical.RespondWithStatusCode(resp, req, http.StatusOK)
	}
}

func (b *teamBackend) handleRestore() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		b.Logger().Debug("restore team", "path", req.Path)
		tx := b.storage.Txn(true)
		defer tx.Abort()

		id := data.Get("uuid").(string)
		var fullRestore bool
		rawFullRestore, ok := data.GetOk("show_archived")
		if ok {
			fullRestore = rawFullRestore.(bool)
		}

		team, err := usecase.Teams(tx).Restore(id, fullRestore)
		if err != nil {
			return responseErr(req, err)
		}

		if err := commit(tx, b.Logger()); err != nil {
			return nil, err
		}

		resp := &logical.Response{Data: map[string]interface{}{
			"team": team,
		}}
		return logical.RespondWithStatusCode(resp, req, http.StatusOK)
	}
}
