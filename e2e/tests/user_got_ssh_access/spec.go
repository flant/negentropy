package user_got_ssh_access

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/tidwall/gjson"

	"github.com/flant/negentropy/e2e/tests/lib"
	"github.com/flant/negentropy/e2e/tests/lib/tools"
	"github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/api"
	"github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/specs"
	model2 "github.com/flant/negentropy/vault-plugins/flant_iam/extensions/extension_server_access/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/uuid"
)

//go:embed client_sock1.yaml
var clientSocketCFG string

//go:embed client_main.yaml
var clientMainCFG string

//go:embed server_sock1.yaml
var serverSocketCFG string

//go:embed server_main.yaml
var serverMainCFG string

// Config vars through envs
var (
	testServerContainerName string
	testClientContainerName string
	authdPath               string
	cliPath                 string
	serverAccessdPath       string
)

// endpoints for calling functions
var dockerCli *client.Client

var (
	testServerContainer *types.Container
	testClientContainer *types.Container
)

var iamVaultClient *http.Client

// var iamAuthVaultClient *http.Client

var _ = BeforeSuite(func() {
	// TODO read vars from envs!
	authdPath = "/opt/authd/bin/authd"
	cliPath = "/opt/cli/bin/cli"
	serverAccessdPath = "/opt/server-access/bin/server-accessd"
	testServerContainerName = "negentropy_test-server_1"
	testClientContainerName = "negentropy_test-client_1"

	// Open connections, create clients
	var err error
	dockerCli, err = client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())

	Expect(err).ToNot(HaveOccurred())

	testServerContainer, err = getContainerByName(dockerCli, testServerContainerName)
	Expect(err).ToNot(HaveOccurred())
	testClientContainer, err = getContainerByName(dockerCli, testClientContainerName)
	Expect(err).ToNot(HaveOccurred())

	// try to read TEST_VAULT_SECOND_TOKEN, ROOT_VAULT_BASE_URL
	iamVaultClient = lib.NewConfiguredIamVaultClient()

	// try to read TEST_VAULT_SECOND_TOKEN, AUTH_VAULT_BASE_URL
	// authVaulyClient = lib.NewConfiguredIamAuthVaultClient()
}, 1.0)

