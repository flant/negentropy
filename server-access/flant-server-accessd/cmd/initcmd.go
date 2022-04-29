package main

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"github.com/hashicorp/vault/api"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/flant/negentropy/cli/pkg"
	model2 "github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_server_access/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
)

const (
	passwordFlagName     = "password"
	projectFlagName      = "project"
	tenantFlagName       = "tenant"
	identifierFlagName   = "identifier"
	labelFlagName        = "label"
	annotationFlagName   = "annotation"
	mutipassPahtFlagName = "store_multipass_to"
	vaultURLFlagName     = "vault_url"
	connectionHostFlag   = "connection_hostname"
	connectionPortFlag   = "connection_port"
)

func InitCMD() *cobra.Command {
	var initErr error
	cmd := &cobra.Command{
		Use:   "init",
		Short: "register a server into negentropy",
		Long:  `register a server into negentropy`,
		Run:   initCmd(&initErr),
		PostRunE: func(command *cobra.Command, args []string) error {
			return initErr
		},
	}
	hostname, err := os.Hostname()
	if err != nil {
		hostname = ""
	}
	//cmd.PersistentFlags().StringP(vaultURLFlagName, string(vaultURLFlagName[0]), "http://localhost:8300",
	cmd.PersistentFlags().StringP(vaultURLFlagName, string(vaultURLFlagName[0]), "http://localhost:8300",
		"specify url root vault with iam plugin")
	cmd.PersistentFlags().String(passwordFlagName, "",
		"base64 encoded service_account_password_uuid:service_account_password_secret")
	cmd.PersistentFlags().StringP(tenantFlagName, string(tenantFlagName[0]), "",
		"specify tenant identifier")
	cmd.PersistentFlags().StringP(projectFlagName, string(projectFlagName[0]), "",
		"specify project identifier")
	cmd.PersistentFlags().String(identifierFlagName, hostname,
		fmt.Sprintf("specify server identifier, by default hostname (%s) will be used", hostname))
	cmd.PersistentFlags().String(mutipassPahtFlagName, "/var/lib/flant/negentropy-authd.jwt",
		"specify path for storing multipassJWT, /var/lib/flant/negentropy-authd.jwt by default")
	cmd.PersistentFlags().StringToStringP(labelFlagName, string(labelFlagName[0]), nil,
		"specify server labels")
	cmd.PersistentFlags().StringToStringP(annotationFlagName, string(annotationFlagName[0]), nil,
		"specify server annotations")
	cmd.PersistentFlags().String(connectionHostFlag, "",
		"specify connection hostname, by default server identifier will be used")
	cmd.PersistentFlags().String(connectionPortFlag, "22",
		"specify connection port, default value 22")
	return cmd
}

