package ssh_session

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	ext "github.com/flant/negentropy/vault-plugins/flant_iam/extensions/extension_server_access/model"

	iam "github.com/flant/negentropy/vault-plugins/flant_iam/model"

	"github.com/google/uuid"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"

	"github.com/flant/negentropy/cli/internals/models"
	"github.com/flant/negentropy/cli/internals/vault"
)

type Session struct {
	UUID               string
	User               iam.User
	ServerList         models.ServerList
	ServerFilter       vault.ServerFilter
	VaultSession       vault.VaultSession
	EnvSSHAuthSock     string
	SSHAgent           agent.Agent
	SSHAgentSocketPath string
	SSHConfigFile      *os.File
	KnownHostsFile     *os.File
	BashRCFile         *os.File
}

const Workdir = "/tmp/flint"

func (s *Session) Destroy() {
	_ = os.Remove(s.SSHAgentSocketPath)
	_ = os.Remove(s.SSHConfigFile.Name())
	_ = os.Remove(s.KnownHostsFile.Name())
	_ = os.Remove(s.BashRCFile.Name())
}

func (s *Session) SyncServersFromVault() {
	sl, err := s.VaultSession.QueryServer(s.ServerFilter)
	if err != nil {
		panic(err)
	}

	for i := range sl.Servers {
		token, err := s.VaultSession.RequestServerToken(&sl.Servers[i])
		if err != nil {
			panic(err)
		}
		models.UpdateSecureData(&sl.Servers[i], token)
	}

	s.ServerList = sl
	// fmt.Println(sl.Projects[0].Tenant.UUID)
}

func (s *Session) RenderKnownHostsToFile() {
	_, err := s.KnownHostsFile.Stat()
	if err != nil {
		file, err := os.Create(fmt.Sprintf("%s/%s-known_hosts", Workdir, s.UUID))
		if err == nil {
			s.KnownHostsFile = file
		} else {
			panic(err)
		}
	}

	s.KnownHostsFile.Seek(0, 0)
	for _, server := range s.ServerList.Servers {
		s.KnownHostsFile.Write([]byte(models.RenderKnownHostsRow(server)))
	}
}

func (s *Session) RenderSSHConfigToFile() {
	_, err := s.SSHConfigFile.Stat()
	if err != nil {
		file, err := os.Create(fmt.Sprintf("%s/%s-ssh_config", Workdir, s.UUID))
		if err == nil {
			s.SSHConfigFile = file
		} else {
			panic(err)
		}
	}
	s.SSHConfigFile.Seek(0, 0)

	for _, server := range s.ServerList.Servers {
		project := s.ServerList.Projects[server.ProjectUUID]
		s.SSHConfigFile.Write([]byte(models.RenderSSHConfigEntry(project, server, s.User)))
	}
}

func (s *Session) RenderBashRCToFile() {
	_, err := s.BashRCFile.Stat()
	if err != nil {
		file, err := os.Create(fmt.Sprintf("%s/%s-bashrc", Workdir, s.UUID))
		if err == nil {
			s.BashRCFile = file
		} else {
			panic(err)
		}
	}

	data := fmt.Sprintf("alias ssh='ssh -o UserKnownHostsFile=%s -F %s';\n. ~/.bashrc;\nPS1=\"[flint] $PS1\"", s.KnownHostsFile.Name(), s.SSHConfigFile.Name())
	s.BashRCFile.Seek(0, 0)
	s.BashRCFile.Write([]byte(data))
}

func (s *Session) generateAndSignSSHCertificateSetForServerBucket(servers []ext.Server) agent.AddedKey {
	principals := []string{}
	identifiers := []string{}

	for _, server := range servers {
		principals = append(principals, models.GenerateUserPrincipal(server, s.User))
		identifiers = append(identifiers, server.Identifier)
	}

	privateRSA, _ := rsa.GenerateKey(rand.Reader, 2048)
	pubkey, err := ssh.NewPublicKey(&privateRSA.PublicKey)
	if err != nil {
		panic(err.Error())
	}

	vaultReq := map[string]interface{}{
		"public_key":       string(ssh.MarshalAuthorizedKey(pubkey)),
		"valid_principals": strings.Join(principals, ","),
	}

	signedPublicSSHCertBytes := s.VaultSession.SignPublicSSHCertificate(vaultReq)

	ak, _, _, _, err := ssh.ParseAuthorizedKey(signedPublicSSHCertBytes)
	if err != nil {
		panic(err.Error())
	}
	signedPublicSSHCert := ak.(*ssh.Certificate)

	return agent.AddedKey{
		PrivateKey:   privateRSA,
		Comment:      strings.Join(identifiers, ","),
		Certificate:  signedPublicSSHCert,
		LifetimeSecs: uint32(signedPublicSSHCert.ValidBefore - uint64(time.Now().UTC().Unix())),
	}
}

