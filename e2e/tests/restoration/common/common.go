package common

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"

	"github.com/flant/negentropy/e2e/tests/lib"
	"github.com/flant/negentropy/vault-plugins/shared/tests"
)

type Suite struct {
	dockerCli *client.Client

	authVault *Vault
	rootVault *Vault
}

type Vault struct {
	ContainerName string
	VaultName     string `json:"name"`
	VaultURL      string `json:"url"`
	VaultToken    string `json:"token"`
	container     *types.Container
	UnsealKeys    []string `json:"keys"`

	dockerCli *client.Client
}

func NewVault(dockerCli *client.Client, containerName string, vaultName string) *Vault {
	vault := ReadVaultFromFile(vaultName)
	vault.ContainerName = containerName
	vault.dockerCli = dockerCli

	// Open connections, create client
	var err error

	vault.container, err = getContainerByName(dockerCli, vault.ContainerName)
	DieOnErr(err)

	return &vault
}

func (v *Vault) Unseal() error {
	println("unsealing: ", v.VaultName)
	for i := 0; i < 3; i++ {
		output, err := executeCommandAtContainer(v.dockerCli, v.container, []string{
			"/bin/sh", "-c", "vault operator unseal " + v.UnsealKeys[i],
		}, nil, []string{"VAULT_TOKEN=" + v.VaultToken})
		if err != nil {
			return err
		}
		println()
		println("vault operator unseal " + v.UnsealKeys[i])
		println()
		for _, l := range output {
			println(l)
		}
	}
	return nil
}

var fiveSeconds = time.Second * 5

func (v *Vault) StopContainer() error {
	return v.dockerCli.ContainerStop(context.TODO(), v.container.ID, &fiveSeconds)
}

func (v *Vault) StartContainer() error {
	return v.dockerCli.ContainerStart(context.TODO(), v.container.ID, types.ContainerStartOptions{})
}

func (v *Vault) TouchIAM() error {
	url := v.VaultURL + "/v1/flant/tenant/"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header["X-Vault-Token"] = []string{v.VaultToken}
	client := lib.HttpClientWithoutInsequireVerifing()

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("wrong response status:%d", resp.StatusCode)
	}
	return nil
}

func (v *Vault) TouchAUTH() error {
	url := v.VaultURL + "/v1/auth/flant/auth_method/multipass"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header["X-Vault-Token"] = []string{v.VaultToken}
	client := lib.HttpClientWithoutInsequireVerifing()

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("wrong response status:%d", resp.StatusCode)
	}
	return nil
}

func (v *Vault) LastFactoryDuration(pluginName string) time.Duration {
	reader, err := v.dockerCli.ContainerLogs(context.TODO(), v.container.ID, types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Details:    true,
	})
	DieOnErr(err)

	sc := bufio.NewScanner(reader)
	startedString := ""
	normalFinishString := ""
	for sc.Scan() {
		l := sc.Text() // GET the line string
		if strings.Contains(l, pluginName+".Factory: started") {
			startedString = l
		}
		if strings.Contains(l, pluginName+".Factory: normal finish") {
			normalFinishString = l
		}
	}
	if err := sc.Err(); err != nil {
		DieOnErr(err)
	}
	startTime, err := timeStamp(startedString)
	DieOnErr(err)
	normalFinishTime, err := timeStamp(normalFinishString)
	DieOnErr(err)
	return normalFinishTime.Sub(startTime)
}

func timeStamp(logLine string) (time.Time, error) {
	a := strings.Split(logLine, " ")
	return time.Parse(time.RFC3339Nano, a[1])
}

func (s *Suite) BeforeSuite() {
	var err error
	s.dockerCli, err = client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())

	DieOnErr(err)

	s.rootVault = NewVault(s.dockerCli, "vault-root", "root")

	s.authVault = NewVault(s.dockerCli, "vault-auth", "auth")
	// check plugins alive
	DieOnErr(s.rootVault.TouchIAM())
	DieOnErr(s.rootVault.TouchAUTH())
	DieOnErr(s.authVault.TouchAUTH())
}

