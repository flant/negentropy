package vault

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/hashicorp/vault/api"

	"github.com/flant/negentropy/server-access/util"
)

type PosixUsersData struct {
	Data PosixUsers `json:"data"`
}

type PosixUsers struct {
	Users []PosixUser `json:"posix_users"`
}

type PosixUser struct {
	UID       int    `json:"uid"`
	Principal string `json:"principal"`

	Name     string `json:"name"`
	HomeDir  string `json:"home_directory"`
	Password string `json:"password"`
	Shell    string `json:"shell"`
	Gecos    string `json:"gecos"`
	Gid      int    `json:"gid"`
}

type ServerAccessSettings struct {
	TenantUUID  string `json:"tenant"`
	ProjectUUID string `json:"project"`
	ServerUUID  string `json:"server"`
}

func AssembleServerAccessSettings(settings ServerAccessSettings) ServerAccessSettings {
	return ServerAccessSettings{
		TenantUUID:  util.FirstNonEmptyString(settings.TenantUUID, os.Getenv("TENANT")),
		ProjectUUID: util.FirstNonEmptyString(settings.ProjectUUID, os.Getenv("PROJECT")),
		ServerUUID:  util.FirstNonEmptyString(settings.ServerUUID, os.Getenv("SERVER")),
	}
}

const FlantIAMMountpoint = "flant"

type FlantIAMAuth struct {
	c *api.Client
}

func NewFlantIAMAuth(client *api.Client) *FlantIAMAuth {
	return &FlantIAMAuth{
		c: client,
	}
}

func (c *FlantIAMAuth) PosixUsers(settings ServerAccessSettings) ([]PosixUser, error) {
	r := c.c.NewRequest("GET", fmt.Sprintf("/v1/auth/%s/tenant/%s/project/%s/server/%s/posix_users", FlantIAMMountpoint, settings.TenantUUID, settings.ProjectUUID, settings.ServerUUID))

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	resp, err := c.c.RawRequestWithContext(ctx, r)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return ParsePosixUsers(resp.Body)
}

func ParsePosixUsers(r io.Reader) ([]PosixUser, error) {
	// First read the data into a buffer. Not super efficient but we want to
	// know if we actually have a body or not.
	var buf bytes.Buffer
	_, err := buf.ReadFrom(r)
	if err != nil {
		return nil, err
	}
	if buf.Len() == 0 {
		return nil, nil
	}

	var users PosixUsersData
	err = json.Unmarshal(buf.Bytes(), &users)
	if err != nil {
		return nil, err
	}

	return users.Data.Users, nil
}
