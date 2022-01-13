package paths

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"

	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/repo"
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/usecase"
	iam_model "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	backentutils "github.com/flant/negentropy/vault-plugins/shared/backent-utils"
	"github.com/flant/negentropy/vault-plugins/shared/consts"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/uuid"
)

type teammateBackend struct {
	*flantFlowExtension
}

func teammatePaths(e *flantFlowExtension) []*framework.Path {
	bb := &teammateBackend{
		flantFlowExtension: e,
	}
	return bb.paths()
}

func teammateBaseAndExtraFields(extraFields map[string]*framework.FieldSchema) map[string]*framework.FieldSchema {
	fs := map[string]*framework.FieldSchema{
		"team_uuid": {
			Type:        framework.TypeNameString,
			Description: "ID of a team",
			Required:    true,
		},
		"identifier": {
			Type:        framework.TypeNameString,
			Description: "Identifier for humans and machines",
			Required:    true,
		},
		"first_name": {
			Type:        framework.TypeString,
			Description: "first_name",
			Required:    true,
		},
		"last_name": {
			Type:        framework.TypeString,
			Description: "last_name",
			Required:    true,
		},
		"display_name": {
			Type:        framework.TypeString,
			Description: "display_name",
			Required:    true,
		},
		"email": {
			Type:        framework.TypeString,
			Description: "email",
			Required:    true,
		},
		"additional_emails": {
			Type:        framework.TypeStringSlice,
			Description: "additional_emails",
			Required:    true,
		},
		"mobile_phone": {
			Type:        framework.TypeString,
			Description: "mobile_phone",
			Required:    true,
		},
		"additional_phones": {
			Type:        framework.TypeStringSlice,
			Description: "additional_phones",
			Required:    true,
		},
		"role_at_team": {
			Type:          framework.TypeString,
			Description:   "role at team",
			Required:      true,
			AllowedValues: model.AllowedRolesAtTeam,
		},
	}
	for fieldName, fieldSchema := range extraFields {
		if _, alreadyDefined := fs[fieldName]; alreadyDefined {
			panic(fmt.Sprintf("path_contact wrong schema: duplicate field name:%s", fieldName))
		}
		fs[fieldName] = fieldSchema
	}
	return fs
}

func (b teammateBackend) paths() []*framework.Path {
	return []*framework.Path{
		// Creation
		{
			Pattern: "team/" + uuid.Pattern("team_uuid") + "/teammate",
			Fields:  teammateBaseAndExtraFields(nil),
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.CreateOperation: &framework.PathOperation{
					Callback: b.checkBaseConfigured(b.handleCreate(false)),
					Summary:  "Create teammate.",
				},
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.checkBaseConfigured(b.handleCreate(false)),
					Summary:  "Create teammate.",
				},
			},
		},
		// Creation with known uuid in advance
		{
			Pattern: "team/" + uuid.Pattern("team_uuid") + "/teammate/privileged",
			Fields: teammateBaseAndExtraFields(map[string]*framework.FieldSchema{
				"uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a teammate",
					Required:    true,
				},
			}),
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.CreateOperation: &framework.PathOperation{
					Callback: b.checkBaseConfigured(b.handleCreate(true)),
					Summary:  "Create teammate with preexistent ID.",
				},
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.checkBaseConfigured(b.handleCreate(true)),
					Summary:  "Create teammate with preexistent ID.",
				},
			},
		},
		// List
		{
			Pattern: "team/" + uuid.Pattern("team_uuid") + "/teammate/?",
			Fields: map[string]*framework.FieldSchema{
				"team_uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a team",
					Required:    true,
				},
				"show_archived": {
					Type:        framework.TypeBool,
					Description: "Option to list archived teammates",
					Required:    false,
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.ReadOperation: &framework.PathOperation{
					Callback: b.checkBaseConfigured(b.handleList),
					Summary:  "Lists all teammates IDs.",
				},
			},
		},
		// Read, update, delete by uuid
		{
			Pattern: "team/" + uuid.Pattern("team_uuid") + "/teammate/" + uuid.Pattern("uuid") + "$",
			Fields: teammateBaseAndExtraFields(map[string]*framework.FieldSchema{
				"uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a teammate",
					Required:    true,
				},
				"resource_version": {
					Type:        framework.TypeString,
					Description: "Resource version",
					Required:    true,
				},
			}),
			ExistenceCheck: b.handleExistence,
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.checkBaseConfigured(b.handleUpdate),
					Summary:  "Update the teammate by ID",
				},
				logical.ReadOperation: &framework.PathOperation{
					Callback: b.checkBaseConfigured(b.handleRead),
					Summary:  "Retrieve the teammate by ID",
				},
				logical.DeleteOperation: &framework.PathOperation{
					Callback: b.checkBaseConfigured(b.handleDelete),
					Summary:  "Deletes the teammate by ID",
				},
			},
		},
		// Restore
		{
			Pattern: "team/" + uuid.Pattern("team_uuid") + "/teammate/" + uuid.Pattern("uuid") + "/restore" + "$",
			Fields: map[string]*framework.FieldSchema{
				"uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a teammate",
					Required:    true,
				},
				"team_uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a team",
					Required:    true,
				},
			},
			ExistenceCheck: b.handleExistence,
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.checkBaseConfigured(b.handleRestore),
					Summary:  "Restore the teammate by ID.",
				},
			},
		},
	}
}

