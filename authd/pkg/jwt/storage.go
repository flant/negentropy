package jwt

import (
	"fmt"
	"io/ioutil"
	"os"
	"sync"
	"syscall"
	"time"
)

const TokenFileMode = 0600

type Storage struct {
	Path  string
	token *Token
	m     sync.RWMutex
}

var DefaultStorage = &Storage{}

/*
- нет файла с JWT;
- файл с JWT есть, но у него неправильные права:
должны быть 0600,
owner и group – соответствовать owner и group, с которым запущен authd;
- JWT истек
*/
func (s *Storage) Load(path string) error {
	// This method should be called once, but guard it anyway.
	s.m.Lock()
	defer s.m.Unlock()

	s.Path = path
	stat, err := checkFileExists(s.Path)
	if err != nil {
		return err
	}
	err = checkFilePerms(stat)
	if err != nil {
		return err
	}

	bytes, err := ioutil.ReadFile(s.Path)
	if err != nil {
		return err
	}

	s.token, err = ParseToken(string(bytes))
	if err != nil {
		return fmt.Errorf("JWT load: %v", err)
	}

	err = checkTokenExpired(s.token)
	if err != nil {
		return err
	}

	return nil
}

// GetJWT returns "raw" JWT for use with vault client.
// TODO We should introduce an "expired" state. This method should reload token from path.
// TODO scenario:
// - authd should refresh token
// - if authd doesn't refresh token in time because of errors or long offline, then:
// - auth should work, no panic allowed.
// - flant open ssh should print warning: Refresh JWT manually.
// - User gets new token, and update file.
// - User starts flant open ssh again and authd should load JWT from file again.
func (s *Storage) GetJWT() (string, error) {
	s.m.RLock()
	err := checkTokenExpired(s.token)
	s.m.RUnlock()
	if err != nil {
		err = s.Load(s.Path)
		if err != nil {
			return "", fmt.Errorf("JWT in '%s' is expired", s.Path)
		}
	}
	s.m.RLock()
	defer s.m.RUnlock()
	return s.token.JWT, nil
}

// Update parses new token and saves it in file.
func (s *Storage) Update(newToken string) error {
	s.m.Lock()
	defer s.m.Unlock()

	t, err := ParseToken(newToken)
	if err != nil {
		return err
	}
	s.token = t
	err = ioutil.WriteFile(s.Path, []byte(newToken), TokenFileMode)
	if err != nil {
		return err
	}
	return nil
}

func checkFileExists(path string) (os.FileInfo, error) {
	stat, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("JWT load: no such file %s", path)
		}
		return nil, fmt.Errorf("JWT load file: %v", err)
	}
	return stat, nil
}

func checkFilePerms(stat os.FileInfo) error {
	/*
	 * if a key owned by the user is accessed, then we check the
	 * permissions of the file. if the key owned by a different user,
	 * then we don't care.
	 */
	var fUID int
	sysStat, ok := stat.Sys().(*syscall.Stat_t)
	if !ok {
		return fmt.Errorf("JWT load: non-unix system is not supported")
	}
	fUID = int(sysStat.Uid)

	if fUID == os.Getuid() && stat.Mode() != 0600 {
		return fmt.Errorf("WARNING: UNPROTECTED JWT FILE! Permissions 0%3.3o are too open.", stat.Mode())
	}
	return nil
}

func checkTokenExpired(t *Token) error {
	if time.Now().After(t.ExpirationDate) {
		return fmt.Errorf("JWT is expired")
	}
	return nil
}
