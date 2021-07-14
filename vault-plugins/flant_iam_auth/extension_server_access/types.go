package extension_server_access

import (
	"crypto/sha256"
	"fmt"

	"github.com/GehirnInc/crypt"
	_ "github.com/GehirnInc/crypt/sha512_crypt"

	iam "github.com/flant/negentropy/vault-plugins/flant_iam/model"
)

type ServerAccessConfig struct {
	RoleForSSHAccess string
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

func newPosixUser(uid int, principal, name, homeDir, pass string) posixUser {
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

func userToPosix(serverID, tenantID string, user *iam.User) (posixUser, error) {
	ext, ok := user.Extensions["server_access"]
	if !ok {
		return posixUser{}, fmt.Errorf("server_access extension not found for user: %s", user.FullIdentifier)
	}

	return buildPosixUser(ext, serverID, tenantID, user.UUID, user.TenantUUID, user.Identifier, user.FullIdentifier)

}

func saToPosix(serverID, tenantID string, sa *iam.ServiceAccount) (posixUser, error) {
	ext, ok := sa.Extensions["server_access"]
	if !ok {
		return posixUser{}, fmt.Errorf("server_access extension not found for service account: %s", sa.FullIdentifier)
	}

	return buildPosixUser(ext, serverID, tenantID, sa.UUID, sa.TenantUUID, sa.Identifier, sa.FullIdentifier)

}

func buildPosixUser(ext *iam.Extension, serverID, tenantID, objectID, objectTenantID, identifier, fullIdentifier string) (posixUser, error) {
	uid, ok := ext.Attributes["UID"]
	if !ok {
		return posixUser{}, fmt.Errorf("UID not found in server_access extension for %s", fullIdentifier)
	}
	principalHash := sha256.New()
	principalHash.Write([]byte(serverID))
	principalHash.Write([]byte(objectID))
	principalSum := principalHash.Sum(nil)
	principal := fmt.Sprintf("%x", principalSum)

	name := identifier
	homeDirRelPath := identifier

	if tenantID != objectTenantID {
		name = fullIdentifier
		homeDirRelPath = fmt.Sprintf("%s/%s", objectTenantID, identifier)
	}

	homeDir := "/home/" + homeDirRelPath

	passwordsRaw, ok := ext.Attributes["passwords"]
	if !ok {
		return posixUser{}, fmt.Errorf("passwords field not found in server_access extension for %q", fullIdentifier)
	}
	passwords, ok := passwordsRaw.([]iam.UserServerPassword)
	if !ok {
		return posixUser{}, fmt.Errorf("passwords field type mismatch in server_access extension for %q", fullIdentifier)
	}
	if len(passwords) == 0 {
		return posixUser{}, fmt.Errorf("no passwords found in server_access extension for %q", fullIdentifier)
	}
	lastPass := passwords[len(passwords)-1]

	crypter := crypt.SHA512.New()
	pass, err := crypter.Generate([]byte(serverID), []byte("$6$"+lastPass.Salt))
	if err != nil {
		return posixUser{}, fmt.Errorf("password crypt failed (%s) for %q", err, fullIdentifier)
	}

	return newPosixUser(uid.(int), principal, name, homeDir, pass), nil
}
