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
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	. "github.com/onsi/gomega"

	fip "github.com/flant/negentropy/e2e/tests/lib/flant_iam_preparing"
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/tests"
)

type Suite struct {
	testServerContainerName string
	testClientContainerName string
	AuthdPath               string
	cliPath                 string
	ServerAccessdPath       string

	dockerCli *client.Client

	TestServerContainer *types.Container
	TestClientContainer *types.Container
}

func (s *Suite) BeforeSuite() {
	// TODO read vars from envs!
	s.AuthdPath = "/opt/authd/bin/authd"
	s.cliPath = "/opt/cli/bin/cli"
	s.ServerAccessdPath = "/opt/server-access/bin/server-accessd"
	s.testServerContainerName = "test-server"
	s.testClientContainerName = "test-client"

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
var ServerSocketCFG string

//go:embed server_main.yaml
var serverMainCFGTPL string

func ServerMainAuthdCFG() string {
	t, err := template.New("").Parse(serverMainCFGTPL)
	Expect(err).ToNot(HaveOccurred(), "template should be ok")
	return mainAuthdCFG(t, nil)
}

func (s *Suite) CheckServerBinariesAndFoldersExists() {
	err := s.CheckFileExistAtContainer(s.TestServerContainer, s.AuthdPath, "f")
	Expect(err).ToNot(HaveOccurred(), "Test_server should have authd")

	err = s.CheckFileExistAtContainer(s.TestServerContainer, s.ServerAccessdPath, "f")
	Expect(err).ToNot(HaveOccurred(), "Test_server should have server-accessd")

	// Authd can be configured and run at Test_server
	err = s.CreateIfNotExistsDirectoryAtContainer(s.TestServerContainer,
		"/etc/flant/negentropy/authd-conf.d")
	Expect(err).ToNot(HaveOccurred(), "folder should be created")
}

func (s *Suite) PrepareServerForSSHTesting(cfg fip.CheckingEnvironment) {
	s.CheckServerBinariesAndFoldersExists()

	err := RunAndCheckAuthdAtServer(*s, cfg.TestServerServiceAccountMultipassJWT)
	Expect(err).ToNot(HaveOccurred())

	// Test_server server_accessd can be configured and run
	err = s.CreateIfNotExistsDirectoryAtContainer(s.TestServerContainer,
		"/opt/serveraccessd")
	Expect(err).ToNot(HaveOccurred(), "folder should be created")

	acccesdCFG := fmt.Sprintf("tenant: %s\n", cfg.Tenant.UUID) +
		fmt.Sprintf("project: %s\n", cfg.Project.UUID) +
		fmt.Sprintf("server: %s\n", cfg.TestServer.UUID) +
		"database: /opt/serveraccessd/server-accessd.db\n" +
		"authdSocketPath: /run/sock1.sock"

	err = s.WriteFileToContainer(s.TestServerContainer,
		"/opt/server-access/config.yaml", acccesdCFG)
	Expect(err).ToNot(HaveOccurred(), "file should be written")

	err = RunAndCheckServerAccessd(*s, cfg.User.Identifier, cfg.TestServer.UUID, cfg.User.UUID)
	Expect(err).ToNot(HaveOccurred())
}

//go:embed client_sock1.yaml
var ClientSocketCFG string

//go:embed client_main.yaml
var сlientMainCFGTPL string

func ClientMainAuthdCFG(mainAuthdCfg *MainAuthdCfgV1) string {
	t, err := template.New("").Parse(сlientMainCFGTPL)
	Expect(err).ToNot(HaveOccurred(), "template should be ok")
	return mainAuthdCFG(t, mainAuthdCfg)
}

func (s *Suite) CheckClientBinariesAndFoldersExists() {
	err := s.CheckFileExistAtContainer(s.TestClientContainer, s.AuthdPath, "f")
	Expect(err).ToNot(HaveOccurred(), "TestClient should have authd")

	err = s.CheckFileExistAtContainer(s.TestClientContainer, s.cliPath, "f")
	Expect(err).ToNot(HaveOccurred(), "Test_сlient should have cli")

	// Authd can be configured and run at Test_server
	err = s.CreateIfNotExistsDirectoryAtContainer(s.TestClientContainer,
		"/etc/flant/negentropy/authd-conf.d")
	Expect(err).ToNot(HaveOccurred(), "folder should be created")
}

func (s *Suite) PrepareClientForSSHTesting(cfg fip.CheckingEnvironment) {
	s.CheckClientBinariesAndFoldersExists()
	clientMainCfg := ClientMainAuthdCFG(nil)
	err := s.WriteFileToContainer(s.TestClientContainer,
		"/etc/flant/negentropy/authd-conf.d/main.yaml", clientMainCfg)
	Expect(err).ToNot(HaveOccurred(), "file should be written")

	err = s.WriteFileToContainer(s.TestClientContainer,
		"/etc/flant/negentropy/authd-conf.d/sock1.yaml", ClientSocketCFG)
	Expect(err).ToNot(HaveOccurred(), "file should be written")

	err = s.WriteFileToContainer(s.TestClientContainer,
		"/opt/authd/client-jwt", cfg.UserMultipassJWT)
	Expect(err).ToNot(HaveOccurred(), "file should be written")

	s.ExecuteCommandAtContainer(s.TestClientContainer,
		[]string{"/bin/bash", "-c", "chmod 600 /opt/authd/client-jwt"}, nil)

	s.KillAllInstancesOfProcessAtContainer(s.TestClientContainer, s.AuthdPath)
	s.RunDaemonAtContainer(s.TestClientContainer, s.AuthdPath, "client_authd.log")
	time.Sleep(time.Second)
	pidAuthd := s.FirstProcessPIDAtContainer(s.TestClientContainer, s.AuthdPath)
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

func (s *Suite) CreateIfNotExistsDirectoryAtContainer(container *types.Container, path string) error {
	lastSeparator := strings.LastIndex(path, string(os.PathSeparator))
	if lastSeparator != 0 {
		err := s.CreateIfNotExistsDirectoryAtContainer(container, path[0:lastSeparator])
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

func (s *Suite) WriteFileToContainer(container *types.Container, path string, content string) error {
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

func (s *Suite) DeleteFileAtContainer(container *types.Container, filePath string) error {
	err := s.CheckFileExistAtContainer(container, filePath, "f")
	if err != nil {
		return nil
	}
	ctx := context.Background()
	config := types.ExecConfig{
		AttachStdin:  true,
		AttachStderr: true,
		AttachStdout: true,
		Cmd:          []string{"/bin/bash", "-c", "rm  " + filePath},
	}

	IDResp, err := s.dockerCli.ContainerExecCreate(ctx, container.ID, config)
	Expect(err).ToNot(HaveOccurred(), "error execution at container")

	resp, err := s.dockerCli.ContainerExecAttach(ctx, IDResp.ID, types.ExecStartCheck{})
	Expect(err).ToNot(HaveOccurred(), "error attaching execution at container")
	defer resp.Close()

	text, err := resp.Reader.ReadString('\n')
	if err != nil && err.Error() == "EOF" {
		fmt.Printf("file  %s was deleted  at container  %s \n", filePath, container.Names)
		return nil
	}
	Expect(err).ToNot(HaveOccurred(), "error response reading at container")
	return fmt.Errorf("unexpected output deleting file: %s", text)
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

func (s *Suite) KillAllInstancesOfProcessAtContainer(container *types.Container, processPath string) {
	for {
		pid := s.FirstProcessPIDAtContainer(container, processPath)
		if pid == 0 {
			break
		}
		s.killProcessAtContainer(container, pid)
	}
}

func (s *Suite) FirstProcessPIDAtContainer(container *types.Container, processPath string) int {
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

func (s *Suite) RunDaemonAtContainer(container *types.Container, daemonPath string, logFilePath string) {
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

func (s *Suite) DirectoryAtContainerNotExistOrEmptyWithRetry(container *types.Container, directoryPath string, maxAttempts int) error {
	return tests.Repeat(func() error {
		if s.DirectoryAtContainerNotExistOrEmpty(container, directoryPath) {
			return nil
		} else {
			return fmt.Errorf("directory %s at container %s is not empty", directoryPath, container.Names[0])
		}
	}, maxAttempts)
}

func calculatePrincipal(serverUUID string, userUUID model.UserUUID) string {
	principalHash := sha256.New()
	principalHash.Write([]byte(serverUUID))
	principalHash.Write([]byte(userUUID))
	principalSum := principalHash.Sum(nil)
	return fmt.Sprintf("%x", principalSum)
}

type MainAuthdCfgV1 struct {
	DefaultSocketDirectory string
	JwtPath                string
	RootVaultInternalURL   string
	AuthVaultInternalURL   string
}

func mainAuthdCFG(tpl *template.Template, mainAuthdCfg *MainAuthdCfgV1) string {
	var mainCFG bytes.Buffer
	if mainAuthdCfg == nil {
		mainAuthdCfg = &MainAuthdCfgV1{
			RootVaultInternalURL: getFromEnv("ROOT_VAULT_INTERNAL_URL"),
			AuthVaultInternalURL: getFromEnv("AUTH_VAULT_INTERNAL_URL"),
		}
	}
	err := tpl.Execute(&mainCFG, mainAuthdCfg)
	Expect(err).ToNot(HaveOccurred(), "template should be executed")
	return mainCFG.String()
}