func (b *teammateBackend) handleExistence(_ context.Context, req *logical.Request, data *framework.FieldData) (bool, error) {
	id := data.Get("uuid").(string)
	teamID := data.Get(repo.TeamForeignPK).(string)
	b.Logger().Debug("checking teammate existence", "path", req.Path, "id", id, "op", req.Operation)

	if !uuid.IsValid(id) {
		return false, fmt.Errorf("id must be valid UUIDv4")
	}

	tx := b.storage.Txn(false)

	obj, err := usecase.Teammates(tx, b.getLiveConfig().FlantTenantUUID).GetByID(id)
	if err != nil {
		return false, err
	}
	exists := obj != nil && obj.TeamUUID == teamID
	return exists, nil
}

func (b *teammateBackend) handleCreate(expectID bool) framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		b.Logger().Debug("create teammate", "path", req.Path)
		id, err := backentutils.GetCreationID(expectID, data)
		if err != nil {
			return backentutils.ResponseErr(req, err)
		}
		teamID := data.Get(repo.TeamForeignPK).(string)
		teammate := &model.FullTeammate{
			User: iam_model.User{
				UUID:             id,
				TenantUUID:       b.getLiveConfig().FlantTenantUUID,
				Identifier:       data.Get("identifier").(string),
				FirstName:        data.Get("first_name").(string),
				LastName:         data.Get("last_name").(string),
				DisplayName:      data.Get("display_name").(string),
				Email:            data.Get("email").(string),
				AdditionalEmails: data.Get("additional_emails").([]string),
				MobilePhone:      data.Get("mobile_phone").(string),
				AdditionalPhones: data.Get("additional_phones").([]string),
				Origin:           consts.OriginFlantFlow,
			},
			TeamUUID:   teamID,
			RoleAtTeam: data.Get("role_at_team").(string),
		}

		tx := b.storage.Txn(true)
		defer tx.Abort()

		if err := usecase.Teammates(tx, b.getLiveConfig().FlantTenantUUID).Create(teammate); err != nil {
			msg := "cannot create teammate"
			b.Logger().Debug(msg, "err", err.Error())
			err = fmt.Errorf("%s:%w", msg, err)
			return backentutils.ResponseErr(req, err)
		}
		if err := io.CommitWithLog(tx, b.Logger()); err != nil {
			return nil, err
		}

		resp := &logical.Response{Data: map[string]interface{}{"teammate": teammate}}
		return logical.RespondWithStatusCode(resp, req, http.StatusCreated)
	}
}