func initCmd(outErr *error) func(*cobra.Command, []string) {
	return func(cmd *cobra.Command, args []string) {
		//var serverIdentifier, tenantIdentifier, projectIdentifier string
		flags := cmd.Flags()
		vaultUrl, err := checkStringNotEmpty(flags.GetString(vaultURLFlagName))
		if err != nil {
			*outErr = fmt.Errorf(vaultURLFlagName+":%w", err)
			return
		}
		serviceAccountPassword, err := getServiceAccountPassword(flags)
		if err != nil {
			*outErr = fmt.Errorf(passwordFlagName+":%w", err)
			return
		}
		tenantIdentifier, err := checkStringNotEmpty(flags.GetString(tenantFlagName))
		if err != nil {
			*outErr = fmt.Errorf(tenantFlagName+":%w", err)
			return
		}
		projectIdentifier, err := checkStringNotEmpty(flags.GetString(projectFlagName))
		if err != nil {
			*outErr = fmt.Errorf(projectFlagName+":%w", err)
			return
		}
		serverIdentifier, err := checkStringNotEmpty(flags.GetString(identifierFlagName))
		if err != nil {
			*outErr = fmt.Errorf(identifierFlagName+":%w", err)
			return
		}
		labels, err := flags.GetStringToString(labelFlagName)
		if err != nil {
			*outErr = fmt.Errorf(labelFlagName+":%w", err)
			return
		}
		annotations, err := flags.GetStringToString(annotationFlagName)
		if err != nil {
			*outErr = fmt.Errorf(annotationFlagName+":%w", err)
			return
		}

		connectionHostname, err := flags.GetString(connectionHostFlag)
		if err != nil {
			*outErr = fmt.Errorf(connectionHostFlag+":%w", err)
			return
		}
		if connectionHostname == "" {
			connectionHostname = serverIdentifier
		}

		connectionPort, err := flags.GetString(connectionPortFlag)
		if err != nil {
			*outErr = fmt.Errorf(connectionPortFlag+":%w", err)
			return
		}
		pathMultipass, err := checkStringNotEmpty(flags.GetString(mutipassPahtFlagName))
		if err != nil {
			*outErr = fmt.Errorf(mutipassPahtFlagName+":%w", err)
			return
		}

		server, multipass, err := registerServer(vaultUrl, *serviceAccountPassword, tenantIdentifier, projectIdentifier,
			serverIdentifier, labels, annotations, connectionHostname, connectionPort)
		if err != nil {
			*outErr = fmt.Errorf("registering server:%w", err)
			return
		}

		err = writeFiles(server, multipass, pathMultipass)

		if err != nil {
			*outErr = err
		}
	}
}

func writeFiles(server *model2.Server, multipass *model.MultipassJWT, pathMultipass string) error {
	fmt.Printf("registration details:%#v\n", *server)
	errBuilder := strings.Builder{}

	err := storeJWTToFile(*multipass, pathMultipass)
	if err != nil {
		msg := fmt.Sprintf("error writing multipass to file:%s, try manually store multipass to file and set file permissions to 600:\n%s\n", err.Error(), *multipass)
		errBuilder.WriteString(msg)
	}

	cfgFilePath := "/opt/server-access/config.yaml"
	acccesdCFG := fmt.Sprintf("tenant: %s\n", server.TenantUUID) +
		fmt.Sprintf("project: %s\n", server.ProjectUUID) +
		fmt.Sprintf("server: %s\n", server.UUID) +
		"database: /opt/serveraccessd/server-accessd.db\n" +
		"authdSocketPath: /run/sock1.sock"
	err = storeCFGToFile(acccesdCFG, cfgFilePath)

	if err != nil {
		msg := fmt.Sprintf("error writing config to file:%s, try manually store config:\n\n%s\n\n", err.Error(), acccesdCFG)
		errBuilder.WriteString(msg)
	}
	if errBuilder.Len() > 0 {
		return fmt.Errorf(errBuilder.String())
	}
	return nil
}

func storeJWTToFile(multipass model.MultipassJWT, multipassPath string) error {
	file, err := os.Create(multipassPath)
	defer file.Close() // nolint:errcheck
	if err != nil {
		return err
	}
	_, err = file.WriteString(multipass)
	if err != nil {
		return err
	}
	err = os.Chmod(multipassPath, 0600)
	return err
}

func storeCFGToFile(cfg string, cfgFilePath string) error {
	file, err := os.Create(cfgFilePath)
	defer file.Close() // nolint:errcheck
	if err != nil {
		return err
	}

	_, err = file.WriteString(cfg)
	return err
}

func getServiceAccountPassword(flags *pflag.FlagSet) (*model.ServiceAccountPassword, error) {
	encodedPasswordPair, err := checkStringNotEmpty(flags.GetString(passwordFlagName))
	if err != nil {
		return nil, fmt.Errorf("getting param: %w", err)
	}
	decodedPasswordPair, err := base64.StdEncoding.DecodeString(encodedPasswordPair)
	if err != nil {
		return nil, fmt.Errorf("decoding:%w", err)
	}
	separeted := strings.SplitN(string(decodedPasswordPair), ":", 2)
	if len(separeted) != 2 {
		return nil, fmt.Errorf("not found ':' in decoded pair")
	}
	serviceAccountPassword := model.ServiceAccountPassword{
		UUID:   separeted[0],
		Secret: separeted[1],
	}
	return &serviceAccountPassword, nil
}

