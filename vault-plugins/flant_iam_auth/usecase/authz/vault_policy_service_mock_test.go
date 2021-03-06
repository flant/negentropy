package authz

// Code generated by http://github.com/gojuno/minimock (dev). DO NOT EDIT.

//go:generate minimock -i github.com/flant/negentropy/vault-plugins/flant_iam_auth/usecase/authz.VaultPolicyService -o ./vault_policy_service_mock_test.go -n VaultPolicyServiceMock

import (
	"sync"
	mm_atomic "sync/atomic"
	mm_time "time"

	"github.com/gojuno/minimock/v3"
	"github.com/hashicorp/go-hclog"
)

// VaultPolicyServiceMock implements VaultPolicyService
type VaultPolicyServiceMock struct {
	t minimock.Tester

	funcDeleteVaultPolicies          func(policiesNames []string, logger hclog.Logger)
	inspectFuncDeleteVaultPolicies   func(policiesNames []string, logger hclog.Logger)
	afterDeleteVaultPoliciesCounter  uint64
	beforeDeleteVaultPoliciesCounter uint64
	DeleteVaultPoliciesMock          mVaultPolicyServiceMockDeleteVaultPolicies

	funcListVaultPolicies          func() (sa1 []string, err error)
	inspectFuncListVaultPolicies   func()
	afterListVaultPoliciesCounter  uint64
	beforeListVaultPoliciesCounter uint64
	ListVaultPoliciesMock          mVaultPolicyServiceMockListVaultPolicies
}

// NewVaultPolicyServiceMock returns a mock for VaultPolicyService
func NewVaultPolicyServiceMock(t minimock.Tester) *VaultPolicyServiceMock {
	m := &VaultPolicyServiceMock{t: t}
	if controller, ok := t.(minimock.MockController); ok {
		controller.RegisterMocker(m)
	}

	m.DeleteVaultPoliciesMock = mVaultPolicyServiceMockDeleteVaultPolicies{mock: m}
	m.DeleteVaultPoliciesMock.callArgs = []*VaultPolicyServiceMockDeleteVaultPoliciesParams{}

	m.ListVaultPoliciesMock = mVaultPolicyServiceMockListVaultPolicies{mock: m}

	return m
}

type mVaultPolicyServiceMockDeleteVaultPolicies struct {
	mock               *VaultPolicyServiceMock
	defaultExpectation *VaultPolicyServiceMockDeleteVaultPoliciesExpectation
	expectations       []*VaultPolicyServiceMockDeleteVaultPoliciesExpectation

	callArgs []*VaultPolicyServiceMockDeleteVaultPoliciesParams
	mutex    sync.RWMutex
}

// VaultPolicyServiceMockDeleteVaultPoliciesExpectation specifies expectation struct of the VaultPolicyService.DeleteVaultPolicies
type VaultPolicyServiceMockDeleteVaultPoliciesExpectation struct {
	mock   *VaultPolicyServiceMock
	params *VaultPolicyServiceMockDeleteVaultPoliciesParams

	Counter uint64
}

// VaultPolicyServiceMockDeleteVaultPoliciesParams contains parameters of the VaultPolicyService.DeleteVaultPolicies
type VaultPolicyServiceMockDeleteVaultPoliciesParams struct {
	policiesNames []string
	logger        hclog.Logger
}

// Expect sets up expected params for VaultPolicyService.DeleteVaultPolicies
func (mmDeleteVaultPolicies *mVaultPolicyServiceMockDeleteVaultPolicies) Expect(policiesNames []string, logger hclog.Logger) *mVaultPolicyServiceMockDeleteVaultPolicies {
	if mmDeleteVaultPolicies.mock.funcDeleteVaultPolicies != nil {
		mmDeleteVaultPolicies.mock.t.Fatalf("VaultPolicyServiceMock.DeleteVaultPolicies mock is already set by Set")
	}

	if mmDeleteVaultPolicies.defaultExpectation == nil {
		mmDeleteVaultPolicies.defaultExpectation = &VaultPolicyServiceMockDeleteVaultPoliciesExpectation{}
	}

	mmDeleteVaultPolicies.defaultExpectation.params = &VaultPolicyServiceMockDeleteVaultPoliciesParams{policiesNames, logger}
	for _, e := range mmDeleteVaultPolicies.expectations {
		if minimock.Equal(e.params, mmDeleteVaultPolicies.defaultExpectation.params) {
			mmDeleteVaultPolicies.mock.t.Fatalf("Expectation set by When has same params: %#v", *mmDeleteVaultPolicies.defaultExpectation.params)
		}
	}

	return mmDeleteVaultPolicies
}

