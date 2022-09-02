package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"

	"github.com/flant/negentropy/e2e/tests/lib"
	"github.com/flant/negentropy/e2e/tests/lib/tools"
	access_token_or_sapass_auth "github.com/flant/negentropy/e2e/tests/renew_vst"
	"github.com/flant/negentropy/e2e/tests/restoration/common"
	"github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/specs"
	ext_model "github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/model"
	specs2 "github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/paths/tests/specs"
	"github.com/flant/negentropy/vault-plugins/flant_iam/fixtures"
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	api "github.com/flant/negentropy/vault-plugins/shared/tests"
	"github.com/flant/negentropy/vault-plugins/shared/uuid"
)

type object = interface{}                                  // usually it is type from model package, sometimes it is gjson.Result, sometimes it is just a string
type objectIdentifier = string                             // usually iy is name of type from model package
type objectChecker = func(*http.Client, *testing.T)        // dynamically created functions for checking is restored object equals to origin
type objectGetter = func(client *http.Client) gjson.Result // dynamically created at objectCreator function for getting created object from vault
type store interface {
	saveObject(object, objectIdentifier)
	getObject(objectIdentifier) object
}
type memStore struct {
	data map[objectIdentifier]object
}

func (m *memStore) saveObject(object object, identifier objectIdentifier) {
	m.data[identifier] = object
}

func (m *memStore) getObject(identifier objectIdentifier) (object object) {
	return m.data[identifier]
}

func newMemStore() store {
	return &memStore{data: map[objectIdentifier]object{}}
}

type objectCreator = func(iamClient *http.Client, store store) (objectIdentifier, object, objectGetter) // create object at vault and create suitable  objectGetter

func tenantCreator(iamClient *http.Client, _ store) (objectIdentifier, object, objectGetter) {
	tenant := specs.CreateRandomTenant(lib.NewTenantAPI(iamClient))
	return "tenant", tenant, func(iamClient *http.Client) gjson.Result {
		return lib.NewTenantAPI(iamClient).Read(api.Params{
			"tenant": tenant.UUID,
		}, nil).Get("tenant")
	}
}

func userCreator(iamClient *http.Client, store store) (objectIdentifier, object, objectGetter) {
	tenant := store.getObject("tenant").(model.Tenant)
	user := specs.CreateRandomUser(lib.NewUserAPI(iamClient), tenant.UUID)
	return "user", user, func(iamClient *http.Client) gjson.Result {
		return lib.NewUserAPI(iamClient).Read(api.Params{
			"tenant": tenant.UUID,
			"user":   user.UUID,
		}, nil).Get("user")
	}
}

func serviceAccountCreator(iamClient *http.Client, store store) (objectIdentifier, object, objectGetter) {
	tenant := store.getObject("tenant").(model.Tenant)
	sa := specs.CreateRandomServiceAccount(lib.NewServiceAccountAPI(iamClient), tenant.UUID)
	return "service_account", sa, func(iamClient *http.Client) gjson.Result {
		return lib.NewServiceAccountAPI(iamClient).Read(api.Params{
			"tenant":          tenant.UUID,
			"service_account": sa.UUID,
		}, nil).Get("service_account")
	}
}

func serviceAccountPasswordCreator(iamClient *http.Client, store store) (objectIdentifier, object, objectGetter) {
	sa := store.getObject("service_account").(model.ServiceAccount)
	sap := specs.CreateServiceAccountPassword(lib.NewServiceAccountPasswordAPI(iamClient), sa, "test_password", time.Minute, []string{"ssh.open"})
	return "service_account_password", sap, func(iamClient *http.Client) gjson.Result {
		return lib.NewServiceAccountPasswordAPI(iamClient).Read(api.Params{
			"tenant":          sa.TenantUUID,
			"service_account": sa.UUID,
			"password":        sap.UUID,
		}, nil).Get("password")
	}
}

