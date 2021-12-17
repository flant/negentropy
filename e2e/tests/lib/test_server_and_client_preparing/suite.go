// Package test_server_and_client_preparing provide preparing and operating
// under two containers, which are run by docker-compose: 1) test-server and 2) test-client
// the first one has authd, server-accessd and server-access-nss
// second one has authd and cli
// it allows to provide e2e testing for all of "backend" components
package test_server_and_client_preparing

import (
	"bytes"
	"context"
	"crypto/sha256"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	. "github.com/onsi/gomega"

	"github.com/flant/negentropy/e2e/tests/lib/flant_iam_preparing"
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
)

type Suite struct {
	testServerContainerName string
	testClientContainerName string
	authdPath               string
	cliPath                 string
	serverAccessdPath       string

	RootVaultInternalURL string
	AuthVaultInternalURL string

	dockerCli *client.Client

	TestServerContainer *types.Container
	TestClientContainer *types.Container
}

func (s *Suite) BeforeSuite() {
	// TODO read vars from envs!
	s.authdPath = "/opt/authd/bin/authd"
	s.cliPath = "/opt/cli/bin/cli"
	s.serverAccessdPath = "/opt/server-access/bin/server-accessd"
	s.testServerContainerName = "test-server"
	s.testClientContainerName = "test-client"
	s.RootVaultInternalURL = getFromEnv("ROOT_VAULT_INTERNAL_URL")
	s.AuthVaultInternalURL = getFromEnv("AUTH_VAULT_INTERNAL_URL")

	// Open connections, create clients
	var err error
	s.dockerCli, err = client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())

	Expect(err).ToNot(HaveOccurred())

	s.TestServerContainer, err = s.getContainerByName(s.testServerContainerName)
	Expect(err).ToNot(HaveOccurred())
	s.TestClientContainer, err = s.getContainerByName(s.testClientContainerName)
	Expect(err).ToNot(HaveOccurred())
}

func getFromEnv(envName string) string {
	value := os.Getenv(envName)
	if value == "" {
		panic(fmt.Sprintf("equired environment variable %s is not set", envName))
	}
	return value
}

//go:embed server_sock1.yaml
var serverSocketCFG string

//go:embed server_main.yaml
var serverMainCFGTPL string