type RestorationDurationResult struct {
	FeedMultiplier  int           // feed multiplier
	RootIAMFactory  time.Duration // root vault IAM factory second run
	RootAUTHFactory time.Duration // root vault AUTH factory second run
	AuthAUTHFactory time.Duration // auth vault AUTH factory second run
}

func (s *Suite) CollectMetrics(caseFeedMultiplier int) RestorationDurationResult {
	return RestorationDurationResult{
		FeedMultiplier:  caseFeedMultiplier,
		RootIAMFactory:  s.rootVault.LastFactoryDuration("IAM"),
		RootAUTHFactory: s.rootVault.LastFactoryDuration("AUTH"),
		AuthAUTHFactory: s.authVault.LastFactoryDuration("AUTH"),
	}
}

func (s *Suite) RestartVaults() error {
	for _, op := range []func() error{
		// stop containers
		s.rootVault.StopContainer,
		s.authVault.StopContainer,
		// up containers
		s.rootVault.StartContainer,
		s.authVault.StartContainer,
		func() error { time.Sleep(time.Second * 5); return nil }, //  time to vaults up
		s.rootVault.Unseal,
		s.authVault.Unseal,
		func() error { time.Sleep(time.Second * 5); return nil },            //  time to vaults up
		func() error { return tests.SlowRepeat(s.authVault.TouchAUTH, 50) }, // wait it first because of longer restoration of root vault due to two plugins on board
		func() error { return tests.SlowRepeat(s.rootVault.TouchIAM, 50) },
	} {
		if err := op(); err != nil {
			return err
		}
	}
	return nil
}

func getContainerByName(dockerCli *client.Client, name string) (*types.Container, error) {
	name = strings.ReplaceAll(name, "_", "-")
	containers, err := dockerCli.ContainerList(context.Background(), types.ContainerListOptions{})
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

func DieOnErr(err error) {
	if err != nil {
		fmt.Printf("critical error: %s", err.Error())
		panic(err)
	}
}

func executeCommandAtContainer(dockerCli *client.Client, container *types.Container, cmd []string,
	extraInputToSTDIN []string, envs []string) ([]string, error) {
	ctx := context.Background()
	config := types.ExecConfig{
		AttachStdin:  true,
		AttachStderr: true,
		AttachStdout: true,
		Cmd:          cmd,
		Env:          envs,
	}

	IDResp, err := dockerCli.ContainerExecCreate(ctx, container.ID, config)
	if err != nil {
		return nil, err
	}

	resp, err := dockerCli.ContainerExecAttach(ctx, IDResp.ID, types.ExecStartCheck{})
	if err != nil {
		return nil, err
	}
	defer resp.Close()

	go func() {
		for _, input := range extraInputToSTDIN {
			time.Sleep(time.Millisecond * 500)
			resp.Conn.Write([]byte(input + "\n"))
		}
	}()
	defer resp.Close()
	output, err := tests.ParseDockerOutput(resp.Reader)
	if err != nil {
		return nil, err
	}

	return strings.Split(string(output), "\n"), nil
}

const vaultsFilePath = "/tmp/vaults"

func ReadVaultFromFile(vaultName string) Vault {
	data, err := os.ReadFile(vaultsFilePath)
	DieOnErr(err)
	var vaults []Vault
	err = json.Unmarshal(data, &vaults)
	DieOnErr(err)
	for _, v := range vaults {
		if v.VaultName == vaultName {
			os.Setenv(strings.ToUpper(vaultName)+"_VAULT_TOKEN", v.VaultToken) // nolint:errcheck
			os.Setenv(strings.ToUpper(vaultName)+"_VAULT_URL", v.VaultURL)     // nolint:errcheck
			return v
		}
	}
	panic(fmt.Sprintf("vault with name %s not found at %s", vaultName, vaultsFilePath))
}