func featureFlagCreator(iamClient *http.Client, _ store) (objectIdentifier, object, objectGetter) {
	ff := lib.NewFeatureFlagAPI(iamClient).Create(tools.Params{}, url.Values{}, fixtures.RandomFeatureFlagCreatePayload()).Get("feature_flag")
	ffName := ff.Get("name").String()
	return "feature_flag", ff, func(iamClient *http.Client) gjson.Result {
		ffs := lib.NewFeatureFlagAPI(iamClient).List(api.Params{}, nil).Get("names")
		for _, f := range ffs.Array() {
			if ffName == f.Get("name").String() {
				fmt.Println(f.String())
				return f
			}
		}
		panic(fmt.Errorf("not found featureFlag by name %q", ffName))
	}
}

func roleCreator(iamClient *http.Client, _ store) (objectIdentifier, object, objectGetter) {
	role := specs.CreateRandomRole(lib.NewRoleAPI(iamClient))
	return "role", role, func(iamClient *http.Client) gjson.Result {
		return lib.NewRoleAPI(iamClient).Read(api.Params{
			"name": role.Name,
		}, nil).Get("role")
	}
}

func projectCreator(iamClient *http.Client, store store) (objectIdentifier, object, objectGetter) {
	tenant := store.getObject("tenant").(model.Tenant)
	project := specs.CreateRandomProject(lib.NewProjectAPI(iamClient), tenant.UUID)
	return "project", project, func(iamClient *http.Client) gjson.Result {
		return lib.NewProjectAPI(iamClient).Read(api.Params{
			"tenant":  tenant.UUID,
			"project": project.UUID,
		}, nil).Get("project")
	}
}

func groupCreator(iamClient *http.Client, store store) (objectIdentifier, object, objectGetter) {
	tenant := store.getObject("tenant").(model.Tenant)
	group := specs.CreateRandomEmptyGroup(lib.NewGroupAPI(iamClient), tenant.UUID)
	return "group", group, func(iamClient *http.Client) gjson.Result {
		return lib.NewGroupAPI(iamClient).Read(api.Params{
			"tenant": tenant.UUID,
			"group":  group.UUID,
		}, nil).Get("group")
	}
}

const flantUUID = "be0ba0d8-7be7-49c8-8609-c62ac1f14597"

func identitySharingCreator(iamClient *http.Client, store store) (objectIdentifier, object, objectGetter) {
	group := store.getObject("group").(model.Group)
	is := lib.NewIdentitySharingAPI(iamClient).Create(api.Params{
		"tenant": group.TenantUUID,
	}, url.Values{}, map[string]interface{}{
		"destination_tenant_uuid": flantUUID,
		"groups":                  []string{group.UUID},
	}).Get("identity_sharing")
	return "identity_sharing", is, func(iamClient *http.Client) gjson.Result {
		return lib.NewIdentitySharingAPI(iamClient).Read(api.Params{
			"tenant": group.TenantUUID,
			"uuid":   is.Get("uuid").String(),
		}, nil).Get("identity_sharing")
	}
}

func multipassCreator(iamClient *http.Client, store store) (objectIdentifier, object, objectGetter) {
	user := store.getObject("user").(model.User)
	multipass, jwt := specs.CreateUserMultipass(lib.NewUserMultipassAPI(iamClient), user,
		"test",
		100*time.Second,
		1000*time.Second,
		[]string{"ssh.open"})

	return "multipass-jwt", jwt, func(iamClient *http.Client) gjson.Result {
		return lib.NewUserMultipassAPI(iamClient).Read(api.Params{
			"tenant":    user.TenantUUID,
			"user":      user.UUID,
			"multipass": multipass.UUID,
		}, nil).Get("multipass")
	}
}

func rolebindingCreator(iamClient *http.Client, store store) (objectIdentifier, object, objectGetter) {
	group := store.getObject("group").(model.Group)
	role := store.getObject("role").(model.Role)
	rbToCreate := model.RoleBinding{
		TenantUUID:  group.TenantUUID,
		Description: "test_rb",
		Members: []model.MemberNotation{{
			Type: "group",
			UUID: group.UUID,
		}},
		Roles: []model.BoundRole{{
			Name:    role.Name,
			Options: map[string]interface{}{},
		}},
	}
	if role.Scope == "project" {
		rbToCreate.AnyProject = true
	}

	rb := specs.CreateRoleBinding(lib.NewRoleBindingAPI(iamClient), rbToCreate)
	return "role_binding", rb, func(iamClient *http.Client) gjson.Result {
		return lib.NewRoleBindingAPI(iamClient).Read(api.Params{
			"tenant":       rb.TenantUUID,
			"role_binding": rb.UUID,
		}, nil).Get("role_binding")
	}
}

