package backend

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"strings"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"

	"github.com/flant/negentropy/vault-plugins/flant_iam/io/kafka_destination"
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	iam_repo "github.com/flant/negentropy/vault-plugins/flant_iam/repo"
	backentutils "github.com/flant/negentropy/vault-plugins/shared/backent-utils"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

type replicaBackend struct {
	logical.Backend
	storage *io.MemoryStore
}

func replicasPaths(b logical.Backend, storage *io.MemoryStore) []*framework.Path {
	bb := &replicaBackend{
		Backend: b,
		storage: storage,
	}
	return bb.paths()
}

func (b replicaBackend) paths() []*framework.Path {
	return []*framework.Path{
		{
			Pattern: "replica/?",
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.ReadOperation: &framework.PathOperation{
					Callback: b.handleReplicaList,
					Summary:  "Lists all flant_iam replication backends",
				},
			},
		},
		{
			Pattern: "replica/" + framework.GenericNameRegex("replica_name"),
			Fields: map[string]*framework.FieldSchema{
				"replica_name": {
					Type:        framework.TypeNameString,
					Description: "replication name",
				},
				"type": {
					Type:          framework.TypeNameString,
					Description:   "replication type",
					AllowedValues: []interface{}{"Vault", "Metadata"},
				},
				"public_key": {
					Type:        framework.TypeString,
					Description: "Public rsa key for encryption",
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				// GET
				logical.ReadOperation: &framework.PathOperation{
					Callback: b.handleReplicaRead,
				},
				logical.UpdateOperation: &framework.PathOperation{
					Callback: b.handleReplicaCreate,
				},
				logical.CreateOperation: &framework.PathOperation{
					Callback: b.handleReplicaCreate,
				},
				logical.DeleteOperation: &framework.PathOperation{
					Callback: b.handleReplicaDelete,
				},
			},
		},
	}
}

func (b replicaBackend) handleReplicaList(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	tx := b.storage.Txn(false)

	var replicaNames []string
	iter, err := tx.Get(model.ReplicaType, iam_repo.PK)
	if err != nil {
		return backentutils.ResponseErrMessage(req, err.Error(), http.StatusInternalServerError)
	}
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		t := raw.(*model.Replica)
		replicaNames = append(replicaNames, t.Name)
	}

	resp := &logical.Response{
		Data: map[string]interface{}{
			"replica_names": replicaNames,
		},
	}

	return resp, nil
}

func (b replicaBackend) handleReplicaCreate(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	replicaName := data.Get("replica_name").(string)
	topicType := data.Get("type").(string)
	if topicType == "" {
		return nil, logical.CodedError(http.StatusBadRequest, "topic_type required")
	}

	publicKeyStr := data.Get("public_key").(string)
	if publicKeyStr == "" {
		return nil, logical.CodedError(http.StatusBadRequest, "public_key required")
	}

	publicKeyStr = strings.ReplaceAll(strings.TrimSpace(publicKeyStr), "\\n", "\n")

	pb, _ := pem.Decode([]byte(publicKeyStr))
	pk, err := x509.ParsePKCS1PublicKey(pb.Bytes)
	if err != nil {
		return backentutils.ResponseErrMessage(req, err.Error(), http.StatusBadRequest)
	}

	r := &model.Replica{
		Name:      replicaName,
		TopicType: topicType,
		PublicKey: pk,
	}

	tx := b.storage.Txn(true)
	defer tx.Abort()
	err = tx.Insert(model.ReplicaType, r)
	if err != nil {
		return backentutils.ResponseErrMessage(req, err.Error(), http.StatusInternalServerError)
	}

	// create topic for replica
	err = b.createTopicForReplica(ctx, replicaName)
	if err != nil {
		return backentutils.ResponseErrMessage(req, err.Error(), http.StatusInternalServerError)
	}
	err = tx.Commit()
	if err != nil {
		return backentutils.ResponseErrMessage(req, err.Error(), http.StatusInternalServerError)
	}

	b.addReplicaToReplications(*r)

	return &logical.Response{}, nil
}

func (b replicaBackend) handleReplicaRead(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	replicaName := data.Get("replica_name").(string)

	tx := b.storage.Txn(false)
	raw, err := tx.First(model.ReplicaType, iam_repo.PK, replicaName)
	if err != nil {
		return backentutils.ResponseErrMessage(req, err.Error(), http.StatusInternalServerError)
	}
	if raw == nil {
		return backentutils.ResponseErrMessage(req, "replica not found", http.StatusNotFound)
	}

	// Respond
	replica := raw.(*model.Replica)

	pemdata := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PUBLIC KEY",
			Bytes: x509.MarshalPKCS1PublicKey(replica.PublicKey),
		},
	)

	return &logical.Response{
		Data: map[string]interface{}{
			"replica_name": replica.Name,
			"type":         replica.TopicType,
			"public_key":   strings.ReplaceAll(string(pemdata), "\n", "\\n"),
		},
	}, nil
}

func (b replicaBackend) handleReplicaDelete(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	replicaName := data.Get("replica_name").(string)

	tx := b.storage.Txn(true)
	defer tx.Abort()

	// Verify existence

	raw, err := tx.First(model.ReplicaType, iam_repo.PK, replicaName)
	if err != nil {
		return backentutils.ResponseErrMessage(req, err.Error(), http.StatusInternalServerError)
	}
	if raw == nil {
		return backentutils.ResponseErrMessage(req, "replica not found", http.StatusNotFound)
	}

	// Delete
	err = tx.Delete(model.ReplicaType, raw)
	if err != nil {
		return backentutils.ResponseErrMessage(req, err.Error(), http.StatusInternalServerError)
	}
	if err := tx.Commit(); err != nil {
		return backentutils.ResponseErrMessage(req, err.Error(), http.StatusInternalServerError)
	}

	replica := raw.(*model.Replica)
	err = b.deleteTopicForReplica(ctx, replica.Name)
	if err != nil {
		return backentutils.ResponseErrMessage(req, err.Error(), http.StatusInternalServerError)
	}

	b.removeReplicaFromReplications(*replica)

	return &logical.Response{}, err
}

func (b replicaBackend) addReplicaToReplications(replica model.Replica) {
	switch replica.TopicType {
	case kafka_destination.VaultTopicType:
		b.storage.AddKafkaDestination(kafka_destination.NewVaultKafkaDestination(b.storage.GetKafkaBroker(), replica))
	case kafka_destination.MetadataTopicType:
		b.storage.AddKafkaDestination(kafka_destination.NewMetadataKafkaDestination(b.storage.GetKafkaBroker(), replica))
	default:
		b.Logger().Debug("unknown replica type", "replicaName", replica.Name, "topicType", replica.TopicType)
	}
}

func (b replicaBackend) removeReplicaFromReplications(replica model.Replica) {
	b.storage.RemoveKafkaDestination(replica.Name)
}

func (b replicaBackend) createTopicForReplica(ctx context.Context, replicaName string) error {
	return b.storage.GetKafkaBroker().CreateTopic(ctx, "root_source."+replicaName, nil)
}

func (b replicaBackend) deleteTopicForReplica(ctx context.Context, replicaName string) error {
	return b.storage.GetKafkaBroker().DeleteTopic(ctx, "root_source."+replicaName)
}
