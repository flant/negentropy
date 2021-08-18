package jwtauth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"

	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/model"
	repo2 "github.com/flant/negentropy/vault-plugins/flant_iam_auth/repo"
	backendutils "github.com/flant/negentropy/vault-plugins/shared/backent-utils"
	"github.com/flant/negentropy/vault-plugins/shared/openapi"
	"github.com/flant/negentropy/vault-plugins/shared/utils"
)

const HttpPathJwtType = "jwt_type"

func pathJwtTypeList(b *flantIamAuthBackend) *framework.Path {
	return &framework.Path{
		Pattern: fmt.Sprintf("%s/?", HttpPathJwtType),
		Operations: map[logical.Operation]framework.OperationHandler{
			logical.ListOperation: &framework.PathOperation{
				Callback:    b.pathJwtTypesList,
				Summary:     "Lists all jwt types registered with the backend",
				Description: "The list will contain the names of the jwt token types.",
			},
		},
	}
}

// pathRole returns the path configurations for the CRUD operations on roles
func pathJwtType(b *flantIamAuthBackend) *framework.Path {
	p := &framework.Path{
		Pattern: fmt.Sprintf("%s/", HttpPathJwtType) + framework.GenericNameRegex("name"),
		Fields: map[string]*framework.FieldSchema{
			"name": {
				Type:        framework.TypeLowerCaseString,
				Description: "Name of the jwt type",
			},
			"ttl": {
				Type:        framework.TypeDurationSecond,
				Description: "Time to life (in seconds) for signed jwt token",
				Required:    true,
			},
			"options_schema": {
				Type:        framework.TypeString,
				Description: "OpenApi schema for validating issuer params",
			},
			// todo rego policy
		},

		ExistenceCheck: b.pathJwtTypeExistenceCheck,
		Operations: map[logical.Operation]framework.OperationHandler{
			logical.ReadOperation: &framework.PathOperation{
				Callback: b.pathJwtTypeRead,
				Summary:  "Read an existing jwt type",
			},

			logical.UpdateOperation: &framework.PathOperation{
				Callback: b.pathJwtTypeCreateUpdate,
				Summary:  "Update an existing jwt type",
			},

			logical.CreateOperation: &framework.PathOperation{
				Callback: b.pathJwtTypeCreateUpdate,
				Summary:  "Crete new jwt type",
			},

			logical.DeleteOperation: &framework.PathOperation{
				Callback: b.pathJwtTypeDelete,
				Summary:  "Delete an existing jwt type",
			},
		},
		HelpSynopsis:    "Manage jwt token types",
		HelpDescription: "Jwt token types using for issue jwt tokens on /issue/* methods",
	}

	return p
}

// pathJwtTypeExistenceCheck returns whether the JwtTypeConfig with the given name exists or not.
func (b *flantIamAuthBackend) pathJwtTypeExistenceCheck(ctx context.Context, req *logical.Request, data *framework.FieldData) (bool, error) {
	typeName := data.Get("name").(string)

	tnx := b.storage.Txn(false)
	repo := repo2.NewJWTIssueTypeRepo(tnx)
	jwtType, err := repo.Get(typeName)
	if err != nil {
		return false, err
	}
	return jwtType != nil, nil
}

func (b *flantIamAuthBackend) pathJwtTypesList(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	tnx := b.storage.Txn(false)
	repo := repo2.NewJWTIssueTypeRepo(tnx)

	var typesNames []string
	err := repo.Iter(func(s *model.JWTIssueType) (bool, error) {
		typesNames = append(typesNames, s.Name)
		return true, nil
	})
	if err != nil {
		return nil, err
	}

	return logical.ListResponse(typesNames), nil
}

// pathJwtTypeRead grabs a read lock and reads the options set on the JwtTypeConfig from the storage
func (b *flantIamAuthBackend) pathJwtTypeRead(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	name := data.Get("name").(string)
	if name == "" {
		return logical.ErrorResponse("missing name"), nil
	}

	tnx := b.storage.Txn(false)
	repo := repo2.NewJWTIssueTypeRepo(tnx)
	jwtType, err := repo.Get(name)
	if err != nil {
		return nil, err
	}
	if jwtType == nil {
		return nil, nil
	}

	// Create a map of data to be returned
	d := map[string]interface{}{
		"uuid": jwtType.UUID,
		"name": jwtType.Name,
		"ttl":  fmt.Sprintf("%ds", int64(jwtType.TTL.Seconds())),

		"options_schema": jwtType.OptionsSchema,
	}

	return &logical.Response{
		Data: d,
	}, nil
}

// pathJwtTypeDelete removes the JwtTypeConfig from storage
func (b *flantIamAuthBackend) pathJwtTypeDelete(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	name := data.Get("name").(string)
	if name == "" {
		return logical.ErrorResponse("jwt type name is required"), nil
	}

	tnx := b.storage.Txn(true)
	defer tnx.Abort()

	repo := repo2.NewJWTIssueTypeRepo(tnx)

	val, err := repo.Get(name)
	if err != nil {
		return nil, err
	}

	if val == nil {
		return nil, nil
	}

	err = repo.Delete(name)
	if err != nil {
		return nil, err
	}

	err = tnx.Commit()
	if err != nil {
		return nil, err
	}

	return nil, nil
}

// pathJwtTypeCreateUpdate registers a new JwtTypeConfig with the backend or updates the options
// of an existing JwtTypeConfig
func (b *flantIamAuthBackend) pathJwtTypeCreateUpdate(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	name, errResp := backendutils.NotEmptyStringParam(data, "name")
	if errResp != nil {
		return errResp, nil
	}

	tnx := b.storage.Txn(true)
	defer tnx.Abort()

	repo := repo2.NewJWTIssueTypeRepo(tnx)
	// Check if the auth already exists
	jwtType, err := repo.Get(name)
	if err != nil {
		return nil, err
	}

	// Create a new entry object if this is a CreateOperation
	if jwtType == nil {
		if req.Operation == logical.UpdateOperation {
			return nil, errors.New("jwt type entry not found during update operation")
		}
		jwtType = new(model.JWTIssueType)
		jwtType.UUID = utils.UUID()
		jwtType.Name = name
	}

	ttlRaw, ok := data.GetOk("ttl")

	if req.Operation == logical.CreateOperation && !ok {
		return logical.ErrorResponse("'ttl' is required"), nil
	}

	if ok {
		ttlInt, ok := ttlRaw.(int)
		if !ok {
			return nil, fmt.Errorf("cannot cast 'ttl' to int")
		}

		ttl := time.Duration(ttlInt) * time.Second
		if ttl < time.Second {
			return logical.ErrorResponse("incorrect ttl minimum 1 second, got %v", ttl), nil
		}

		jwtType.TTL = ttl
	}

	specRaw, ok := data.GetOk("options_schema")
	var validator openapi.Validator
	if ok {
		spec, ok := specRaw.(string)
		if !ok {
			return nil, fmt.Errorf("cannot cast 'options_schema' to string")
		}

		validator, err = openapi.SchemaValidator(spec)
		if err != nil {
			return logical.ErrorResponse(fmt.Sprintf("incorrect 'options_schema': %v", err)), nil
		}

		jwtType.OptionsSchema = spec
	}

	resp := &logical.Response{}

	err = repo.Put(jwtType)
	if err != nil {
		return nil, err
	}

	err = tnx.Commit()
	if err != nil {
		return nil, err
	}

	b.setJWTTypeValidator(jwtType, validator)

	return resp, nil
}
