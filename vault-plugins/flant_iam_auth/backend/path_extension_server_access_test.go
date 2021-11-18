package backend

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"path"
	"testing"

	"github.com/hashicorp/vault/sdk/logical"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/square/go-jose.v2"

	ext_model "github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_server_access/model"
	iam_model "github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/uuid"
)

func Test_ExtensionServer_PosixUsers(t *testing.T) {
	b, storage := getBackend(t)
	err := storage.Put(context.TODO(), &logical.StorageEntry{
		Key:      "iam_auth.extensions.server_access.ssh_role",
		Value:    []byte("ssh"),
		SealWrap: true,
	})
	require.NoError(t, err)

	tx := b.storage.Txn(true)
	defer tx.Abort()

	tenant := uuid.New()

	err = createUserAndSa(tx, tenant)
	require.NoError(t, err)
	_ = tx.Commit()

	req := &logical.Request{
		Operation: logical.ReadOperation,
		Path:      path.Join("tenant", tenant, "project", uuid.New(), "server", uuid.New(), "posix_users"),
		Storage:   storage,
	}

	resp, err := b.HandleRequest(context.Background(), req)
	require.NoError(t, err)

	type response struct {
		Data struct {
			PosixUsers []interface{} `json:"posix_users"`
		} `json:"data"`
	}

	var respData response
	err = json.Unmarshal([]byte(resp.Data["http_raw_body"].(string)), &respData)
	require.NoError(t, err)
	assert.Len(t, respData.Data.PosixUsers, 2)
}