var _ = Describe("Process of getting ssh access to server by a user", func() {
	const SSHRole = "ssh"
	const ServerRole = "servers"
	const testServerIdentifier = "test-server"
	const testClientIdentifier = "test-client"

	var (
		tenant      model.Tenant
		user        model.User
		project     model.Project
		group       model.Group
		rolebinding model.RoleBinding
		testServer  specs.ServerRegistrationResult
		// testClient  specs.ServerRegistrationResult
		userJWToken string
	)

	Context("initially at flant_iam and flant_iam_auth", func() {
		// TODO here many things are omitted
		// by start.sh already done:
		/*
				function initialize() {
				  docker-exec "vault write flant_iam/configure_extension/server_access roles_for_servers=servers role_for_ssh_access=ssh name=ssh delete_expired_password_seeds_after=1000000 expire_password_seed_after_reveal_in=1000000 last_allocated_uid=10000 --format=json"
				  docker-exec "vault write auth/flant_iam_auth/configure_extension/server_access role_for_ssh_access=ssh name=ssh --format=json"

				  docker-exec "vault write -force flant_iam/jwt/enable" >/dev/null 2>&1
				  docker-exec "vault write -force auth/flant_iam_auth/jwt/enable" >/dev/null 2>&1

				  docker-exec "vault token create -orphan -policy=root -field=token" > /tmp/root_token
				  export VAULT_TOKEN="$(cat /tmp/root_token)"

				  docker-exec "vault token create -orphan -policy=root -field=token > /vault/testdata/token"

				  docker-exec 'cat <<'EOF' > full.hcl
				path "*" {
				  capabilities = ["create", "read", "update", "delete", "list"]
				}
				EOF'

				  docker-exec "vault auth enable approle"
				  docker-exec "vault policy write full full.hcl"
				  docker-exec "vault write auth/approle/role/full secret_id_ttl=30m token_ttl=900s token_policies=full"
				  secretID=$(docker-exec "vault write -format=json -f auth/approle/role/full/secret-id" | jq -r '.data.secret_id')
				  roleID=$(docker-exec "vault read -format=json auth/approle/role/full/role-id" | jq -r '.data.role_id')

				  docker-exec "vault write auth/flant_iam_auth/configure_vault_access \
				    vault_addr=\"http://127.0.0.1:8200\" \
				    vault_tls_server_name=\"vault_host\" \
				    role_name=\"full\" \
				    secret_id_ttl=\"120m\" \
				    approle_mount_point=\"/auth/approle/\" \
				    secret_id=\"$secretID\" \
				    role_id=\"$roleID\" \
				    vault_api_ca=\"\""

				  docker-exec "vault write auth/flant_iam_auth/auth_method/multipass \
				    token_ttl=\"30m\" \
				    token_policies=\"full\" \
				    token_no_default_policy=true \
				    method_type=\"multipass_jwt\""

			  docker-exec 'vault write ssh/config/ca \
				  private_key="-----BEGIN RSA PRIVATE KEY-----
			MIIEogIBAAKCAQEA0/G1wVnF9ufvio1W1XBAD51EU6UP+p0otMVfpap/7DgkyZY0
			WEzJNFGxmR271VdnnWGKYApAyjlhfXheYaY5j2rMmKLJFTCc/X2ntfnJfqZsnJxk
			2S7KYNK+fTa/++68o2tipJZWOAl3O85Zrv0ft9elYM6Vj8keNNO5SGZdvAQGoW3X
			yif4zaaZFWS+Nd60hWeYEwZTCFZmataVLzgbWoTKx9ig71nYNFCVoeao8h8Ynwvi
			797x1pSqsC64CRUPOfVeLG306obeNV8LfNJ5CkgO8ji+BZ8RcMSauQ0iW+chk2J7
			b902JcJpWZi9yYNeEt2kM1vNCG1bkcJw38L9JQIDAQABAoIBABSABaeNCmPmbToG
			j8aXU+ruuEQq7A++kchiauz4P+VWTOCewbNkwfVojXgU8y0ghion3B2MAFZPFInx
			UZe6X0jq+J0u6ao+CIFQXR9x6LZyXIENc4e6SeLxn3E3EXzJy782zNTEodRLvhev
			zubpHt9GYX2qnbbJqj1L2VkSZbCgufku+4y4UbFINMImzwU9kZpc3rbqsYCSzNH+
			x7cCsj1yuXK4Du+k5NX16jFnuZfES05h6Rq26egSkBSrhzTd8eP6YVun6JnJEVOw
			vOqGyGVFMu5toOb8Wnjp5PEj6/c4oRzg+t1tXr1YUoo3RAA17JnqeHopVb8gz1d+
			83bxpEUCgYEA8PiWUZ8Za++w1iE1XPt499504pwSzPTh5vbTl0nbE17YSfA0Dc4S
			vyrZZLjmYKezqebM1Sw9/IJWblk4e6UbcRu0+XsQpeH2Yxv0h/fJTi43tYVyzSKP
			70+IYJJBFJ3xfA8dPN8HqvkKUMHcQvdwU2DEC47wg15yrD0+sETF83cCgYEA4Smr
			603VY5HB/Ic+ehAXMc/CFRB6bs2ytxJL254bmPWJablqHH25xYbe7weJEPGJedaw
			Ek1r3hFjGddxLC4ix5i6YfH4NwRMBh0rU8YmAWHVyHVFlZecGTv+42dBxXzVxPS9
			Hf/DFLy6r3L0FL+pcVxRy9Mm63e3ydnF54ptI0MCgYAHQDOluRfWu5uildU5Owfk
			zXjO6MtYB3ZUsNClGL/S0WPItcWbNLwzrGJmOXoVJnatghhfwbkLxBA9ucmNTuaI
			fMDxUNarZyU2zjyJatdP1uwuNhnCOmwCU25TGZODv0zo4ruKfVuJtXyt+WdbTH7A
			w4SipGZwTYM904nzW95o+QKBgHRWmbO8xZLqzvZx0sAy7CkalcdYekoiEkMxOuzA
			prXDuDpeSQtrkr8SzsFmfVW51zSSzurGAgP9q9zASoNvWx0SNstAwOV8XOOT0r04
			Vo7ERDeNEGUYrtkC/NH2mi82LyXS5pxHeD6QvUzF8oN9/EjMUJ8l/KgRdW7gDLdz
			+KwNAoGAQkNO/RWEsJYUkEUkkObfSqGN75s78fjT1yZ7CX0dUvHv6KC3+f7RmNHM
			2zNxHZ+s+x9hfasJaduoV/hksluY4KUMuZjkfih8CaRIqCY8E/wEYjsyYJzJ4f1u
			C+iz1LopgyIrKSebDzl13Yx9/J6dP3LrC+TiYyYl0bf4a4AStLw=
			-----END RSA PRIVATE KEY-----" \
			          public_key="ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDT8bXBWcX25++KjVbVcEAPnURTpQ/6nSi0xV+lqn/sOCTJljRYTMk0UbGZHbvVV2edYYpgCkDKOWF9eF5hpjmPasyYoskVMJz9fae1+cl+pmycnGTZLspg0r59Nr/77ryja2KkllY4CXc7zlmu/R+316VgzpWPyR4007lIZl28BAahbdfKJ/jNppkVZL413rSFZ5gTBlMIVmZq1pUvOBtahMrH2KDvWdg0UJWh5qjyHxifC+Lv3vHWlKqwLrgJFQ859V4sbfTqht41Xwt80nkKSA7yOL4FnxFwxJq5DSJb5yGTYntv3TYlwmlZmL3Jg14S3aQzW80IbVuRwnDfwv0l"
				  '
			  docker-exec 'vault write ssh/roles/signer -<<"EOH"
			{
			  "allow_user_certificates": true,
			  "allowed_users": "*",
			  "allowed_extensions": "permit-pty,permit-agent-forwarding",
			  "default_extensions": [
			    {
			      "permit-pty": "",
			      "permit-agent-forwarding": ""
			    }
			  ],
			  "key_type": "ca",
			  "ttl": "2m0s"
			}
			EOH"'  >/dev/null 2>&1
			}
		*/
		It("can create some tenant", func() {
			tenant = specs.CreateRandomTenant(lib.NewTenantAPI(iamVaultClient))
			fmt.Printf("Created tenant:%#v\n", tenant)
		})

		It("can create some project at the tenant", func() {
			project = specs.CreateRandomProject(lib.NewProjectAPI(iamVaultClient), tenant.UUID)
			fmt.Printf("Created project:%#v\n", project)
		})

		It("can create some user at the tenant", func() {
			user = specs.CreateRandomUser(lib.NewUserAPI(iamVaultClient), tenant.UUID)
			fmt.Printf("Created user:%#v\n", user)
		})

		It("can create a group with the user", func() {
			group = specs.CreateRandomGroupWithUser(lib.NewGroupAPI(iamVaultClient), user.TenantUUID, user.UUID)
			fmt.Printf("Created group:%#v\n", group)
		})

		It("can create a role 'ssh' if not exists", func() {
			createRoleIfNotExist(SSHRole)
		})

		It("can create a role 'servers' if not exists", func() {
			createRoleIfNotExist(ServerRole)
		})

		It("can create rolebinding for a user in project with the ssh role", func() {
			rolebinding = specs.CreateRoleBinding(lib.NewRoleBindingAPI(iamVaultClient),
				model.RoleBinding{
					TenantUUID: user.TenantUUID,
					Version:    "",
					Identifier: uuid.New(),
					ValidTill:  1000000,
					RequireMFA: false,
					Members:    group.Members,
					AnyProject: true,
					Roles:      []model.BoundRole{{Name: SSHRole, Options: map[string]interface{}{}}},
				})
			fmt.Printf("Created rolebinding:%#v\n", rolebinding)
		})

		It("can register as a server 'test_server'", func() {
			testServer = specs.RegisterServer(lib.NewServerAPI(iamVaultClient),
				model2.Server{
					TenantUUID:  tenant.UUID,
					ProjectUUID: project.UUID,
					Identifier:  testServerIdentifier,
				})
			fmt.Printf("Created testServer Server:%#v\n", testServer)
		})

		It("can add connection_info for a server 'test_server'", func() {
			s := specs.UpdateConnectionInfo(lib.NewConnectionInfoAPI(iamVaultClient),
				model2.Server{
					UUID:        testServer.ServerUUID,
					TenantUUID:  tenant.UUID,
					ProjectUUID: project.UUID,
				},
				model2.ConnectionInfo{
					Hostname: testServerIdentifier,
				},
			)
			fmt.Printf("connection_info is updated: %#v\n", s.ConnectionInfo)
		})

		It("can create and get multipass for a user", func() {
			_, userJWToken = specs.CreateUserMultipass(lib.NewUserMultipassAPI(iamVaultClient),
				user, "test", 100*time.Second, 1000*time.Second, []string{"ssh"})
			fmt.Printf("user JWToken: : %#v\n", userJWToken)
		})
	})

	Describe("Test_server_configuring", func() {
		It("Test_server has authd", func() {
			err := checkFileExistAtContainer(dockerCli, testServerContainer, authdPath, "f")
			Expect(err).ToNot(HaveOccurred())
		})

		It("Test_server has server-accessd", func() {
			err := checkFileExistAtContainer(dockerCli, testServerContainer, serverAccessdPath, "f")
			Expect(err).ToNot(HaveOccurred())
		})

		It("Authd can be configured and run at Test_server", func() {
			err := createIfNotExistsDirectoryAtContainer(dockerCli, testServerContainer,
				"/etc/flant/negentropy/authd-conf.d")
			Expect(err).ToNot(HaveOccurred())

			err = writeFileToContainer(dockerCli, testServerContainer,
				"/etc/flant/negentropy/authd-conf.d/main.yaml", serverMainCFG)
			Expect(err).ToNot(HaveOccurred())

			err = writeFileToContainer(dockerCli, testServerContainer,
				"/etc/flant/negentropy/authd-conf.d/sock1.yaml", serverSocketCFG)
			Expect(err).ToNot(HaveOccurred())

			err = writeFileToContainer(dockerCli, testServerContainer,
				"/opt/authd/server-jwt", testServer.MultipassJWT)
			Expect(err).ToNot(HaveOccurred())

			executeCommandAtContainer(dockerCli, testServerContainer,
				[]string{"/bin/bash", "-c", "chmod 600 /opt/authd/server-jwt"}, nil)

			killAllInstancesOfProcessAtContainer(dockerCli, testServerContainer, authdPath)
			runDaemonAtContainer(dockerCli, testServerContainer, authdPath, "server_authd.log")
			time.Sleep(time.Second)
			pidAuthd := firstProcessPIDAtContainer(dockerCli, testServerContainer, authdPath)
			Expect(pidAuthd).Should(BeNumerically(">", 0), "pid greater 0")
		})
		It("Test_server nsswitch is configured properly", func() {
			// TODO check content /etc/nsswitch.conf
		})

		It("Test_server server_accessd can be configured and run", func() {
			err := createIfNotExistsDirectoryAtContainer(dockerCli, testServerContainer,
				"/opt/serveraccessd")
			Expect(err).ToNot(HaveOccurred())

			acccesdCFG := fmt.Sprintf("tenant: %s\n", tenant.UUID) +
				fmt.Sprintf("project: %s\n", project.UUID) +
				fmt.Sprintf("server: %s\n", testServer.ServerUUID) +
				"database: /opt/serveraccessd/server-accessd.db\n" +
				"authdSocketPath: /run/sock1.sock"

			err = writeFileToContainer(dockerCli, testServerContainer,
				"/opt/server-access/config.yaml", acccesdCFG)
			Expect(err).ToNot(HaveOccurred())

			killAllInstancesOfProcessAtContainer(dockerCli, testServerContainer, serverAccessdPath)
			runDaemonAtContainer(dockerCli, testServerContainer, serverAccessdPath, "server_accessd.log")
			time.Sleep(time.Second)
			pidServerAccessd := firstProcessPIDAtContainer(dockerCli, testServerContainer, serverAccessdPath)
			Expect(pidServerAccessd).Should(BeNumerically(">", 0), "pid greater 0")

			authKeysFilePath := filepath.Join("/home", user.Identifier, ".ssh", "authorized_keys")
			contentAuthKeysFile := executeCommandAtContainer(dockerCli, testServerContainer,
				[]string{"/bin/bash", "-c", "cat " + authKeysFilePath}, nil)
			Expect(contentAuthKeysFile).To(HaveLen(1), "cat authorize should have one line text")
			principal := calculatePrincipal(testServer.ServerUUID, user.UUID)
			Expect(contentAuthKeysFile[0]).To(MatchRegexp(".+cert-authority,principals=\"" + principal + "\" ssh-rsa.{373}"))
		})
	})

	Describe("Test_сlient configuring", func() {
		It("TestClient has authd", func() {
			err := checkFileExistAtContainer(dockerCli, testClientContainer, authdPath, "f")
			Expect(err).ToNot(HaveOccurred())
		})

		It("Test_сlient has cli", func() {
			err := checkFileExistAtContainer(dockerCli, testClientContainer, cliPath, "f")
			Expect(err).ToNot(HaveOccurred())
		})

		It("Authd can be configured and runned at Test_сlient", func() {
			err := createIfNotExistsDirectoryAtContainer(dockerCli, testClientContainer,
				"/etc/flant/negentropy/authd-conf.d")
			Expect(err).ToNot(HaveOccurred())

			err = writeFileToContainer(dockerCli, testClientContainer,
				"/etc/flant/negentropy/authd-conf.d/main.yaml", clientMainCFG)
			Expect(err).ToNot(HaveOccurred())

			err = writeFileToContainer(dockerCli, testClientContainer,
				"/etc/flant/negentropy/authd-conf.d/sock1.yaml", clientSocketCFG)
			Expect(err).ToNot(HaveOccurred())

			err = writeFileToContainer(dockerCli, testClientContainer,
				"/opt/authd/client-jwt", userJWToken)
			Expect(err).ToNot(HaveOccurred())

			executeCommandAtContainer(dockerCli, testClientContainer,
				[]string{"/bin/bash", "-c", "chmod 600 /opt/authd/client-jwt"}, nil)

			killAllInstancesOfProcessAtContainer(dockerCli, testClientContainer, authdPath)
			runDaemonAtContainer(dockerCli, testClientContainer, authdPath, "client_authd.log")
			time.Sleep(time.Second)
			pidAuthd := firstProcessPIDAtContainer(dockerCli, testClientContainer, authdPath)
			Expect(pidAuthd).Should(BeNumerically(">", 0), "pid greater 0")
		})

		It("Cli ssh can run, write through ssh, and remove tmp files, [-t XXX --all-projects]", func() {
			Expect(directoryAtContainerNotExistOrEmpty(dockerCli, testClientContainer, "/tmp/flint")).To(BeTrue(),
				"/tmp/flint files doesn't exist before start")

			// TODO Redo after design CLI
			runningCliCmd := fmt.Sprintf("/opt/cli/bin/cli  ssh -t %s --all-projects\n", tenant.Identifier)
			sshCmd := fmt.Sprintf("ssh -oStrictHostKeyChecking=accept-new %s.%s", project.Identifier, testServerIdentifier)
			testFilePath := fmt.Sprintf("/home/%s/test.txt", user.Identifier)
			touchCommand := "touch " + testFilePath
			fmt.Printf("\n%s\n", runningCliCmd)
			fmt.Printf("\n%s\n", sshCmd)
			fmt.Printf("\n%s\n", touchCommand)
			output := executeCommandAtContainer(dockerCli, testClientContainer, []string{
				"/bin/bash", "-c",
				runningCliCmd,
			},
				[]string{
					sshCmd,
					touchCommand,
					"exit", "exit",
				})

			writeLogToFile(output, "cli.log")

			Expect(directoryAtContainerNotExistOrEmpty(dockerCli, testClientContainer, "/tmp/flint")).To(BeTrue(),
				"/tmp/flint files doesn't exist after closing cli")

			Expect(checkFileExistAtContainer(dockerCli, testServerContainer, testFilePath, "f")).
				ToNot(HaveOccurred(), "after run cli ssh - test file is created at server")

			Expect(checkFileExistAtContainer(dockerCli, testClientContainer, "/tmp/flint", "d")).
				ToNot(HaveOccurred(), "after run cli ssh - tmp dir exists at client container")

			Expect(directoryAtContainerNotExistOrEmpty(dockerCli, testClientContainer, "/tmp/flint")).To(BeTrue(),
				"/tmp/flint is empty  after closing cli")
		})

		//It("Cli ssh can run, write through ssh, and remove tmp files, [-t XXX -p YYY]", func() {
		//	Expect(directoryAtContainerNotExistOrEmpty(dockerCli, testClientContainer, "/tmp/flint")).To(BeTrue(),
		//		"/tmp/flint files doesn't exist before start")
		//
		//	// TODO Redo after design CLI
		//	runningCliCmd := fmt.Sprintf("/opt/cli/bin/cli  ssh -t %s -p %s\n", tenant.Identifier, project.Identifier)
		//	sshCmd := fmt.Sprintf("ssh -oStrictHostKeyChecking=accept-new %s.%s", project.Identifier, testServerIdentifier)
		//	testFilePath := fmt.Sprintf("/home/%s/test2.txt", user.Identifier)
		//	touchCommand := "touch " + testFilePath
		//	fmt.Printf("\n%s\n", runningCliCmd)
		//	fmt.Printf("\n%s\n", sshCmd)
		//	fmt.Printf("\n%s\n", touchCommand)
		//	output := executeCommandAtContainer(dockerCli, testClientContainer, []string{
		//		"/bin/bash", "-c",
		//		runningCliCmd,
		//	},
		//		[]string{
		//			sshCmd,
		//			touchCommand,
		//			"exit", "exit",
		//		})
		//
		//	writeLogToFile(output, "cli.log")
		//
		//	Expect(directoryAtContainerNotExistOrEmpty(dockerCli, testClientContainer, "/tmp/flint")).To(BeTrue(),
		//		"/tmp/flint files doesn't exist after closing cli")
		//
		//	Expect(checkFileExistAtContainer(dockerCli, testServerContainer, testFilePath, "f")).
		//		ToNot(HaveOccurred(), "after run cli ssh - test file is created at server")
		//
		//	Expect(checkFileExistAtContainer(dockerCli, testClientContainer, "/tmp/flint", "d")).
		//		ToNot(HaveOccurred(), "after run cli ssh - tmp dir exists at client container")
		//
		//	Expect(directoryAtContainerNotExistOrEmpty(dockerCli, testClientContainer, "/tmp/flint")).To(BeTrue(),
		//		"/tmp/flint is empty  after closing cli")
		//})
		//
		//It("Cli ssh can run, write through ssh, and remove tmp files, [--all-tenants --all-projects ]", func() {
		//	Expect(directoryAtContainerNotExistOrEmpty(dockerCli, testClientContainer, "/tmp/flint")).To(BeTrue(),
		//		"/tmp/flint files doesn't exist before start")
		//
		//	// TODO Redo after design CLI
		//	runningCliCmd := fmt.Sprintf("/opt/cli/bin/cli  ssh --all-tenants --all-projects\n")
		//	sshCmd := fmt.Sprintf("ssh -oStrictHostKeyChecking=accept-new %s.%s", project.Identifier, testServerIdentifier)
		//	testFilePath := fmt.Sprintf("/home/%s/test3.txt", user.Identifier)
		//	touchCommand := "touch " + testFilePath
		//	fmt.Printf("\n%s\n", runningCliCmd)
		//	fmt.Printf("\n%s\n", sshCmd)
		//	fmt.Printf("\n%s\n", touchCommand)
		//	output := executeCommandAtContainer(dockerCli, testClientContainer, []string{
		//		"/bin/bash", "-c",
		//		runningCliCmd,
		//	},
		//		[]string{
		//			sshCmd,
		//			touchCommand,
		//			"exit", "exit",
		//		})
		//
		//	writeLogToFile(output, "cli.log")
		//
		//	Expect(directoryAtContainerNotExistOrEmpty(dockerCli, testClientContainer, "/tmp/flint")).To(BeTrue(),
		//		"/tmp/flint files doesn't exist after closing cli")
		//
		//	Expect(checkFileExistAtContainer(dockerCli, testServerContainer, testFilePath, "f")).
		//		ToNot(HaveOccurred(), "after run cli ssh - test file is created at server")
		//
		//	Expect(checkFileExistAtContainer(dockerCli, testClientContainer, "/tmp/flint", "d")).
		//		ToNot(HaveOccurred(), "after run cli ssh - tmp dir exists at client container")
		//
		//	Expect(directoryAtContainerNotExistOrEmpty(dockerCli, testClientContainer, "/tmp/flint")).To(BeTrue(),
		//		"/tmp/flint is empty  after closing cli")
		//})
		//
		//It("Cli ssh can run, write through ssh, and remove tmp files, [--all-tenants -p XXX ]", func() {
		//	Expect(directoryAtContainerNotExistOrEmpty(dockerCli, testClientContainer, "/tmp/flint")).To(BeTrue(),
		//		"/tmp/flint files doesn't exist before start")
		//
		//	// TODO Redo after design CLI
		//	runningCliCmd := fmt.Sprintf("/opt/cli/bin/cli  ssh --all-tenants -p %s\n", project.Identifier)
		//	sshCmd := fmt.Sprintf("ssh -oStrictHostKeyChecking=accept-new %s.%s", project.Identifier, testServerIdentifier)
		//	testFilePath := fmt.Sprintf("/home/%s/test4.txt", user.Identifier)
		//	touchCommand := "touch " + testFilePath
		//	fmt.Printf("\n%s\n", runningCliCmd)
		//	fmt.Printf("\n%s\n", sshCmd)
		//	fmt.Printf("\n%s\n", touchCommand)
		//	output := executeCommandAtContainer(dockerCli, testClientContainer, []string{
		//		"/bin/bash", "-c",
		//		runningCliCmd,
		//	},
		//		[]string{
		//			sshCmd,
		//			touchCommand,
		//			"exit", "exit",
		//		})
		//
		//	writeLogToFile(output, "cli.log")
		//
		//	Expect(directoryAtContainerNotExistOrEmpty(dockerCli, testClientContainer, "/tmp/flint")).To(BeTrue(),
		//		"/tmp/flint files doesn't exist after closing cli")
		//
		//	Expect(checkFileExistAtContainer(dockerCli, testServerContainer, testFilePath, "f")).
		//		ToNot(HaveOccurred(), "after run cli ssh - test file is created at server")
		//
		//	Expect(checkFileExistAtContainer(dockerCli, testClientContainer, "/tmp/flint", "d")).
		//		ToNot(HaveOccurred(), "after run cli ssh - tmp dir exists at client container")
		//
		//	Expect(directoryAtContainerNotExistOrEmpty(dockerCli, testClientContainer, "/tmp/flint")).To(BeTrue(),
		//		"/tmp/flint is empty  after closing cli")
		//})
	})
})

