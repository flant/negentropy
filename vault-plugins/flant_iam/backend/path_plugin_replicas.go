package backend

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/http"
	"strings"

	memdb2 "github.com/hashicorp/go-memdb"
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
	model.RoleType, // need to be special processed
	model.FeatureFlagType,
	model.TenantType,
	model.ProjectType,
	model.UserType,
	model.GroupType, // need to be special processed
	model.ServiceAccountType,
	model.ServiceAccountPasswordType,
	model.IdentitySharingType,
	model.MultipassType,
	model.RoleBindingType,
	model.RoleBindingApprovalType,
	// ext server_access
	ext_model.ServerType,
	// ext flant_flow
	ext_model_ff.TeamType, // need to be special processed
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
		dbIter, err := txn.Get(typeToSend, iam_repo.PK)
		if err != nil {
			return fmt.Errorf(fmt.Sprintf("sendCurrentState: type %q: %s", typesToSend, err.Error()))
		}
		iter := iteratorForType(typeToSend, dbIter)

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

// provide suitable iterator
func iteratorForType(typeToSend string, dbIter memdb2.ResultIterator) memdb2.ResultIterator {
	switch typeToSend {
	case model.RoleType:
		return &regularizedResultIterator{
			Source: dbIter,
			ObjectDescriptor: func(object interface{}) (objectID, []predecessorID) {
				role := object.(*model.Role)
				predecessors := make([]predecessorID, len(role.IncludedRoles))
				for i, ir := range role.IncludedRoles {
					predecessors[i] = ir.Name
				}
				return role.Name, predecessors
			},
		}
	case model.GroupType:
		return &regularizedResultIterator{
			Source: dbIter,
			ObjectDescriptor: func(object interface{}) (objectID, []predecessorID) {
				group := object.(*model.Group)
				return group.UUID, group.Groups
			},
		}
	case ext_model_ff.TeamType:
		return &regularizedResultIterator{
			Source: dbIter,
			ObjectDescriptor: func(object interface{}) (objectID, []predecessorID) {
				team := object.(*ext_model_ff.Team)
				var predecessors []predecessorID
				if team.ParentTeamUUID == "" {
					predecessors = nil
				} else {
					predecessors = []predecessorID{team.ParentTeamUUID}
				}
				return team.UUID, predecessors
			},
		}
	default:
		return dbIter
	}
}

type (
	objectID      = string
	predecessorID = string
)

// regularizedResultIterator provide valid sequence for items which has self-links
// items appear at next only if oll links are already showed previously
type regularizedResultIterator struct {
	// provide Source og objects to regularize
	Source memdb2.ResultIterator
	// ObjectDescriptor should provide object id, and ids of same type linked object
	ObjectDescriptor func(object interface{}) (objectID, []predecessorID)
	// internals
	processedObjects map[objectID]struct{}
	postponedObjects []interface{}
}

func (r *regularizedResultIterator) WatchCh() <-chan struct{} {
	return r.Source.WatchCh()
}

func (r *regularizedResultIterator) Next() interface{} {
	if r.processedObjects == nil {
		r.processedObjects = map[objectID]struct{}{}
	}
	for {
		raw := r.Source.Next()
		if raw == nil {
			break
		}
		objectID, predecessors := r.ObjectDescriptor(raw)
		if r.allPredecessorsAreSent(predecessors) {
			r.processedObjects[objectID] = struct{}{}
			return raw
		} else {
			r.postponedObjects = append(r.postponedObjects, raw)
		}
	}
	if len(r.postponedObjects) == 0 {
		return nil
	}
	for { // iterate over postponed objects
		obj := r.postponedObjects[0]
		objectID, predecessors := r.ObjectDescriptor(obj)
		if r.allPredecessorsAreSent(predecessors) {
			r.postponedObjects = r.postponedObjects[1:]
			r.processedObjects[objectID] = struct{}{}
			return obj
		} else {
			r.postponedObjects = append(r.postponedObjects, obj)
			r.postponedObjects = r.postponedObjects[1:]
		}
	}
}

func (r *regularizedResultIterator) allPredecessorsAreSent(predecessors []predecessorID) bool {
	for _, predecessor := range predecessors {
		if _, ok := r.processedObjects[predecessor]; !ok {
			return false
		}
	}
	return true
}