func (s *Session) RefreshClientCertificates() {
	maxSize := 256
	for i, j := 0, 0; i < len(s.ServerList.Servers); {
		j += maxSize
		if j > len(s.ServerList.Servers) {
			j = len(s.ServerList.Servers)
		}

		serversBucket := s.ServerList.Servers[i:j]
		signedCertificateForBucket := s.generateAndSignSSHCertificateSetForServerBucket(serversBucket)

		// TODO remove after refresh
		s.SSHAgent.Add(signedCertificateForBucket)
		i += maxSize
	}
}

func (s *Session) StartSSHAgent() {
	s.SSHAgent = agent.NewKeyring()
	s.SSHAgentSocketPath = fmt.Sprintf("%s/%s-ssh_agent.sock", Workdir, s.UUID)

	// TODO ошибки
	agentListener, _ := net.Listen("unix", s.SSHAgentSocketPath)

	// Close unix socket properly
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, os.Interrupt, os.Kill, syscall.SIGTERM)
	go func(c chan os.Signal) {
		sig := <-c
		log.Printf("Caught signal %s: shutting down.", sig)
		agentListener.Close()
		s.Destroy()
		os.Exit(0)
	}(sigc)

	go func() {
		defer agentListener.Close()
		for {
			// TODO break если что-то пошло не так
			conn, _ := agentListener.Accept()
			go func() {
				_ = agent.ServeAgent(s.SSHAgent, conn)
				conn.Close()
			}()
		}
	}()
}

func (s *Session) StartShell() {
	s.RenderBashRCToFile()

	os.Setenv("SSH_AUTH_SOCK", s.SSHAgentSocketPath)
	os.Setenv("FLINT_SESSION_UUID", s.UUID)

	// Todo redo
	cmdsJson := os.Getenv("COMMANDS")
	if cmdsJson == "" {
		cmd := exec.Command("/bin/bash", "--rcfile", s.BashRCFile.Name())

		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Env = os.Environ()
		_ = cmd.Run()
	} else {
		var cmds []string
		err := json.Unmarshal([]byte(cmdsJson), &cmds)
		if err != nil {
			panic(err)
		}

		cmd := exec.Command("/bin/bash", "--rcfile", s.BashRCFile.Name(), "-i")

		// cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Env = os.Environ()
		pipe, err := cmd.StdinPipe()
		if err != nil {
			panic(err)
		}
		//_ = cmd.Run()
		cmd.Start()
		pipe.Write([]byte("alias ssh='ssh -o UserKnownHostsFile=/tmp/flint/" + s.UUID + "-known_hosts -F /tmp/flint/" + s.UUID + "-ssh_config';\n"))

		for _, c := range cmds {
			pipe.Write([]byte(c))
			pipe.Write([]byte("\n"))
		}
		pipe.Close()
		time.Sleep(time.Second * 1) // Need some time to execute commands
	}
}

func (s *Session) syncRoutine() {
	s.SyncServersFromVault()
	s.RenderKnownHostsToFile()
	s.RenderSSHConfigToFile()
	s.RefreshClientCertificates()
}

func (s *Session) syncRoutineEveryMinute() {
	for {
		time.Sleep(time.Minute)
		s.syncRoutine()
	}
}

func (s *Session) Go() {
	os.MkdirAll(Workdir, os.ModePerm)
	s.UUID = uuid.Must(uuid.NewRandom()).String()
	// TODO Hardcoded

	tenantIdentifier := os.Getenv("TENANT_ID")
	if tenantIdentifier == "" {
		tenantIdentifier = "1tv"
	}
	fmt.Printf("Tenant identifier %s\n", tenantIdentifier)
	s.ServerFilter = vault.ServerFilter{
		TenantIdentifier: tenantIdentifier,
	}

	s.VaultSession.Init()
	s.User = s.VaultSession.GetSSHUser()
	s.StartSSHAgent()

	s.syncRoutine()
	go s.syncRoutineEveryMinute()

	s.StartShell()

	s.Destroy()
}