func checkStringNotEmpty(s string, err error) (string, error) {
	if err != nil {
		return s, err
	}
	if s == "" {
		return "", fmt.Errorf("empty string is passed")
	}
	return s, nil
}

func registerServer(vaultURL string, saPassword model.ServiceAccountPassword, tIdentifier string, pIdentifier string,
	sIdentifier string, labels map[string]string, annotations map[string]string,
	connectionHostname string, connectionPort string) (*model2.Server, *model.MultipassJWT, error) {
	// get client just for tenants list
	cl, err := getVaultClientAuthorizedWithSAPass(vaultURL, saPassword, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("gettting vault client: %w", err)
	}
	tenant, err := cl.GetTenantByIdentifier(tIdentifier)
	if err != nil {
		return nil, nil, fmt.Errorf("gettting tenant by identifier: %w", err)
	}
	// get client just for project list
	cl, err = getVaultClientAuthorizedWithSAPass(vaultURL, saPassword, []map[string]interface{}{
		{"role": "iam_auth_read", "tenant_uuid": tenant.UUID}})
	if err != nil {
		return nil, nil, fmt.Errorf("gettting vault client: %w", err)
	}
	project, err := cl.GetProjectByIdentifier(tenant.UUID, pIdentifier)
	if err != nil {
		return nil, nil, fmt.Errorf("gettting project by identifier: %w", err)
	}
	// get client for registering server
	cl, err = getVaultClientAuthorizedWithSAPass(vaultURL, saPassword, []map[string]interface{}{
		{"role": "servers.register", "tenant_uuid": tenant.UUID, "project_uuid": project.UUID}})
	if err != nil {
		return nil, nil, fmt.Errorf("gettting vault client: %w", err)
	}
	serverUUID, multipassJWT, err := cl.RegisterServer(
		model2.Server{
			TenantUUID:  tenant.UUID,
			ProjectUUID: project.UUID,
			Identifier:  sIdentifier,
			Labels:      labels,
			Annotations: annotations,
		})
	if err != nil {
		return nil, nil, fmt.Errorf("registering server: %w", err)
	}
	server, err := cl.UpdateServerConnectionInfo(tenant.UUID, project.UUID, serverUUID, model2.ConnectionInfo{
		Hostname: connectionHostname,
		Port:     connectionPort,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("updating connection_info: %w", err)
	}

	return server, &multipassJWT, nil
}

func getVaultClientAuthorizedWithSAPass(vaultURL string, password model.ServiceAccountPassword,
	roles []map[string]interface{}) (*pkg.VaultClient, error) {
	cl, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		return nil, err
	}
	err = cl.SetAddress(vaultURL)
	if err != nil {
		return nil, err
	}

	secret, err := cl.Logical().Write("/auth/flant_iam_auth/login", map[string]interface{}{
		"method":                          "sapassword",
		"service_account_password_uuid":   password.UUID,
		"service_account_password_secret": password.Secret,
		"roles":                           roles,
	})
	if err != nil {
		return nil, err
	}

	if secret == nil || secret.Auth == nil {
		return nil, fmt.Errorf("expect not nil secret.Auth, got secret:%#v", secret)
	}
	cl.SetToken(secret.Auth.ClientToken)
	return &pkg.VaultClient{Client: cl}, nil
}

// init -t tenant_for_authd_tests -p project_XXX  --password Mjc5MDUxYjItZTVmOC00MTVlLTlhNmQtMzcyMmQ5MTZlOTZmOjlPQWV7S0RpSDprQDRiajJMSklsd0NoRiFVcVBhY1ReVnVCWDFZWk10ZCVtTmcwPDVXUXZFMyk2b1J6OHMsKzc= --identifier tralala4 -l l1=vl1 -l l2=vl2 -a a1=av1 -a a2=av2 -h localhost --connection_port 222