func writeLogToFile(output []string, logFilePath string) {
	logFile, err := os.Create(logFilePath)
	if err != nil {
		panic(err)
	}
	for _, s := range output {
		logFile.WriteString(s)
	}
	logFile.Close()
}

func directoryAtContainerNotExistOrEmpty(cli *client.Client, container *types.Container, directoryPath string) bool {
	ctx := context.Background()
	config := types.ExecConfig{
		AttachStdin:  true,
		AttachStderr: true,
		AttachStdout: true,
		Cmd:          []string{"/bin/bash", "-c", "ls " + directoryPath},
	}

	IDResp, err := cli.ContainerExecCreate(ctx, container.ID, config)
	Expect(err).ToNot(HaveOccurred(), "error execution at container")

	resp, err := cli.ContainerExecAttach(ctx, IDResp.ID, types.ExecStartCheck{})
	defer resp.Close()
	Expect(err).ToNot(HaveOccurred(), "error attaching execution at container")

	output := []string{}
	var text string
	for err == nil {
		text, err = resp.Reader.ReadString('\n')
		if text != "" {
			output = append(output, text)
		}
	}
	if err.Error() != "EOF" {
		Expect(err).ToNot(HaveOccurred(), "error response reading at container")
		return false
	}
	if len(output) == 0 ||
		(len(output) == 1 && strings.HasSuffix(output[0], "ls: cannot access '"+directoryPath+"': No such file or directory\n")) {
		return true
	}
	return false
}