func (s *Suite) PrepareServerForSSHTesting(cfg flant_iam_preparing.CheckingEnvironment) {
	err := s.CheckFileExistAtContainer(s.TestServerContainer, s.authdPath, "f")
	Expect(err).ToNot(HaveOccurred(), "Test_server should have authd")

	err = s.CheckFileExistAtContainer(s.TestServerContainer, s.serverAccessdPath, "f")
	Expect(err).ToNot(HaveOccurred(), "Test_server should have server-accessd")

	// Authd can be configured and run at Test_server
	err = s.createIfNotExistsDirectoryAtContainer(s.TestServerContainer,
		"/etc/flant/negentropy/authd-conf.d")
	Expect(err).ToNot(HaveOccurred(), "folder should be created")

	t, err := template.New("").Parse(serverMainCFGTPL)
	Expect(err).ToNot(HaveOccurred(), "template should be ok")
	var serverMainCFG bytes.Buffer
	err = t.Execute(&serverMainCFG, *s)
	Expect(err).ToNot(HaveOccurred(), "template should be executed")

	err = s.writeFileToContainer(s.TestServerContainer,
		"/etc/flant/negentropy/authd-conf.d/main.yaml", serverMainCFG.String())
	Expect(err).ToNot(HaveOccurred(), "file should be written")

	err = s.writeFileToContainer(s.TestServerContainer,
		"/etc/flant/negentropy/authd-conf.d/sock1.yaml", serverSocketCFG)
	Expect(err).ToNot(HaveOccurred(), "file should be written")

	err = s.writeFileToContainer(s.TestServerContainer,
		"/opt/authd/server-jwt", cfg.TestServer.MultipassJWT)
	Expect(err).ToNot(HaveOccurred(), "file should be written")

	s.ExecuteCommandAtContainer(s.TestServerContainer,
		[]string{"/bin/bash", "-c", "chmod 600 /opt/authd/server-jwt"}, nil)

	s.killAllInstancesOfProcessAtContainer(s.TestServerContainer, s.authdPath)
	s.runDaemonAtContainer(s.TestServerContainer, s.authdPath, "server_authd.log")
	time.Sleep(time.Second)
	pidAuthd := s.firstProcessPIDAtContainer(s.TestServerContainer, s.authdPath)
	Expect(pidAuthd).Should(BeNumerically(">", 0), "pid greater 0")

	// TODO check content /etc/nsswitch.conf

	// Test_server server_accessd can be configured and run
	err = s.createIfNotExistsDirectoryAtContainer(s.TestServerContainer,
		"/opt/serveraccessd")
	Expect(err).ToNot(HaveOccurred(), "folder should be created")

	acccesdCFG := fmt.Sprintf("tenant: %s\n", cfg.Tenant.UUID) +
		fmt.Sprintf("project: %s\n", cfg.Project.UUID) +
		fmt.Sprintf("server: %s\n", cfg.TestServer.ServerUUID) +
		"database: /opt/serveraccessd/server-accessd.db\n" +
		"authdSocketPath: /run/sock1.sock"

	err = s.writeFileToContainer(s.TestServerContainer,
		"/opt/server-access/config.yaml", acccesdCFG)
	Expect(err).ToNot(HaveOccurred(), "file should be written")

	s.killAllInstancesOfProcessAtContainer(s.TestServerContainer, s.serverAccessdPath)
	s.runDaemonAtContainer(s.TestServerContainer, s.serverAccessdPath, "server_accessd.log")
	time.Sleep(time.Second)
	pidServerAccessd := s.firstProcessPIDAtContainer(s.TestServerContainer, s.serverAccessdPath)
	Expect(pidServerAccessd).Should(BeNumerically(">", 0), "pid greater 0")

	authKeysFilePath := filepath.Join("/home", cfg.User.Identifier, ".ssh", "authorized_keys")
	contentAuthKeysFile := s.ExecuteCommandAtContainer(s.TestServerContainer,
		[]string{"/bin/bash", "-c", "cat " + authKeysFilePath}, nil)
	Expect(contentAuthKeysFile).To(HaveLen(1), "cat authorize should have one line text")
	principal := calculatePrincipal(cfg.TestServer.ServerUUID, cfg.User.UUID)
	Expect(contentAuthKeysFile[0]).To(MatchRegexp(".+cert-authority,principals=\""+principal+"\" ssh-rsa.{373}"),
		"content should be specific")
}

//go:embed client_sock1.yaml
var clientSocketCFG string

//go:embed client_main.yaml
var clientMainCFGTPL string

func (s *Suite) PrepareClientForSSHTesting(cfg flant_iam_preparing.CheckingEnvironment) {
	err := s.CheckFileExistAtContainer(s.TestClientContainer, s.authdPath, "f")
	Expect(err).ToNot(HaveOccurred(), "TestClient should have authd")

	err = s.CheckFileExistAtContainer(s.TestClientContainer, s.cliPath, "f")
	Expect(err).ToNot(HaveOccurred(), "Test_сlient should have cli")

	// Authd can be configured and run at Test_server
	err = s.createIfNotExistsDirectoryAtContainer(s.TestClientContainer,
		"/etc/flant/negentropy/authd-conf.d")
	Expect(err).ToNot(HaveOccurred(), "folder should be created")

	t, err := template.New("").Parse(clientMainCFGTPL)
	Expect(err).ToNot(HaveOccurred(), "template should be ok")
	var clientMainCFG bytes.Buffer
	err = t.Execute(&clientMainCFG, *s)
	Expect(err).ToNot(HaveOccurred(), "template should be executed")

	err = s.writeFileToContainer(s.TestClientContainer,
		"/etc/flant/negentropy/authd-conf.d/main.yaml", clientMainCFG.String())
	Expect(err).ToNot(HaveOccurred(), "file should be written")

	err = s.writeFileToContainer(s.TestClientContainer,
		"/etc/flant/negentropy/authd-conf.d/sock1.yaml", clientSocketCFG)
	Expect(err).ToNot(HaveOccurred(), "file should be written")

	err = s.writeFileToContainer(s.TestClientContainer,
		"/opt/authd/client-jwt", cfg.UserJWToken)
	Expect(err).ToNot(HaveOccurred(), "file should be written")

	s.ExecuteCommandAtContainer(s.TestClientContainer,
		[]string{"/bin/bash", "-c", "chmod 600 /opt/authd/client-jwt"}, nil)

	s.killAllInstancesOfProcessAtContainer(s.TestClientContainer, s.authdPath)
	s.runDaemonAtContainer(s.TestClientContainer, s.authdPath, "client_authd.log")
	time.Sleep(time.Second)
	pidAuthd := s.firstProcessPIDAtContainer(s.TestClientContainer, s.authdPath)
	Expect(pidAuthd).Should(BeNumerically(">", 0), "pid greater 0")
}

