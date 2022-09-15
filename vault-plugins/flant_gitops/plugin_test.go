package flant_gitops

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/vault/command"
	"github.com/hashicorp/vault/sdk/logical"

	"github.com/flant/negentropy/vault-plugins/flant_gitops/pkg/util"
)

const (
	secondaryVaultDevServerAddress = "0.0.0.0:8201"
	secondaryVaultDevServerToken   = "root"
)

func TestPlugin_VaultRequestsOperation(t *testing.T) {
	go func() {
		if _, err := RunVaultCommandWithError("server", "-dev", "-dev-listen-address", secondaryVaultDevServerAddress, "-dev-root-token-id", secondaryVaultDevServerToken); err != nil {
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

	systemClockMock := util.NewFixedClock(time.Now())
	systemClock = systemClockMock

	ctx := context.Background()

	b, storage, testLogger := getTestBackend(t, ctx)

	go func() {
		time.Sleep(1 * time.Minute)

		req := &logical.Request{
			Storage:    storage,
			Connection: &logical.Connection{},
		}
		if err := b.AccessVaultClientProvider.OnPeriodical(context.Background(), req); err != nil {
			panic(err.Error())
		}
	}()

	var dockerBridgeIPAddr string
	var vaultCaCrtData []byte

	// configure vault access
	{
		RunVaultCommand(t, "auth", "enable", "-address", "http://127.0.0.1:8201", "-ca-cert", "examples/conf/ca-cert.pem", "approle")
		RunVaultCommand(t, "policy", "write", "-address", "http://127.0.0.1:8201", "-ca-cert", "examples/conf/ca-cert.pem", "good", "examples/conf/good.hcl")
		RunVaultCommand(t, "write", "-address", "http://127.0.0.1:8201", "-ca-cert", "examples/conf/ca-cert.pem", "auth/approle/role/good", " secret_id_ttl=30m", "token_ttl=90s", "token_policies=good")

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
		dockerBridgeIPAddr = getDockerBridgeIPAddr(t)
		fmt.Printf("Got dockerBridgeIPAddr=%s\n", dockerBridgeIPAddr)

		var err error
		vaultCaCrtData, err = ioutil.ReadFile("examples/conf/ca-cert.pem")
		if err != nil {
			t.Fatal(err.Error())
		}

		req := &logical.Request{
			Operation: logical.UpdateOperation,
			Path:      "configure_vault_access",
			Data: map[string]interface{}{
				"vault_addr":            fmt.Sprintf("http://%s:8201", dockerBridgeIPAddr),
				"vault_tls_server_name": "localhost",
				"role_name":             "good",
				"secret_id_ttl":         "120m",
				"approle_mount_point":   "auth/approle",
				"secret_id":             secretID,
				"role_id":               roleID,
				"vault_cacert":          vaultCaCrtData,
			},
			Storage:    storage,
			Connection: &logical.Connection{},
		}
		resp, err := b.HandleRequest(ctx, req)
		if err != nil || (resp != nil && resp.IsError()) {
			t.Fatalf("err:%v resp:%#v\n", err, resp)
		}
	}

	// configure flant_gitops plugin itself
	{
		testGitRepoDir := util.GenerateTmpGitRepo(t, "flant_gitops_test_repo")
		defer os.RemoveAll(testGitRepoDir)

		flantGitopsScript := `#!/bin/sh
	
	set -e
	
	echo flant_gitops.sh BEGIN
	
	echo VAULT_ADDR="${VAULT_ADDR}"
	echo VAULT_CACERT="${VAULT_CACERT}"
	echo VAULT_TLS_SERVER_NAME="${VAULT_TLS_SERVER_NAME}"
	echo VAULT_REQUEST_TOKEN_GET_BUCKET1="${VAULT_REQUEST_TOKEN_GET_BUCKET1}"
	echo VAULT_REQUEST_TOKEN_GET_BUCKET2="${VAULT_REQUEST_TOKEN_GET_BUCKET2}"
	
	echo VAULT_CACERT BEGIN
	cat $VAULT_CACERT
	echo VAULT_CACERT END
	
	echo root | vault login -

	echo VAULT_REQUEST_TOKEN_GET_BUCKET1 UNWRAP BEGIN
	vault write -format=json sys/wrapping/unwrap token="${VAULT_REQUEST_TOKEN_GET_BUCKET1}"
	echo VAULT_REQUEST_TOKEN_GET_BUCKET1 UNWRAP END
	
	echo VAULT_REQUEST_TOKEN_GET_BUCKET2 UNWRAP BEGIN
	vault write -format=json sys/wrapping/unwrap token="${VAULT_REQUEST_TOKEN_GET_BUCKET2}"
	echo VAULT_REQUEST_TOKEN_GET_BUCKET2 UNWRAP END

	echo flant_gitops.sh END
	`

		if err := ioutil.WriteFile(filepath.Join(testGitRepoDir, "flant_gitops.sh"), []byte(flantGitopsScript), os.ModePerm); err != nil {
			t.Fatal(err.Error())
		}

		util.ExecGitCommand(t, testGitRepoDir, "add", ".")
		util.ExecGitCommand(t, testGitRepoDir, "commit", "-m", "go")

		req := &logical.Request{
			Operation: logical.UpdateOperation,
			Path:      "configure",
			Data: map[string]interface{}{
				fieldNameGitRepoUrl:    testGitRepoDir,
				fieldNameGitBranch:     "master",
				fieldNameGitPollPeriod: "5m",
				fieldNameRequiredNumberOfVerifiedSignaturesOnCommit: "0",
				fieldNameInitialLastSuccessfulCommit:                "",
				fieldNameDockerImage:                                "vault:1.7.3@sha256:53e509aaa6f72c54418b2f65f23fdd8a5ddd22bf6521c4b5bf82a8ae4edd0e53",
				fieldNameCommands:                                   "./flant_gitops.sh",
			},
			Storage:    storage,
			Connection: &logical.Connection{},
		}
		resp, err := b.HandleRequest(ctx, req)
		if err != nil || (resp != nil && resp.IsError()) {
			t.Fatalf("err:%v resp:%#v\n", err, resp)
		}
	}

	// configure test vault requests
	{
		RunVaultCommand(t, "secrets", "enable", "-address", "http://127.0.0.1:8201", "kv")
		RunVaultCommand(t, "write", "-address", "http://127.0.0.1:8201", "kv/bucket1", "key1=value1")
		RunVaultCommand(t, "write", "-address", "http://127.0.0.1:8201", "kv/bucket2", "key2=value2")

		req := &logical.Request{
			Operation: logical.UpdateOperation,
			Path:      "configure/vault_request/get_bucket1",
			Data: map[string]interface{}{
				"name":     "get_bucket1",
				"method":   "GET",
				"path":     "/v1/kv/bucket1",
				"wrap_ttl": "1m",
			},
			Storage:    storage,
			Connection: &logical.Connection{},
		}
		resp, err := b.HandleRequest(ctx, req)
		if err != nil || (resp != nil && resp.IsError()) {
			t.Fatalf("err:%v resp:%#v\n", err, resp)
		}

		req = &logical.Request{
			Operation: logical.UpdateOperation,
			Path:      "configure/vault_request/get_bucket2",
			Data: map[string]interface{}{
				"name":     "get_bucket2",
				"method":   "GET",
				"path":     "/v1/kv/bucket2",
				"wrap_ttl": "1m",
			},
			Storage:    storage,
			Connection: &logical.Connection{},
		}
		resp, err = b.HandleRequest(ctx, req)
		if err != nil || (resp != nil && resp.IsError()) {
			t.Fatalf("err:%v resp:%#v\n", err, resp)
		}
	}

	var periodicTaskUUIDs []string

	invokePeriodicRun(t, ctx, b, testLogger, storage)
	periodicTaskUUIDs = append(periodicTaskUUIDs, b.LastPeriodicTaskUUID)
	WaitForTaskSuccess(t, ctx, b, storage, periodicTaskUUIDs[len(periodicTaskUUIDs)-1])

	if match, lines := testLogger.Grep("VAULT_ADDR="); !match {
		t.Fatalf("task %s output not contains expected output:\n%s\n", periodicTaskUUIDs[len(periodicTaskUUIDs)-1], strings.Join(testLogger.GetLines(), "\n"))
	} else {
		expectedVaultAddr := fmt.Sprintf("http://%s:8201", dockerBridgeIPAddr)
		if !strings.Contains(lines[0], expectedVaultAddr) {
			t.Fatalf("expected VAULT_ADDR to equal %q, got: %s", expectedVaultAddr, lines[0])
		}
	}

	if match, _ := testLogger.Grep("VAULT_CACERT"); !match {
		t.Fatalf("task %s output not contains expected output:\n%s\n", periodicTaskUUIDs[len(periodicTaskUUIDs)-1], strings.Join(testLogger.GetLines(), "\n"))
	}

	if match, _ := testLogger.Grep("VAULT_TLS_SERVER_NAME=localhost"); !match {
		t.Fatalf("task %s output not contains expected output:\n%s\n", periodicTaskUUIDs[len(periodicTaskUUIDs)-1], strings.Join(testLogger.GetLines(), "\n"))
	}

	if match, _ := testLogger.Grep("VAULT_REQUEST_TOKEN_GET_BUCKET1"); !match {
		t.Fatalf("task %s output not contains expected output:\n%s\n", periodicTaskUUIDs[len(periodicTaskUUIDs)-1], strings.Join(testLogger.GetLines(), "\n"))
	}

	if match, _ := testLogger.Grep("VAULT_REQUEST_TOKEN_GET_BUCKET2"); !match {
		t.Fatalf("task %s output not contains expected output:\n%s\n", periodicTaskUUIDs[len(periodicTaskUUIDs)-1], strings.Join(testLogger.GetLines(), "\n"))
	}

	if match, data := testLogger.GetDataByMarkers("VAULT_CACERT BEGIN", "VAULT_CACERT END"); !match {
		t.Fatalf("task %s output not contains expected output:\n%s\n", periodicTaskUUIDs[len(periodicTaskUUIDs)-1], strings.Join(testLogger.GetLines(), "\n"))
	} else if string(data) != string(vaultCaCrtData) {
		t.Fatalf("expected vault ca cert:\n%s\ngot:\n%s\n", vaultCaCrtData, data)
	}

	if match, data := testLogger.GetDataByMarkers("VAULT_REQUEST_TOKEN_GET_BUCKET1 UNWRAP BEGIN", "VAULT_REQUEST_TOKEN_GET_BUCKET1 UNWRAP END"); !match {
		t.Fatalf("task %s output not contains expected output:\n%s\n", periodicTaskUUIDs[len(periodicTaskUUIDs)-1], strings.Join(testLogger.GetLines(), "\n"))
	} else {
		var dataMap map[string]interface{}
		if err := json.Unmarshal(data, &dataMap); err != nil {
			t.Fatalf("expected valid json, got error: %s\n%s\n", err, data)
		}

		key1Value := dataMap["data"].(map[string]interface{})["key1"].(string)

		if key1Value != "value1" {
			t.Fatalf("expected data.key1 to equal value1, got:\n%s\n", data)
		}
	}

	if match, data := testLogger.GetDataByMarkers("VAULT_REQUEST_TOKEN_GET_BUCKET2 UNWRAP BEGIN", "VAULT_REQUEST_TOKEN_GET_BUCKET2 UNWRAP END"); !match {
		t.Fatalf("task %s output not contains expected output:\n%s\n", periodicTaskUUIDs[len(periodicTaskUUIDs)-1], strings.Join(testLogger.GetLines(), "\n"))
	} else {
		var dataMap map[string]interface{}
		if err := json.Unmarshal(data, &dataMap); err != nil {
			t.Fatalf("expected valid json, got error: %s\n%s\n", err, data)
		}

		key2Value := dataMap["data"].(map[string]interface{})["key2"].(string)

		if key2Value != "value2" {
			t.Fatalf("expected data.key2 to equal value2, got:\n%s\n", data)
		}
	}
}

// returns IP-address or url to link from docker-container to host
func getDockerBridgeIPAddr(t *testing.T) string {
	if runtime.GOOS == "darwin" {
		return "docker.for.mac.localhost"
	}
	cmd := exec.Command("docker", "network", "inspect", "bridge", "-f", "{{ (index .IPAM.Config 0).Gateway }}")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("docker network inspect command failed: %s\n%s\n", err, output)
	}
	return strings.TrimSpace(string(output))
}

func RunVaultCommand(t *testing.T, args ...string) []byte {
	output, err := RunVaultCommandWithError(args...)
	if err != nil {
		t.Fatalf("error running vault command '%s': %s", strings.Join(args, " "), err)
	}
	return output
}

func RunVaultCommandWithError(args ...string) ([]byte, error) {
	var output bytes.Buffer

	opts := &command.RunOptions{
		Stdout: io.MultiWriter(os.Stdout, &output),
		Stderr: io.MultiWriter(os.Stderr, &output),
	}

	rc := command.RunCustom(args, opts)
	if rc != 0 {
		return output.Bytes(), fmt.Errorf("vault failed with rc=%d:\n%s\n", rc, output.String())
	}

	return output.Bytes(), nil
}