func calculatePrincipal(serverUUID string, userUUID model.UserUUID) string {
	principalHash := sha256.New()
	principalHash.Write([]byte(serverUUID))
	principalHash.Write([]byte(userUUID))
	principalSum := principalHash.Sum(nil)
	return fmt.Sprintf("%x", principalSum)
}

func createRoleIfNotExist(roleName string) {
	roleAPI := lib.NewRoleAPI(iamVaultClient)
	var roleNotExists bool
	rawRole := roleAPI.Read(tools.Params{
		"name":         roleName,
		"expectStatus": api.ExpectStatus("%d > 0"),
		"expectPayload": func(json gjson.Result) {
			roleNotExists = json.String() == "{\"error\":\"not found\"}"
		},
	}, nil)
	if roleNotExists {
		rawRole = roleAPI.Create(tools.Params{}, url.Values{},
			map[string]interface{}{
				"name":        roleName,
				"description": roleName,
				"scope":       model.RoleScopeProject,
			})
	}
	fmt.Printf("role: %s\n", rawRole.String())
}

func checkFileExistAtContainer(cli *client.Client, container *types.Container, path string, fileTypeFlag string) error {
	ctx := context.Background()
	config := types.ExecConfig{
		AttachStdin:  true,
		AttachStderr: true,
		AttachStdout: true,
		Cmd:          []string{"/bin/bash", "-c", "test -" + fileTypeFlag + " " + path + " && echo true"},
	}

	IDResp, err := cli.ContainerExecCreate(ctx, container.ID, config)
	Expect(err).ToNot(HaveOccurred(), "error execution at container")

	resp, err := cli.ContainerExecAttach(ctx, IDResp.ID, types.ExecStartCheck{})
	defer resp.Close()
	Expect(err).ToNot(HaveOccurred(), "error attaching execution at container")

	text, err := resp.Reader.ReadString('\n')
	if err != nil && err.Error() == "EOF" {
		return fmt.Errorf("file %s not found", path)
	}
	Expect(err).ToNot(HaveOccurred(), "error response reading at container")
	if strings.HasSuffix(text, "true\n") {
		fmt.Printf("file %s at container %s exists\n", path, container.Names)
		return nil
	}
	return fmt.Errorf("unexpected output checking file exists: %s", text)
}

