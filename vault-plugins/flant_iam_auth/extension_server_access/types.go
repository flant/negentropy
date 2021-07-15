package extension_server_access

import (
	"crypto/sha256"
	"fmt"

	"github.com/GehirnInc/crypt"
	_ "github.com/GehirnInc/crypt/sha512_crypt"

	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/extension_server_access/model"
	iam "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

type ServerAccessConfig struct {
	RoleForSSHAccess string
}

type posixUserBuilder struct {
	tx *io.MemoryStoreTxn

	serverID string
	tenantID string
}

func newPosixUserBuilder(tx *io.MemoryStoreTxn, serverID, tenantID string) *posixUserBuilder {
	return &posixUserBuilder{tx, serverID, tenantID}
}

type posixUser struct {
	UID       int    `json:"uid"`
	Principal string `json:"principal"`

	Name     string `json:"name"`
	HomeDir  string `json:"home_directory"`
	Password string `json:"password"`
	Shell    string `json:"shell"`
	Gecos    string `json:"gecos"`
	Gid      int    `json:"gid"`
}

func (pb *posixUserBuilder) newPosixUser(uid int, principal, name, homeDir, pass string) posixUser {
	return posixUser{
		UID:       uid,
		Principal: principal,
		Name:      name,
		HomeDir:   homeDir,
		Password:  pass,
		Shell:     "/bin/bash",
		Gecos:     "",
		Gid:       999,
	}
}

func (pb *posixUserBuilder) userToPosix(user *iam.User) (posixUser, error) {
	ext, ok := user.Extensions["server_access"]
	if !ok {
		return posixUser{}, fmt.Errorf("server_access extension not found for user: %s", user.FullIdentifier)
	}

	return pb.buildPosixUser(ext, user.UUID, user.TenantUUID, user.Identifier, user.FullIdentifier)
}

func (pb *posixUserBuilder) serviceAccountToPosix(sa *iam.ServiceAccount) (posixUser, error) {
	ext, ok := sa.Extensions["server_access"]
	if !ok {
		return posixUser{}, fmt.Errorf("server_access extension not found for service account: %s", sa.FullIdentifier)
	}

	return pb.buildPosixUser(ext, sa.UUID, sa.TenantUUID, sa.Identifier, sa.FullIdentifier)
}

func (pb *posixUserBuilder) buildPosixUser(ext *iam.Extension, objectID, objectTenantID, identifier, fullIdentifier string) (posixUser, error) {
	uid, ok := ext.Attributes["UID"]
	if !ok {
		return posixUser{}, fmt.Errorf("UID not found in server_access extension for %s", fullIdentifier)
	}
	principalHash := sha256.New()
	principalHash.Write([]byte(pb.serverID))
	principalHash.Write([]byte(objectID))
	principalSum := principalHash.Sum(nil)
	principal := fmt.Sprintf("%x", principalSum)

	name := identifier
	homeDirRelPath := identifier

	if pb.tenantID != objectTenantID {
		name = fullIdentifier
		repo := iam.NewTenantRepository(pb.tx)
		tenant, err := repo.GetByID(objectTenantID)
		if err != nil {
			return posixUser{}, fmt.Errorf("tenant error: %s", err)
		}
		homeDirRelPath = fmt.Sprintf("%s/%s", tenant.Identifier, identifier)
	}

	homeDir := "/home/" + homeDirRelPath

	passwordsRaw, ok := ext.Attributes["passwords"]
	if !ok {
		return posixUser{}, fmt.Errorf("passwords field not found in server_access extension for %q", fullIdentifier)
	}
	passwords, ok := passwordsRaw.([]model.UserServerPassword)
	if !ok {
		return posixUser{}, fmt.Errorf("passwords field type mismatch in server_access extension for %q", fullIdentifier)
	}
	if len(passwords) == 0 {
		return posixUser{}, fmt.Errorf("no passwords found in server_access extension for %q", fullIdentifier)
	}
	lastPass := passwords[len(passwords)-1]

	crypter := crypt.SHA512.New()
	pass, err := crypter.Generate([]byte(pb.serverID), []byte("$6$"+string(lastPass.Salt)))
	if err != nil {
		return posixUser{}, fmt.Errorf("password crypt failed (%s) for %q", err, fullIdentifier)
	}

	return pb.newPosixUser(uid.(int), principal, name, homeDir, pass), nil
}
