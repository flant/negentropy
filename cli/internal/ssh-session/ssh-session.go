package ssh_session

import (
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"

	authdapi "github.com/flant/negentropy/authd/pkg/api/v1"
	"github.com/flant/negentropy/cli/internal/consts"
	"github.com/flant/negentropy/cli/internal/model"
	"github.com/flant/negentropy/cli/internal/vault"
	ext "github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_server_access/model"
	iam "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	auth "github.com/flant/negentropy/vault-plugins/flant_iam_auth/extensions/extension_server_access/model"
)

type VaultAuthReader interface {
	// GetUser returns user which credentials are recognized by vault
	GetUser() (*auth.User, error)
	// UpdateServersByFilter returns ServerList synchronized with vault, according filter, using given ServerList as cache
	UpdateServersByFilter(model.ServerFilter, *model.ServerList) (*model.ServerList, error)
	// UpdateTenants update oldTenants by vault requests, according specified identifiers given by args
	UpdateTenants(map[iam.TenantUUID]iam.Tenant, model.StringSet) (map[iam.TenantUUID]iam.Tenant, error)
	// UpdateProjects update oldProjects by vault requests, according specified identifiers given by args
	UpdateProjects(map[iam.ProjectUUID]iam.Project, map[iam.TenantUUID]iam.Tenant,
		model.StringSet) (map[iam.ProjectUUID]iam.Project, error)
}

type Session struct {
	UUID               string
	PermanentCache     model.Cache
	CachePath          string
	User               auth.User
	ServerList         *model.ServerList
	ServerFilter       model.ServerFilter
	VaultAuthReader    VaultAuthReader
	SSHAgent           agent.Agent
	SSHAgentSocketPath string
	SSHConfigFile      *os.File
	KnownHostsFile     *os.File
	BashRCFile         *os.File
	cacheTTL           time.Duration
}

func (s *Session) Close() error {
	err := os.Remove(s.SSHAgentSocketPath)
	if err != nil {
		return fmt.Errorf("close: %w", err)
	}
	err = os.Remove(s.SSHConfigFile.Name())
	if err != nil {
		return fmt.Errorf("close: %w", err)
	}
	err = os.Remove(s.KnownHostsFile.Name())
	if err != nil {
		return fmt.Errorf("close: %w", err)
	}
	err = os.Remove(s.BashRCFile.Name())
	if err != nil {
		return fmt.Errorf("close: %w", err)
	}
	return nil
}

func (s *Session) SyncServersFromVault() error {
	if s.ServerList == nil {
		cache, err := model.TryReadCacheFromFile(s.CachePath, s.cacheTTL)
		if err != nil {
			return fmt.Errorf("SyncServersFromVault, reading permanent cache: %w", err)
		}
		s.PermanentCache = *cache
		s.ServerList = &cache.ServerList
	}
	sl, err := s.VaultAuthReader.UpdateServersByFilter(s.ServerFilter, s.ServerList)
	if err != nil {
		return fmt.Errorf("SyncServersFromVault: %w", err)
	}
	s.ServerList = sl
	err = s.updateCache()
	if err != nil {
		return fmt.Errorf("SyncServersFromVault, saving permanent cache: %w", err)
	}
	return nil
}

func (s *Session) RenderKnownHostsToFile() error {
	_, err := s.KnownHostsFile.Stat()
	if err != nil {
		file, err := os.Create(fmt.Sprintf("%s/%s-known_hosts", consts.SSHWorkdir, s.UUID))
		if err != nil {
			return fmt.Errorf("RenderKnownHostsToFile: %w", err)
		}
		s.KnownHostsFile = file
	}

	s.KnownHostsFile.Seek(0, 0)
	for _, server := range s.ServerList.Servers {
		_, err := s.KnownHostsFile.Write([]byte(RenderKnownHostsRow(server)))
		if err != nil {
			return fmt.Errorf("RenderKnownHostsToFile: %w", err)
		}
	}
	return nil
}

