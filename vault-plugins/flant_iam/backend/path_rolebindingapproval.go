package backend

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	iam_repo "github.com/flant/negentropy/vault-plugins/flant_iam/repo"
	"github.com/flant/negentropy/vault-plugins/flant_iam/usecase"
	backentutils "github.com/flant/negentropy/vault-plugins/shared/backent-utils"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/uuid"
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

func rbaBaseAndExtraFields(extraFields map[string]*framework.FieldSchema) map[string]*framework.FieldSchema {
	fs := map[string]*framework.FieldSchema{
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
		// body params
		"approvers": {
			Type:        framework.TypeSlice,
			Description: "Approvers list",
			Required:    true,
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
	}
	for fieldName, fieldSchema := range extraFields {
		if _, alreadyDefined := fs[fieldName]; alreadyDefined {
			panic(fmt.Sprintf("path_rolebinding_approval wrong schema: duplicate field name:%s", fieldName))
		}
		fs[fieldName] = fieldSchema
	}
	return fs
}

func (b roleBindingApprovalBackend) paths() []*framework.Path {
	return []*framework.Path{
		{
			// Create
			Pattern:        "tenant/" + uuid.Pattern("tenant_uuid") + "/role_binding/" + uuid.Pattern("role_binding_uuid") + "/approval" + "$",
			Fields:         rbaBaseAndExtraFields(nil),
			ExistenceCheck: b.handleExistence(),
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.handleCreate(),
					Summary:  "Create the role binding approval.",
				},
				logical.CreateOperation: &framework.PathOperation{
					Callback: b.handleCreate(),
					Summary:  "Create the role binding approval.",
				},
			},
		},
		{
			// Read, update, delete by uuid
			Pattern: "tenant/" + uuid.Pattern("tenant_uuid") + "/role_binding/" + uuid.Pattern("role_binding_uuid") + "/approval/" + uuid.Pattern("uuid") + "$",
			Fields: rbaBaseAndExtraFields(map[string]*framework.FieldSchema{
				"uuid": {
					Type:        framework.TypeNameString,
					Description: "ID of a role binding approval",
					Required:    true,
				},
				"resource_version": {
					Type:        framework.TypeString,
					Description: "Resource version",
					Required:    true,
				},
			}),
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
		// List
		{
			Pattern: "tenant/" + uuid.Pattern("tenant_uuid") + "/role_binding/" + uuid.Pattern("role_binding_uuid") + "/approval/?",
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
				"show_archived": {
					Type:        framework.TypeBool,
					Description: "Option to list archived approvals",
					Required:    false,
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.ReadOperation: &framework.PathOperation{
					Callback: b.handleList(),
					Summary:  "Lists all approvals for role_binding",
				},
			},
		},
	}
}

func (b *roleBindingApprovalBackend) handleExistence() framework.ExistenceFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (bool, error) {
		tenantID := data.Get(iam_repo.TenantForeignPK).(string)
		roleBindingID := data.Get(iam_repo.RoleBindingForeignPK).(string)

		if !uuid.IsValid(roleBindingID) {
			return false, fmt.Errorf("roleBindingID must be valid UUIDv4")
		}

		tx := b.storage.Txn(false)

		rb, err := usecase.RoleBindings(tx).GetByID(roleBindingID)
		if err != nil {
			return false, err
		}
		exists := rb != nil && rb.TenantUUID == tenantID

		return exists, nil
	}
}

func (b *roleBindingApprovalBackend) handleCreate() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		b.Logger().Debug("create role_binding_approval", "path", req.Path)
		id := uuid.New()
		requiredVotes := data.Get("required_votes").(int)
		requireMFA := data.Get("require_mfa").(bool)
		requireUniqueApprover := data.Get("require_unique_approver").(bool)

		if requiredVotes <= 0 {
			return nil, logical.CodedError(http.StatusBadRequest, "required_votes must be greater then zero")
		}

		approvers, err := parseMembers(data.Get("approvers"))

		roleBindingApproval := &model.RoleBindingApproval{
			UUID:                  id,
			TenantUUID:            data.Get(iam_repo.TenantForeignPK).(string),
			RoleBindingUUID:       data.Get(iam_repo.RoleBindingForeignPK).(string),
			Approvers:             approvers,
			RequiredVotes:         requiredVotes,
			RequireMFA:            requireMFA,
			RequireUniqueApprover: requireUniqueApprover,
		}

		tx := b.storage.Txn(true)
		defer tx.Abort()

		err = usecase.RoleBindingApprovals(tx).Create(roleBindingApproval)
		if err != nil {
			return backentutils.ResponseErr(req, err)
		}
		if err = io.CommitWithLog(tx, b.Logger()); err != nil {
			return backentutils.ResponseErrMessage(req, err.Error(), http.StatusInternalServerError)
		}

		resp := &logical.Response{Data: map[string]interface{}{"approval": roleBindingApproval}}
		return logical.RespondWithStatusCode(resp, req, http.StatusCreated)
	}
}

