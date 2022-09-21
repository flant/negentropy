package flant_gitops

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/vault/command"
)

const (
	confVaultAddress = "0.0.0.0:8201"
	confVaultToken   = "root"

	coreVaultAddress = "0.0.0.0:8202"
	coreVaultToken   = "root"
)

func runVaultInstance(address, token string) {
	go func() {
		if _, err := RunVaultCommandWithError("server", "-dev", "-dev-listen-address", confVaultAddress, "-dev-root-token-id", confVaultToken); err != nil {
			panic(fmt.Sprintf("vault dev server failed: %s", err))
		}
	}()

	// Wait until dev server started
	for {
		time.Sleep(1 * time.Second)
		if _, err := RunVaultCommandWithError("status", "-address", "http://127.0.0.1:8201"); err != nil {
			continue
		}
		break
	}
}

func Test_vault(t *testing.T) {
	//runVaultInstance(coreVaultAddress, coreVaultToken)
	runVaultInstance(confVaultAddress, confVaultToken)

	//ctx := context.Background()
	// configure vault access
	{
		out := RunVaultCommand(t, "auth", "enable", "-address", "http://127.0.0.1:8201", "approle")
		println("=============")
		fmt.Println(string(out))

		out = RunVaultCommand(t, "policy", "write", "-address", "http://127.0.0.1:8201", "good", "examples/conf/good.hcl")
		println("=============")
		fmt.Println(string(out))
		out = RunVaultCommand(t, "write", "-address", "http://127.0.0.1:8201", "auth/approle/role/good", " secret_id_ttl=30m", "token_ttl=90s", "token_policies=good")
		println("=============")
		fmt.Println(string(out))

		var secretID string
		{
			var data map[string]interface{}

			jsonData := RunVaultCommand(t, "write", "-address", "http://127.0.0.1:8201", "-ca-cert", "examples/conf/ca-cert.pem", "-format", "json", "-f", "auth/approle/role/good/secret-id")

			if err := json.Unmarshal(jsonData, &data); err != nil {
				t.Fatalf("bad json: %s\n%s\n", err, jsonData)
			}

			secretID = data["data"].(map[string]interface{})["secret_id"].(string)

			fmt.Printf("Got secretID: %s\n", secretID)
		}

		var roleID string
		{
			var data map[string]interface{}

			jsonData := RunVaultCommand(t, "read", "-address", "http://127.0.0.1:8201", "-ca-cert", "examples/conf/ca-cert.pem", "-format", "json", "auth/approle/role/good/role-id")

			if err := json.Unmarshal(jsonData, &data); err != nil {
				t.Fatalf("bad json: %s\n%s\n", err, jsonData)
			}

			roleID = data["data"].(map[string]interface{})["role_id"].(string)

			fmt.Printf("Got roleID: %s\n", roleID)
		}

		// Get docker bridge ip address to access secondary vault dev server from inside docker container

		vaultCaCrtData, err := ioutil.ReadFile("examples/conf/ca-cert.pem")
		if err != nil {
			t.Fatal(err.Error())
		}

		println(vaultCaCrtData)

		//req := &logical.Request{
		//	Operation: logical.UpdateOperation,
		//	Path:      "configure_vault_access",
		//	Data: map[string]interface{}{
		//		"vault_addr":            coreVaultAddress,
		//		"vault_tls_server_name": "localhost",
		//		"role_name":             "good",
		//		"secret_id_ttl":         "120m",
		//		"approle_mount_point":   "auth/approle",
		//		"secret_id":             secretID,
		//		"role_id":               roleID,
		//		"vault_cacert":          vaultCaCrtData,
		//	},
		//	//Storage:    storage,
		//	//Connection: &logical.Connection{},
		//}
		//resp, err := b.HandleRequest(ctx, req)
		//if err != nil || (resp != nil && resp.IsError()) {
		//	t.Fatalf("err:%v resp:%#v\n", err, resp)
		//}
	}

	// configure test vault requests
	{
		RunVaultCommand(t, "secrets", "enable", "-address", "http://127.0.0.1:8201", "kv")
		RunVaultCommand(t, "write", "-address", "http://127.0.0.1:8201", "kv/bucket1", "key1=value1")
		RunVaultCommand(t, "write", "-address", "http://127.0.0.1:8201", "kv/bucket2", "key2=value2")

	}
}

func RunVaultCommandAtVault(t *testing.T, vaultAddr string, args ...string) []byte {
	var output bytes.Buffer

	opts := &command.RunOptions{
		Stdout:  io.MultiWriter(os.Stdout, &output),
		Stderr:  io.MultiWriter(os.Stderr, &output),
		Address: vaultAddr,
	}

	rc := command.RunCustom(args, opts)
	if rc != 0 {
		t.Fatalf("vault failed with rc=%d:\n%s\n", rc, output.String())
	}

	return output.Bytes()
}
