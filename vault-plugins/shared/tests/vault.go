// run and control docker vault https instance for testing purposes (image=vault:1.11.4),
// for running needs 'conf-folder':
// good.hcl - policy for access
// one or more XXX.hcl - config for running vault instance
// ca.crt - CA cert
// tls.crt - used in XXX.hcl
// tls.key - used in XXX.hcl

package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

// Vault represents docker container with standard vault
type Vault struct {
	ContainerID string
	Port        string
	Token       string
	Addr        string
	Name        string
}

// Remove removes docker container
func (v *Vault) Remove() {
	ctx := context.Background()
	if v == nil || v.ContainerID == "" {
		return
	}
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		println(err.Error())
	}
	err = cli.ContainerRemove(ctx, v.ContainerID, types.ContainerRemoveOptions{
		RemoveVolumes: true,
		Force:         true,
	})
	if err != nil {
		println(err.Error())
	}
}

func (v *Vault) RunVaultCmd(args ...string) ([]byte, error) {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("creating docker cli: %w", err)
	}

	config := types.ExecConfig{
		AttachStdin:  true,
		AttachStderr: true,
		AttachStdout: true,
		Env:          []string{"VAULT_ADDR=https://127.0.0.1:" + v.Port, "VAULT_TOKEN=" + v.Token, "VAULT_CACERT=/etc/vault/ca.crt"},
		Cmd:          append([]string{"vault"}, args...),
	}

	IDResp, err := cli.ContainerExecCreate(ctx, v.ContainerID, config)
	if err != nil {
		return nil, fmt.Errorf("executing vault cmd: %v: %w", args, err)
	}
	resp, err := cli.ContainerExecAttach(ctx, IDResp.ID, types.ExecStartCheck{})
	if err != nil {
		return nil, fmt.Errorf("collecting vault cmd output: %v: %w", args, err)
	}
	defer resp.Close()

	return ParseDockerOutput(resp.Reader)
}

// RunAndWaitVaultUp run docker instance vault
// confFolderPath - relative path to 'conf-folder' (specified at the beginning)
// hclFileName - short file name for vault configuration file in  'conf-folder'
// vaultName - name for vault in code internals and docker-container name
func RunAndWaitVaultUp(confFolderPath string, hclFileName string, vaultName string) (*Vault, error) {
	confFolderFullPath, err := fullPath(confFolderPath)
	if err != nil {
		return nil, err
	}
	port, err := parsePort(confFolderPath, hclFileName)
	if err != nil {
		return nil, err
	}

	containerID, err := runVaultContainer(confFolderFullPath, hclFileName, port, vaultName)
	if err != nil {
		return nil, err
	}

	vault := &Vault{
		Name:        vaultName,
		ContainerID: containerID,
		Port:        string(port),
		Addr:        "https://127.0.0.1:" + string(port),
	}
	// init+_unseal
	for {
		time.Sleep(1 * time.Second)
		out, err := vault.RunVaultCmd("operator", "init")
		if err != nil {
			fmt.Printf("%#v\n", err)
			continue
		}
		vault.Token = unseal(*vault, out)
		break
	}
	return vault, nil
}

func parsePort(folderPath string, hclFileName string) (nat.Port, error) {
	fileName := filepath.Join(folderPath, hclFileName)
	data, err := ioutil.ReadFile(fileName)
	if err != nil {
		return "", fmt.Errorf("reading file: %s :%w", fileName, err)
	}
	for _, l := range strings.Split(string(data), "\n") {
		if strings.Contains(l, "address") {
			elems := strings.Split(l, "\"")
			if len(elems) == 3 {
				hostAndPort := strings.Split(elems[1], ":")
				if len(hostAndPort) == 2 {
					return nat.Port(hostAndPort[1]), nil
				}
			}
		}
	}
	return "", fmt.Errorf("can't find substring like: `address = \"0.0.0.0:8300\"` in file: %s", fileName)
}

func fullPath(relativePath string) (string, error) {
	pwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	fullpath := path.Join(pwd, relativePath)
	fullpath, err = filepath.Abs(fullpath)
	if err != nil {
		return "", err
	}
	_, err = os.Stat(fullpath)
	if err != nil {
		return "", err
	}
	return fullpath, nil
}

// runVaultContainer runs vault container with port forwarding and with specified name
func runVaultContainer(confFolderFullPath string, hclShortFileName string, port nat.Port, containerName string) (string, error) {
	imageName := "vault:1.11.4"

	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return "", fmt.Errorf("creating docker cli: %w", err)
	}
	reader, err := cli.ImagePull(ctx, imageName, types.ImagePullOptions{})
	if err != nil {
		return "", fmt.Errorf("pulling image: %w", err)
	}
	defer reader.Close()       // nolint:errcheck
	io.Copy(os.Stdout, reader) // nolint:errcheck

	// https://stackoverflow.com/questions/48470194/defining-a-mount-point-for-volumes-in-golang-docker-sdk
	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: imageName,
		// Volumes: map[string]struct{}{"/Users/admin/flant/negentropy/probs/docker/msg.txt:/msg.txt": {}},
		Cmd: []string{"vault", "server", "-config", "/etc/vault/" + hclShortFileName},
		Tty: false,
		ExposedPorts: nat.PortSet{
			port: struct{}{},
		},
	},
		&container.HostConfig{
			PortBindings: map[nat.Port][]nat.PortBinding{port: {{
				HostIP:   "0.0.0.0",
				HostPort: string(port),
			}}},
			Mounts: []mount.Mount{
				{
					Type:   mount.TypeBind,
					Source: confFolderFullPath,
					Target: "/etc/vault",
				},
			},
			CapAdd: []string{"IPC_LOCK"},
		},
		nil, nil, containerName)
	if err != nil {
		return "", fmt.Errorf("creating container: %w", err)
	}
	go func() {
		if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
			fmt.Printf("starting container: %s \n", err.Error())
		}
	}()

	statusCh, errCh := cli.ContainerWait(ctx, resp.ID, container.WaitCondition("running"))
	select {
	case err := <-errCh:
		if err != nil {
			return "", err
		}
	case <-statusCh:
	}
	return resp.ID, nil
}

// unseal vault using output of init command, returns root token, collected from  initOut
func unseal(vault Vault, initOut []byte) (rootToken string) {
	outs := strings.Split(string(initOut), "\n")
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
		vault.RunVaultCmd("operator", "unseal", k) //nolint:errcheck
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
	_, err = vault.RunVaultCmd("policy", "write", "good", "/etc/vault/good.hcl")
	if err != nil {
		return
	}
	_, err = vault.RunVaultCmd("write", "auth/approle/role/good", "secret_id_ttl=360h", "token_ttl=15m", "token_policies=good")
	if err != nil {
		return
	}

	var responseData []byte
	var data map[string]interface{}
	{ // secretID
		responseData, err = vault.RunVaultCmd("write", "-format", "json", "-f", "auth/approle/role/good/secret-id")
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
		responseData, err = vault.RunVaultCmd("read", "-format", "json", "auth/approle/role/good/role-id")
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
	resp, err := vault.RunVaultCmd("auth", "list")
	if err != nil {
		return err
	}
	if !strings.Contains(string(resp), "approle/") {
		_, err = vault.RunVaultCmd("auth", "enable", "approle")
	}
	return err
}
