// run and control local vault instance for testing purposes,
// for running needs inside folder `examples/conf`:
// good.hcl - policy for access
// one or more XXX.hcl - config for running vault instance
// ca.crt - CA cert
// tls.crt - used in XXX.hcl
// tls.key - used in XXX.hcl

// WARNING! port in XXX.hcl and  RunAndWaitVaultUp should be the same

package tests

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/vault/command"
	"github.com/hashicorp/vault/sdk/physical"
	physInmem "github.com/hashicorp/vault/sdk/physical/inmem"
	"github.com/mitchellh/cli"
)

type Vault struct {
	Name  string
	Addr  string
	Token string
}

func RunVaultCommandAtVault(vault Vault, args ...string) ([]byte, error) {
	err := os.Setenv("VAULT_ADDR", vault.Addr)
	// needs drop to valid work other staff from hashicorp
	defer os.Setenv("VAULT_ADDR", "") //nolint:errcheck
	if err != nil {
		return nil, err
	}
	err = os.Setenv("VAULT_TOKEN", vault.Token)
	defer os.Setenv("VAULT_TOKEN", "") //nolint:errcheck
	if err != nil {
		return nil, err
	}
	err = os.Setenv("VAULT_CACERT", "examples/conf/ca.crt")
	defer os.Setenv("VAULT_CACERT", "") //nolint:errcheck
	if err != nil {
		return nil, err
	}
	output, err := RunVaultCommandWithError(args...)
	if err != nil {
		return nil, err
	}
	return output, nil
}

var vaultCLiMutex = &sync.Mutex{}

func RunVaultCommandWithError(args ...string) ([]byte, error) {
	vaultCLiMutex.Lock()
	defer vaultCLiMutex.Unlock()

	var output bytes.Buffer
	var errOutput bytes.Buffer

	opts := &command.RunOptions{
		Stdout: io.MultiWriter(os.Stdout, &output),
		Stderr: io.MultiWriter(os.Stderr, &errOutput),
	}

	rc := command.RunCustom(args, opts)
	if rc != 0 {
		return output.Bytes(), fmt.Errorf("vault failed with rc=%d:\n%s\n", rc, errOutput.String())
	}

	return output.Bytes(), nil
}

// RunAndWaitVaultUp run vault with specified config at specified port
// port should be the same as at the config
func RunAndWaitVaultUp(configPath string, port string, name string) Vault {
	// this function is too complex due to data race problems
	vault := Vault{
		Name: name,
		Addr: "https://127.0.0.1:" + port,
	}
	tokenChan := make(chan string)
	go func() {
		// run init and unseal
		go func() {
			for {
				time.Sleep(1 * time.Second)
				d, err := RunVaultCommandWithError("operator", "init", "-address="+vault.Addr, "-ca-cert=examples/conf/ca.crt")
				if err != nil {
					continue
				}
				tokenChan <- unseal(vault, d)
				break
			}
		}()
		srvcmd, output, errOutput := srvCmd()
		outcode := srvcmd.Run([]string{"-config", configPath})
		if outcode != 0 {
			panic(fmt.Sprintf("vault server failed: %d \noutput:\n%s \nerrOutput:\n %s", outcode, output.String(), errOutput.String()))
		}
		println("HELLO !!!!")
	}()
	vault.Token = <-tokenChan
	for {
		time.Sleep(1 * time.Second)
		if _, err := RunVaultCommandAtVault(vault, "status"); err != nil {
			continue
		}
		break
	}
	return vault
}

// srvCmd returns configured vault server command for running server and
// errOutput & output
func srvCmd() (*command.ServerCommand, *bytes.Buffer, *bytes.Buffer) {
	var output bytes.Buffer
	var errOutput bytes.Buffer

	runOpts := &command.RunOptions{
		Stdout: io.MultiWriter(&output),
		Stderr: io.MultiWriter(&errOutput),
	}

	serverCmdUi := &command.VaultUI{
		Ui: &cli.ColoredUi{
			ErrorColor: cli.UiColorRed,
			WarnColor:  cli.UiColorYellow,
			Ui: &cli.BasicUi{
				Reader: bufio.NewReader(os.Stdin),
				Writer: runOpts.Stdout,
			},
		},
	}
	srvcmd := &command.ServerCommand{
		BaseCommand: &command.BaseCommand{
			UI: serverCmdUi,
		},
		PhysicalBackends: map[string]physical.Factory{
			"inmem": physInmem.NewInmem,
		},
	}

	return srvcmd, &output, &errOutput
}

// unseal vault using output of init command, returns root token, collected from  initOut
func unseal(vault Vault, initOut []byte) (rootToken string) {
	outs := strings.Split(string(initOut), "\n")
	// remove garbage in case of debug
	for i := range outs {
		outs[i] = strings.ReplaceAll(outs[i], "\u001B[0m", "")
	}
	// collect keys
	if len(outs) < 5 {
		panic(fmt.Sprintf("not found 5 keys at:%s", string(initOut)))
	}
	shamir := []string{}
	for _, s := range outs[0:5] {
		aims := strings.Split(s, ":")
		if len(aims) == 2 {
			k := strings.TrimSpace(aims[1])
			shamir = append(shamir, k)
		}
	}
	if len(shamir) != 5 {
		panic(fmt.Sprintf("not found 5 keys at:%s", string(initOut)))
	}
	// unseal
	for _, k := range shamir {
		RunVaultCommandAtVault(vault, "operator", "unseal", k) //nolint:errcheck
	}
	// got root_key
	for _, s := range outs {
		if strings.Contains(s, "Initial Root Token") {
			aims := strings.Split(s, ":")
			rootToken = strings.TrimSpace(aims[1])
			return
		}
	}
	panic(fmt.Sprintf("not found Initial Root Token at:%s", string(initOut)))
}

// GotSecretIDAndRoleIDatApprole activates approle and returns secretID an roleID
func GotSecretIDAndRoleIDatApprole(vault Vault) (secretID string, roleID string, err error) {
	err = provideApprole(vault)
	if err != nil {
		return
	}
	_, err = RunVaultCommandAtVault(vault, "policy", "write", "good", "examples/conf/good.hcl")
	if err != nil {
		return
	}
	_, err = RunVaultCommandAtVault(vault, "write", "auth/approle/role/good", "secret_id_ttl=360h", "token_ttl=15m", "token_policies=good")
	if err != nil {
		return
	}

	var responseData []byte
	var data map[string]interface{}
	{ // secretID
		responseData, err = RunVaultCommandAtVault(vault, "write", "-format", "json", "-f", "auth/approle/role/good/secret-id")
		if err != nil {
			return
		}

		if err = json.Unmarshal(responseData, &data); err != nil {
			return
		}

		secretID = data["data"].(map[string]interface{})["secret_id"].(string)

		fmt.Printf("Got secretID: %s\n", secretID)
	}

	{ // roleID
		responseData, err = RunVaultCommandAtVault(vault, "read", "-format", "json", "auth/approle/role/good/role-id")
		if err != nil {
			return
		}
		if err = json.Unmarshal(responseData, &data); err != nil {
			return
		}

		roleID = data["data"].(map[string]interface{})["role_id"].(string)

		fmt.Printf("Got roleID: %s\n", roleID)
	}
	return
}

func provideApprole(vault Vault) error {
	resp, err := RunVaultCommandAtVault(vault, "auth", "list")
	if err != nil {
		return err
	}
	if !strings.Contains(string(resp), "approle/") {
		_, err = RunVaultCommandAtVault(vault, "auth", "enable", "approle")
	}
	return err
}
