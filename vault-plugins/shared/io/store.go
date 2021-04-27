package io

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	log "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-memdb"
	"github.com/segmentio/kafka-go"

	sharedkafka "github.com/flant/negentropy/vault-plugins/shared/kafka"
)

type MemoryStorableObject interface {
	ObjType() string
	ObjId() string
	Marshal(includeSensitive bool) ([]byte, error)
}

type KafkaSource interface {
	Restore(txn *memdb.Txn) error
	// Run(ms MemoryStore) // not implemented yet
	// Stop()
}

type KafkaDestination interface {
	ProcessObject(ms *MemoryStore, tnx *memdb.Txn, obj MemoryStorableObject) ([]kafka.Message, error)
	ProcessObjectDelete(ms *MemoryStore, tnx *memdb.Txn, obj MemoryStorableObject) ([]kafka.Message, error)
	ReplicaName() string // which replica it belongs to
}

type MemoryStore struct {
	*memdb.MemDB

	kafkaConnection *sharedkafka.MessageBroker

	kafkaMutex          sync.RWMutex
	kafkaSources        []KafkaSource
	replicaDestinations map[string]KafkaDestination
	kafkaDestinations   []KafkaDestination

	// vaultStorage      VaultStorage
	// downstreamApis    []DownstreamApi
	logger log.Logger
}

type MemoryStoreTxn struct {
	*memdb.Txn

	memstore *MemoryStore // crosslink
}

func (ms *MemoryStore) Txn(write bool) *MemoryStoreTxn {
	mTxn := ms.MemDB.Txn(write)
	if write {
		mTxn.TrackChanges()
	}
	return &MemoryStoreTxn{mTxn, ms}
}

func (mst *MemoryStoreTxn) commitWithBlackjack() error {
	changes := mst.Txn.Changes()

	kafkaMessages := make([]kafka.Message, 0)
	for _, change := range changes {
		mst.memstore.kafkaMutex.RLock()
		for _, dest := range mst.memstore.kafkaDestinations {
			var msgs []kafka.Message
			var err error
			if change.After == nil {
				object, ok := change.Before.(MemoryStorableObject)
				if !ok {
					mst.memstore.kafkaMutex.RUnlock()
					return fmt.Errorf("object does not implement MemoryStorableObject: %s", reflect.TypeOf(change.Before))
				}
				msgs, err = dest.ProcessObjectDelete(mst.memstore, mst.Txn, object)
			} else {
				object, ok := change.After.(MemoryStorableObject)
				if !ok {
					mst.memstore.kafkaMutex.RUnlock()
					return fmt.Errorf("object does not implement MemoryStorableObject: %s", reflect.TypeOf(change.Before))
				}
				msgs, err = dest.ProcessObject(mst.memstore, mst.Txn, object)
			}
			if err != nil {
				mst.memstore.kafkaMutex.RUnlock()
				return err
			}
			kafkaMessages = append(kafkaMessages, msgs...)
		}
		mst.memstore.kafkaMutex.RUnlock()
	}

	// TODO: проверка атомарности
	if mst.memstore.kafkaConnection.Configured() && len(kafkaMessages) > 0 {
		wr := mst.memstore.kafkaConnection.GetKafkaWriter()

		return wr.WriteMessages(context.Background(), kafkaMessages...)
	}

	return nil
}

func (mst *MemoryStoreTxn) Commit() error {
	// все посчитать!
	err := mst.commitWithBlackjack()
	if err != nil {
		mst.memstore.logger.Error("transaction aborted", err)
		mst.Txn.Abort()
		return err
	}

	mst.Txn.Commit()
	return nil
}

func (mst *MemoryStoreTxn) Abort() {
	mst.Txn.Abort()
}

func NewMemoryStore(schema *memdb.DBSchema, conn *sharedkafka.MessageBroker) (*MemoryStore, error) {
	db, err := memdb.NewMemDB(schema)
	if err != nil {
		return nil, err
	}

	return &MemoryStore{
		db,
		conn,
		sync.RWMutex{},
		make([]KafkaSource, 0),
		make(map[string]KafkaDestination),
		make([]KafkaDestination, 0),
		log.New(nil),
	}, nil
}

func (ms *MemoryStore) SetLogger(l log.Logger) {

}

func (ms *MemoryStore) AddKafkaSource(s KafkaSource) {
	ms.kafkaMutex.Lock()
	ms.kafkaSources = append(ms.kafkaSources, s)
	ms.kafkaMutex.Unlock()
}
func (ms *MemoryStore) AddKafkaDestination(s KafkaDestination) {
	ms.kafkaMutex.Lock()
	ms.replicaDestinations[s.ReplicaName()] = s
	ms.kafkaDestinations = append(ms.kafkaDestinations, s)
	ms.kafkaMutex.Unlock()
}

func (ms *MemoryStore) GetKafkaBroker() *sharedkafka.MessageBroker {
	return ms.kafkaConnection
}

func (ms *MemoryStore) RemoveKafkaDestination(replicaName string) {
	ms.kafkaMutex.Lock()
	ks, ok := ms.replicaDestinations[replicaName]
	if !ok {
		return
	}
	var index int
	for i, dest := range ms.kafkaDestinations {
		if dest == ks {
			index = i
			break
		}
	}

	ms.kafkaDestinations = append(ms.kafkaDestinations[:index], ms.kafkaDestinations[index+1:]...)
	ms.kafkaMutex.Unlock()
}

func (ms *MemoryStore) Restore() error {
	if !ms.kafkaConnection.Configured() {
		ms.logger.Warn("Kafka is not configured. Skipping restore")
		return nil
	}
	txn := ms.MemDB.Txn(true)
	ms.kafkaMutex.RLock()
	defer ms.kafkaMutex.RUnlock()
	for _, ks := range ms.kafkaSources {
		err := ks.Restore(txn)
		if err != nil {
			txn.Abort()
			return err
		}
	}
	txn.Commit()

	return nil
}
