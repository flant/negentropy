package backend

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	log "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/helper/jsonutil"
	"github.com/hashicorp/vault/sdk/logical"

	"github.com/flant/negentropy/explugin/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

type userBackend struct {
	logical.Backend
	storage *io.MemoryStore
}

func userPaths(b logical.Backend, storage *io.MemoryStore) []*framework.Path {
	bb := &userBackend{
		Backend: b,
		storage: storage,
	}
	return bb.paths()
}

func (b userBackend) paths() []*framework.Path {
	return []*framework.Path{
		// Creation
		{
			Pattern: "user",
			Fields: map[string]*framework.FieldSchema{
				"identifier": {
					Type:        framework.TypeNameString,
					Description: "Identifier for humans and machines",
					Required:    true,
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.CreateOperation: &framework.PathOperation{
					Callback: b.handleCreate(false),
					Summary:  "Create user.",
				},
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.handleCreate(false),
					Summary:  "Create user.",
				},
			},
		},
		// Creation with known uuid in advance
		// Listing
		{
			Pattern: "user/?$",
			Fields:  map[string]*framework.FieldSchema{},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.ListOperation: &framework.PathOperation{
					Callback: b.handleList(),
					Summary:  "Lists all users IDs.",
				},
			},
		},
	}
}

func (b *userBackend) handleCreate(expectID bool) framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		id := getCreationID(expectID, data)
		user := &model.User{
			UUID:       id,
			Identifier: data.Get("identifier").(string),
			TenantUUID: "0000",
			Version:    uuid.NewString(),
		}

		tx := b.storage.Txn(true)
		defer tx.Abort()

		err := tx.Insert(model.UserType, user)
		if err != nil {
			return logical.ErrorResponse("cannot create user: %s", err), nil
		}

		if err := commit(tx, b.Logger()); err != nil {
			return nil, err
		}

		return responseWithDataAndCode(req, user, http.StatusCreated)
	}
}

func (b *userBackend) handleList() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		tx := b.storage.Txn(false)

		iter, err := tx.Get(model.UserType, model.PK)
		if err != nil {
			b.Logger().Error("get users", "error", err)
			return nil, err
		}

		ids := make([]string, 0)
		for {
			raw := iter.Next()
			if raw == nil {
				break
			}
			u := raw.(*model.User)
			ids = append(ids, u.UUID)
		}

		resp := &logical.Response{
			Data: map[string]interface{}{
				"uuids": ids,
			},
		}
		return resp, nil
	}
}

func Pattern(name string) string {
	const (
		uuidPattern = "(?i:[0-9A-F]{8}-[0-9A-F]{4}-[4][0-9A-F]{3}-[89AB][0-9A-F]{3}-[0-9A-F]{12})"
	)
	return fmt.Sprintf(`(?P<%s>%s)`, name, uuidPattern)
}

func getCreationID(expectID bool, data *framework.FieldData) string {
	var id string

	if expectID {
		// for privileged access
		id = data.Get("uuid").(string)
	}

	if id == "" {
		id = uuid.New().String()
	}

	return id
}

func commit(tx *io.MemoryStoreTxn, logger log.Logger) error {
	err := tx.Commit()
	if err != nil {
		logger.Error("failed to commit", "err", err)
		return fmt.Errorf("request failed, try again")
	}
	return nil
}

func responseWithDataAndCode(req *logical.Request, m model.Marshaller, status int) (*logical.Response, error) {
	resp, err := responseWithData(m)
	if err != nil {
		return nil, err
	}
	return logical.RespondWithStatusCode(resp, req, status)
}

func responseWithData(m model.Marshaller) (*logical.Response, error) {
	json, err := m.Marshal(false)
	if err != nil {
		return nil, err
	}

	var data map[string]interface{}
	err = jsonutil.DecodeJSON(json, &data)
	if err != nil {
		return nil, err
	}

	resp := &logical.Response{
		Data: data,
	}

	return resp, err
}