func (s *Suite) getContainerByName(name string) (*types.Container, error) {
	name = strings.ReplaceAll(name, "_", "-")
	containers, err := s.dockerCli.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil {
		return nil, err
	}
	for _, c := range containers {
		for _, n := range c.Names {
			if strings.ReplaceAll(n, "_", "-") == "/"+name {
				if c.State != "running" {
					return nil, fmt.Errorf("container with name %s has state: %s", name, c.State)
				}
				return &c, nil
			}
		}
	}

	return nil, errors.New("Container with name " + name + " not found")
}

func (s *Suite) CheckFileExistAtContainer(container *types.Container, path string, fileTypeFlag string) error {
	ctx := context.Background()
	config := types.ExecConfig{
		AttachStdin:  true,
		AttachStderr: true,
		AttachStdout: true,
		Cmd:          []string{"/bin/bash", "-c", "test -" + fileTypeFlag + " " + path + " && echo true"},
	}

	IDResp, err := s.dockerCli.ContainerExecCreate(ctx, container.ID, config)
	Expect(err).ToNot(HaveOccurred(), "error execution at container")

	resp, err := s.dockerCli.ContainerExecAttach(ctx, IDResp.ID, types.ExecStartCheck{})
	Expect(err).ToNot(HaveOccurred(), "error attaching execution at container")
	defer resp.Close()

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

func (s *Suite) createIfNotExistsDirectoryAtContainer(container *types.Container, path string) error {
	lastSeparator := strings.LastIndex(path, string(os.PathSeparator))
	if lastSeparator != 0 {
		err := s.createIfNotExistsDirectoryAtContainer(container, path[0:lastSeparator])
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

	IDResp, err := s.dockerCli.ContainerExecCreate(ctx, container.ID, config)
	Expect(err).ToNot(HaveOccurred(), "error execution at container")

	resp, err := s.dockerCli.ContainerExecAttach(ctx, IDResp.ID, types.ExecStartCheck{})
	Expect(err).ToNot(HaveOccurred(), "error attaching execution at container")
	defer resp.Close()

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

func (s *Suite) writeFileToContainer(container *types.Container, path string, content string) error {
	ctx := context.Background()
	config := types.ExecConfig{
		AttachStdin:  true,
		AttachStderr: true,
		AttachStdout: true,
		Cmd:          []string{"/bin/bash", "-c", "echo \"" + content + "\" > " + path},
	}

	IDResp, err := s.dockerCli.ContainerExecCreate(ctx, container.ID, config)
	Expect(err).ToNot(HaveOccurred(), "error execution at container")

	resp, err := s.dockerCli.ContainerExecAttach(ctx, IDResp.ID, types.ExecStartCheck{})
	Expect(err).ToNot(HaveOccurred(), "error attaching execution at container")
	defer resp.Close()

	text, err := resp.Reader.ReadString('\n')
	if err != nil && err.Error() == "EOF" {
		fmt.Printf("this content: \n %s \n ==> has been written to file %s at container  %s \n", content, path, container.Names)
		return nil
	}
	Expect(err).ToNot(HaveOccurred(), "error response reading at container")
	return fmt.Errorf("unexpected output creating directory: %s", text)
}

func (s *Suite) ExecuteCommandAtContainer(container *types.Container, cmd []string, extraInputToSTDIN []string) []string {
	ctx := context.Background()
	config := types.ExecConfig{
		AttachStdin:  true,
		AttachStderr: true,
		AttachStdout: true,
		Cmd:          cmd,
	}

	IDResp, err := s.dockerCli.ContainerExecCreate(ctx, container.ID, config)
	Expect(err).ToNot(HaveOccurred(), "error execution at container")

	resp, err := s.dockerCli.ContainerExecAttach(ctx, IDResp.ID, types.ExecStartCheck{})
	Expect(err).ToNot(HaveOccurred(), "error attaching execution at container")
	defer resp.Close()

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

func (s *Suite) killProcessAtContainer(container *types.Container, processPid int) {
	ctx := context.Background()
	config := types.ExecConfig{
		AttachStdin:  true,
		AttachStderr: true,
		AttachStdout: true,
		Cmd:          []string{"/bin/bash", "-c", "kill -9 " + strconv.Itoa(processPid)},
	}

	IDResp, err := s.dockerCli.ContainerExecCreate(ctx, container.ID, config)
	Expect(err).ToNot(HaveOccurred(), "error execution at container")

	_, err = s.dockerCli.ContainerExecAttach(ctx, IDResp.ID, types.ExecStartCheck{})
	Expect(err).ToNot(HaveOccurred(), "error attaching execution at container")
}

func (s *Suite) killAllInstancesOfProcessAtContainer(container *types.Container, processPath string) {
	for {
		pid := s.firstProcessPIDAtContainer(container, processPath)
		if pid == 0 {
			break
		}
		s.killProcessAtContainer(container, pid)
	}
}

func (s *Suite) firstProcessPIDAtContainer(container *types.Container, processPath string) int {
	ctx := context.Background()
	config := types.ExecConfig{
		AttachStdin:  true,
		AttachStderr: true,
		AttachStdout: true,
		Cmd:          []string{"/bin/bash", "-c", "ps ax"},
	}

	IDResp, err := s.dockerCli.ContainerExecCreate(ctx, container.ID, config)
	Expect(err).ToNot(HaveOccurred(), "error execution at container")

	resp, err := s.dockerCli.ContainerExecAttach(ctx, IDResp.ID, types.ExecStartCheck{})
	Expect(err).ToNot(HaveOccurred(), "error attaching execution at container")
	defer resp.Close()

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

func (s *Suite) runDaemonAtContainer(container *types.Container, daemonPath string, logFilePath string) {
	ctx := context.Background()
	config := types.ExecConfig{
		AttachStdin:  true,
		AttachStderr: true,
		AttachStdout: true,
		Cmd:          []string{daemonPath},
	}

	IDResp, err := s.dockerCli.ContainerExecCreate(ctx, container.ID, config)
	Expect(err).ToNot(HaveOccurred(), "error execution at container")

	resp, err := s.dockerCli.ContainerExecAttach(ctx, IDResp.ID, types.ExecStartCheck{})
	Expect(err).ToNot(HaveOccurred(), "error attaching execution at container")
	// It is an independent goroutine due to we run daemon and should return control back
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
		if !errors.Is(err, io.EOF) {
			logFile.Write([]byte(fmt.Sprintf("reading from container %s:%s", container.Names, err)))
		}
		logFile.Close()
		resp.Close()
	}()
}

func (s *Suite) DirectoryAtContainerNotExistOrEmpty(container *types.Container, directoryPath string) bool {
	ctx := context.Background()
	config := types.ExecConfig{
		AttachStdin:  true,
		AttachStderr: true,
		AttachStdout: true,
		Cmd:          []string{"/bin/bash", "-c", "ls " + directoryPath},
	}

	IDResp, err := s.dockerCli.ContainerExecCreate(ctx, container.ID, config)
	Expect(err).ToNot(HaveOccurred(), "error execution at container")

	resp, err := s.dockerCli.ContainerExecAttach(ctx, IDResp.ID, types.ExecStartCheck{})
	Expect(err).ToNot(HaveOccurred(), "error attaching execution at container")
	defer resp.Close()

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
