package kube

import (
	"context"
	"fmt"

	"github.com/hashicorp/vault/sdk/logical"
)

type MockKubeService struct {
	// by hashCommit = name
	ActiveJobs map[string]Job
	// by hashCommit = name
	FinishedJobs map[string]Job
}

type Job struct {
	HashCommit    string
	VaultsB64Json string
}

func (m *MockKubeService) RunJob(_ context.Context, hashCommit string, vaultsB64Json string) error {
	m.ActiveJobs[hashCommit] = Job{
		HashCommit:    hashCommit,
		VaultsB64Json: vaultsB64Json,
	}
	return nil
}

func (m *MockKubeService) IsJobFinished(_ context.Context, hashCommit string) (bool, error) {
	_, ok := m.ActiveJobs[hashCommit]
	if ok {
		return false, nil
	}
	_, ok = m.FinishedJobs[hashCommit]
	if ok {
		return true, nil
	}
	return false, fmt.Errorf("job by name: %s: not found", hashCommit)
}

func (m *MockKubeService) FinishJob(_ context.Context, hashCommit string) error {
	job, ok := m.ActiveJobs[hashCommit]
	if !ok {
		return fmt.Errorf("job by name: %s: not found", hashCommit)
	}
	delete(m.ActiveJobs, hashCommit)
	m.FinishedJobs[hashCommit] = job
	return nil
}

func NewMock() (func(context.Context, logical.Storage) (KubeService, error), *MockKubeService) {
	mock := &MockKubeService{
		ActiveJobs:   map[string]Job{},
		FinishedJobs: map[string]Job{},
	}
	return func(context.Context, logical.Storage) (KubeService, error) { return mock, nil }, mock
}
