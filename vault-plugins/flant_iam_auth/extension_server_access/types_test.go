package extension_server_access

import (
	"testing"

	"github.com/stretchr/testify/assert"

	iam "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/uuid"
)

func TestUserToPosix(t *testing.T) {
	tenant1 := uuid.New()
	tenant2 := uuid.New()

	user1 := &iam.User{
		UUID:           uuid.New(),
		TenantUUID:     tenant1,
		Identifier:     "vasya",
		FullIdentifier: "vasya@tenant1",
		Extensions: map[iam.ObjectOrigin]*iam.Extension{
			"server_access": {
				Origin: "server_access",
				Attributes: map[string]interface{}{
					"UID": 42,
					"passwords": []iam.UserServerPassword{
						{
							Seed: "1",
							Salt: "1",
						},
						{
							Seed: "2",
							Salt: "2",
						},
					},
				},
			},
		},
	}

	user2 := &iam.User{
		UUID:           uuid.New(),
		TenantUUID:     tenant2,
		Identifier:     "vasya",
		FullIdentifier: "vasya@tenant2",
		Extensions: map[iam.ObjectOrigin]*iam.Extension{
			"server_access": {
				Origin: "server_access",
				Attributes: map[string]interface{}{
					"UID": 56,
					"passwords": []iam.UserServerPassword{
						{
							Seed: "3",
							Salt: "3",
						},
						{
							Seed: "4",
							Salt: "4",
						},
					},
				},
			},
		},
	}

	serverID := "serverX"
	posix1, _ := userToPosix(serverID, tenant1, user1)
	assert.Equal(t, "vasya", posix1.Name)
	assert.Equal(t, 42, posix1.UID)
	assert.Equal(t, "/home/vasya", posix1.HomeDir)
	assert.Contains(t, posix1.Password, "$6$")

	posix2, _ := userToPosix(serverID, tenant1, user2)
	assert.Equal(t, "vasya@tenant2", posix2.Name)
	assert.Equal(t, 56, posix2.UID)
	assert.Equal(t, "/home/"+tenant2+"/vasya", posix2.HomeDir)
	assert.Contains(t, posix2.Password, "$6$")
}
