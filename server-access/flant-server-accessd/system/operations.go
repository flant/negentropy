package system

import (
	"bufio"
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	copier "github.com/otiai10/copy"
)

type Interface interface {
	// Create home directory for new user
	CreateHomeDir(dir string, uid, gid int) error
	CreateAuthorizedKeysFile(homeDir string, principal string) error
	FixChown(homeDir string, uid, gid int) error
	// Recursively delete home directory if exists.
	DeleteHomeDir(dir string) error
	// Kill all user processes.
	PurgeUserLegacy(username string) error
}

type SystemOperator struct {
	dryRun bool
}

func NewSystemOperator() *SystemOperator {
	dryRun := os.Getenv("USER_DRY_RUN")
	return &SystemOperator{
		dryRun: dryRun == "yes",
	}
}

func (s *SystemOperator) CreateHomeDir(dir string, uid, gid int) error {
	if s.dryRun {
		fmt.Printf("Create dir '%s' for %d:%d\n", dir, uid, gid)
		return nil
	}
	_, err := os.Stat(dir)
	if os.IsNotExist(err) {
		err := copier.Copy("/etc/skel", dir)
		if err != nil {
			return err
		}

		err = os.Chown(dir, uid, gid)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	err = os.Chown(dir, uid, gid)
	if err != nil {
		return err
	}

	return nil
}

func (s *SystemOperator) DeleteHomeDir(dir string) error {
	if s.dryRun {
		fmt.Printf("Delete dir '%s'\n", dir)
		return nil
	}

	return os.RemoveAll(dir)
}

func (s *SystemOperator) PurgeUserLegacy(username string) error {
	if s.dryRun {
		fmt.Printf("Purge user legacy for '%s'\n", username)
		return nil
	}

	// we can't parse /var/run/utmp directly, since its serialization format is platform-dependent
	cmd := exec.Command("ps", "-o", "pid", "-u", username)

	out, err := cmd.Output()
	if err != nil {
		fmt.Println(err.Error())
		fmt.Println(string(out))
		return err
	}
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		line := scanner.Text()
		fmt.Println(line)
		fields := strings.Fields(line)
		if len(fields) != 1 {
			return fmt.Errorf(`"ps -o pid -u $USER" command output doesn't have 1 fields: %q`, line)
		}
		if fields[0] == "PID" {
			continue
		}
		pidToKill, err := strconv.Atoi(fields[0])
		if err != nil {
			return err
		}
		process, err := os.FindProcess(pidToKill)
		if err != nil {
			return nil
		}

		err = process.Kill()
		if err != nil {
			return nil
		}
	}

	return nil
}

func (s *SystemOperator) CreateAuthorizedKeysFile(homeDir string, principal string) error {
	content := GenerateAuthorizedKeysFile(principal, "")
	sshDir := filepath.Join(homeDir, ".ssh")
	filePath := filepath.Join(sshDir, "authorized_keys")
	err := os.MkdirAll(sshDir, 0o700)
	if err != nil {
		return fmt.Errorf("create .ssh dir: %v")
	}

	err = CreateAuthorizedKeysFile(filePath, content)
	if err != nil {
		return fmt.Errorf("create .ssh/authorized_keys file: %v")
	}

	err = os.Chmod(filePath, 0o600)
	if err != nil {
		return fmt.Errorf("change .ssh/authorized_keys mode: %v")
	}

	return nil
}

func (s *SystemOperator) FixChown(homeDir string, uid, gid int) error {
	err := os.Chown(homeDir, uid, gid)
	if err != nil {
		return err
	}

	err = filepath.Walk(homeDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			// prevent panic by handling failure accessing a path
			return err
		}

		return os.Chown(path, uid, gid)
	})

	return err
}
