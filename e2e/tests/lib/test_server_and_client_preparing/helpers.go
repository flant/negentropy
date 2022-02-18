package test_server_and_client_preparing

import (
	"fmt"
	"path/filepath"
	"regexp"
	"time"
)

func Try(limitSeconds int, action func() error) error {
	var err error
	var i int
	for i = 0; i < limitSeconds*4; i++ {
		err = action()
		if err == nil {
			return nil
		}
		time.Sleep(time.Millisecond * 250)
	}
	if err != nil {
		return fmt.Errorf("spend %f seconds for retries, got:%w", float32(i)/4, err)
	}
	return nil
}

func RunAndCheckServerAccessd(s Suite, posixUserName string, testServerUUID string, userUUID string) error {
	// TODO check content /etc/nsswitch.conf
	path := "/opt/serveraccessd"
	err := s.CreateIfNotExistsDirectoryAtContainer(s.TestServerContainer, path)
	if err != nil {
		return fmt.Errorf("folder:%s should be created, but got error: %w", path, err)
	}
	s.KillAllInstancesOfProcessAtContainer(s.TestServerContainer, s.ServerAccessdPath)
	s.RunDaemonAtContainer(s.TestServerContainer, s.ServerAccessdPath, "server_accessd.log")
	err = Try(10, func() error {
		pidServerAccessd := s.FirstProcessPIDAtContainer(s.TestServerContainer, s.ServerAccessdPath)
		if pidServerAccessd == 0 {
			return fmt.Errorf("cant find running process of %s at container %s", s.ServerAccessdPath, s.TestServerContainer.Names[0])
		}
		return nil
	})
	if err != nil {
		return err
	}
	authKeysFilePath := filepath.Join("/home", posixUserName, ".ssh", "authorized_keys")
	err = Try(10, func() error {
		contentAuthKeysFile := s.ExecuteCommandAtContainer(s.TestServerContainer,
			[]string{"/bin/bash", "-c", "cat " + authKeysFilePath}, nil)
		if len(contentAuthKeysFile) != 1 {
			return fmt.Errorf("file %s should have 1 line, but, actually got:%s", authKeysFilePath, contentAuthKeysFile)
		}
		principal := calculatePrincipal(testServerUUID, userUUID)
		matched, err := regexp.Match(`cert-authority,principals="`+principal+`" ssh-rsa.{373}$`, []byte(contentAuthKeysFile[0]))
		if err != nil {
			return err
		}
		if !matched {
			return fmt.Errorf("file %s has wrong content: %s", authKeysFilePath, contentAuthKeysFile[0])
		}
		return nil
	})
	return err
}
