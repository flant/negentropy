package io

type HookEvent int

const (
	HookEventInsert HookEvent = iota
	HookEventDelete
)

type HookCallbackFn func(txn *MemoryStoreTxn, event HookEvent, obj interface{}) error

type ObjectHook struct {
	Events     []HookEvent // insert || delete
	ObjType    string      // model.Type
	CallbackFn HookCallbackFn
}

func (ms *MemoryStore) RegisterHook(hookConfig ObjectHook) {
	if len(hookConfig.Events) == 0 {
		return
	}

	ms.hookMutex.Lock()
	defer ms.hookMutex.Unlock()

	existHooks, ok := ms.hooks[hookConfig.ObjType]
	if !ok {
		ms.hooks[hookConfig.ObjType] = []ObjectHook{hookConfig}
		return
	}

	existHooks = append(existHooks, hookConfig)
	ms.hooks[hookConfig.ObjType] = existHooks
}