func Test_ExtensionServer_QueryServers(t *testing.T) {
	b, storage := getBackend(t)
	err := storage.Put(context.TODO(), &logical.StorageEntry{
		Key:      "iam_auth.extensions.server_access.ssh_role",
		Value:    []byte("ssh"),
		SealWrap: true,
	})
	require.NoError(t, err)

	tx := b.storage.Txn(true)
	defer tx.Abort()

	tenant := uuid.New()
	project := uuid.New()

	serverUUIDs, err := createServers(tx, tenant, project)
	require.NoError(t, err)
	_ = tx.Commit()

	type response struct {
		Warnings []string `json:"warnings"`
		Data     struct {
			Servers []ext_model.Server `json:"servers"`
		} `json:"data"`
	}

	t.Run("tenant and project are set", func(t *testing.T) {
		t.Run("by name", func(t *testing.T) {
			req := &logical.Request{
				Operation: logical.ReadOperation,
				Path:      path.Join("tenant", tenant, "project", project, "query_server"),
				Storage:   storage,
				Data:      map[string]interface{}{"names": "db-1"},
			}

			resp, err := b.HandleRequest(context.Background(), req)
			require.NoError(t, err)

			var respData response
			err = json.Unmarshal([]byte(resp.Data["http_raw_body"].(string)), &respData)
			require.NoError(t, err)
			require.Len(t, respData.Data.Servers, 1)
			assert.Equal(t, "db-1", respData.Data.Servers[0].Identifier)
			assert.Equal(t, serverUUIDs[0], respData.Data.Servers[0].UUID)
		})

		t.Run("by name with warnings", func(t *testing.T) {
			req := &logical.Request{
				Operation: logical.ReadOperation,
				Path:      path.Join("tenant", tenant, "project", project, "query_server"),
				Storage:   storage,
				Data:      map[string]interface{}{"names": "db-1,db-3"},
			}

			resp, err := b.HandleRequest(context.Background(), req)
			require.NoError(t, err)

			var respData response
			err = json.Unmarshal([]byte(resp.Data["http_raw_body"].(string)), &respData)
			require.NoError(t, err)
			require.Len(t, respData.Data.Servers, 1)
			assert.Equal(t, "db-1", respData.Data.Servers[0].Identifier)
			assert.Equal(t, serverUUIDs[0], respData.Data.Servers[0].UUID)

			require.Len(t, respData.Warnings, 1)
			assert.Equal(t, respData.Warnings[0], `Server: "db-3" not found`)
		})

		t.Run("by labels", func(t *testing.T) {
			req := &logical.Request{
				Operation: logical.ReadOperation,
				Path:      path.Join("tenant", tenant, "project", project, "query_server"),
				Storage:   storage,
				Data:      map[string]interface{}{"labelSelector": "foo=bar"},
			}

			resp, err := b.HandleRequest(context.Background(), req)
			require.NoError(t, err)

			var respData response
			err = json.Unmarshal([]byte(resp.Data["http_raw_body"].(string)), &respData)
			require.NoError(t, err)
			require.Len(t, respData.Data.Servers, 1)
			assert.Equal(t, "db-2", respData.Data.Servers[0].Identifier)
			assert.Equal(t, serverUUIDs[1], respData.Data.Servers[0].UUID)
		})

		t.Run("by IN labels", func(t *testing.T) {
			req := &logical.Request{
				Operation: logical.ReadOperation,
				Path:      path.Join("tenant", tenant, "project", project, "query_server"),
				Storage:   storage,
				Data:      map[string]interface{}{"labelSelector": "foo in (bar)"},
			}

			resp, err := b.HandleRequest(context.Background(), req)
			require.NoError(t, err)

			var respData response
			err = json.Unmarshal([]byte(resp.Data["http_raw_body"].(string)), &respData)
			require.NoError(t, err)
			require.Len(t, respData.Data.Servers, 1)
			assert.Equal(t, "db-2", respData.Data.Servers[0].Identifier)
			assert.Equal(t, serverUUIDs[1], respData.Data.Servers[0].UUID)
		})

		t.Run("names and labelSelector at once are forbidden", func(t *testing.T) {
			req := &logical.Request{
				Operation: logical.ReadOperation,
				Path:      path.Join("tenant", tenant, "project", project, "query_server"),
				Storage:   storage,
				Data:      map[string]interface{}{"labelSelector": "foo in (bar)", "names": "db-1"},
			}

			_, err := b.HandleRequest(context.Background(), req)
			require.Error(t, err)
			assert.EqualError(t, err, "only names or labelSelector must be set")
		})
	})

	t.Run("only tenant pass is set", func(t *testing.T) {
		t.Run("by name is not working here(return all servers)", func(t *testing.T) {
			req := &logical.Request{
				Operation: logical.ReadOperation,
				Path:      path.Join("tenant", tenant, "query_server"),
				Storage:   storage,
				Data:      map[string]interface{}{"names": "db-1"},
			}

			resp, err := b.HandleRequest(context.Background(), req)
			require.NoError(t, err)

			var respData response
			err = json.Unmarshal([]byte(resp.Data["http_raw_body"].(string)), &respData)
			require.NoError(t, err)
			require.Len(t, respData.Data.Servers, 2)
		})

		t.Run("by labels", func(t *testing.T) {
			req := &logical.Request{
				Operation: logical.ReadOperation,
				Path:      path.Join("tenant", tenant, "query_server"),
				Storage:   storage,
				Data:      map[string]interface{}{"labelSelector": "foo=bar"},
			}

			resp, err := b.HandleRequest(context.Background(), req)
			require.NoError(t, err)

			var respData response
			err = json.Unmarshal([]byte(resp.Data["http_raw_body"].(string)), &respData)
			require.NoError(t, err)
			require.Len(t, respData.Data.Servers, 1)
			assert.Equal(t, "", respData.Data.Servers[0].Identifier) // no unsafe data by unsafe path
			assert.Equal(t, serverUUIDs[1], respData.Data.Servers[0].UUID)
		})
	})

	t.Run("no tenant is set", func(t *testing.T) {
		t.Run("by name is not working here(return all servers)", func(t *testing.T) {
			req := &logical.Request{
				Operation: logical.ReadOperation,
				Path:      path.Join("query_server"),
				Storage:   storage,
				Data:      map[string]interface{}{"names": "db-1"},
			}

			resp, err := b.HandleRequest(context.Background(), req)
			require.NoError(t, err)

			var respData response
			err = json.Unmarshal([]byte(resp.Data["http_raw_body"].(string)), &respData)
			require.NoError(t, err)
			require.Len(t, respData.Data.Servers, 2)
		})

		t.Run("by labels", func(t *testing.T) {
			req := &logical.Request{
				Operation: logical.ReadOperation,
				Path:      path.Join("query_server"),
				Storage:   storage,
				Data:      map[string]interface{}{"labelSelector": "foo=bar"},
			}

			resp, err := b.HandleRequest(context.Background(), req)
			require.NoError(t, err)

			var respData response
			err = json.Unmarshal([]byte(resp.Data["http_raw_body"].(string)), &respData)
			require.NoError(t, err)
			require.Len(t, respData.Data.Servers, 1)
			assert.Equal(t, "", respData.Data.Servers[0].Identifier)
			assert.Equal(t, "", respData.Data.Servers[0].Identifier)
		})
	})
}

