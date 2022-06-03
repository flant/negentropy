package authz

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/vault/sdk/logical"

	iam "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	iam_repo "github.com/flant/negentropy/vault-plugins/flant_iam/repo"
	iam_usecase "github.com/flant/negentropy/vault-plugins/flant_iam/usecase"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/io/downstream/vault"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/io/downstream/vault/api"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam_auth/repo"
	authn2 "github.com/flant/negentropy/vault-plugins/flant_iam_auth/usecase/authn"
	"github.com/flant/negentropy/vault-plugins/shared/client"
	"github.com/flant/negentropy/vault-plugins/shared/consts"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/memdb"
)

type Authorizator struct {
	UserRepo      *iam_repo.UserRepository
	SaRepo        *iam_repo.ServiceAccountRepository
	EntityRepo    *repo.EntityRepo
	EaRepo        *repo.EntityAliasRepo
	RoleRepo      *iam_repo.RoleRepository
	PolicyRepo    *repo.PolicyRepository
	RolesResolver iam_usecase.RoleResolver

	ExtensionsDataProvider ExtensionsDataProvider

	MountAccessor *vault.MountAccessorGetter

	Logger              hclog.Logger
	vaultClientProvider client.VaultClientController
}

func MakeSubject(data map[string]interface{}) model.Subject {
	subjectType, _ := data["type"].(string)
	uuid, _ := data["uuid"].(string)
	tenantUUID, _ := data["tenant_uuid"].(string)
	return model.Subject{
		Type:       subjectType,
		UUID:       uuid,
		TenantUUID: tenantUUID,
	}
}

func NewAutorizator(txn *io.MemoryStoreTxn, vaultClientController client.VaultClientController, aGetter *vault.MountAccessorGetter, logger hclog.Logger) *Authorizator {
	return &Authorizator{
		Logger: logger.Named("AuthoriZator"),

		SaRepo:   iam_repo.NewServiceAccountRepository(txn),
		UserRepo: iam_repo.NewUserRepository(txn),

		EaRepo:        repo.NewEntityAliasRepo(txn),
		EntityRepo:    repo.NewEntityRepo(txn),
		RoleRepo:      iam_repo.NewRoleRepository(txn),
		PolicyRepo:    repo.NewPolicyRepository(txn),
		RolesResolver: iam_usecase.NewRoleResolver(txn),

		ExtensionsDataProvider: NewExtensionsDataProvider(txn),

		MountAccessor:       aGetter,
		vaultClientProvider: vaultClientController,
	}
}

func (a *Authorizator) identityApi() *api.IdentityAPI {
	return api.NewIdentityAPI(a.vaultClientProvider, a.Logger.Named("LoginIdentityApi"))
}

type RoleClaimResult struct {
	model.RoleClaim
	RolebindingExists bool   `json:"rolebinding_exists"`
	AllowLogin        bool   `json:"allow_login"`
	RequireMFA        bool   `json:"require_mfa,omitempty"`
	NeedApprovals     bool   `json:"need_approvals,omitempty"`
	Err               string `json:"error,omitempty"`
}

func (a *Authorizator) CheckPermissions(authMethodName string, subject model.Subject, roleClaims []model.RoleClaim) []RoleClaimResult {
	rowResults := a.checkPermissions(authMethodName, subject, roleClaims)
	results := make([]RoleClaimResult, 0, len(rowResults))
	for _, rowResult := range rowResults {
		result := RoleClaimResult{
			RoleClaim:         rowResult.loginClaim,
			RolebindingExists: len(rowResult.effectiveRoles) > 0,
			AllowLogin:        rowResult.regoresult.Allow,
		}
		if rowResult.err != nil {
			result.Err = rowResult.err.Error()
		}
		if rowResult.regoresult.BestEffectiveRole != nil {
			result.RequireMFA = rowResult.regoresult.BestEffectiveRole.RequireMFA
			result.NeedApprovals = rowResult.regoresult.BestEffectiveRole.NeedApprovals > 0
		}

		results = append(results, result)
	}
	return results
}

type tryLoginResult struct {
	loginClaim     model.RoleClaim
	regoresult     RegoResult
	effectiveRoles []iam_usecase.EffectiveRole
	err            error
}

