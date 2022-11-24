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

	ext_model_ff "github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_flant_flow/model"
	ext_model "github.com/flant/negentropy/vault-plugins/flant_iam/extensions/ext_server_access/model"
	"github.com/flant/negentropy/vault-plugins/flant_iam/io/kafka_destination"
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
	iam_repo "github.com/flant/negentropy/vault-plugins/flant_iam/repo"
	backentutils "github.com/flant/negentropy/vault-plugins/shared/backent-utils"
	"github.com/flant/negentropy/vault-plugins/shared/io"
	"github.com/flant/negentropy/vault-plugins/shared/memdb"
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
		{ // create read delete
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
				"send_current_state_at_start": {
					Type:        framework.TypeBool,
					Description: "Send state of negentropy at start of replication",
				},
				"show_archived_in_current_state_at_start": {
					Type:        framework.TypeBool,
					Description: "Send deleted items at start of replication, only valuable if  send_current_state_at_start",
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
	if pb == nil {
		return backentutils.ResponseErrMessage(req, "can't decode given public_key", http.StatusBadRequest)
	}
	pk, err := x509.ParsePKCS1PublicKey(pb.Bytes)
	if err != nil {
		return backentutils.ResponseErrMessage(req, err.Error(), http.StatusBadRequest)
	}
	sendCurrentStateAtStart := data.Get("send_current_state_at_start").(bool)
	showArchivedInCurrentStateAtStart := data.Get("show_archived_in_current_state_at_start").(bool)

	r := &model.Replica{
		Name:                              replicaName,
		TopicType:                         topicType,
		PublicKey:                         pk,
		SendCurrentStateAtStart:           sendCurrentStateAtStart,
		ShowArchivedInCurrentStateAtStart: showArchivedInCurrentStateAtStart && sendCurrentStateAtStart,
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

	err = b.addReplicaToReplications(*r)
	if err != nil {
		b.Logger().Error("addReplicaToReplications", "error", err.Error())
		return backentutils.ResponseErr(req, err)
	}

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
			"replica_name":                replica.Name,
			"type":                        replica.TopicType,
			"public_key":                  strings.ReplaceAll(string(pemdata), "\n", "\\n"),
			"send_current_state_at_start": replica.SendCurrentStateAtStart,
			"show_archived_in_current_state_at_start": replica.ShowArchivedInCurrentStateAtStart,
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

func (b replicaBackend) addReplicaToReplications(replica model.Replica) error {
	var kafkaDestination io.KafkaDestination
	switch replica.TopicType {
	case kafka_destination.VaultTopicType:
		kafkaDestination = kafka_destination.NewVaultKafkaDestination(b.storage.GetKafkaBroker(), replica)
	case kafka_destination.MetadataTopicType:
		kafkaDestination = kafka_destination.NewMetadataKafkaDestination(b.storage.GetKafkaBroker(), replica)
	default:
		return fmt.Errorf("unknown replica type, replicaName: %s, topicType: %s", replica.Name, replica.TopicType)
	}
	if replica.SendCurrentStateAtStart {
		err := b.sendCurrentState(kafkaDestination, replica)
		if err != nil {
			return err
		}
	}
	b.storage.AddKafkaDestination(kafkaDestination)
	return nil
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

var typesToSend = []string{
	model.RoleType,
	model.FeatureFlagType,
	model.TenantType,
	model.ProjectType,
	model.UserType,
	model.GroupType,
	model.ServiceAccountType,
	model.ServiceAccountPasswordType,
	model.IdentitySharingType,
	model.MultipassType,
	model.RoleBindingType,
	model.RoleBindingApprovalType,
	// ext server_access
	ext_model.ServerType,
	// ext flant_flow
	ext_model_ff.TeamType,
	ext_model_ff.TeammateType,
	ext_model_ff.ServicePackType,
}

func (b replicaBackend) sendCurrentState(destination io.KafkaDestination, replica model.Replica) error {
	ms := b.storage
	txn := b.storage.Txn(false)
	mb := ms.GetKafkaBroker()
	for _, typeToSend := range typesToSend {
		b.Logger().Info(fmt.Sprintf("start sending %q objects to %s", typesToSend, replica.Name))
		counter := 0
		iter, err := txn.Get(typeToSend, iam_repo.PK)
		if err != nil {
			return fmt.Errorf(fmt.Sprintf("sendCurrentState: type %q: %s", typesToSend, err.Error()))
		}
		for {
			raw := iter.Next()
			if raw == nil {
				break
			}
			obj, isArchivable := raw.(memdb.Archivable)
			if isArchivable && !replica.ShowArchivedInCurrentStateAtStart && obj.Archived() {
				continue
			}
			storableObject, isStorable := raw.(io.MemoryStorableObject)
			if isStorable {
				msgs, err := destination.ProcessObject(ms, txn.Txn, storableObject)
				if err != nil {
					return fmt.Errorf(fmt.Sprintf("building kafka messages: type %q: %s", typesToSend, err.Error()))
				}
				err = mb.SendMessages(msgs, nil)
				if err != nil {
					return fmt.Errorf(fmt.Sprintf("sending kafka messages: type %q: %s", typesToSend, err.Error()))
				}
				counter++
			}
		}
		b.Logger().Info(fmt.Sprintf("end sending %q objects to %s, send %d", typesToSend, replica.Name, counter))
	}
	return nil
}
