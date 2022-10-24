package io

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/hashicorp/go-hclog"

	"github.com/flant/negentropy/vault-plugins/shared/kafka"
	"github.com/flant/negentropy/vault-plugins/shared/memdb"
)

type MemoryStorableObject interface {
	ObjType() string
	ObjId() string
}

type KafkaSource interface {
	// Name returns topic name
	Name() string
	// Restore gets data from topic and store it to txn
	// if called after Run, before Stop: do nothing
	Restore(txn *memdb.Txn) error
	// Run starts infinite loop processing incoming messages, will returns after using Stop
	// if called second time  - without intermediate call Stop:  do nothing
	Run(ms *MemoryStore)
	// Stop finish running infinite loop, if used before Run, will do nothing
	Stop()
}

type KafkaDestination interface {
	ProcessObject(ms *MemoryStore, txn *memdb.Txn, obj MemoryStorableObject) ([]kafka.Message, error)
	ProcessObjectDelete(ms *MemoryStore, txn *memdb.Txn, obj MemoryStorableObject) ([]kafka.Message, error)
	ReplicaName() string // which replica it belongs to
}

// MemoryStore is complex logic structure to store data in memdb and interact with kafka
// typical use:
// a) create Memstore (NewMemoryStore) with nil connection to kafka, use it without synchronizing with kafka (Txn, Delete, Insert, Commit, Abort)
// b) create Memstore with actual connection to kafka, add kafka destinations(AddKafkaDestination), add kafka sources (AddKafkaSource), Restore data from kafka,
//  RunKafkaSourceMainLoops to run main reading kafka loops. Be free to use ReinitializeKafka to reRstore data from kafka
// Use Close to close all connections to kafka
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
	logger hclog.Logger
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

	if len(kafkaMessages) > 0 || len(sourceMsg) > 0 {
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

func NewMemoryStore(schema *memdb.DBSchema, conn *kafka.MessageBroker, parentLogger hclog.Logger) (*MemoryStore, error) {
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
		parentLogger.Named("MemStore"),
	}, nil
}

func (ms *MemoryStore) AddKafkaSource(s KafkaSource) {
	name := s.Name()
	ms.logger.Debug(fmt.Sprintf("Add kafka source '%s'", name))
	ms.RemoveKafkaSource(name)

	ms.kafkaMutex.Lock()
	ms.kafkaSources = append(ms.kafkaSources, s)
	if name != "" {
		ms.kafkaMapSources[name] = s
	}
	ms.kafkaMutex.Unlock()
	ms.logger.Debug(fmt.Sprintf("kafka source '%s', AddKafkaSource finished", name))
}

func (ms *MemoryStore) ReinitializeKafka() {
	ms.kafkaMutex.RLock()
	// need sync restore before kafka sources rewatch
	ms.logger.Debug("Call reinitialize kafka")
	defer ms.logger.Debug("Kafka reinitialized")
	ms.logger.Debug("Stopping run loops")
	for _, s := range ms.kafkaSources {
		name := s.Name()
		ms.logger.Debug(fmt.Sprintf("Stop source %s", name))
		s.Stop()
	}

	err := ms.Restore()
	if err != nil {
		// it is critical error, if it happens, there are no guarantees memStore consistency,
		// also there is no way to anyhow restore it
		ms.logger.Error(fmt.Sprintf("critical error: %s", err.Error()))
	}
	ms.logger.Debug("running loops")
	for _, s := range ms.kafkaSources {
		name := s.Name()
		if name == "" {
			ms.logger.Debug("empty name source")
		} else {
			ms.kafkaMapSources[name] = s
		}
		ms.logger.Debug(fmt.Sprintf("run source %s", name))
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
	ms.logger.Debug(fmt.Sprintf("kafka source '%s' RemoveKafkaSource is started", name))
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
	ms.logger.Debug(fmt.Sprintf("kafka source '%s' RemoveKafkaSource finish", name))
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
	logger := ms.logger.Named("StorageRestore")
	logger.Debug("started")
	defer logger.Debug("exit")

	if !ms.kafkaConnection.Configured() {
		ms.logger.Warn("Kafka is not configured. Skipping restore")
		return nil
	}
	txn := ms.MemDB.Txn(true).WithSkippingInsertForeignKeysCheck() // turn off checking due to possible compaction problems and restoring items with archived relations
	ms.kafkaMutex.RLock()
	defer ms.kafkaMutex.RUnlock()
	for _, ks := range ms.kafkaSources {
		ms.logger.Debug(fmt.Sprintf("kafka_source %#v start", ks))
		err := ks.Restore(txn)
		if err != nil {
			txn.Abort()
			return err
		}
		ms.logger.Debug(fmt.Sprintf("kafka_source %#v end", ks))
	}
	txn.Commit()
	logger.Debug("normal finish")
	return nil
}

// RunKafkaSourceMainLoops try run main loops
func (ms *MemoryStore) RunKafkaSourceMainLoops() {
	ms.logger.Info("TryRunKafkaSourceMainLoops started")
	defer ms.logger.Info("TryRunKafkaSourceMainLoops exit")
	if ms.kafkaConnection.Configured() {
		for _, ks := range ms.kafkaSources {
			ms.logger.Debug(fmt.Sprintf("kafka_source %s main loop starting", ks.Name()))
			go ks.Run(ms)
		}
	} else {
		ms.logger.Warn("TryRunKafkaSourceMainLoops: kafka connection is not configured")
	}
}

func (ms *MemoryStore) removeAllKafkaSources() {
	ms.kafkaMutex.Lock()
	defer ms.kafkaMutex.Unlock()
	ms.logger.Debug("Call removeAllKafkaSources")
	defer ms.logger.Debug("removeAllKafkaSources finished")
	for _, k := range ms.kafkaSources {
		delete(ms.kafkaMapSources, k.Name())
		k.Stop()
	}
	ms.kafkaSources = []KafkaSource{}
}

func (ms *MemoryStore) Close() {
	ms.removeAllKafkaSources()
	ms.kafkaConnection.Close()
}

// commit wraps the committing and error logging
func CommitWithLog(txn *MemoryStoreTxn, logger hclog.Logger) error {
	err := txn.Commit()
	if err != nil {
		logger.Error("failed to commit", "err", err)
		return fmt.Errorf("request failed, try again")
	}
	return nil
}