// checkPermissions validate all permissions request and store results
func (a *Authorizator) checkPermissions(authMethodName string, subject model.Subject, roleClaims []model.RoleClaim) []tryLoginResult {
	result := make([]tryLoginResult, 0, len(roleClaims))
	var err error
	for _, rc := range roleClaims {
		item := tryLoginResult{
			loginClaim: rc,
		}

		err = a.checkTenantUUID(rc, subject)
		if err != nil {
			item.err = err
			result = append(result, item)
			continue
		}

		item.effectiveRoles, err = a.checkScopeAndCollectEffectiveRoles(rc, subject)
		if err != nil {
			item.err = err
			result = append(result, item)
			continue
		}

		negentropyPolicy, err := a.seekAndValidatePolicy(rc.Role, authMethodName)
		if err != nil {
			item.err = err
			result = append(result, item)
			continue
		}

		regoResult, err := a.applyNegentropyPolicy(subject, rc, *negentropyPolicy, item.effectiveRoles)
		if err != nil {
			item.err = err
			result = append(result, item)
			continue
		}
		item.regoresult = *regoResult
		result = append(result, item)
	}
	return result
}

func (a *Authorizator) Authorize(authnResult *authn2.Result, method *model.AuthMethod, source *model.AuthSource,
	roleClaims []model.RoleClaim) (*logical.Auth, error) {
	subjectUUID := authnResult.UUID
	a.Logger.Debug(fmt.Sprintf("Start authz for %s", subjectUUID))

	authzRes, fullId, err := a.authorizeTokenOwner(subjectUUID, method, source)
	if err != nil {
		return nil, err
	}

	if authzRes == nil {
		a.Logger.Warn(fmt.Sprintf("Nil autzRes %s", subjectUUID))
		return nil, fmt.Errorf("not authz %s", subjectUUID)
	}

	a.Logger.Debug(fmt.Sprintf("Start getting vault entity and entity alias %s", fullId))
	vaultAlias, entityId, err := a.getAlias(subjectUUID, source)
	if err != nil {
		return nil, err
	}

	a.Logger.Debug(fmt.Sprintf("Got entityId %s and entity alias %s", entityId, vaultAlias.ID))

	authzRes.Alias = vaultAlias
	authzRes.EntityID = entityId
	subject := authzRes.InternalData["subject"].(model.Subject)

	method.PopulateTokenAuth(authzRes)

	err = a.addDynamicPolicy(authzRes, roleClaims, subject, method.Name)
	if err != nil {
		return nil, err
	}

	authzRes.InternalData["flantIamAuthMethod"] = method.Name

	a.Logger.Debug(fmt.Sprintf("Token auth populated %s", fullId))

	a.populateAuthnData(authzRes, authnResult)

	a.Logger.Debug(fmt.Sprintf("Authn data populated %s", fullId))

	return authzRes, nil
}

func (a *Authorizator) authorizeTokenOwner(uuid string, method *model.AuthMethod, source *model.AuthSource) (authzRes *logical.Auth, tokenOwnerFullIdentifier string, err error) {
	user, err := a.UserRepo.GetByID(uuid)
	if err != nil && !errors.Is(err, consts.ErrNotFound) {
		return nil, "", err
	}
	if user != nil && user.NotArchived() {
		tokenOwnerFullIdentifier = user.FullIdentifier
		a.Logger.Debug(fmt.Sprintf("Found user %s for %s uuid", tokenOwnerFullIdentifier, uuid))
		authzRes, err = a.authorizeUser(user, method, source)
	} else {
		// not found user try to found service account
		a.Logger.Debug(fmt.Sprintf("Not found active user for %s uuid. Try find service account", uuid))
		var sa *iam.ServiceAccount
		sa, err = a.SaRepo.GetByID(uuid)
		if errors.Is(err, consts.ErrNotFound) || sa.Archived() {
			return nil, "", fmt.Errorf("not found active iam entity %s", uuid)
		}
		if err != nil {
			return nil, "", err
		}

		tokenOwnerFullIdentifier = sa.FullIdentifier

		a.Logger.Debug(fmt.Sprintf("Found service account %s for %s uuid", tokenOwnerFullIdentifier, uuid))
		authzRes, err = a.authorizeServiceAccount(sa, method, source)
	}
	return authzRes, tokenOwnerFullIdentifier, nil
}