// Inspect accepts an inspector function that has same arguments as the VaultPolicyService.DeleteVaultPolicies
func (mmDeleteVaultPolicies *mVaultPolicyServiceMockDeleteVaultPolicies) Inspect(f func(policiesNames []string, logger hclog.Logger)) *mVaultPolicyServiceMockDeleteVaultPolicies {
	if mmDeleteVaultPolicies.mock.inspectFuncDeleteVaultPolicies != nil {
		mmDeleteVaultPolicies.mock.t.Fatalf("Inspect function is already set for VaultPolicyServiceMock.DeleteVaultPolicies")
	}

	mmDeleteVaultPolicies.mock.inspectFuncDeleteVaultPolicies = f

	return mmDeleteVaultPolicies
}

// Return sets up results that will be returned by VaultPolicyService.DeleteVaultPolicies
func (mmDeleteVaultPolicies *mVaultPolicyServiceMockDeleteVaultPolicies) Return() *VaultPolicyServiceMock {
	if mmDeleteVaultPolicies.mock.funcDeleteVaultPolicies != nil {
		mmDeleteVaultPolicies.mock.t.Fatalf("VaultPolicyServiceMock.DeleteVaultPolicies mock is already set by Set")
	}

	if mmDeleteVaultPolicies.defaultExpectation == nil {
		mmDeleteVaultPolicies.defaultExpectation = &VaultPolicyServiceMockDeleteVaultPoliciesExpectation{mock: mmDeleteVaultPolicies.mock}
	}

	return mmDeleteVaultPolicies.mock
}

//Set uses given function f to mock the VaultPolicyService.DeleteVaultPolicies method
func (mmDeleteVaultPolicies *mVaultPolicyServiceMockDeleteVaultPolicies) Set(f func(policiesNames []string, logger hclog.Logger)) *VaultPolicyServiceMock {
	if mmDeleteVaultPolicies.defaultExpectation != nil {
		mmDeleteVaultPolicies.mock.t.Fatalf("Default expectation is already set for the VaultPolicyService.DeleteVaultPolicies method")
	}

	if len(mmDeleteVaultPolicies.expectations) > 0 {
		mmDeleteVaultPolicies.mock.t.Fatalf("Some expectations are already set for the VaultPolicyService.DeleteVaultPolicies method")
	}

	mmDeleteVaultPolicies.mock.funcDeleteVaultPolicies = f
	return mmDeleteVaultPolicies.mock
}

// DeleteVaultPolicies implements VaultPolicyService
func (mmDeleteVaultPolicies *VaultPolicyServiceMock) DeleteVaultPolicies(policiesNames []string, logger hclog.Logger) {
	mm_atomic.AddUint64(&mmDeleteVaultPolicies.beforeDeleteVaultPoliciesCounter, 1)
	defer mm_atomic.AddUint64(&mmDeleteVaultPolicies.afterDeleteVaultPoliciesCounter, 1)

	if mmDeleteVaultPolicies.inspectFuncDeleteVaultPolicies != nil {
		mmDeleteVaultPolicies.inspectFuncDeleteVaultPolicies(policiesNames, logger)
	}

	mm_params := &VaultPolicyServiceMockDeleteVaultPoliciesParams{policiesNames, logger}

	// Record call args
	mmDeleteVaultPolicies.DeleteVaultPoliciesMock.mutex.Lock()
	mmDeleteVaultPolicies.DeleteVaultPoliciesMock.callArgs = append(mmDeleteVaultPolicies.DeleteVaultPoliciesMock.callArgs, mm_params)
	mmDeleteVaultPolicies.DeleteVaultPoliciesMock.mutex.Unlock()

	for _, e := range mmDeleteVaultPolicies.DeleteVaultPoliciesMock.expectations {
		if minimock.Equal(e.params, mm_params) {
			mm_atomic.AddUint64(&e.Counter, 1)
			return
		}
	}

	if mmDeleteVaultPolicies.DeleteVaultPoliciesMock.defaultExpectation != nil {
		mm_atomic.AddUint64(&mmDeleteVaultPolicies.DeleteVaultPoliciesMock.defaultExpectation.Counter, 1)
		mm_want := mmDeleteVaultPolicies.DeleteVaultPoliciesMock.defaultExpectation.params
		mm_got := VaultPolicyServiceMockDeleteVaultPoliciesParams{policiesNames, logger}
		if mm_want != nil && !minimock.Equal(*mm_want, mm_got) {
			mmDeleteVaultPolicies.t.Errorf("VaultPolicyServiceMock.DeleteVaultPolicies got unexpected parameters, want: %#v, got: %#v%s\n", *mm_want, mm_got, minimock.Diff(*mm_want, mm_got))
		}

		return

	}
	if mmDeleteVaultPolicies.funcDeleteVaultPolicies != nil {
		mmDeleteVaultPolicies.funcDeleteVaultPolicies(policiesNames, logger)
		return
	}
	mmDeleteVaultPolicies.t.Fatalf("Unexpected call to VaultPolicyServiceMock.DeleteVaultPolicies. %v %v", policiesNames, logger)

}

