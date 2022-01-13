package paths

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"

	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/usecase"
	iam_model "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	backentutils "github.com/flant/negentropy/vault-plugins/shared/backent-utils"
	"github.com/flant/negentropy/vault-plugins/shared/consts"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/uuid"
)

type contactBackend struct {
	*flantFlowExtension
}

func contactPaths(e *flantFlowExtension) []*framework.Path {
	bb := &contactBackend{
		flantFlowExtension: e,
	}
	return bb.paths()
}

func baseAndExtraFields(extraFields map[string]*framework.FieldSchema) map[string]*framework.FieldSchema {
	fs := map[string]*framework.FieldSchema{
		clientUUIDKey: {
			Type:        framework.TypeNameString,
			Description: "ID of a client",
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
		"credentials": {
			Type: framework.TypeKVPairs,
			Description: "credentials per projectUUID, allowed values: " +
				strings.TrimSuffix(fmt.Sprintf("%#v", model.AllowedContactRoles)[9:], "}"),
			Required: true,
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

func (b contactBackend) paths() []*framework.Path {
	return []*framework.Path{
		// Creation
		{
			Pattern: "client/" + uuid.Pattern(clientUUIDKey) + "/contact",
			Fields:  baseAndExtraFields(nil),
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.CreateOperation: &framework.PathOperation{
					Callback: b.checkConfigured(b.handleCreate(false)),
					Summary:  "Create contact.",
				},
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.checkConfigured(b.handleCreate(false)),
					Summary:  "Create contact.",
				},
			},
		},
		// Creation with known uuid in advance
		{
			Pattern: "client/" + uuid.Pattern(clientUUIDKey) + "/contact/privileged",
			Fields: baseAndExtraFields(map[string]*framework.FieldSchema{
				"uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a contact",
					Required:    true,
				},
			}),
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.CreateOperation: &framework.PathOperation{
					Callback: b.checkConfigured(b.handleCreate(true)),
					Summary:  "Create contact with preexistent ID.",
				},
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.checkConfigured(b.handleCreate(true)),
					Summary:  "Create contact with preexistent ID.",
				},
			},
		},
		// List
		{
			Pattern: "client/" + uuid.Pattern(clientUUIDKey) + "/contact/?",
			Fields: map[string]*framework.FieldSchema{
				clientUUIDKey: {
					Type:        framework.TypeNameString,
					Description: "ID of a client",
					Required:    true,
				},
				"show_archived": {
					Type:        framework.TypeBool,
					Description: "Option to list archived contacts",
					Required:    false,
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.ReadOperation: &framework.PathOperation{
					Callback: b.checkConfigured(b.handleList),
					Summary:  "Lists all contacts IDs.",
				},
			},
		},
		// Read, update, delete by uuid
		{

			Pattern: "client/" + uuid.Pattern(clientUUIDKey) + "/contact/" + uuid.Pattern("uuid") + "$",
			Fields: baseAndExtraFields(map[string]*framework.FieldSchema{
				"uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a contact",
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
					Callback: b.checkConfigured(b.handleUpdate),
					Summary:  "Update the contact by ID",
				},
				logical.ReadOperation: &framework.PathOperation{
					Callback: b.checkConfigured(b.handleRead),
					Summary:  "Retrieve the contact by ID",
				},
				logical.DeleteOperation: &framework.PathOperation{
					Callback: b.checkConfigured(b.handleDelete),
					Summary:  "Deletes the contact by ID",
				},
			},
		},
		// Restore
		{
			Pattern: "client/" + uuid.Pattern(clientUUIDKey) + "/contact/" + uuid.Pattern("uuid") + "/restore" + "$",
			Fields: map[string]*framework.FieldSchema{
				"uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a contact",
					Required:    true,
				},
				clientUUIDKey: {
					Type:        framework.TypeNameString,
					Description: "ID of a client",
					Required:    true,
				},
			},
			ExistenceCheck: b.handleExistence,
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.checkConfigured(b.handleRestore),
					Summary:  "Restore the contact by ID.",
				},
			},
		},
	}
}

// neverExisting  is a useful existence check handler to always trigger create operation
func neverExisting(context.Context, *logical.Request, *framework.FieldData) (bool, error) {
	return false, nil
}

func (b *contactBackend) handleExistence(_ context.Context, req *logical.Request, data *framework.FieldData) (bool, error) {
	id := data.Get("uuid").(string)
	clientID := data.Get(clientUUIDKey).(string)
	b.Logger().Debug("checking contact existence", "path", req.Path, "id", id, "op", req.Operation)

	if !uuid.IsValid(id) {
		return false, fmt.Errorf("id must be valid UUIDv4")
	}

	tx := b.storage.Txn(false)

	obj, err := usecase.Contacts(tx, clientID).GetByID(id)
	if err != nil {
		return false, err
	}
	exists := obj != nil && obj.TenantUUID == clientID
	return exists, nil
}