func Test_ExtensionServer_JWT(t *testing.T) {
	t.Skip("Need to riwrete to memdb")
	b, storage := getBackend(t)
	err := storage.Put(context.TODO(), &logical.StorageEntry{
		Key:      "iam_auth.extensions.server_access.ssh_role",
		Value:    []byte("ssh"),
		SealWrap: true,
	})
	require.NoError(t, err)

	issuer := map[string]interface{}{
		"issuer": "test",
	}
	data, _ := json.Marshal(issuer)

	err = storage.Put(context.TODO(), &logical.StorageEntry{
		Key:   "jwt/configuration",
		Value: data,
	})
	require.NoError(t, err)

	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}
	jwk := jose.JSONWebKey{
		Key: privateKey,
	}

	keysSet := jose.JSONWebKeySet{
		Keys: []jose.JSONWebKey{
			jwk,
		},
	}
	data, _ = json.Marshal(keysSet)

	err = storage.Put(context.TODO(), &logical.StorageEntry{
		Key:   "jwt/private_keys",
		Value: data,
	})
	require.NoError(t, err)

	tx := b.storage.Txn(true)
	defer tx.Abort()

	tenant := uuid.New()
	project := uuid.New()
	serverID := uuid.New()

	_, err = createServers(tx, tenant, project, serverID)
	require.NoError(t, err)
	_ = tx.Commit()

	req := &logical.Request{
		Operation: logical.ReadOperation,
		Path:      path.Join("tenant", tenant, "project", project, "server", serverID),
		Storage:   storage,
	}

	resp, err := b.HandleRequest(context.Background(), req)
	require.NoError(t, err)

	type response struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}

	var respData response
	err = json.Unmarshal([]byte(resp.Data["http_raw_body"].(string)), &respData)
	require.NoError(t, err)
	assert.NotEmpty(t, respData.Data.Token)
}

// returns servers uuids
func createServers(tx *io.MemoryStoreTxn, tenantID, projectID string, serverID ...string) ([]string, error) {
	predefinedID := uuid.New()
	if len(serverID) > 0 {
		predefinedID = serverID[0]
	}
	serverDB1 := &ext_model.Server{
		UUID:        predefinedID,
		TenantUUID:  tenantID,
		ProjectUUID: projectID,
		Version:     uuid.New(),
		Identifier:  "db-1",
		Fingerprint: "F1",
		Labels:      nil,
		Annotations: nil,
	}

	serverWithLabels := &ext_model.Server{
		UUID:        uuid.New(),
		TenantUUID:  tenantID,
		ProjectUUID: projectID,
		Version:     uuid.New(),
		Identifier:  "db-2",
		Fingerprint: "F2",
		Labels: map[string]string{
			"foo": "bar",
		},
		Annotations: nil,
	}

	err := tx.Insert(ext_model.ServerType, serverDB1)
	if err != nil {
		return nil, err
	}

	err = tx.Insert(ext_model.ServerType, serverWithLabels)
	if err != nil {
		return nil, err
	}

	tenant := iam_model.Tenant{
		UUID:       tenantID,
		Version:    "v1",
		Identifier: "i1",
	}

	err = tx.Insert(iam_model.TenantType, &tenant)
	if err != nil {
		return nil, err
	}

	project := iam_model.Project{
		UUID:       projectID,
		TenantUUID: tenantID,
		Version:    "v1",
		Identifier: "i2",
	}

	err = tx.Insert(iam_model.ProjectType, &project)
	if err != nil {
		return nil, err
	}

	return []string{serverDB1.UUID, serverWithLabels.UUID}, nil
}

func createUserAndSa(tx *io.MemoryStoreTxn, tenant string) error {
	userAttr := map[string]interface{}{
		"UID": 42,
		"passwords": []ext_model.UserServerPassword{
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
	attrs, err := marshallUnmarshal(userAttr)
	if err != nil {
		return err
	}

	user := &iam_model.User{
		UUID:           uuid.New(),
		TenantUUID:     tenant,
		Identifier:     "vasya",
		FullIdentifier: "vasya@tenant1",
		Version:        uuid.New(),
		Extensions: map[iam_model.ObjectOrigin]*iam_model.Extension{
			iam_model.OriginServerAccess: {
				Origin:     iam_model.OriginServerAccess,
				Attributes: attrs,
			},
		},
	}

	saAttr := map[string]interface{}{
		"UID": 42,
		"passwords": []ext_model.UserServerPassword{
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
	attrs, err = marshallUnmarshal(saAttr)
	if err != nil {
		return err
	}
	sa := &iam_model.ServiceAccount{
		UUID:           uuid.New(),
		TenantUUID:     tenant,
		Identifier:     "serviceacc",
		FullIdentifier: "serviceacc@tenant1",
		Version:        uuid.New(),
		Extensions: map[iam_model.ObjectOrigin]*iam_model.Extension{
			iam_model.OriginServerAccess: {
				Origin:     iam_model.OriginServerAccess,
				Attributes: attrs,
			},
		},
	}

	err = tx.Insert(iam_model.UserType, user)
	if err != nil {
		return err
	}

	err = tx.Insert(iam_model.ServiceAccountType, sa)

	return err
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
