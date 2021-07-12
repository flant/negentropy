package backend

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/uuid"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

type roleBindingApprovalBackend struct {
	logical.Backend
	storage *io.MemoryStore
}

func roleBindingApprovalPaths(b logical.Backend, storage *io.MemoryStore) []*framework.Path {
	bb := &roleBindingApprovalBackend{
		Backend: b,
		storage: storage,
	}
	return bb.paths()
}

func (b roleBindingApprovalBackend) paths() []*framework.Path {
	return []*framework.Path{
		{
			// Read, update, delete by uuid
			Pattern: "tenant/" + uuid.Pattern("tenant_uuid") + "/role_binding/" + uuid.Pattern("role_binding_uuid") + "/approval/" + uuid.Pattern("uuid") + "$",
			Fields: map[string]*framework.FieldSchema{
				"role_binding_uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a roleBinding",
					Required:    true,
				},
				"tenant_uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a tenant",
					Required:    true,
				},
				"uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a role binding approval",
					Required:    true,
				},

				// body params
				"users": {
					Type:        framework.TypeStringSlice,
					Description: "Array of user IDs.",
				},
				"groups": {
					Type:        framework.TypeStringSlice,
					Description: "Array of group IDs.",
				},
				"service_accounts": {
					Type:        framework.TypeStringSlice,
					Description: "Array of service account IDs.",
				},
				"required_votes": {
					Type:        framework.TypeInt,
					Description: "Cound of required approves.",
					Required:    true,
				},
				"require_mfa": {
					Type:        framework.TypeBool,
					Description: "Necessity to approve second auth factor.",
					Default:     false,
				},
				"require_unique_approver": {
					Type:        framework.TypeBool,
					Description: "Whether the approver is required to be unique among all approvals.",
					Default:     true,
				},
			},
			ExistenceCheck: b.handleExistence(),
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.handleUpdate(),
					Summary:  "Update the role binding approval by ID.",
				},
				logical.ReadOperation: &framework.PathOperation{
					Callback: b.handleRead(),
					Summary:  "Retrieve the role binding approval by ID.",
				},
				logical.DeleteOperation: &framework.PathOperation{
					Callback: b.handleDelete(),
					Summary:  "Deletes the role binding approval by ID.",
				},
			},
		},
	}
}

func (b *roleBindingApprovalBackend) handleExistence() framework.ExistenceFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (bool, error) {
		tenantID := data.Get(model.TenantForeignPK).(string)
		roleBindingID := data.Get(model.RoleBindingForeignPK).(string)

		if !uuid.IsValid(roleBindingID) {
			return false, fmt.Errorf("roleBindingID must be valid UUIDv4")
		}

		tx := b.storage.Txn(false)
		repo := model.NewRoleBindingRepository(tx)

		rb, err := repo.GetByID(roleBindingID)
		if err != nil {
			return false, err
		}
		exists := rb != nil && rb.TenantUUID == tenantID

		return exists, nil
	}
}

func (b *roleBindingApprovalBackend) handleUpdate() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		id := data.Get("uuid").(string)
		userIDs := data.Get("users").([]string)
		groupIDs := data.Get("groups").([]string)
		serviceAccountIDs := data.Get("service_accounts").([]string)
		requiredVotes := data.Get("required_votes").(int)
		requireMFA := data.Get("require_mfa").(bool)
		requireUniqueApprover := data.Get("require_unique_approver").(bool)

		if requiredVotes <= 0 {
			return nil, logical.CodedError(http.StatusBadRequest, "required_votes must be greater then zero")
		}

		roleBindingApproval := &model.RoleBindingApproval{
			UUID:                  id,
			TenantUUID:            data.Get(model.TenantForeignPK).(string),
			RoleBindingUUID:       data.Get(model.RoleBindingForeignPK).(string),
			Users:                 userIDs,
			Groups:                groupIDs,
			ServiceAccounts:       serviceAccountIDs,
			RequiredVotes:         requiredVotes,
			RequireMFA:            requireMFA,
			RequireUniqueApprover: requireUniqueApprover,
		}

		tx := b.storage.Txn(true)
		defer tx.Abort()

		repo := model.NewRoleBindingApprovalRepository(tx)
		err := repo.Update(roleBindingApproval)
		if err != nil {
			return responseErr(req, err)
		}
		if err := commit(tx, b.Logger()); err != nil {
			return nil, err
		}

		resp := &logical.Response{Data: map[string]interface{}{"role_binding_approval": roleBindingApproval}}
		return logical.RespondWithStatusCode(resp, req, http.StatusOK)
	}
}

func (b *roleBindingApprovalBackend) handleDelete() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		id := data.Get("uuid").(string)

		tx := b.storage.Txn(true)
		defer tx.Abort()
		repo := model.NewRoleBindingApprovalRepository(tx)

		err := repo.Delete(id)
		if err != nil {
			return responseErr(req, err)
		}
		if err := commit(tx, b.Logger()); err != nil {
			return nil, err
		}

		return logical.RespondWithStatusCode(nil, req, http.StatusNoContent)
	}
}

func (b *roleBindingApprovalBackend) handleRead() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		id := data.Get("uuid").(string)

		tx := b.storage.Txn(false)
		repo := model.NewRoleBindingApprovalRepository(tx)

		roleBindingApproval, err := repo.GetByID(id)
		if err != nil {
			return responseErr(req, err)
		}

		resp := &logical.Response{Data: map[string]interface{}{"role_binding_approval": roleBindingApproval}}
		return logical.RespondWithStatusCode(resp, req, http.StatusOK)
	}
}

func (b *roleBindingApprovalBackend) handleList() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		tenantID := data.Get(model.TenantForeignPK).(string)
		rbID := data.Get(model.RoleBindingForeignPK).(string)

		tx := b.storage.Txn(false)
		repo := model.NewRoleBindingApprovalRepository(tx)

		uuids := make([]string, 0)
		err := repo.Iter(func(approval *model.RoleBindingApproval) (bool, error) {
			if approval.TenantUUID == tenantID && approval.RoleBindingUUID == rbID {
				uuids = append(uuids, approval.UUID)
			}

			return true, nil
		})
		if err != nil {
			return nil, err
		}

		resp := &logical.Response{
			Data: map[string]interface{}{
				"uuids": uuids,
			},
		}
		return resp, nil
	}
}