func (b *contactBackend) handleCreate(expectID bool) framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		b.Logger().Debug("create contact", "path", req.Path)
		id, err := backentutils.GetCreationID(expectID, data)
		if err != nil {
			return backentutils.ResponseErr(req, err)
		}
		clientID := data.Get(clientUUIDKey).(string)
		contact := &model.FullContact{
			User: iam_model.User{
				UUID:             id,
				TenantUUID:       clientID,
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
			Credentials: data.Get("credentials").(map[string]string),
		}

		tx := b.storage.Txn(true)
		defer tx.Abort()

		if err := usecase.Contacts(tx, clientID).Create(contact); err != nil {
			msg := "cannot create contact"
			b.Logger().Debug(msg, "err", err.Error())
			return logical.ErrorResponse(msg), nil
		}
		if err := io.CommitWithLog(tx, b.Logger()); err != nil {
			return nil, err
		}

		resp := &logical.Response{Data: map[string]interface{}{"contact": contact}}
		return logical.RespondWithStatusCode(resp, req, http.StatusCreated)
	}
}

func (b *contactBackend) handleUpdate(_ context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	b.Logger().Debug("update contact", "path", req.Path)
	id := data.Get("uuid").(string)
	clientID := data.Get(clientUUIDKey).(string)
	tx := b.storage.Txn(true)
	defer tx.Abort()

	contact := &model.FullContact{
		User: iam_model.User{
			UUID:             id,
			TenantUUID:       clientID,
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
		Credentials: data.Get("credentials").(map[string]string),
	}

	err := usecase.Contacts(tx, clientID).Update(contact)
	if err != nil {
		return backentutils.ResponseErr(req, err)
	}
	if err := io.CommitWithLog(tx, b.Logger()); err != nil {
		return nil, err
	}

	resp := &logical.Response{Data: map[string]interface{}{"contact": contact}}
	return logical.RespondWithStatusCode(resp, req, http.StatusOK)
}

func (b *contactBackend) handleDelete(_ context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	b.Logger().Debug("delete contact", "path", req.Path)
	id := data.Get("uuid").(string)
	clientID := data.Get(clientUUIDKey).(string)

	tx := b.storage.Txn(true)
	defer tx.Abort()

	// TODO pass origin to use in client
	err := usecase.Contacts(tx, clientID).Delete(id)
	if err != nil {
		return backentutils.ResponseErr(req, err)
	}
	if err := io.CommitWithLog(tx, b.Logger()); err != nil {
		return nil, err
	}

	return logical.RespondWithStatusCode(nil, req, http.StatusNoContent)
}

func (b *contactBackend) handleRead(_ context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	b.Logger().Debug("read contact", "path", req.Path)
	id := data.Get("uuid").(string)
	clientID := data.Get(clientUUIDKey).(string)

	tx := b.storage.Txn(false)

	contact, err := usecase.Contacts(tx, clientID).GetByID(id)
	if err != nil {
		return backentutils.ResponseErr(req, err)
	}

	resp := &logical.Response{Data: map[string]interface{}{"contact": contact}}
	return logical.RespondWithStatusCode(resp, req, http.StatusOK)
}

func (b *contactBackend) handleList(_ context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	b.Logger().Debug("list contacts", "path", req.Path)
	var showArchived bool
	rawShowArchived, ok := data.GetOk("show_archived")
	if ok {
		showArchived = rawShowArchived.(bool)
	}
	clientID := data.Get(clientUUIDKey).(string)

	tx := b.storage.Txn(false)

	contacts, err := usecase.Contacts(tx, clientID).List(showArchived)
	if err != nil {
		return nil, err
	}

	resp := &logical.Response{
		Data: map[string]interface{}{
			"contacts": contacts,
		},
	}
	return logical.RespondWithStatusCode(resp, req, http.StatusOK)
}

func (b *contactBackend) handleRestore(_ context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	b.Logger().Debug("restore contact", "path", req.Path)
	tx := b.storage.Txn(true)
	defer tx.Abort()

	id := data.Get("uuid").(string)
	clientID := data.Get(clientUUIDKey).(string)

	contact, err := usecase.Contacts(tx, clientID).Restore(id)
	if err != nil {
		return backentutils.ResponseErr(req, err)
	}

	if err := io.CommitWithLog(tx, b.Logger()); err != nil {
		return nil, err
	}

	resp := &logical.Response{Data: map[string]interface{}{
		"contact": contact,
	}}
	return logical.RespondWithStatusCode(resp, req, http.StatusOK)
}