// DeleteVaultPoliciesAfterCounter returns a count of finished VaultPolicyServiceMock.DeleteVaultPolicies invocations
func (mmDeleteVaultPolicies *VaultPolicyServiceMock) DeleteVaultPoliciesAfterCounter() uint64 {
	return mm_atomic.LoadUint64(&mmDeleteVaultPolicies.afterDeleteVaultPoliciesCounter)
}

// DeleteVaultPoliciesBeforeCounter returns a count of VaultPolicyServiceMock.DeleteVaultPolicies invocations
func (mmDeleteVaultPolicies *VaultPolicyServiceMock) DeleteVaultPoliciesBeforeCounter() uint64 {
	return mm_atomic.LoadUint64(&mmDeleteVaultPolicies.beforeDeleteVaultPoliciesCounter)
}

// Calls returns a list of arguments used in each call to VaultPolicyServiceMock.DeleteVaultPolicies.
// The list is in the same order as the calls were made (i.e. recent calls have a higher index)
func (mmDeleteVaultPolicies *mVaultPolicyServiceMockDeleteVaultPolicies) Calls() []*VaultPolicyServiceMockDeleteVaultPoliciesParams {
	mmDeleteVaultPolicies.mutex.RLock()

	argCopy := make([]*VaultPolicyServiceMockDeleteVaultPoliciesParams, len(mmDeleteVaultPolicies.callArgs))
	copy(argCopy, mmDeleteVaultPolicies.callArgs)

	mmDeleteVaultPolicies.mutex.RUnlock()

	return argCopy
}

// MinimockDeleteVaultPoliciesDone returns true if the count of the DeleteVaultPolicies invocations corresponds
// the number of defined expectations
func (m *VaultPolicyServiceMock) MinimockDeleteVaultPoliciesDone() bool {
	for _, e := range m.DeleteVaultPoliciesMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			return false
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.DeleteVaultPoliciesMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterDeleteVaultPoliciesCounter) < 1 {
		return false
	}
	// if func was set then invocations count should be greater than zero
	if m.funcDeleteVaultPolicies != nil && mm_atomic.LoadUint64(&m.afterDeleteVaultPoliciesCounter) < 1 {
		return false
	}
	return true
}

// MinimockDeleteVaultPoliciesInspect logs each unmet expectation
func (m *VaultPolicyServiceMock) MinimockDeleteVaultPoliciesInspect() {
	for _, e := range m.DeleteVaultPoliciesMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			m.t.Errorf("Expected call to VaultPolicyServiceMock.DeleteVaultPolicies with params: %#v", *e.params)
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.DeleteVaultPoliciesMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterDeleteVaultPoliciesCounter) < 1 {
		if m.DeleteVaultPoliciesMock.defaultExpectation.params == nil {
			m.t.Error("Expected call to VaultPolicyServiceMock.DeleteVaultPolicies")
		} else {
			m.t.Errorf("Expected call to VaultPolicyServiceMock.DeleteVaultPolicies with params: %#v", *m.DeleteVaultPoliciesMock.defaultExpectation.params)
		}
	}
	// if func was set then invocations count should be greater than zero
	if m.funcDeleteVaultPolicies != nil && mm_atomic.LoadUint64(&m.afterDeleteVaultPoliciesCounter) < 1 {
		m.t.Error("Expected call to VaultPolicyServiceMock.DeleteVaultPolicies")
	}
}