// addDynamicPolicy build ONE vault policy for all roleClaims if all are allowed
func (a *Authorizator) addDynamicPolicy(authzRes *logical.Auth, roleClaims []model.RoleClaim, subject model.Subject, authMethod string) error {
	loginItems := a.checkPermissions(authMethod, subject, roleClaims)
	multiError := multierror.Error{}
	allow := true
	// TODO what about multifactor ?
	for _, loginItem := range loginItems {
		if !loginItem.regoresult.Allow {
			allow = false
		}
		if loginItem.err != nil {
			multiError.Errors = append(multiError.Errors, loginItem.err)
		}
	}
	if !allow || multiError.Len() > 0 {
		return fmt.Errorf("not allowed: %s", multiError.Error())
	}

	var ttl, maxTTL time.Duration

	extraPolicy := VaultPolicy{}
	for _, loginItem := range loginItems {
		extraPolicy.Rules = append(extraPolicy.Rules, loginItem.regoresult.VaultRules...)
		if ttl > loginItem.regoresult.TTL || ttl == 0 {
			ttl = loginItem.regoresult.TTL
		}
		if maxTTL > loginItem.regoresult.MaxTTL || maxTTL == 0 {
			maxTTL = loginItem.regoresult.MaxTTL
		}
	}

	extraPolicy.AddValidTillToName(time.Now().Add(maxTTL))

	err := a.createDynamicPolicy(extraPolicy)
	if err != nil {
		return err
	}
	authzRes.Policies = append(authzRes.Policies, extraPolicy.Name)
	authzRes.MaxTTL = maxTTL
	authzRes.TTL = ttl
	return nil
}

func (a *Authorizator) applyNegentropyPolicy(subject model.Subject, rc model.RoleClaim, negentropyPolicy model.Policy,
	effectiveRoles []iam_usecase.EffectiveRole) (*RegoResult, error) {
	role, err := a.RoleRepo.GetByID(rc.Role)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	regoClaims := map[string]interface{}{
		"role":         rc.Role,
		"tenant_uuid":  rc.TenantUUID,
		"project_uuid": rc.ProjectUUID,
	}
	for k, v := range rc.Claim {
		regoClaims[k] = v
	}

	extensionsData, err := a.ExtensionsDataProvider.CollectExtensionsData(role.EnrichingExtensions, subject, rc)

	regoResult, err := ApplyRegoPolicy(ctx, negentropyPolicy, subject, extensionsData, effectiveRoles, regoClaims)

	if err != nil {
		err = fmt.Errorf("error appliing rego policy:%w", err)
		a.Logger.Error(err.Error())
		return nil, err
	} else {
		a.Logger.Debug(fmt.Sprintf("regoResult:%#v\n", *regoResult))
	}

	if !regoResult.Allow {
		err = fmt.Errorf("not allowed: subject_type=%s, subject_uuid=%s, rolename=%s, claims=%v, errors, returned by rego:%v",
			subject.Type, subject.UUID, rc.Role, rc, regoResult.Errors)
		a.Logger.Error(err.Error())
		return nil, err
	}

	return regoResult, nil
}

// check role scope and passed claim, if correct, collect  effectiveRoles
func (a *Authorizator) checkScopeAndCollectEffectiveRoles(rc model.RoleClaim, subject model.Subject) ([]iam_usecase.EffectiveRole, error) {
	role, err := a.RoleRepo.GetByID(rc.Role)
	if err != nil {
		err = fmt.Errorf("error catching role %s:%w", rc.Role, err)
		a.Logger.Error(err.Error())
		return nil, err
	}
	scope, err := checkAndEvaluateScope(role, rc.TenantUUID, rc.ProjectUUID)
	if err != nil {
		a.Logger.Error(err.Error())
		return nil, err
	}

	var effectiveRoles []iam_usecase.EffectiveRole
	var found bool
	switch {

	case subject.Type == "user" && scope == projectScope:
		found, effectiveRoles, err = a.RolesResolver.CheckUserForRolebindingsAtProject(subject.UUID, rc.Role, rc.ProjectUUID)
	case subject.Type == "user" && scope == tenantScope:
		found, effectiveRoles, err = a.RolesResolver.CheckUserForRolebindingsAtTenant(subject.UUID, rc.Role, rc.TenantUUID)
	case subject.Type == "user" && scope == globalScope:
		found, effectiveRoles, err = a.RolesResolver.CheckUserForRolebindings(subject.UUID, rc.Role)

	case subject.Type == "service_account" && scope == projectScope:
		found, effectiveRoles, err = a.RolesResolver.CheckServiceAccountForRolebindingsAtProject(subject.UUID, rc.Role,
			rc.ProjectUUID)
	case subject.Type == "service_account" && scope == tenantScope:
		found, effectiveRoles, err = a.RolesResolver.CheckServiceAccountForRolebindingsAtTenant(subject.UUID, rc.Role,
			rc.TenantUUID)
	case subject.Type == "service_account" && scope == globalScope:
		found, effectiveRoles, err = a.RolesResolver.CheckServiceAccountForRolebindings(subject.UUID, rc.Role)
	}
	if !found {
		a.Logger.Warn(fmt.Sprintf("not found rolebindings, subject_type=%s, subject_uuid=%s, role=%s, if policy needs rolebindings, login fail",
			subject.Type, subject.UUID, role.Name))
	}
	if err != nil {
		err = fmt.Errorf("error searching, subject_type=%s, subject_uuid=%s, role=%s :%w",
			subject.Type, subject.UUID, role.Name, err)
		a.Logger.Error(err.Error())
		return nil, err
	}
	return effectiveRoles, nil
}