func (b *teammateBackend) handleUpdate(_ context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	b.Logger().Debug("update teammate", "path", req.Path)
	id := data.Get("uuid").(string)
	teamID := data.Get(repo.TeamForeignPK).(string)
	tx := b.storage.Txn(true)
	defer tx.Abort()

	teammate := &model.FullTeammate{
		User: iam_model.User{
			UUID:             id,
			TenantUUID:       b.getLiveConfig().FlantTenantUUID,
			Identifier:       data.Get("identifier").(string),
			FirstName:        data.Get("first_name").(string),
			LastName:         data.Get("last_name").(string),
			DisplayName:      data.Get("display_name").(string),
			Email:            data.Get("email").(string),
			AdditionalEmails: data.Get("additional_emails").([]string),
			MobilePhone:      data.Get("mobile_phone").(string),
			AdditionalPhones: data.Get("additional_phones").([]string),
			Version:          data.Get("resource_version").(string),
			Origin:           consts.OriginFlantFlow,
		},
		TeamUUID:   teamID,
		RoleAtTeam: data.Get("role_at_team").(string),
	}

	err := usecase.Teammates(tx, b.getLiveConfig().FlantTenantUUID).Update(teammate)
	if err != nil {
		return backentutils.ResponseErr(req, err)
	}
	if err := io.CommitWithLog(tx, b.Logger()); err != nil {
		return nil, err
	}

	resp := &logical.Response{Data: map[string]interface{}{"teammate": teammate}}
	return logical.RespondWithStatusCode(resp, req, http.StatusOK)
}

func (b *teammateBackend) handleDelete(_ context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	b.Logger().Debug("delete teammate", "path", req.Path)
	id := data.Get("uuid").(string)

	tx := b.storage.Txn(true)
	defer tx.Abort()
	err := usecase.Teammates(tx, b.getLiveConfig().FlantTenantUUID).Delete(id)
	if err != nil {
		return backentutils.ResponseErr(req, err)
	}
	if err := io.CommitWithLog(tx, b.Logger()); err != nil {
		return nil, err
	}

	return logical.RespondWithStatusCode(nil, req, http.StatusNoContent)
}

func (b *teammateBackend) handleRead(_ context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	b.Logger().Debug("read teammate", "path", req.Path)
	id := data.Get("uuid").(string)

	tx := b.storage.Txn(false)

	teammate, err := usecase.Teammates(tx, b.getLiveConfig().FlantTenantUUID).GetByID(id)
	if err != nil {
		return backentutils.ResponseErr(req, err)
	}

	resp := &logical.Response{Data: map[string]interface{}{"teammate": teammate}}
	return logical.RespondWithStatusCode(resp, req, http.StatusOK)
}

func (b *teammateBackend) handleList(_ context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	b.Logger().Debug("list teammates", "path", req.Path)
	var showArchived bool
	rawShowArchived, ok := data.GetOk("show_archived")
	if ok {
		showArchived = rawShowArchived.(bool)
	}

	tx := b.storage.Txn(false)
	teamID := data.Get(repo.TeamForeignPK).(string)
	teammates, err := usecase.Teammates(tx, b.getLiveConfig().FlantTenantUUID).List(teamID, showArchived)
	if err != nil {
		return nil, err
	}

	resp := &logical.Response{
		Data: map[string]interface{}{
			"teammates": teammates,
		},
	}
	return logical.RespondWithStatusCode(resp, req, http.StatusOK)
}

func (b *teammateBackend) handleRestore(_ context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	b.Logger().Debug("restore teammate", "path", req.Path)
	tx := b.storage.Txn(true)
	defer tx.Abort()

	id := data.Get("uuid").(string)

	teammate, err := usecase.Teammates(tx, b.getLiveConfig().FlantTenantUUID).Restore(id)
	if err != nil {
		return backentutils.ResponseErr(req, err)
	}

	if err := io.CommitWithLog(tx, b.Logger()); err != nil {
		return nil, err
	}

	resp := &logical.Response{Data: map[string]interface{}{
		"teammate": teammate,
	}}
	return logical.RespondWithStatusCode(resp, req, http.StatusOK)
}