type mVaultPolicyServiceMockListVaultPolicies struct {
	mock               *VaultPolicyServiceMock
	defaultExpectation *VaultPolicyServiceMockListVaultPoliciesExpectation
	expectations       []*VaultPolicyServiceMockListVaultPoliciesExpectation
}

// VaultPolicyServiceMockListVaultPoliciesExpectation specifies expectation struct of the VaultPolicyService.ListVaultPolicies
type VaultPolicyServiceMockListVaultPoliciesExpectation struct {
	mock *VaultPolicyServiceMock

	results *VaultPolicyServiceMockListVaultPoliciesResults
	Counter uint64
}

// VaultPolicyServiceMockListVaultPoliciesResults contains results of the VaultPolicyService.ListVaultPolicies
type VaultPolicyServiceMockListVaultPoliciesResults struct {
	sa1 []string
	err error
}

// Expect sets up expected params for VaultPolicyService.ListVaultPolicies
func (mmListVaultPolicies *mVaultPolicyServiceMockListVaultPolicies) Expect() *mVaultPolicyServiceMockListVaultPolicies {
	if mmListVaultPolicies.mock.funcListVaultPolicies != nil {
		mmListVaultPolicies.mock.t.Fatalf("VaultPolicyServiceMock.ListVaultPolicies mock is already set by Set")
	}

	if mmListVaultPolicies.defaultExpectation == nil {
		mmListVaultPolicies.defaultExpectation = &VaultPolicyServiceMockListVaultPoliciesExpectation{}
	}

	return mmListVaultPolicies
}

// Inspect accepts an inspector function that has same arguments as the VaultPolicyService.ListVaultPolicies
func (mmListVaultPolicies *mVaultPolicyServiceMockListVaultPolicies) Inspect(f func()) *mVaultPolicyServiceMockListVaultPolicies {
	if mmListVaultPolicies.mock.inspectFuncListVaultPolicies != nil {
		mmListVaultPolicies.mock.t.Fatalf("Inspect function is already set for VaultPolicyServiceMock.ListVaultPolicies")
	}

	mmListVaultPolicies.mock.inspectFuncListVaultPolicies = f

	return mmListVaultPolicies
}

// Return sets up results that will be returned by VaultPolicyService.ListVaultPolicies
func (mmListVaultPolicies *mVaultPolicyServiceMockListVaultPolicies) Return(sa1 []string, err error) *VaultPolicyServiceMock {
	if mmListVaultPolicies.mock.funcListVaultPolicies != nil {
		mmListVaultPolicies.mock.t.Fatalf("VaultPolicyServiceMock.ListVaultPolicies mock is already set by Set")
	}

	if mmListVaultPolicies.defaultExpectation == nil {
		mmListVaultPolicies.defaultExpectation = &VaultPolicyServiceMockListVaultPoliciesExpectation{mock: mmListVaultPolicies.mock}
	}
	mmListVaultPolicies.defaultExpectation.results = &VaultPolicyServiceMockListVaultPoliciesResults{sa1, err}
	return mmListVaultPolicies.mock
}

//Set uses given function f to mock the VaultPolicyService.ListVaultPolicies method
func (mmListVaultPolicies *mVaultPolicyServiceMockListVaultPolicies) Set(f func() (sa1 []string, err error)) *VaultPolicyServiceMock {
	if mmListVaultPolicies.defaultExpectation != nil {
		mmListVaultPolicies.mock.t.Fatalf("Default expectation is already set for the VaultPolicyService.ListVaultPolicies method")
	}

	if len(mmListVaultPolicies.expectations) > 0 {
		mmListVaultPolicies.mock.t.Fatalf("Some expectations are already set for the VaultPolicyService.ListVaultPolicies method")
	}

	mmListVaultPolicies.mock.funcListVaultPolicies = f
	return mmListVaultPolicies.mock
}