const (
	errorScope = iota
	globalScope
	tenantScope
	projectScope
)

// checkAndEvaluateScope according role.Scope, role.TenantIsOptional and role.ProjectIsOptional evaluate scope:
// a) globalScope, b) "tenant" c) "project" and check is all necessary passed
func checkAndEvaluateScope(role *iam.Role, tenantUUID iam.TenantUUID, projectUUID iam.ProjectUUID) (int, error) {
	// global
	if ((role.Scope == iam.RoleScopeProject && role.TenantIsOptional && role.ProjectIsOptional) || (role.Scope == iam.RoleScopeTenant && role.TenantIsOptional)) &&
		tenantUUID == "" && projectUUID == "" {
		return globalScope, nil
	}
	// tenant
	if ((role.Scope == iam.RoleScopeProject && role.ProjectIsOptional) || (role.Scope == iam.RoleScopeTenant)) &&
		tenantUUID != "" && projectUUID == "" {
		return tenantScope, nil
	}
	// project
	if (role.Scope == iam.RoleScopeProject) &&
		tenantUUID != "" && projectUUID != "" {
		return projectScope, nil
	}
	return errorScope, fmt.Errorf("error scope: role: %#v, passed tenant_uuid='%s', project_uuid='%s'", role, tenantUUID, projectUUID)
}

// authorizeServiceAccount called from authorizeTokenOwner in case token is owned by service_account
func (a *Authorizator) authorizeServiceAccount(sa *iam.ServiceAccount, method *model.AuthMethod, source *model.AuthSource) (*logical.Auth, error) {
	// todo some logic for sa here
	// todo collect rba for user
	return &logical.Auth{
		DisplayName: sa.FullIdentifier,
		InternalData: map[string]interface{}{
			"subject": model.Subject{
				Type:       "service_account",
				UUID:       sa.UUID,
				TenantUUID: sa.TenantUUID,
			},
		},
	}, nil
}

// authorizeUser called from authorizeTokenOwner in case token is owned by user
func (a *Authorizator) authorizeUser(user *iam.User, method *model.AuthMethod, source *model.AuthSource) (*logical.Auth, error) {
	// todo some logic for user here
	// todo collect rba for user
	return &logical.Auth{
		DisplayName: user.FullIdentifier,
		InternalData: map[string]interface{}{
			"subject": model.Subject{
				Type:       "user",
				UUID:       user.UUID,
				TenantUUID: user.TenantUUID,
			},
		},
	}, nil
}

func (a *Authorizator) populateAuthnData(authzRes *logical.Auth, authnResult *authn2.Result) {
	if len(authnResult.Metadata) > 0 {
		for k, v := range authnResult.Metadata {
			if _, ok := authzRes.Metadata[k]; ok {
				a.Logger.Warn(fmt.Sprintf("Key %s already exists in authz metadata. Skip", k))
				continue
			}

			authzRes.Metadata[k] = v
		}
	}

	if len(authnResult.InternalData) > 0 {
		for k, v := range authnResult.InternalData {
			if _, ok := authzRes.Metadata[k]; ok {
				a.Logger.Warn(fmt.Sprintf("Key %s already exists in authz internal data. Skip", k))
				continue
			}

			authzRes.InternalData[k] = v
		}
	}

	if len(authnResult.Policies) > 0 {
		authzRes.Policies = append(authzRes.Policies, authnResult.Policies...)
	}

	if len(authnResult.GroupAliases) > 0 {
		for i, p := range authnResult.GroupAliases {
			k := fmt.Sprintf("group_alias_%v", i)
			authzRes.Metadata[k] = p
		}
	}
}