func getContainerByName(cli *client.Client, name string) (*types.Container, error) {
	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil {
		return nil, err
	}
	for _, c := range containers {
		for _, n := range c.Names {
			if n == "/"+name {
				if c.State != "running" {
					return nil, errors.New("Container with name " + name + " has state: " + c.State)
				}
				return &c, nil
			}
		}
	}

	return nil, errors.New("Container with name " + name + " not found")
}

func createIfNotExistsDirectoryAtContainer(cli *client.Client, container *types.Container, path string) error {
	lastSeparator := strings.LastIndex(path, string(os.PathSeparator))
	if lastSeparator != 0 {
		err := createIfNotExistsDirectoryAtContainer(cli, container, path[0:lastSeparator])
		if err != nil {
			return err
		}
	}
	ctx := context.Background()
	config := types.ExecConfig{
		AttachStdin:  true,
		AttachStderr: true,
		AttachStdout: true,
		Cmd:          []string{"/bin/bash", "-c", "mkdir " + path},
	}

	IDResp, err := cli.ContainerExecCreate(ctx, container.ID, config)
	Expect(err).ToNot(HaveOccurred(), "error execution at container")

	resp, err := cli.ContainerExecAttach(ctx, IDResp.ID, types.ExecStartCheck{})
	defer resp.Close()
	Expect(err).ToNot(HaveOccurred(), "error attaching execution at container")

	text, err := resp.Reader.ReadString('\n')
	if err != nil && err.Error() == "EOF" {
		fmt.Printf("Directory %s at container %s created \n", path, container.Names)
		return nil
	}
	Expect(err).ToNot(HaveOccurred(), "error response reading at container")
	if strings.Contains(text, "File exists") {
		fmt.Printf("Directory %s at container %s exists\n", path, container.Names)
		return nil
	}
	return fmt.Errorf("unexpected output creating directory: %s", text)
}