// ListVaultPolicies implements VaultPolicyService
func (mmListVaultPolicies *VaultPolicyServiceMock) ListVaultPolicies() (sa1 []string, err error) {
	mm_atomic.AddUint64(&mmListVaultPolicies.beforeListVaultPoliciesCounter, 1)
	defer mm_atomic.AddUint64(&mmListVaultPolicies.afterListVaultPoliciesCounter, 1)

	if mmListVaultPolicies.inspectFuncListVaultPolicies != nil {
		mmListVaultPolicies.inspectFuncListVaultPolicies()
	}

	if mmListVaultPolicies.ListVaultPoliciesMock.defaultExpectation != nil {
		mm_atomic.AddUint64(&mmListVaultPolicies.ListVaultPoliciesMock.defaultExpectation.Counter, 1)

		mm_results := mmListVaultPolicies.ListVaultPoliciesMock.defaultExpectation.results
		if mm_results == nil {
			mmListVaultPolicies.t.Fatal("No results are set for the VaultPolicyServiceMock.ListVaultPolicies")
		}
		return (*mm_results).sa1, (*mm_results).err
	}
	if mmListVaultPolicies.funcListVaultPolicies != nil {
		return mmListVaultPolicies.funcListVaultPolicies()
	}
	mmListVaultPolicies.t.Fatalf("Unexpected call to VaultPolicyServiceMock.ListVaultPolicies.")
	return
}

// ListVaultPoliciesAfterCounter returns a count of finished VaultPolicyServiceMock.ListVaultPolicies invocations
func (mmListVaultPolicies *VaultPolicyServiceMock) ListVaultPoliciesAfterCounter() uint64 {
	return mm_atomic.LoadUint64(&mmListVaultPolicies.afterListVaultPoliciesCounter)
}

// ListVaultPoliciesBeforeCounter returns a count of VaultPolicyServiceMock.ListVaultPolicies invocations
func (mmListVaultPolicies *VaultPolicyServiceMock) ListVaultPoliciesBeforeCounter() uint64 {
	return mm_atomic.LoadUint64(&mmListVaultPolicies.beforeListVaultPoliciesCounter)
}

// MinimockListVaultPoliciesDone returns true if the count of the ListVaultPolicies invocations corresponds
// the number of defined expectations
func (m *VaultPolicyServiceMock) MinimockListVaultPoliciesDone() bool {
	for _, e := range m.ListVaultPoliciesMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			return false
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.ListVaultPoliciesMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterListVaultPoliciesCounter) < 1 {
		return false
	}
	// if func was set then invocations count should be greater than zero
	if m.funcListVaultPolicies != nil && mm_atomic.LoadUint64(&m.afterListVaultPoliciesCounter) < 1 {
		return false
	}
	return true
}

// MinimockListVaultPoliciesInspect logs each unmet expectation
func (m *VaultPolicyServiceMock) MinimockListVaultPoliciesInspect() {
	for _, e := range m.ListVaultPoliciesMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			m.t.Error("Expected call to VaultPolicyServiceMock.ListVaultPolicies")
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.ListVaultPoliciesMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterListVaultPoliciesCounter) < 1 {
		m.t.Error("Expected call to VaultPolicyServiceMock.ListVaultPolicies")
	}
	// if func was set then invocations count should be greater than zero
	if m.funcListVaultPolicies != nil && mm_atomic.LoadUint64(&m.afterListVaultPoliciesCounter) < 1 {
		m.t.Error("Expected call to VaultPolicyServiceMock.ListVaultPolicies")
	}
}

// MinimockFinish checks that all mocked methods have been called the expected number of times
func (m *VaultPolicyServiceMock) MinimockFinish() {
	if !m.minimockDone() {
		m.MinimockDeleteVaultPoliciesInspect()

		m.MinimockListVaultPoliciesInspect()
		m.t.FailNow()
	}
}

// MinimockWait waits for all mocked methods to be called the expected number of times
func (m *VaultPolicyServiceMock) MinimockWait(timeout mm_time.Duration) {
	timeoutCh := mm_time.After(timeout)
	for {
		if m.minimockDone() {
			return
		}
		select {
		case <-timeoutCh:
			m.MinimockFinish()
			return
		case <-mm_time.After(10 * mm_time.Millisecond):
		}
	}
}

func (m *VaultPolicyServiceMock) minimockDone() bool {
	done := true
	return done &&
		m.MinimockDeleteVaultPoliciesDone() &&
		m.MinimockListVaultPoliciesDone()
}
