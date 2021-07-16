package util

import (
	"fmt"
	"os"
	"os/user"
	"strconv"
)

// FileExists returns true if path exists
func FileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func DirExists(path string) (bool, error) {
	fileInfo, err := os.Stat(path)
	if err != nil && os.IsNotExist(err) {
		return false, nil
	}

	return fileInfo.IsDir(), nil
}

func IsFileExecutable(f os.FileInfo) bool {
	return f.Mode()&0111 != 0
}

func ChangeFilePermissions(fileName string, userName string, groupName string, mode int) error {
	var err error
	var uid int64 = -1
	var gid int64 = -1

	if userName != "" {
		u, err := user.Lookup(userName)
		if err != nil {
			return fmt.Errorf("lookup user id for '%s': %v", userName, err)
		}
		if uid, err = strconv.ParseInt(u.Uid, 10, 32); err != nil {
			return fmt.Errorf("'%s' uid is invalid: %v", u.Uid, err)
		}
	}

	if groupName != "" {
		g, err := user.LookupGroup(groupName)
		if err != nil {
			return fmt.Errorf("lookup user id for '%s': %v", userName, err)
		}
		if gid, err = strconv.ParseInt(g.Gid, 10, 32); err != nil {
			return fmt.Errorf("'%s' uid is invalid: %v", g.Gid, err)
		}
	}

	err = os.Chown(fileName, int(uid), int(gid))
	if err != nil {
		return fmt.Errorf("change owner: %v", err)
	}

	err = os.Chmod(fileName, os.FileMode(mode))
	if err != nil {
		return fmt.Errorf("change mode: %v", err)
	}

	return nil
}