func rolebindingApprovalCreator(iamClient *http.Client, store store) (objectIdentifier, object, objectGetter) {
	user := store.getObject("user").(model.User)
	roleBinding := store.getObject("role_binding").(model.RoleBinding)

	rba := lib.NewRoleBindingApprovalAPI(iamClient).Create(api.Params{
		"expectStatus": api.ExpectExactStatus(201),
		"tenant":       user.TenantUUID,
		"role_binding": roleBinding.UUID,
	}, url.Values{}, map[string]interface{}{
		"required_votes": 1,
		"approvers": []map[string]interface{}{
			{"type": "user", "uuid": user.UUID},
		},
	}).Get("approval")
	return "approval", rba, func(iamClient *http.Client) gjson.Result {
		return lib.NewRoleBindingApprovalAPI(iamClient).Read(api.Params{
			"tenant":       roleBinding.TenantUUID,
			"role_binding": roleBinding.UUID,
			"uuid":         rba.Get("uuid").String(),
		}, nil).Get("approval")
	}
}

// serverAccess
func serverCreator(iamClient *http.Client, store store) (objectIdentifier, object, objectGetter) {
	project := store.getObject("project").(model.Project)
	server, _ := specs.CreateRandomServer(lib.NewServerAPI(iamClient), project.TenantUUID, project.UUID)
	return "server", server, func(iamClient *http.Client) gjson.Result {
		return lib.NewServerAPI(iamClient).Read(api.Params{
			"tenant":  project.TenantUUID,
			"project": project.UUID,
			"server":  server.UUID,
		}, nil).Get("server")
	}
}

// flant_flow
func clientCreator(iamClient *http.Client, store store) (objectIdentifier, object, objectGetter) {
	user := store.getObject("user").(model.User)
	client := specs2.CreateRandomClient(lib.NewFlowClientAPI(iamClient), user.UUID)
	return "client", client, func(vaultClient *http.Client) gjson.Result {
		return lib.NewFlowClientAPI(vaultClient).Read(api.Params{
			"client": client.UUID,
		}, nil).Get("client")
	}
}

func teamCreator(iamClient *http.Client, store store) (objectIdentifier, object, objectGetter) {
	devopsTeam := specs2.CreateRandomTeamWithSpecificType(lib.NewFlowTeamAPI(iamClient), ext_model.DevopsTeam)
	return "team", devopsTeam, func(vaultClient *http.Client) gjson.Result {
		return lib.NewFlowTeamAPI(vaultClient).Read(api.Params{
			"team": devopsTeam.UUID,
		}, nil).Get("team")
	}
}

func flantFlowProjectCreator(iamClient *http.Client, store store) (objectIdentifier, object, objectGetter) {
	client := store.getObject("client").(ext_model.Client)
	devopsTeam := store.getObject("team").(ext_model.Team)
	createPayload := fixtures.RandomProjectCreatePayload()
	createPayload["tenant_uuid"] = client.UUID
	createPayload["devops_team"] = devopsTeam.UUID
	createPayload["service_packs"] = []string{ext_model.DevOps}
	project := lib.NewFlowProjectAPI(iamClient).Create(
		api.Params{
			"client": client.UUID,
		},
		url.Values{},
		createPayload,
	).Get("project")

	return "flow_project", project, func(vaultClient *http.Client) gjson.Result {
		data := lib.NewFlowProjectAPI(vaultClient).Read(api.Params{
			"client":  client.UUID,
			"project": project.Get("uuid").String(),
		}, nil)
		return data.Get("project")
	}
}

func teammateCreator(iamClient *http.Client, store store) (objectIdentifier, object, objectGetter) {
	team := store.getObject("team").(ext_model.Team)
	teammate := specs2.CreateRandomTeammate(lib.NewFlowTeammateAPI(iamClient), team)
	return "teammate", teammate, func(iamClient *http.Client) gjson.Result {
		return lib.NewFlowTeammateAPI(iamClient).Read(api.Params{
			"team":     team.UUID,
			"teammate": teammate.UUID,
		}, nil).Get("teammate")
	}
}

