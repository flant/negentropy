package extension_server_access

import (
	"encoding/json"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/assert"

	"github.com/flant/negentropy/vault-plugins/flant_iam/extensions/extension_server_access/model"
	iam_model "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	iam_repo "github.com/flant/negentropy/vault-plugins/flant_iam/repo"
	"github.com/flant/negentropy/vault-plugins/flant_iam/uuid"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

func TestUserToPosix(t *testing.T) {
	tenant1 := uuid.New()
	tenant2ID := uuid.New()

	attrUser1 := map[string]interface{}{
		"UID": 42,
		"passwords": []model.UserServerPassword{
			{
				Seed: []byte("1"),
				Salt: []byte("1"),
			},
			{
				Seed: []byte("2"),
				Salt: []byte("2"),
			},
		},
	}
	attrs1, err := marshallUnmarshal(attrUser1)
	assert.NoError(t, err)

	user1 := &iam_model.User{
		UUID:           uuid.New(),
		TenantUUID:     tenant1,
		Identifier:     "vasya",
		FullIdentifier: "vasya@tenant1",
		Extensions: map[iam_model.ObjectOrigin]*iam_model.Extension{
			iam_model.OriginServerAccess: {
				Origin:     iam_model.OriginServerAccess,
				Attributes: attrs1,
			},
		},
	}

	attrUser2 := map[string]interface{}{
		"UID": 56,
		"passwords": []model.UserServerPassword{
			{
				Seed: []byte("3"),
				Salt: []byte("3"),
			},
			{
				Seed: []byte("4"),
				Salt: []byte("4"),
			},
		},
	}
	attrs2, err := marshallUnmarshal(attrUser2)
	assert.NoError(t, err)

	user2 := &iam_model.User{
		UUID:           uuid.New(),
		TenantUUID:     tenant2ID,
		Identifier:     "vasya",
		FullIdentifier: "vasya@tenant2",
		Extensions: map[iam_model.ObjectOrigin]*iam_model.Extension{
			iam_model.OriginServerAccess: {
				Origin:     iam_model.OriginServerAccess,
				Attributes: attrs2,
			},
		},
	}
	tenant2 := &iam_model.Tenant{
		UUID:       tenant2ID,
		Version:    uuid.New(),
		Identifier: "tenant2",
	}

	st, _ := io.NewMemoryStore(iam_repo.TenantSchema(), nil, hclog.NewNullLogger())
	tx := st.Txn(true)
	_ = tx.Insert(iam_model.TenantType, tenant2)
	_ = tx.Commit()

	serverID := "serverX"
	builder := newPosixUserBuilder(st.Txn(false), serverID, tenant1)

	posix1, _ := builder.userToPosix(user1)
	assert.Equal(t, "vasya", posix1.Name)
	assert.Equal(t, 42, posix1.UID)
	assert.Equal(t, "/home/vasya", posix1.HomeDir)
	assert.Contains(t, posix1.Password, "$6$")

	posix2, _ := builder.userToPosix(user2)
	assert.Equal(t, "vasya@tenant2", posix2.Name)
	assert.Equal(t, 56, posix2.UID)
	assert.Equal(t, "/home/tenant2/vasya", posix2.HomeDir)
	assert.Contains(t, posix2.Password, "$6$")
}

// emulates pipeline flant_iam -> kafka -> flant_iam_auth
func marshallUnmarshal(in map[string]interface{}) (map[string]interface{}, error) {
	tmp, err := json.Marshal(in)
	if err != nil {
		return nil, err
	}
	var out map[string]interface{}
	err = json.Unmarshal(tmp, &out)
	if err != nil {
		return nil, err
	}
	return out, nil
}