func writeFileToContainer(cli *client.Client, container *types.Container, path string, content string) error {
	ctx := context.Background()
	config := types.ExecConfig{
		AttachStdin:  true,
		AttachStderr: true,
		AttachStdout: true,
		Cmd:          []string{"/bin/bash", "-c", "echo \"" + content + "\" > " + path},
	}

	IDResp, err := cli.ContainerExecCreate(ctx, container.ID, config)
	Expect(err).ToNot(HaveOccurred(), "error execution at container")

	resp, err := cli.ContainerExecAttach(ctx, IDResp.ID, types.ExecStartCheck{})
	defer resp.Close()
	Expect(err).ToNot(HaveOccurred(), "error attaching execution at container")

	text, err := resp.Reader.ReadString('\n')
	if err != nil && err.Error() == "EOF" {
		fmt.Printf("this content: \n %s \n ==> has been written to file %s at container  %s \n", content, path, container.Names)
		return nil
	}
	Expect(err).ToNot(HaveOccurred(), "error response reading at container")
	return fmt.Errorf("unexpected output creating directory: %s", text)
}

func executeCommandAtContainer(cli *client.Client, container *types.Container, cmd []string, extraInputToSTDIN []string) []string {
	ctx := context.Background()
	config := types.ExecConfig{
		AttachStdin:  true,
		AttachStderr: true,
		AttachStdout: true,
		Cmd:          cmd,
	}

	IDResp, err := cli.ContainerExecCreate(ctx, container.ID, config)
	Expect(err).ToNot(HaveOccurred(), "error execution at container")

	resp, err := cli.ContainerExecAttach(ctx, IDResp.ID, types.ExecStartCheck{})
	defer resp.Close()
	Expect(err).ToNot(HaveOccurred(), "error attaching execution at container")

	go func() {
		for _, input := range extraInputToSTDIN {
			time.Sleep(time.Millisecond * 500)
			resp.Conn.Write([]byte(input + "\n"))
		}
	}()

	output := []string{}
	var text string
	for err == nil {
		text, err = resp.Reader.ReadString('\n')
		if text != "" {
			output = append(output, text)
		}
	}

	if err != nil && err.Error() == "EOF" {
		fmt.Printf("command: \n %s \n ==> has been succeseed at  at container  %s \n", cmd, container.Names)
		return output
	}
	Expect(err).ToNot(HaveOccurred(), "error response reading at container")
	return nil
}

