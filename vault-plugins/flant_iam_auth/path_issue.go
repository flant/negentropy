package jwtauth

import (
	"context"
	"fmt"
	iam "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/usecase"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"

	repos "github.com/flant/negentropy/vault-plugins/flant_iam_auth/model/repo"
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
	tnx := b.storage.Txn(false)
	defer tnx.Abort()

	isEnabled, err := b.jwtController.IsEnabled(tnx)
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

	repo := repos.NewJWTIssueTypeRepo(tnx)
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

	signedJwt, err := b.jwtController.IssuePayloadAsJwt(tnx, mapOptions, &jwt.TokenOptions{
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

	tnx := b.storage.Txn(true)
	defer tnx.Abort()

	isEnabled, err := b.jwtController.IsEnabled(tnx)
	if err != nil {
		return nil, err
	}

	if !isEnabled {
		return logical.ErrorResponse("jwt is not enabled"), nil
	}

	multipassService := &usecase.Multipass{
		JwtController:    b.jwtController,
		MultipassRepo:    iam.NewMultipassRepository(tnx),
		GenMultipassRepo: model.NewMultipassGenerationNumberRepository(tnx),
		Logger:           b.NamedLogger("MultipassNewGen"),
	}

	token, err := multipassService.IssueNewMultipassGeneration(tnx, multipassUUID)
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
