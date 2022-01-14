package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/flant/negentropy/e2e/tests/lib"
	"github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/specs"
)

var feedingMultipliers = []int{10, 20, 30}

// var feedingMultipliers = []int{1_000, 10_000, 50_000, 100_000}

type Result struct {
	FeedMultiplier  int           // feed multiplier
	RootIAMFactory  time.Duration // root vault IAM factory second run
	RootAUTHFactory time.Duration // root vault AUTH factory second run
	AuthAUTHFactory time.Duration // auth vault AUTH factory second run
}

func main() {
	// to use e2e test libs
	RegisterFailHandler(Fail)
	defer GinkgoRecover()

	result := []Result{}
	s := Suite{}
	s.BeforeSuite()
	// check plugins alive
	dieOnErr(s.rootVault.TouchIAM())
	dieOnErr(s.rootVault.TouchAUTH())
	dieOnErr(s.authVault.TouchAUTH())
	feedingAmmount := 0
	for _, multiplier := range feedingMultipliers {
		toFeed := multiplier - feedingAmmount
		feed(toFeed)

		// stop containers
		dieOnErr(s.rootVault.StopContainer())
		dieOnErr(s.authVault.StopContainer())

		// up containers
		dieOnErr(s.rootVault.StartContainer())
		dieOnErr(s.authVault.StartContainer())

		time.Sleep(time.Second * 3)

		// reader := bufio.NewReader(os.Stdin)
		// fmt.Print("Check topics and press Enter")
		// reader.ReadString('\n')

		// unseal vaults
		dieOnErr(s.rootVault.Unseal())
		dieOnErr(s.authVault.Unseal())

		// wake up plugins
		s.rootVault.TouchIAM()
		s.rootVault.TouchAUTH()
		s.authVault.TouchAUTH()

		// wait plugins
		dieOnErr(repeat(s.rootVault.TouchIAM, 10))
		dieOnErr(repeat(s.rootVault.TouchAUTH, 10))
		dieOnErr(repeat(s.authVault.TouchAUTH, 10))

		result = append(result, s.CollectMetrics(multiplier))
		feedingAmmount += toFeed
	}
	fmt.Printf("         N     RootIAMFactory    RootAUTHFactory    AuthAUTHFactory\n")
	for _, r := range result {
		fmt.Printf("%10d %18s %18s %18s\n", r.FeedMultiplier, r.RootIAMFactory.String(), r.RootAUTHFactory.String(), r.AuthAUTHFactory.String())
	}
}

func repeat(f func() error, maxAttempts int) error {
	err := f()
	counter := 1
	for err != nil {
		if counter > maxAttempts {
			return fmt.Errorf("exceeded attempts, last err:%w", err)
		}
		counter++
		time.Sleep(time.Second)
		err = f()
	}
	return nil
}

type Suite struct {
	dockerCli *client.Client

	authVault *Vault
	rootVault *Vault
}

type Vault struct {
	containerName string
	vaultURL      string
	vaultToken    string
	container     *types.Container
	unsealKeys    []string

	dockerCli *client.Client
}

func NewVault(dockerCli *client.Client, containerName string, vaultURLEnv string, vaultTokenEnv string, unsealKeysFileName string) *Vault {
	vault := &Vault{
		containerName: containerName,
		vaultURL:      getFromEnv(vaultURLEnv),
		vaultToken:    getFromEnv(vaultTokenEnv),
		dockerCli:     dockerCli,
	}

	// Open connections, create client
	var err error

	vault.container, err = getContainerByName(dockerCli, vault.containerName)
	dieOnErr(err)

	vault.unsealKeys = readUnsealKeys(unsealKeysFileName)
	return vault
}