func replicaCreator(iamClient *http.Client, _ store) (objectIdentifier, object, objectGetter) {
	replicaName := "replica_" + uuid.New()
	payload := map[string]interface{}{
		"type":       "Vault",
		"public_key": "-----BEGIN RSA PUBLIC KEY-----\\nMIIBigKCAYEA4cS4zynvKjYPzVVz921JXWLuElks/cs6CBvJK9UAWdapAg4P+Hb8\\ni2ZycG/r4UEjeffpfBQlwqbE75v29mpxhidE+c6Qs5zJfe5+lyIh0AW+m9TC9IFO\\n6o6NV/Z8foyH+oPzf1ZgKcuTXUc7xlRNK2niun9HJHzrUOLVN1CmBbwu0jyXY+Jq\\n8hl5NYsHLuvGwciyBLERtrIM6bp6a0fLl1ypsloZYW80MyTl7oX6V+sdoQlIIBcJ\\nlCevWMqn9NqhlFSCtL0fdQHJLXOqo6H6WZrEIwWbWGjd0iMTtXIcUPbZ04YUEtCf\\nlsV4YewaoXdANZDJRc798UeBuya8AjWiCt+4/TKdCjlpYmhJ2eCrAhGU0sAFoc81\\nmfJmJb/8OgfwOAzJ8BgGYshukwEXUvQX6V8P5EbTQT97N/rjPQyBFkZh61qv5+MM\\naiIfu2D/wOprDg2mibhehbMV7SarUdVLgIhd8FJ46CsA9riuAR0w0ICe5ndt2M6s\\n80Vn72rBbU47AgMBAAE=\\n-----END RSA PUBLIC KEY-----",
	}
	_ = postToPlugin(iamClient, "/replica/"+replicaName, payload) // this endpoint doesn't return anything
	return "replica", replicaName, func(iamClient *http.Client) gjson.Result {
		return getFromPlugin(iamClient, "/replica/"+replicaName)
	}
}

// flant_iam_auth plugin

func authSourceCreator(_ *http.Client, _ store) (objectIdentifier, object, objectGetter) {
	authSourceName := "auth_source_" + uuid.New()
	payload := map[string]interface{}{
		"oidc_discovery_url": "http://oidc-mock:9998",
		"default_role":       "test",
		"entity_alias_name":  "full_identifier",
	}
	authClient := lib.NewConfiguredIamAuthVaultClient()
	_ = postToPlugin(authClient, "/auth_source/"+authSourceName, payload) // this endpoint doesn't return anything
	return "auth_source", authSourceName, func(_ *http.Client) gjson.Result {
		authClient := lib.NewConfiguredIamAuthVaultClient()
		return getFromPlugin(authClient, "/auth_source/"+authSourceName)
	}
}

func authMethodCreator(_ *http.Client, store store) (objectIdentifier, object, objectGetter) {
	authSourceName := store.getObject("auth_source").(string)
	authMethodName := "auth_method_" + uuid.New()
	payload := map[string]interface{}{
		"method_type":             "access_token",
		"source":                  authSourceName,
		"bound_audiences":         "ttps://login.flant.com",
		"token_ttl":               "30m",
		"token_max_ttl":           "1440m",
		"user_claim":              "email",
		"token_policies":          []string{"vst_owner", "list_tenants", "token_renew"},
		"token_no_default_policy": true,
	}
	authClient := lib.NewConfiguredIamAuthVaultClient()
	_ = postToPlugin(authClient, "/auth_method/"+authMethodName, payload) // this endpoint doesn't return anything
	return "auth_method", authMethodName, func(_ *http.Client) gjson.Result {
		authClient := lib.NewConfiguredIamAuthVaultClient()
		return getFromPlugin(authClient, "/auth_method/"+authMethodName)
	}
}

