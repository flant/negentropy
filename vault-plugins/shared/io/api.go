package io

type DownstreamAPIAction interface {
	Execute() error
}
type DownstreamApi interface {
	ProcessObject(ms *MemoryStore, txn *MemoryStoreTxn, obj MemoryStorableObject) ([]DownstreamAPIAction, error)
	ProcessObjectDelete(ms *MemoryStore, txn *MemoryStoreTxn, obj MemoryStorableObject) ([]DownstreamAPIAction, error)
}
