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
	uid, ok := ext.Attributes["UID"]
	if !ok {
		return posixUser{}, fmt.Errorf("UID not found in server_access extension for user: %s", user.FullIdentifier)
	}
	principalHash := sha256.New()
	principalHash.Write([]byte(serverID))
	principalHash.Write([]byte(user.UUID))
	principalSum := principalHash.Sum(nil)
	principal := fmt.Sprintf("%x", principalSum)

	name := user.Identifier
	if tenantID != user.TenantUUID {
		name = user.FullIdentifier
	}

	homeDir := "/home/" + user.Identifier
	if tenantID != user.TenantUUID {
		homeDir = fmt.Sprintf("/home/%s/%s", user.TenantUUID, user.Identifier)
	}

	passwordsRaw, ok := ext.Attributes["passwords"]
	if !ok {
		return posixUser{}, fmt.Errorf("passwords field not found in server_access extension for user: %s", user.FullIdentifier)
	}
	passwords, ok := passwordsRaw.([]iam.UserServerPassword)
	if !ok {
		return posixUser{}, fmt.Errorf("passwords field type mismatch in server_access extension for user: %s", user.FullIdentifier)
	}
	if len(passwords) == 0 {
		return posixUser{}, fmt.Errorf("no passwords found in server_access extension for user: %s", user.FullIdentifier)
	}
	lastPass := passwords[len(passwords)-1]

	crypter := crypt.SHA512.New()
	pass, err := crypter.Generate([]byte(serverID), []byte("$6$"+lastPass.Salt))
	if err != nil {
		return posixUser{}, fmt.Errorf("password crypt failed (%s) for user: %s", err, user.FullIdentifier)
	}

	return newPosixUser(uid.(int), principal, name, homeDir, pass), nil
}

func saToPosix(serverID, tenantID string, sa *iam.ServiceAccount) (posixUser, error) {
	ext, ok := sa.Extensions["server_access"]
	if !ok {
		return posixUser{}, fmt.Errorf("server_access extension not found for service account: %s", sa.FullIdentifier)
	}
	uid, ok := ext.Attributes["UID"]
	if !ok {
		return posixUser{}, fmt.Errorf("UID not found in server_access extension for service account: %s", sa.FullIdentifier)
	}
	principalHash := sha256.New()
	principalHash.Write([]byte(serverID))
	principalHash.Write([]byte(sa.UUID))
	principalSum := principalHash.Sum(nil)
	principal := fmt.Sprintf("%x", principalSum)

	name := sa.Identifier
	if tenantID != sa.TenantUUID {
		name = sa.FullIdentifier
	}

	homeDir := "/home/" + sa.Identifier
	if tenantID != sa.TenantUUID {
		homeDir = fmt.Sprintf("/home/%s/%s", sa.TenantUUID, sa.Identifier)
	}

	passwordsRaw, ok := ext.Attributes["passwords"]
	if !ok {
		return posixUser{}, fmt.Errorf("passwords field not found in server_access extension for service account: %s", sa.FullIdentifier)
	}
	passwords, ok := passwordsRaw.([]iam.UserServerPassword)
	if !ok {
		return posixUser{}, fmt.Errorf("passwords field type mismatch in server_access extension for service account: %s", sa.FullIdentifier)
	}
	if len(passwords) == 0 {
		return posixUser{}, fmt.Errorf("no passwords found in server_access extension for service account: %s", sa.FullIdentifier)
	}
	lastPass := passwords[len(passwords)-1]

	crypter := crypt.SHA512.New()
	pass, err := crypter.Generate([]byte(serverID), []byte("$6$"+lastPass.Salt))
	if err != nil {
		return posixUser{}, fmt.Errorf("password crypt failed (%s) for service account: %s", err, sa.FullIdentifier)
	}

	return newPosixUser(uid.(int), principal, name, homeDir, pass), nil
}
