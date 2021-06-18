package system

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	copier "github.com/otiai10/copy"
)

type Interface interface {
	CreateHomeDir(dir string, uid, gid int) error
	DeleteHomeDir(dir string) error
	BootUser(username string) error
}

type SystemOperator struct{}

func NewSystemOperator() *SystemOperator {
	return &SystemOperator{}
}

func (*SystemOperator) CreateHomeDir(dir string, uid, gid int) error {
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

func (*SystemOperator) DeleteHomeDir(dir string) error {
	return os.RemoveAll(dir)
}

func (*SystemOperator) BootUser(username string) error {
	// we can't parse /var/run/utmp directly, since its serialization format is platform-dependent
	cmd := exec.Command("who", "-u")
	out, err := cmd.Output()
	if err != nil {
		return err
	}

	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		line := scanner.Text()

		fields := strings.Fields(line)
		if len(fields) != 7 {
			return fmt.Errorf(`"who" command output doesn't have 7 fields: %q`, line)
		}

		pidToKill, err := strconv.Atoi(fields[6])
		if err != nil {
			return err
		}

		if fields[0] == username {
			process, err := os.FindProcess(pidToKill)
			if err != nil {
				return nil
			}

			err = process.Kill()
			if err != nil {
				return nil
			}
		}
	}

	return nil
}