func runDaemonAtContainer(cli *client.Client, container *types.Container, daemonPath string, logFilePath string) {
	ctx := context.Background()
	config := types.ExecConfig{
		AttachStdin:  true,
		AttachStderr: true,
		AttachStdout: true,
		Cmd:          []string{daemonPath},
	}

	IDResp, err := cli.ContainerExecCreate(ctx, container.ID, config)
	Expect(err).ToNot(HaveOccurred(), "error execution at container")

	resp, err := cli.ContainerExecAttach(ctx, IDResp.ID, types.ExecStartCheck{})
	Expect(err).ToNot(HaveOccurred(), "error attaching execution at container")
	go func() {
		logFile, err := os.Create(logFilePath)
		if err != nil {
			panic(err)
		}
		var text string
		for err == nil {
			text, err = resp.Reader.ReadString('\n')
			logFile.WriteString(text)
			logFile.Sync()
		}
		if err.Error() != "EOF" {
			logFile.Write([]byte(fmt.Sprintf("reading from container %s:%s", container.Names, err)))
		}
		logFile.Close()
		defer resp.Close()
	}()
}

func firstProcessPIDAtContainer(cli *client.Client, container *types.Container, processPath string) int {
	ctx := context.Background()
	config := types.ExecConfig{
		AttachStdin:  true,
		AttachStderr: true,
		AttachStdout: true,
		Cmd:          []string{"/bin/bash", "-c", "ps ax"},
	}

	IDResp, err := cli.ContainerExecCreate(ctx, container.ID, config)
	Expect(err).ToNot(HaveOccurred(), "error execution at container")

	resp, err := cli.ContainerExecAttach(ctx, IDResp.ID, types.ExecStartCheck{})
	defer resp.Close()
	Expect(err).ToNot(HaveOccurred(), "error attaching execution at container")

	text, err := resp.Reader.ReadString('\n')
	for err == nil {
		if strings.HasSuffix(text, processPath+"\n") {
			arr := strings.Split(text, " ")
			for _, c := range arr {
				if c != "" {
					pid, err := strconv.ParseInt(c, 10, 32)
					Expect(err).ToNot(HaveOccurred(), "error attaching execution at container")
					return int(pid)
				}
			}
		}
		text, err = resp.Reader.ReadString('\n')
	}
	if err != nil && err.Error() == "EOF" {
		return 0
	}
	Expect(err).ToNot(HaveOccurred(), "error response reading at container")
	return 0
}

func killProcessAtContainer(cli *client.Client, container *types.Container, processPid int) {
	ctx := context.Background()
	config := types.ExecConfig{
		AttachStdin:  true,
		AttachStderr: true,
		AttachStdout: true,
		Cmd:          []string{"/bin/bash", "-c", "kill -9 " + strconv.Itoa(processPid)},
	}

	IDResp, err := cli.ContainerExecCreate(ctx, container.ID, config)
	Expect(err).ToNot(HaveOccurred(), "error execution at container")

	resp, err := cli.ContainerExecAttach(ctx, IDResp.ID, types.ExecStartCheck{})
	defer resp.Close()
	Expect(err).ToNot(HaveOccurred(), "error attaching execution at container")
	fmt.Println(err)
}

func killAllInstancesOfProcessAtContainer(cli *client.Client, container *types.Container, processPath string) {
	for {
		pid := firstProcessPIDAtContainer(dockerCli, container, processPath)
		if pid == 0 {
			break
		}
		killProcessAtContainer(cli, container, pid)
	}
}