func (s *Session) RenderSSHConfigToFile() error {
	_, err := s.SSHConfigFile.Stat()
	if err != nil {
		file, err := os.Create(fmt.Sprintf("%s/%s-ssh_config", consts.SSHWorkdir, s.UUID))
		if err != nil {
			return fmt.Errorf("RenderSSHConfigToFile: %w", err)
		}
		s.SSHConfigFile = file
	}
	_, err = s.SSHConfigFile.Seek(0, 0)
	if err != nil {
		return fmt.Errorf("RenderSSHConfigToFile: %w", err)
	}
	for _, server := range s.ServerList.Servers {
		project := s.ServerList.Projects[server.ProjectUUID]
		_, err := s.SSHConfigFile.Write([]byte(RenderSSHConfigEntry(project, server, s.User)))
		if err != nil {
			return fmt.Errorf("RenderSSHConfigToFile: %w", err)
		}

	}
	return nil
}

func (s *Session) RenderBashRCToFile() error {
	_, err := s.BashRCFile.Stat()
	if err != nil {
		file, err := os.Create(fmt.Sprintf("%s/%s-bashrc", consts.SSHWorkdir, s.UUID))
		if err != nil {
			return fmt.Errorf("RenderBashRCToFile: %w", err)
		}
		s.BashRCFile = file
	}

	data := fmt.Sprintf("alias ssh='ssh -o UserKnownHostsFile=%s -F %s';\n. ~/.bashrc;\nPS1=\"[flint] $PS1\"", s.KnownHostsFile.Name(), s.SSHConfigFile.Name())
	_, err = s.BashRCFile.Seek(0, 0)
	if err != nil {
		return fmt.Errorf("RenderBashRCToFile: %w", err)
	}
	_, err = s.BashRCFile.Write([]byte(data))
	if err != nil {
		return fmt.Errorf("RenderBashRCToFile: %w", err)
	}
	return nil
}

func (s *Session) generateAndSignSSHCertificateSetForServerBucket(servers []ext.Server) (*agent.AddedKey, error) {
	principals := []string{}
	serverIdentifiers := []string{}
	projects := map[iam.ProjectUUID]iam.TenantUUID{}

	for _, server := range servers {
		principals = append(principals, GenerateUserPrincipal(server.UUID, s.User.UUID))
		serverIdentifiers = append(serverIdentifiers, server.Identifier)
		projects[server.ProjectUUID] = server.TenantUUID
	}

	privateRSA, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("generateAndSignSSHCertificateSetForServerBucket: %w", err)
	}

	pubkey, err := ssh.NewPublicKey(&privateRSA.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("generateAndSignSSHCertificateSetForServerBucket: %w", err)
	}

	vaultReq := model.VaultSSHSignRequest{
		PublicKey:       string(ssh.MarshalAuthorizedKey(pubkey)),
		ValidPrincipals: strings.Join(principals, ","),
	}
	roles := make([]authdapi.RoleWithClaim, 0, len(projects))
	for projectUUID, tenantUUID := range projects {
		roles = append(roles, authdapi.RoleWithClaim{
			Role:        "ssh",
			TenantUUID:  tenantUUID,
			ProjectUUID: projectUUID,
			Claim:       map[string]interface{}{"ttl": "720m", "max_ttl": "1440m"},
		})
	}
	sshSigner := vault.NewService(roles...)

	signedPublicSSHCertBytes, err := sshSigner.SignPublicSSHCertificate(vaultReq)
	if err != nil {
		return nil, fmt.Errorf("generateAndSignSSHCertificateSetForServerBucket: %w", err)
	}

	ak, _, _, _, err := ssh.ParseAuthorizedKey(signedPublicSSHCertBytes)
	if err != nil {
		return nil, fmt.Errorf("generateAndSignSSHCertificateSetForServerBucket: %w", err)
	}
	signedPublicSSHCert := ak.(*ssh.Certificate)

	return &agent.AddedKey{
		PrivateKey:   privateRSA,
		Comment:      strings.Join(serverIdentifiers, ","),
		Certificate:  signedPublicSSHCert,
		LifetimeSecs: uint32(signedPublicSSHCert.ValidBefore - uint64(time.Now().UTC().Unix())),
	}, nil
}