func policyCreator(_ *http.Client, store store) (objectIdentifier, object, objectGetter) {
	role := store.getObject("role").(model.Role)
	authClient := lib.NewConfiguredIamAuthVaultClient()
	policy := lib.NewPolicyAPI(authClient).Create(api.Params{}, url.Values{},
		map[string]interface{}{
			"name":         "policy_" + uuid.New(),
			"rego":         "not_real_rego",
			"roles":        []string{role.Name},
			"claim_schema": "not_real_schema",
		}).Get("policy")
	return "policy", policy, func(_ *http.Client) gjson.Result {
		authClient := lib.NewConfiguredIamAuthVaultClient()
		return lib.NewPolicyAPI(authClient).Read(api.Params{
			"policy": policy.Get("name").String(),
		}, nil).Get("policy")
	}
}

var creators = []objectCreator{
	tenantCreator,
	userCreator, multipassCreator,
	serviceAccountCreator, serviceAccountPasswordCreator,
	featureFlagCreator,
	groupCreator, identitySharingCreator,
	roleCreator, rolebindingCreator, rolebindingApprovalCreator,
	projectCreator,
	serverCreator,
	teamCreator, teammateCreator,
	clientCreator, flantFlowProjectCreator,
	replicaCreator,
	authSourceCreator, authMethodCreator,
	policyCreator,
}

func TestRestoration(t *testing.T) {
	// to use e2e test libs
	RegisterFailHandler(Fail)

	checkers := map[objectIdentifier]objectChecker{}
	s := common.Suite{}
	s.BeforeSuite()
	iamClient := lib.NewConfiguredIamVaultClient()
	store := newMemStore()

	for _, creator := range creators {
		identifier, obj, getter := creator(iamClient, store)
		var oldObj gjson.Result
		store.saveObject(obj, identifier)
		t.Run("creating_"+identifier, func(t *testing.T) {
			oldObj = getter(iamClient)
			if len(oldObj.String()) < 10 {
				checkers[identifier] = func(_ *http.Client, t *testing.T) {
					t.Fatalf("restoring %s not checked due to wrong response of objectGetter", identifier)
				}
				t.Fatalf("getter of %s returns very short result: %q", identifier, oldObj.String())
			}
			checkers[identifier] = func(iamClient *http.Client, t *testing.T) {
				restoredObj := getter(iamClient)
				require.JSONEq(t, oldObj.String(), restoredObj.String())
			}

		})
		fmt.Println(identifier, "is created")
	}
	defer GinkgoRecover()
	fmt.Println("=== restarting vaults ===")
	// time.Sleep(time.Second * 3)
	s.RestartVaults()
	for objectIdentifier, checker := range checkers {
		t.Run("restoration_"+objectIdentifier, func(t *testing.T) { checker(iamClient, t) })
	}
	fmt.Println("===  todo remove vaults ===") //TODO remove
	t.Run("check restoration jwks/jwt and entity/entity_alias staff",
		func(t *testing.T) {
			multipassJWT := store.getObject("multipass-jwt").(string)
			println("==TODO REMOVE========") // TODO
			println(multipassJWT)
			println("==========")
			vst := access_token_or_sapass_auth.Login(true, map[string]interface{}{
				"method": "multipass", "jwt": multipassJWT,
				"roles": []map[string]interface{}{},
			}, lib.GetRootVaultUrl(), 10).ClientToken
			fmt.Printf("vst:%#v", vst)
		})
}

func postToPlugin(vaultClient *http.Client, url string, payload map[string]interface{}) gjson.Result {
	data, err := json.Marshal(payload)
	common.DieOnErr(err)
	r := bytes.NewReader(data)
	resp, err := vaultClient.Post(url, "application/json", r)
	common.DieOnErr(err)
	data, err = io.ReadAll(resp.Body)
	common.DieOnErr(err)
	defer resp.Body.Close()
	return gjson.Parse(string(data)).Get("data")
}

func getFromPlugin(vaultClient *http.Client, url string) gjson.Result {
	resp, err := vaultClient.Get(url)
	common.DieOnErr(err)
	data, err := io.ReadAll(resp.Body)
	common.DieOnErr(err)
	defer resp.Body.Close()
	return gjson.Parse(string(data)).Get("data")
}