func (b *roleBindingApprovalBackend) handleUpdate() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		b.Logger().Debug("update role_binding_approval", "path", req.Path)
		id := data.Get("uuid").(string)
		if id == "" {
			id = uuid.New()
		}
		requiredVotes := data.Get("required_votes").(int)
		requireMFA := data.Get("require_mfa").(bool)
		requireUniqueApprover := data.Get("require_unique_approver").(bool)

		if requiredVotes <= 0 {
			return nil, logical.CodedError(http.StatusBadRequest, "required_votes must be greater then zero")
		}
		approvers, err := parseMembers(data.Get("approvers"))
		if err != nil {
			return backentutils.ResponseErrMessage(req, err.Error(), http.StatusBadRequest)
		}

		roleBindingApproval := &model.RoleBindingApproval{
			UUID:                  id,
			TenantUUID:            data.Get(iam_repo.TenantForeignPK).(string),
			RoleBindingUUID:       data.Get(iam_repo.RoleBindingForeignPK).(string),
			Version:               data.Get("resource_version").(string),
			Approvers:             approvers,
			RequiredVotes:         requiredVotes,
			RequireMFA:            requireMFA,
			RequireUniqueApprover: requireUniqueApprover,
		}

		tx := b.storage.Txn(true)
		defer tx.Abort()

		err = usecase.RoleBindingApprovals(tx).Update(roleBindingApproval)
		if err != nil {
			return backentutils.ResponseErr(req, err)
		}
		if err = io.CommitWithLog(tx, b.Logger()); err != nil {
			return backentutils.ResponseErrMessage(req, err.Error(), http.StatusInternalServerError)
		}

		resp := &logical.Response{Data: map[string]interface{}{"approval": roleBindingApproval}}
		return logical.RespondWithStatusCode(resp, req, http.StatusOK)
	}
}

func (b *roleBindingApprovalBackend) handleDelete() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		b.Logger().Debug("delete role_binding_approval", "path", req.Path)
		id := data.Get("uuid").(string)

		tx := b.storage.Txn(true)
		defer tx.Abort()

		err := usecase.RoleBindingApprovals(tx).Delete(id)
		if err != nil {
			return backentutils.ResponseErr(req, err)
		}
		if err = io.CommitWithLog(tx, b.Logger()); err != nil {
			return backentutils.ResponseErrMessage(req, err.Error(), http.StatusInternalServerError)
		}

		return logical.RespondWithStatusCode(nil, req, http.StatusNoContent)
	}
}

func (b *roleBindingApprovalBackend) handleRead() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		b.Logger().Debug("read role_binding_approval", "path", req.Path)
		id := data.Get("uuid").(string)

		tx := b.storage.Txn(true) // need writable for fixing approvers
		defer tx.Abort()

		roleBindingApproval, err := usecase.RoleBindingApprovals(tx).GetByID(id)
		if err != nil {
			return backentutils.ResponseErr(req, err)
		}
		if err = io.CommitWithLog(tx, b.Logger()); err != nil {
			return backentutils.ResponseErrMessage(req, err.Error(), http.StatusInternalServerError)
		}

		resp := &logical.Response{Data: map[string]interface{}{"approval": roleBindingApproval}}
		return logical.RespondWithStatusCode(resp, req, http.StatusOK)
	}
}

func (b *roleBindingApprovalBackend) handleList() framework.OperationFunc {
	return func(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
		b.Logger().Debug("list role_binding_approval", "path", req.Path)
		var showArchived bool
		rawShowArchived, ok := data.GetOk("show_archived")
		if ok {
			showArchived = rawShowArchived.(bool)
		}
		rbID := data.Get(iam_repo.RoleBindingForeignPK).(string)

		tx := b.storage.Txn(true) // need writable for fixing approvers
		defer tx.Abort()

		rolebindingApprovals, err := usecase.RoleBindingApprovals(tx).List(rbID, showArchived)
		if err != nil {
			return backentutils.ResponseErrMessage(req, err.Error(), http.StatusInternalServerError)
		}
		if err = io.CommitWithLog(tx, b.Logger()); err != nil {
			return backentutils.ResponseErrMessage(req, err.Error(), http.StatusInternalServerError)
		}

		resp := &logical.Response{
			Data: map[string]interface{}{
				"approvals": rolebindingApprovals,
			},
		}
		return logical.RespondWithStatusCode(resp, req, http.StatusOK)
	}
}
