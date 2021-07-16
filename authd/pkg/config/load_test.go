package config

import (
	"fmt"
	"testing"
)

func Test_LoadConfigFiles(t *testing.T) {
	paths, err := RecursiveFindConfFiles("testdata/conf-dir")

	if err != nil {
		t.Fatalf("find files in testdata/conf-dir: %v", err)
	}

	authConfig, authSocketConfig, err := LoadConfigFiles(paths)
	if err != nil {
		t.Fatalf("load files from testdata/conf-dir: %v", err)
	}

	if authConfig == nil {
		t.Fatalf("authConfig should not be nil")
	}
	if authSocketConfig == nil {
		t.Fatalf("authSocketConfig should not be nil")
	}
	if len(authSocketConfig) != 3 {
		t.Fatalf("should be 3 authSocketConfig")
	}

	socketConfigs := make(map[string]*AuthdSocketConfig)
	for _, sock := range authSocketConfig {
		socketConfigs[sock.GetPath()] = sock
	}

	mode := socketConfigs["/var/run/my.sock"].GetMode()
	if mode != 0600 {
		t.Fatalf("mode should be 0600. Got %04o", mode)
	}

	mode = socketConfigs["/var/run/sock2.sock"].GetMode()
	if mode != 0754 {
		t.Fatalf("mode should be 0754. Got %04o", mode)
	}
}

func Test_DetectServer(t *testing.T) {
	paths, err := RecursiveFindConfFiles("testdata/conf-dir")

	if err != nil {
		t.Fatalf("find files in testdata/conf-dir: %v", err)
	}

	authConfig, _, err := LoadConfigFiles(paths)
	if err != nil {
		t.Fatalf("load files from testdata/conf-dir: %v", err)
	}

	if authConfig == nil {
		t.Fatalf("authConfig should not be nil")
	}

	s, e := DetectServerAddr(authConfig.GetServers(), "auth", "")
	if e != nil {
		t.Fatalf("authConfig should contain default domain: %v", err)
	}
	if s == "" {
		t.Fatalf("authConfig should contain non-empty default domain for 'auth' type")
	}
	fmt.Printf("default server: '%s'\n", s)
}