func (a *Authorizator) getAlias(uuid string, source *model.AuthSource) (*logical.Alias, string, error) {
	entity, err := a.EntityRepo.GetByUserId(uuid)
	if err != nil {
		return nil, "", err
	}

	if entity == nil {
		return nil, "", fmt.Errorf("not found entity for %s", uuid)
	}

	a.Logger.Debug(fmt.Sprintf("Got entity from db %s", uuid))

	ea, err := a.EaRepo.GetForUser(uuid, source)
	if err != nil {
		return nil, "", err
	}

	if ea == nil {
		return nil, "", fmt.Errorf("not found entity alias for %s with source %s", uuid, source.Name)
	}

	a.Logger.Debug(fmt.Sprintf("Got entity alias from db %s", uuid))

	entityId, err := a.identityApi().EntityApi().GetID(entity.Name)
	if err != nil {
		return nil, "", fmt.Errorf("getting entity_id:%w", err)
	}

	if entityId == "" {
		return nil, "", fmt.Errorf("can not get entity id %s/%s", uuid, entity.Name)
	}

	a.Logger.Debug(fmt.Sprintf("Got entity id from vault %s", uuid))

	accessorId, err := a.MountAccessor.MountAccessor()
	if err != nil {
		return nil, "", err
	}

	eaId, err := a.identityApi().AliasApi().FindAliasIDByName(ea.Name, accessorId)
	if err != nil {
		return nil, "", err
	}

	if eaId == "" {
		return nil, "", fmt.Errorf("can not get entity alias id %s/%s/%s", uuid, ea.Name, source.Name)
	}

	a.Logger.Debug(fmt.Sprintf("Got entity alias id from db %s", uuid))

	return &logical.Alias{
		ID:            eaId,
		MountAccessor: accessorId,
		Name:          ea.Name,
	}, entityId, nil
}

func (a *Authorizator) createDynamicPolicy(p VaultPolicy) error {
	err := backoff.Retry(func() error {
		client, err := a.vaultClientProvider.APIClient(nil)
		if err != nil {
			return err
		}
		return client.Sys().PutPolicy(p.Name, p.PolicyRules())
	}, io.FiveSecondsBackoff())
	if err != nil {
		return fmt.Errorf("put policy %s:%w", p.Name, err)
	}
	return nil
}

func (a *Authorizator) Renew(method *model.AuthMethod, auth *logical.Auth, txn *io.MemoryStoreTxn, subject model.Subject) (*logical.Auth, error) {
	// check is user/sa still active
	// TODO check rolebinding still active
	var owner memdb.Archivable
	var err error
	switch subject.Type {
	case iam.UserType:
		owner, err = iam_repo.NewUserRepository(txn).GetByID(subject.UUID)
	case iam.ServiceAccountType:
		owner, err = iam_repo.NewServiceAccountRepository(txn).GetByID(subject.UUID)
	default:
		return nil, fmt.Errorf("wrong type of tokenOwnerType:%s", subject.Type)
	}
	if err != nil {
		return nil, err
	}
	if owner.Archived() {
		return nil, fmt.Errorf("tokenOwner is deleted")
	}
	authzRes := *auth
	authzRes.TTL = method.TokenTTL
	authzRes.MaxTTL = method.TokenMaxTTL
	authzRes.Period = method.TokenPeriod
	return &authzRes, nil
}

func (a *Authorizator) seekAndValidatePolicy(roleName iam.RoleName, authMethod string) (*model.Policy, error) {
	policies, err := a.PolicyRepo.ListActiveForRole(roleName)
	if err != nil {
		return nil, err
	}
	if len(policies) == 0 {
		return nil, fmt.Errorf("no one negentropy policy for role:%s", roleName)
	}
	if len(policies) > 1 {
		return nil, fmt.Errorf("more the one negentropy policy for role:%s", roleName)
	}
	policy := policies[0]

	if len(policy.AllowedAuthMethods) == 0 {
		return policy, nil
	}
	for _, m := range policy.AllowedAuthMethods {
		if m == authMethod {
			return policy, nil
		}
	}
	return nil, fmt.Errorf("for role:%s authMethod %s is not allowed", roleName, authMethod)
}

// checkTenantUUID checks  tenantUUID in RoleClaim:
// if it filled - it checks is it owner of subject or is subject shared to this tenant
func (a *Authorizator) checkTenantUUID(rc model.RoleClaim, subject model.Subject) error {
	if rc.TenantUUID == "" {
		return nil
	}
	if rc.TenantUUID == subject.TenantUUID {
		return nil
	}
	var isShared bool
	var err error
	if subject.Type == "user" {
		isShared, err = a.RolesResolver.IsUserSharedWithTenant(subject.UUID, rc.TenantUUID)
	} else {
		isShared, err = a.RolesResolver.IsServiceAccountSharedWithTenant(subject.UUID, rc.TenantUUID)
	}
	if err != nil {
		return err
	}
	if !isShared {
		return fmt.Errorf("role_claim: %#v has invalid tenant_uuid", rc)
	}
	return nil
}