func (v *Vault) Unseal() error {
	for i := 0; i < 3; i++ {
		executeCommandAtContainer(v.dockerCli, v.container, []string{
			"/bin/sh", "-c", "vault operator unseal " + v.unsealKeys[i],
		}, nil, []string{"VAULT_TOKEN=" + v.vaultToken})
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
	url := v.vaultURL + "/v1/flant_iam/tenant/"
	req, err := http.NewRequest("GET", url, nil)
	dieOnErr(err)
	req.Header["X-Vault-Token"] = []string{v.vaultToken}
	client := http.DefaultClient

	resp, err := client.Do(req)
	dieOnErr(err)
	if resp.StatusCode != 200 {
		return fmt.Errorf("wrong response status:%d", resp.StatusCode)
	}
	return nil
}

func (v *Vault) TouchAUTH() error {
	url := v.vaultURL + "/v1/auth/flant_iam_auth/auth_method/multipass"
	req, err := http.NewRequest("GET", url, nil)
	dieOnErr(err)
	req.Header["X-Vault-Token"] = []string{v.vaultToken}
	client := http.DefaultClient

	resp, err := client.Do(req)
	dieOnErr(err)
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
	dieOnErr(err)

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
		dieOnErr(err)
	}
	startTime, err := timeStamp(startedString)
	dieOnErr(err)
	normalFinishTime, err := timeStamp(normalFinishString)
	dieOnErr(err)
	return normalFinishTime.Sub(startTime)
}

func timeStamp(logLine string) (time.Time, error) {
	a := strings.Split(logLine, " ")
	return time.Parse(time.RFC3339Nano, a[1])
}

func (s *Suite) BeforeSuite() {
	var err error
	s.dockerCli, err = client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())

	dieOnErr(err)

	s.rootVault = NewVault(s.dockerCli,
		"vault-root",
		"ROOT_VAULT_URL",
		"ROOT_VAULT_TOKEN",
		"/tmp/vault_root_operator_output")

	s.authVault = NewVault(s.dockerCli,
		"vault-auth",
		"AUTH_VAULT_URL",
		"AUTH_VAULT_TOKEN",
		"/tmp/vault_auth_operator_output")
}

func (s *Suite) CollectMetrics(caseFeedMultiplier int) Result {
	return Result{
		FeedMultiplier:  caseFeedMultiplier,
		RootIAMFactory:  s.rootVault.LastFactoryDuration("IAM"),
		RootAUTHFactory: s.rootVault.LastFactoryDuration("AUTH"),
		AuthAUTHFactory: s.authVault.LastFactoryDuration("AUTH"),
	}
}

func readUnsealKeys(file string) []string {
	dat, err := os.ReadFile(file)
	dieOnErr(err)
	ss := strings.Split(string(dat), "\n")
	var result []string
	for _, s := range ss {
		if strings.HasPrefix(s, "Unseal Key ") {
			l := strings.Split(s, ":")
			if len(l) != 2 {
				panic("Invalid string: " + s)
			}
			result = append(result, strings.TrimSpace(l[1]))
		}
	}
	if len(result) < 3 {
		panic("wrong amount Unseal keys, need at least 3, collect:" + strconv.Itoa(len(result)))
	}
	return result
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

func getFromEnv(envName string) string {
	value := os.Getenv(envName)
	if value == "" {
		panic(fmt.Sprintf("required environment variable %s is not set", envName))
	}
	return value
}

func dieOnErr(err error) {
	if err != nil {
		panic(err)
	}
}

func executeCommandAtContainer(dockerCli *client.Client, container *types.Container, cmd []string,
	extraInputToSTDIN []string, envs []string) []string {
	ctx := context.Background()
	config := types.ExecConfig{
		AttachStdin:  true,
		AttachStderr: true,
		AttachStdout: true,
		Cmd:          cmd,
		Env:          envs,
	}

	IDResp, err := dockerCli.ContainerExecCreate(ctx, container.ID, config)
	dieOnErr(err)

	resp, err := dockerCli.ContainerExecAttach(ctx, IDResp.ID, types.ExecStartCheck{})
	dieOnErr(err)
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
	dieOnErr(err)
	return nil
}

func feed(n int) {
	rootClient := lib.NewConfiguredIamVaultClient()
	tenantApi := lib.NewTenantAPI(rootClient)
	userApi := lib.NewUserAPI(rootClient)
	tenant := specs.CreateRandomTenant(tenantApi)
	for i := 0; i < n; i++ {
		if i%10 == 0 {
			fmt.Printf("%d/%d\n", n, i)
		}
		specs.CreateRandomUser(userApi, tenant.UUID)
	}
}
