package io

import (
	"fmt"
	"reflect"
	"sync"

	log "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-memdb"

	"github.com/flant/negentropy/vault-plugins/shared/kafka"
)

type MemoryStorableObject interface {
	ObjType() string
	ObjId() string
}

type KafkaSource interface {
	Name() string
	Restore(txn *memdb.Txn) error
	Run(ms *MemoryStore)
	Stop()
}

type KafkaDestination interface {
	ProcessObject(ms *MemoryStore, txn *memdb.Txn, obj MemoryStorableObject) ([]kafka.Message, error)
	ProcessObjectDelete(ms *MemoryStore, txn *memdb.Txn, obj MemoryStorableObject) ([]kafka.Message, error)
	ReplicaName() string // which replica it belongs to
}

type MemoryStore struct {
	*memdb.MemDB

	kafkaConnection *kafka.MessageBroker

	kafkaMutex          sync.RWMutex
	kafkaSources        []KafkaSource
	kafkaMapSources     map[string]KafkaSource
	kafkaDestinations   []KafkaDestination
	replicaDestinations map[string]KafkaDestination

	hookMutex sync.Mutex
	hooks     map[string][]ObjectHook // by objectType
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

func (mst *MemoryStoreTxn) Insert(table string, obj interface{}) error {
	mobj, ok := obj.(MemoryStorableObject)
	if !ok {
		mst.memstore.logger.Warn("object does not implement MemoryStorableObject. Can not trigger hooks")
		return mst.Txn.Insert(table, obj)
	}
	typ := mobj.ObjType()
	hooks, ok := mst.memstore.hooks[typ]
	if !ok {
		return mst.Txn.Insert(table, obj)
	}

	for _, hook := range hooks {
		for _, event := range hook.Events {
			if event == HookEventInsert {
				err := hook.CallbackFn(mst, event, obj)
				if err != nil {
					return fmt.Errorf("hook execution failed: %s", err)
				}
				break
			}
		}
	}

	return mst.Txn.Insert(table, obj)
}

func (mst *MemoryStoreTxn) Delete(table string, obj interface{}) error {
	mobj, ok := obj.(MemoryStorableObject)
	if !ok {
		mst.memstore.logger.Warn("object does not implement MemoryStorableObject. Can not trigger hooks")
		return mst.Txn.Delete(table, obj)
	}

	typ := mobj.ObjType()
	hooks, ok := mst.memstore.hooks[typ]
	if !ok {
		return mst.Txn.Delete(table, obj)
	}

	for _, hook := range hooks {
		for _, event := range hook.Events {
			if event == HookEventDelete {
				err := hook.CallbackFn(mst, event, obj)
				if err != nil {
					return fmt.Errorf("hook execution failed: %s", err)
				}
				break
			}
		}
	}

	return mst.Txn.Delete(table, obj)
}

func (mst *MemoryStoreTxn) commitWithSourceInput(sourceMsg ...*kafka.SourceInputMessage) error {
	changes := mst.Txn.Changes()

	kafkaMessages := make([]kafka.Message, 0)
	for _, change := range changes {
		if mst.memstore.kafkaConnection == nil || !mst.memstore.kafkaConnection.Configured() {
			mst.memstore.logger.Warn("kafka not configured")
			continue
		}
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
					return fmt.Errorf("object does not implement MemoryStorableObject: %s", reflect.TypeOf(change.After))
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

	// TODO: atomic check

	if len(kafkaMessages) > 0 {
		var sm *kafka.SourceInputMessage
		if len(sourceMsg) > 0 {
			sm = sourceMsg[0]
		}
		mst.memstore.logger.Info("Messages count", "count", len(kafkaMessages), "source", sm)
		return mst.memstore.kafkaConnection.SendMessages(kafkaMessages, sm)
	}

	return nil
}

func (mst *MemoryStoreTxn) Commit(sourceMsg ...*kafka.SourceInputMessage) error {
	// все посчитать!
	err := mst.commitWithSourceInput(sourceMsg...)
	if err != nil {
		mst.memstore.logger.Error("transaction aborted", err)
		mst.Txn.Abort()
		return err
	}

	mst.memstore.logger.Debug("Commit transaction")
	mst.Txn.Commit()
	return nil
}

func (mst *MemoryStoreTxn) Abort() {
	mst.Txn.Abort()
}

func NewMemoryStore(schema *memdb.DBSchema, conn *kafka.MessageBroker) (*MemoryStore, error) {
	db, err := memdb.NewMemDB(schema)
	if err != nil {
		return nil, err
	}

	return &MemoryStore{
		db,
		conn,
		sync.RWMutex{},
		make([]KafkaSource, 0),
		make(map[string]KafkaSource),
		make([]KafkaDestination, 0),
		make(map[string]KafkaDestination),
		sync.Mutex{},
		make(map[string][]ObjectHook),
		log.New(nil),
	}, nil
}

func (ms *MemoryStore) SetLogger(l log.Logger) {
	ms.logger = l
}

func (ms *MemoryStore) AddKafkaSource(s KafkaSource) {
	name := s.Name()
	ms.logger.Debug(fmt.Sprintf("Add kafka source '%s'", s.Name()))
	ms.RemoveKafkaSource(name)

	ms.kafkaMutex.Lock()
	ms.kafkaSources = append(ms.kafkaSources, s)
	if name != "" {
		ms.kafkaMapSources[name] = s
	}
	ms.kafkaMutex.Unlock()

	if ms.kafkaConnection.Configured() {
		go s.Run(ms)
	}
}

func (ms *MemoryStore) ReinitializeKafka() {
	ms.kafkaMutex.RLock()
	// need sync restore before kafka sources rewatch
	ms.logger.Debug("Call reinitialize kafka")
	defer ms.logger.Debug("Kafka reinitialized")
	ms.Restore() // nolint: errcheck
	for _, s := range ms.kafkaSources {
		name := s.Name()
		if name == "" {
			ms.logger.Debug("Reinitialize with empty name")
		} else {
			ms.kafkaMapSources[name] = s
		}
		ms.logger.Debug(fmt.Sprintf("Reinitialize source %s", name))
		go s.Run(ms)
	}
	ms.kafkaMutex.RUnlock()
}

func (ms *MemoryStore) AddKafkaDestination(s KafkaDestination) {
	ms.RemoveKafkaDestination(s.ReplicaName())

	ms.kafkaMutex.Lock()
	ms.replicaDestinations[s.ReplicaName()] = s
	ms.kafkaDestinations = append(ms.kafkaDestinations, s)
	ms.kafkaMutex.Unlock()
}

func (ms *MemoryStore) GetKafkaBroker() *kafka.MessageBroker {
	return ms.kafkaConnection
}

func (ms *MemoryStore) RemoveKafkaSource(name string) {
	if name == "" {
		return
	}

	ms.kafkaMutex.Lock()
	defer ms.kafkaMutex.Unlock()

	ks, ok := ms.kafkaMapSources[name]
	if !ok {
		return
	}
	index := -1
	for i, dest := range ms.kafkaSources {
		if dest == ks {
			index = i
			break
		}
	}

	if index == -1 {
		return
	}

	ks.Stop()

	ms.kafkaSources = append(ms.kafkaSources[:index], ms.kafkaSources[index+1:]...)
	delete(ms.kafkaMapSources, name)
}

func (ms *MemoryStore) RemoveKafkaDestination(replicaName string) {
	if replicaName == "" {
		return
	}

	ms.kafkaMutex.Lock()
	defer ms.kafkaMutex.Unlock()

	ks, ok := ms.replicaDestinations[replicaName]
	if !ok {
		return
	}
	index := -1
	for i, dest := range ms.kafkaDestinations {
		if dest == ks {
			index = i
			break
		}
	}

	if index == -1 {
		return
	}

	ms.kafkaDestinations = append(ms.kafkaDestinations[:index], ms.kafkaDestinations[index+1:]...)
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
