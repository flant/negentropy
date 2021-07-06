package backend

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"

	"github.com/flant/negentropy/vault-plugins/flant_iam/io/kafka_source"
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

type extensionBackend struct {
	logical.Backend
	storage *io.MemoryStore
}

func extensionPaths(b logical.Backend, storage *io.MemoryStore) []*framework.Path {
	bb := &extensionBackend{
		Backend: b,
		storage: storage,
	}
	return bb.paths()
}

func (b extensionBackend) paths() []*framework.Path {
	return []*framework.Path{
		{
			Pattern: "extension/?$",
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.ListOperation: &framework.PathOperation{
					Callback: b.handleExtensionList,
					Summary:  "Lists all flant_iam extension backends",
				},
			},
		},
		{
			Pattern: "extension/" + framework.GenericNameRegex("extension_name"),
			Fields: map[string]*framework.FieldSchema{
				"extension_name": {
					Type:        framework.TypeString,
					Description: "replication name",
				},
				"owned_types": {
					Type:          framework.TypeStringSlice,
					Description:   "types owned by extension",
					AllowedValues: []interface{}{"user", "group", "service_account", "role_binding", "multipass"},
				},
				"extended_types": {
					Type:          framework.TypeStringSlice,
					Description:   "types that could be extended by extension",
					AllowedValues: []interface{}{"user", "group", "service_account", "role_binding", "multipass"},
				},
				"allowed_roles": {
					Type:        framework.TypeStringSlice,
					Description: "allowed roles for extension bindings. Only if RoleBinding type is set",
				},
				"public_key": {
					Type:        framework.TypeString,
					Description: "Public rsa key for encryption",
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				// GET
				logical.ReadOperation: &framework.PathOperation{
					Callback: b.handleExtensionRead,
				},
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.handleExtensionCreate,
				},
				logical.CreateOperation: &framework.PathOperation{
					Callback: b.handleExtensionCreate,
				},
				logical.DeleteOperation: &framework.PathOperation{
					Callback: b.handleExtensionDelete,
				},
			},
		},
	}
}

func (b extensionBackend) handleExtensionList(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	tx := b.storage.Txn(false)

	var extensionNames []string
	iter, err := tx.Get(model.PluginExtensionType, model.PK)
	if err != nil {
		return nil, err
	}
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		t := raw.(*model.Replica)
		extensionNames = append(extensionNames, t.Name)
	}

	resp := &logical.Response{
		Data: map[string]interface{}{
			"extension_names": extensionNames,
		},
	}

	return resp, nil
}

func (b extensionBackend) handleExtensionCreate(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	extensionName := data.Get("extension_name").(string)
	if extensionName == "" {
		return nil, logical.CodedError(http.StatusBadRequest, "extension_name required")
	}

	publicKeyStr := data.Get("public_key").(string)
	if publicKeyStr == "" {
		return nil, logical.CodedError(http.StatusBadRequest, "public_key required")
	}

	publicKeyStr = strings.ReplaceAll(strings.TrimSpace(publicKeyStr), "\\n", "\n")

	pb, _ := pem.Decode([]byte(publicKeyStr))
	pubKey, err := x509.ParsePKCS1PublicKey(pb.Bytes)
	if err != nil {
		return nil, logical.CodedError(http.StatusBadRequest, err.Error())
	}

	ownTypes := data.Get("owned_types").([]string)
	extTypes := data.Get("extended_types").([]string)
	roles := data.Get("allowed_roles").([]string)

	extTypes = append(extTypes, ownTypes...)

	var hasRoleBinding bool
	for _, typ := range extTypes {
		if typ == model.RoleBindingType {
			hasRoleBinding = true
			break
		}
	}

	if hasRoleBinding && len(roles) == 0 {
		return nil, logical.CodedError(http.StatusBadRequest, "allowed_roles required")
	}

	ext := &model.PluginExtension{
		Name:          extensionName,
		PublicKey:     pubKey,
		OwnedTypes:    ownTypes,
		ExtendedTypes: extTypes,
		AllowedRoles:  roles,
	}

	encPubkey := b.storage.GetKafkaBroker().EncryptionPublicKey()

	if encPubkey == nil {
		return nil, logical.CodedError(http.StatusNotFound, "public key does not exist. Run /kafka/configure_access first")
	}

	tx := b.storage.Txn(true)
	err = tx.Insert(model.PluginExtensionType, ext)
	if err != nil {
		tx.Abort()
		return nil, logical.CodedError(http.StatusInternalServerError, err.Error())
	}

	// create topic for replica
	err = b.createTopicForExtension(ctx, ext.Name)
	if err != nil {
		return nil, logical.CodedError(http.StatusInternalServerError, err.Error())
	}
	err = tx.Commit()
	if err != nil {
		return nil, logical.CodedError(http.StatusInternalServerError, err.Error())
	}

	b.addExtensionToSources(ext)

	return &logical.Response{}, nil
}

func (b extensionBackend) handleExtensionRead(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	extensionName := data.Get("extension_name").(string)

	tx := b.storage.Txn(false)
	raw, err := tx.First(model.PluginExtensionType, model.PK, extensionName)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		rr := logical.ErrorResponse("extension not found")
		return logical.RespondWithStatusCode(rr, req, http.StatusNotFound)
	}

	// Respond
	ext := raw.(*model.PluginExtension)

	pemdata := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PUBLIC KEY",
			Bytes: x509.MarshalPKCS1PublicKey(ext.PublicKey),
		},
	)

	return &logical.Response{
		Data: map[string]interface{}{
			"replica_name":   ext.Name,
			"owned_types":    ext.OwnedTypes,
			"extended_types": ext.ExtendedTypes,
			"allowed_roles":  ext.AllowedRoles,
			"public_key":     strings.ReplaceAll(string(pemdata), "\n", "\\n"),
		},
	}, nil
}

func (b extensionBackend) handleExtensionDelete(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	extName := data.Get("extension_name").(string)
	if extName == "" {
		return nil, logical.CodedError(http.StatusBadRequest, "extension_name required")
	}

	tx := b.storage.Txn(true)

	raw, err := tx.First(model.PluginExtensionType, model.PK, extName)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		rr := logical.ErrorResponse("extension not found")
		return logical.RespondWithStatusCode(rr, req, http.StatusNotFound)
	}

	// Delete
	err = tx.Delete(model.PluginExtensionType, raw)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit changes: %v", err)
	}

	ext := raw.(*model.PluginExtension)
	b.removeExtensionFromSources(ext)

	return &logical.Response{}, err
}

func (b extensionBackend) addExtensionToSources(ext *model.PluginExtension) {
	b.storage.AddKafkaSource(kafka_source.NewExtensionKafkaSource(b.storage.GetKafkaBroker(), ext.Name, ext.PublicKey, ext.OwnedTypes, ext.ExtendedTypes, ext.AllowedRoles))
}

func (b extensionBackend) removeExtensionFromSources(ext *model.PluginExtension) {
	b.storage.RemoveKafkaSource(ext.Name)
}

func (b extensionBackend) createTopicForExtension(ctx context.Context, extName string) error {
	return b.storage.GetKafkaBroker().CreateTopic(ctx, "extension."+extName)
}
