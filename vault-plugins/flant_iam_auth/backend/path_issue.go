package backend

import (
	"context"
	"fmt"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"

	iam "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	iam_repo "github.com/flant/negentropy/vault-plugins/flant_iam/repo"
	repo2 "github.com/flant/negentropy/vault-plugins/flant_iam_auth/repo"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/usecase"
	backendutils "github.com/flant/negentropy/vault-plugins/shared/backent-utils"
	jwt "github.com/flant/negentropy/vault-plugins/shared/jwt/usecase"
)

const HttpPathIssue = "issue"

// pathRole returns the path configurations for the CRUD operations on roles
func pathIssueJwtType(b *flantIamAuthBackend) *framework.Path {
	p := &framework.Path{
		Pattern: fmt.Sprintf("%s/jwt/", HttpPathIssue) + framework.GenericNameRegex("name"),
		Fields: map[string]*framework.FieldSchema{
			"name": {
				Type:        framework.TypeLowerCaseString,
				Description: "Name of the jwt type",
				Required:    true,
			},
			"options": {
				Type:        framework.TypeMap,
				Description: "Options for jwt type sign",
				Required:    true,
			},
		},

		Operations: map[logical.Operation]framework.OperationHandler{
			logical.UpdateOperation: &framework.PathOperation{
				Callback: b.pathIssueJwt,
				Summary:  "Update an existing jwt type",
			},
		},
		HelpSynopsis:    "Issue options as jwt token",
		HelpDescription: "",
	}

	return p
}

func pathIssueMultipassJwt(b *flantIamAuthBackend) *framework.Path {
	p := &framework.Path{
		Pattern: fmt.Sprintf("%s/multipass_jwt/", HttpPathIssue) + framework.GenericNameRegex("uuid"),
		Fields: map[string]*framework.FieldSchema{
			"uuid": {
				Type:        framework.TypeString,
				Description: "Name of the jwt type",
				Required:    true,
			},
		},

		Operations: map[logical.Operation]framework.OperationHandler{
			logical.UpdateOperation: &framework.PathOperation{
				Callback: b.pathIssueMultipassJwt,
				Summary:  "Update an existing jwt type",
			},
		},
		HelpSynopsis:    "Issue multipass jwt token with new generation number",
		HelpDescription: "",
	}

	return p
}

// pathJwtTypeCreateUpdate registers a new JwtTypeConfig with the backend or updates the options
// of an existing JwtTypeConfig
func (b *flantIamAuthBackend) pathIssueJwt(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	txn := b.storage.Txn(false)
	defer txn.Abort()

	isEnabled, err := b.jwtController.IsEnabled(txn)
	if err != nil {
		return nil, err
	}

	if !isEnabled {
		return logical.ErrorResponse("jwt is not enabled"), nil
	}

	name, errResp := backendutils.NotEmptyStringParam(data, "name")
	if errResp != nil {
		return errResp, nil
	}

	optionsRaw, ok := data.GetOk("options")
	if !ok {
		return logical.ErrorResponse("'options' is required"), nil
	}

	options, ok := optionsRaw.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("cannot cast 'options' to map[string]interface{}")
	}

	repo := repo2.NewJWTIssueTypeRepo(txn)
	jwtType, err := repo.Get(name)
	if err != nil {
		return nil, err
	}

	if jwtType == nil {
		return logical.RespondWithStatusCode(
			logical.ErrorResponse("not found jwt type %s", name),
			req,
			404,
		)
	}

	validator, err := b.jwtTypeValidator(jwtType)
	if err != nil {
		return nil, err
	}

	optionsWithDefaults, err := validator.Validate(options)
	if err != nil {
		return logical.ErrorResponse("validate options error: %v", err), nil
	}

	mapOptions, ok := optionsWithDefaults.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("cannot cast 'optionsWithDefaults' to map[string]interface{}")
	}

	signedJwt, err := b.jwtController.IssuePayloadAsJwt(txn, mapOptions, &jwt.TokenOptions{
		TTL: jwtType.TTL,
	})
	if err != nil {
		return nil, fmt.Errorf("cannot sign options: %v", err)
	}

	resp := &logical.Response{
		Data: map[string]interface{}{
			"token": signedJwt,
		},
	}

	return resp, nil
}

// pathJwtTypeCreateUpdate registers a new JwtTypeConfig with the backend or updates the options
// of an existing JwtTypeConfig
func (b *flantIamAuthBackend) pathIssueMultipassJwt(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	multipassUUID, errResp := backendutils.NotEmptyStringParam(data, "uuid")
	if errResp != nil {
		return errResp, nil
	}

	txn := b.storage.Txn(true)
	defer txn.Abort()

	isEnabled, err := b.jwtController.IsEnabled(txn)
	if err != nil {
		return nil, err
	}

	if !isEnabled {
		return logical.ErrorResponse("jwt is not enabled"), nil
	}

	multipassService := &usecase.Multipass{
		JwtController:    b.jwtController,
		MultipassRepo:    iam_repo.NewMultipassRepository(txn),
		GenMultipassRepo: repo2.NewMultipassGenerationNumberRepository(txn),
		Logger:           b.NamedLogger("MultipassNewGen"),
	}

	vstOwnerType, vstOwnerUUID, err := b.revealVSTOwner(req)
	if err != nil {
		return logical.ErrorResponse(err.Error()), nil
	}

	token, err := multipassService.IssueNewMultipassGeneration(txn, multipassUUID, vstOwnerType, vstOwnerUUID)
	if err != nil {
		return logical.ErrorResponse(err.Error()), nil
	}

	resp := &logical.Response{
		Data: map[string]interface{}{
			"token": token,
		},
	}

	return resp, nil
}

func (b *flantIamAuthBackend) revealVSTOwner(req *logical.Request) (iam.MultipassOwnerType, iam.OwnerUUID, error) {
	entityIDOwner, err := b.entityIDResolver.RevealEntityIDOwner(req.EntityID, b.storage.Txn(false), req.Storage)
	if err != nil {
		return "", "", err
	}
	var multipassOwnerUUID iam.OwnerUUID
	var multipassOwnerType iam.MultipassOwnerType
	switch entityIDOwner.OwnerType {
	case iam.UserType:
		{
			user, ok := entityIDOwner.Owner.(*iam.User)
			if !ok {
				return "", "", fmt.Errorf("can't cast, need *model.User, got: %T", entityIDOwner.Owner)
			}
			multipassOwnerUUID = user.UUID
			multipassOwnerType = iam.UserType
		}

	case iam.ServiceAccountType:
		{
			sa, ok := entityIDOwner.Owner.(*iam.ServiceAccount)
			if !ok {
				return "", "", fmt.Errorf("can't cast, need *model.ServiceAccount, got: %T", entityIDOwner.Owner)
			}
			multipassOwnerUUID = sa.UUID
			multipassOwnerType = iam.ServiceAccountType
		}
	default:
		return "", "", fmt.Errorf("wrong entityIDOwner.OwnerType:%s", entityIDOwner.OwnerType)
	}
	return multipassOwnerType, multipassOwnerUUID, nil
}