func (s *Session) RefreshClientCertificates() error {
	servers := make([]ext.Server, 0, len(s.ServerList.Servers))
	for _, s := range s.ServerList.Servers {
		servers = append(servers, s)
	}
	maxSize := 256
	for i, j := 0, 0; i < len(servers); {
		j += maxSize
		if j > len(servers) {
			j = len(servers)
		}

		serversBucket := servers[i:j]
		signedCertificateForBucket, err := s.generateAndSignSSHCertificateSetForServerBucket(serversBucket)
		if err != nil {
			return fmt.Errorf("RefreshClientCertificates: %w", err)
		}

		// TODO remove after refresh
		err = s.SSHAgent.Add(*signedCertificateForBucket)
		if err != nil {
			return fmt.Errorf("RefreshClientCertificates: %w", err)
		}
		i += maxSize
	}
	return nil
}

func (s *Session) StartSSHAgent() error {
	s.SSHAgent = agent.NewKeyring()
	s.SSHAgentSocketPath = fmt.Sprintf("%s/%s-ssh_agent.sock", consts.SSHWorkdir, s.UUID)

	agentListener, err := net.Listen("unix", s.SSHAgentSocketPath)
	if err != nil {
		return fmt.Errorf("StartSSHAgent: %w", err)
	}

	// Close unix socket properly
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, os.Interrupt, os.Kill, syscall.SIGTERM)
	go func(c chan os.Signal) {
		sig := <-c
		log.Printf("Caught signal %s: shutting down.", sig)
		agentListener.Close()
		s.Close()
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
	return nil
}

func (s *Session) StartShell() error {
	err := s.RenderBashRCToFile()
	if err != nil {
		return fmt.Errorf("StartShell: %w", err)
	}

	os.Setenv("SSH_AUTH_SOCK", s.SSHAgentSocketPath)
	os.Setenv("FLINT_SESSION_UUID", s.UUID)

	cmd := exec.Command("/bin/bash", "--rcfile", s.BashRCFile.Name(), "-i")

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("StartShell: %w", err)
	}
	return nil
}

func (s *Session) syncRoutine() error {
	err := s.SyncServersFromVault()
	if err != nil {
		return fmt.Errorf("syncRoutine: %w", err)
	}
	err = s.RenderKnownHostsToFile()
	if err != nil {
		return fmt.Errorf("syncRoutine: %w", err)
	}
	err = s.RenderSSHConfigToFile()
	if err != nil {
		return fmt.Errorf("syncRoutine: %w", err)
	}
	err = s.RefreshClientCertificates()
	if err != nil {
		return fmt.Errorf("syncRoutine: %w", err)
	}
	return nil
}

func (s *Session) syncRoutineEveryMinute() {
	for {
		time.Sleep(time.Minute)
		s.syncRoutine()
	}
}

func (s *Session) Start() error {
	err := s.StartSSHAgent()
	if err != nil {
		return err
	}
	err = s.syncRoutine()
	if err != nil {
		return err
	}
	go s.syncRoutineEveryMinute()

	err = s.StartShell()
	if err != nil {
		return err
	}
	err = s.Close()
	return err
}

func (s *Session) updateCache() error {
	s.PermanentCache.Update(*s.ServerList)
	return s.PermanentCache.SaveToFile(s.CachePath)
}

func New(vaultService vault.VaultService, serverFilter model.ServerFilter, cacheFilePath string, cacheTTL time.Duration) (*Session, error) {
	os.MkdirAll(consts.SSHWorkdir, os.ModePerm)
	session := Session{VaultAuthReader: vaultService, ServerFilter: serverFilter, CachePath: cacheFilePath}
	session.UUID = uuid.Must(uuid.NewRandom()).String()
	session.SSHAgentSocketPath = fmt.Sprintf("%s/%s-ssh_agent.sock", consts.SSHWorkdir, session.UUID)
	user, err := session.VaultAuthReader.GetUser()
	if err != nil {
		return nil, fmt.Errorf("getting user: %w", err)
	}
	session.User = *user
	session.cacheTTL = cacheTTL
	return &session, nil
}
