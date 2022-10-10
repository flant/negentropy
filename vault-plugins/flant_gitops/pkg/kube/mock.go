package kube

import (
	"context"
	"fmt"
	"sync"

	"github.com/hashicorp/vault/sdk/logical"
)

type MockKubeService struct {
	// by hashCommit = name
	activeJobs map[string]Job
	// by hashCommit = name
	finishedJobs map[string]Job
	mutex        *sync.Mutex
}

type Job struct {
	HashCommit    string
	VaultsB64Json string
}

func (m *MockKubeService) RunJob(_ context.Context, hashCommit string, vaultsB64Json string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.activeJobs[hashCommit] = Job{
		HashCommit:    hashCommit,
		VaultsB64Json: vaultsB64Json,
	}
	return nil
}

func (m *MockKubeService) CheckJob(_ context.Context, hashCommit string) (exist, finished, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	_, ok := m.activeJobs[hashCommit]
	if !ok {
		return false, false, nil
	}
	_, ok = m.finishedJobs[hashCommit]
	if ok {
		return true, true, nil
	}
	return true, false, nil
}

func (m *MockKubeService) FinishJob(_ context.Context, hashCommit string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	job, ok := m.activeJobs[hashCommit]
	if !ok {
		return fmt.Errorf("job by name: %s: not found", hashCommit)
	}
	delete(m.activeJobs, hashCommit)
	m.finishedJobs[hashCommit] = job
	return nil
}

func (m *MockKubeService) GetFinishedJob(hashCommit string) (Job, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	job, ok := m.finishedJobs[hashCommit]
	if !ok {
		return Job{}, fmt.Errorf("job by name: %s: not found", hashCommit)
	}
	return Job{HashCommit: job.HashCommit, VaultsB64Json: job.VaultsB64Json}, nil
}

func (m *MockKubeService) HasActiveJob(hashCommit string) bool {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	_, has := m.activeJobs[hashCommit]
	return has
}

func (m *MockKubeService) HasFinishedJob(hashCommit string) bool {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	_, has := m.finishedJobs[hashCommit]
	return has
}

func (m *MockKubeService) LenActiveJobs() int {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return len(m.activeJobs)
}

func (m *MockKubeService) LenFinishedJobs() int {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return len(m.finishedJobs)
}

func NewMock() (func(context.Context, logical.Storage) (KubeService, error), *MockKubeService) {
	mock := &MockKubeService{
		activeJobs:   map[string]Job{},
		finishedJobs: map[string]Job{},
		mutex:        &sync.Mutex{},
	}
	return func(context.Context, logical.Storage) (KubeService, error) { return mock, nil }, mock
}
