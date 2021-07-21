package vault

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_ParsePosixUsers(t *testing.T) {
	posixUsersData := []byte(`
{"data":{
 "posix_users": [
   {
	 "uid": 42,
	 "principal": "abc12beade1122",
	 "name": "testuser",
	 "home_directory": "/home/testuser",
	 "password": "$6$AAAABBBBBBBCCCCCCCCCC", 
	 "shell": "/bin/bash",
	 "gecos": "",
	 "gid": 999
   }
 ]
}}
`)

	users, err := ParsePosixUsers(bytes.NewReader(posixUsersData))

	require.NoError(t, err)
	require.NotNil(t, users)

	require.Len(t, users, 1)
	require.Equal(t, "testuser", users[0].Name)
}
