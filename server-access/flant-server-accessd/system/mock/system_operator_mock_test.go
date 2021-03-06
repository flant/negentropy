package mock

// Code generated by http://github.com/gojuno/minimock (dev). DO NOT EDIT.

//go:generate minimock -i github.com/flant/negentropy/server-access/flant-server-accessd/system.Interface -o ./mock/system_operator_mock_test.go

import (
	"sync"
	mm_atomic "sync/atomic"
	mm_time "time"

	"github.com/gojuno/minimock/v3"
)

// InterfaceMock implements system.Interface
type InterfaceMock struct {
	t minimock.Tester

	funcCreateHomeDir          func(dir string, uid int, gid int) (err error)
	inspectFuncCreateHomeDir   func(dir string, uid int, gid int)
	afterCreateHomeDirCounter  uint64
	beforeCreateHomeDirCounter uint64
	CreateHomeDirMock          mInterfaceMockCreateHomeDir

	funcDeleteHomeDir          func(dir string) (err error)
	inspectFuncDeleteHomeDir   func(dir string)
	afterDeleteHomeDirCounter  uint64
	beforeDeleteHomeDirCounter uint64
	DeleteHomeDirMock          mInterfaceMockDeleteHomeDir

	funcPurgeUserLegacy          func(username string) (err error)
	inspectFuncPurgeUserLegacy   func(username string)
	afterPurgeUserLegacyCounter  uint64
	beforePurgeUserLegacyCounter uint64
	PurgeUserLegacyMock          mInterfaceMockPurgeUserLegacy
}

// NewInterfaceMock returns a mock for system.Interface
func NewInterfaceMock(t minimock.Tester) *InterfaceMock {
	m := &InterfaceMock{t: t}
	if controller, ok := t.(minimock.MockController); ok {
		controller.RegisterMocker(m)
	}

	m.CreateHomeDirMock = mInterfaceMockCreateHomeDir{mock: m}
	m.CreateHomeDirMock.callArgs = []*InterfaceMockCreateHomeDirParams{}

	m.DeleteHomeDirMock = mInterfaceMockDeleteHomeDir{mock: m}
	m.DeleteHomeDirMock.callArgs = []*InterfaceMockDeleteHomeDirParams{}

	m.PurgeUserLegacyMock = mInterfaceMockPurgeUserLegacy{mock: m}
	m.PurgeUserLegacyMock.callArgs = []*InterfaceMockPurgeUserLegacyParams{}

	return m
}

type mInterfaceMockCreateHomeDir struct {
	mock               *InterfaceMock
	defaultExpectation *InterfaceMockCreateHomeDirExpectation
	expectations       []*InterfaceMockCreateHomeDirExpectation

	callArgs []*InterfaceMockCreateHomeDirParams
	mutex    sync.RWMutex
}

// InterfaceMockCreateHomeDirExpectation specifies expectation struct of the Interface.CreateHomeDir
type InterfaceMockCreateHomeDirExpectation struct {
	mock    *InterfaceMock
	params  *InterfaceMockCreateHomeDirParams
	results *InterfaceMockCreateHomeDirResults
	Counter uint64
}

// InterfaceMockCreateHomeDirParams contains parameters of the Interface.CreateHomeDir
type InterfaceMockCreateHomeDirParams struct {
	dir string
	uid int
	gid int
}

// InterfaceMockCreateHomeDirResults contains results of the Interface.CreateHomeDir
type InterfaceMockCreateHomeDirResults struct {
	err error
}

// Expect sets up expected params for Interface.CreateHomeDir
func (mmCreateHomeDir *mInterfaceMockCreateHomeDir) Expect(dir string, uid int, gid int) *mInterfaceMockCreateHomeDir {
	if mmCreateHomeDir.mock.funcCreateHomeDir != nil {
		mmCreateHomeDir.mock.t.Fatalf("InterfaceMock.CreateHomeDir mock is already set by Set")
	}

	if mmCreateHomeDir.defaultExpectation == nil {
		mmCreateHomeDir.defaultExpectation = &InterfaceMockCreateHomeDirExpectation{}
	}

	mmCreateHomeDir.defaultExpectation.params = &InterfaceMockCreateHomeDirParams{dir, uid, gid}
	for _, e := range mmCreateHomeDir.expectations {
		if minimock.Equal(e.params, mmCreateHomeDir.defaultExpectation.params) {
			mmCreateHomeDir.mock.t.Fatalf("Expectation set by When has same params: %#v", *mmCreateHomeDir.defaultExpectation.params)
		}
	}

	return mmCreateHomeDir
}

// Inspect accepts an inspector function that has same arguments as the Interface.CreateHomeDir
func (mmCreateHomeDir *mInterfaceMockCreateHomeDir) Inspect(f func(dir string, uid int, gid int)) *mInterfaceMockCreateHomeDir {
	if mmCreateHomeDir.mock.inspectFuncCreateHomeDir != nil {
		mmCreateHomeDir.mock.t.Fatalf("Inspect function is already set for InterfaceMock.CreateHomeDir")
	}

	mmCreateHomeDir.mock.inspectFuncCreateHomeDir = f

	return mmCreateHomeDir
}

// Return sets up results that will be returned by Interface.CreateHomeDir
func (mmCreateHomeDir *mInterfaceMockCreateHomeDir) Return(err error) *InterfaceMock {
	if mmCreateHomeDir.mock.funcCreateHomeDir != nil {
		mmCreateHomeDir.mock.t.Fatalf("InterfaceMock.CreateHomeDir mock is already set by Set")
	}

	if mmCreateHomeDir.defaultExpectation == nil {
		mmCreateHomeDir.defaultExpectation = &InterfaceMockCreateHomeDirExpectation{mock: mmCreateHomeDir.mock}
	}
	mmCreateHomeDir.defaultExpectation.results = &InterfaceMockCreateHomeDirResults{err}
	return mmCreateHomeDir.mock
}

//Set uses given function f to mock the Interface.CreateHomeDir method
func (mmCreateHomeDir *mInterfaceMockCreateHomeDir) Set(f func(dir string, uid int, gid int) (err error)) *InterfaceMock {
	if mmCreateHomeDir.defaultExpectation != nil {
		mmCreateHomeDir.mock.t.Fatalf("Default expectation is already set for the Interface.CreateHomeDir method")
	}

	if len(mmCreateHomeDir.expectations) > 0 {
		mmCreateHomeDir.mock.t.Fatalf("Some expectations are already set for the Interface.CreateHomeDir method")
	}

	mmCreateHomeDir.mock.funcCreateHomeDir = f
	return mmCreateHomeDir.mock
}

// When sets expectation for the Interface.CreateHomeDir which will trigger the result defined by the following
// Then helper
func (mmCreateHomeDir *mInterfaceMockCreateHomeDir) When(dir string, uid int, gid int) *InterfaceMockCreateHomeDirExpectation {
	if mmCreateHomeDir.mock.funcCreateHomeDir != nil {
		mmCreateHomeDir.mock.t.Fatalf("InterfaceMock.CreateHomeDir mock is already set by Set")
	}

	expectation := &InterfaceMockCreateHomeDirExpectation{
		mock:   mmCreateHomeDir.mock,
		params: &InterfaceMockCreateHomeDirParams{dir, uid, gid},
	}
	mmCreateHomeDir.expectations = append(mmCreateHomeDir.expectations, expectation)
	return expectation
}

// Then sets up Interface.CreateHomeDir return parameters for the expectation previously defined by the When method
func (e *InterfaceMockCreateHomeDirExpectation) Then(err error) *InterfaceMock {
	e.results = &InterfaceMockCreateHomeDirResults{err}
	return e.mock
}

// CreateHomeDir implements system.Interface
func (mmCreateHomeDir *InterfaceMock) CreateHomeDir(dir string, uid int, gid int) (err error) {
	mm_atomic.AddUint64(&mmCreateHomeDir.beforeCreateHomeDirCounter, 1)
	defer mm_atomic.AddUint64(&mmCreateHomeDir.afterCreateHomeDirCounter, 1)

	if mmCreateHomeDir.inspectFuncCreateHomeDir != nil {
		mmCreateHomeDir.inspectFuncCreateHomeDir(dir, uid, gid)
	}

	mm_params := &InterfaceMockCreateHomeDirParams{dir, uid, gid}

	// Record call args
	mmCreateHomeDir.CreateHomeDirMock.mutex.Lock()
	mmCreateHomeDir.CreateHomeDirMock.callArgs = append(mmCreateHomeDir.CreateHomeDirMock.callArgs, mm_params)
	mmCreateHomeDir.CreateHomeDirMock.mutex.Unlock()

	for _, e := range mmCreateHomeDir.CreateHomeDirMock.expectations {
		if minimock.Equal(e.params, mm_params) {
			mm_atomic.AddUint64(&e.Counter, 1)
			return e.results.err
		}
	}

	if mmCreateHomeDir.CreateHomeDirMock.defaultExpectation != nil {
		mm_atomic.AddUint64(&mmCreateHomeDir.CreateHomeDirMock.defaultExpectation.Counter, 1)
		mm_want := mmCreateHomeDir.CreateHomeDirMock.defaultExpectation.params
		mm_got := InterfaceMockCreateHomeDirParams{dir, uid, gid}
		if mm_want != nil && !minimock.Equal(*mm_want, mm_got) {
			mmCreateHomeDir.t.Errorf("InterfaceMock.CreateHomeDir got unexpected parameters, want: %#v, got: %#v%s\n", *mm_want, mm_got, minimock.Diff(*mm_want, mm_got))
		}

		mm_results := mmCreateHomeDir.CreateHomeDirMock.defaultExpectation.results
		if mm_results == nil {
			mmCreateHomeDir.t.Fatal("No results are set for the InterfaceMock.CreateHomeDir")
		}
		return (*mm_results).err
	}
	if mmCreateHomeDir.funcCreateHomeDir != nil {
		return mmCreateHomeDir.funcCreateHomeDir(dir, uid, gid)
	}
	mmCreateHomeDir.t.Fatalf("Unexpected call to InterfaceMock.CreateHomeDir. %v %v %v", dir, uid, gid)
	return
}

// CreateHomeDirAfterCounter returns a count of finished InterfaceMock.CreateHomeDir invocations
func (mmCreateHomeDir *InterfaceMock) CreateHomeDirAfterCounter() uint64 {
	return mm_atomic.LoadUint64(&mmCreateHomeDir.afterCreateHomeDirCounter)
}

// CreateHomeDirBeforeCounter returns a count of InterfaceMock.CreateHomeDir invocations
func (mmCreateHomeDir *InterfaceMock) CreateHomeDirBeforeCounter() uint64 {
	return mm_atomic.LoadUint64(&mmCreateHomeDir.beforeCreateHomeDirCounter)
}

// Calls returns a list of arguments used in each call to InterfaceMock.CreateHomeDir.
// The list is in the same order as the calls were made (i.e. recent calls have a higher index)
func (mmCreateHomeDir *mInterfaceMockCreateHomeDir) Calls() []*InterfaceMockCreateHomeDirParams {
	mmCreateHomeDir.mutex.RLock()

	argCopy := make([]*InterfaceMockCreateHomeDirParams, len(mmCreateHomeDir.callArgs))
	copy(argCopy, mmCreateHomeDir.callArgs)

	mmCreateHomeDir.mutex.RUnlock()

	return argCopy
}

// MinimockCreateHomeDirDone returns true if the count of the CreateHomeDir invocations corresponds
// the number of defined expectations
func (m *InterfaceMock) MinimockCreateHomeDirDone() bool {
	for _, e := range m.CreateHomeDirMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			return false
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.CreateHomeDirMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterCreateHomeDirCounter) < 1 {
		return false
	}
	// if func was set then invocations count should be greater than zero
	if m.funcCreateHomeDir != nil && mm_atomic.LoadUint64(&m.afterCreateHomeDirCounter) < 1 {
		return false
	}
	return true
}

// MinimockCreateHomeDirInspect logs each unmet expectation
func (m *InterfaceMock) MinimockCreateHomeDirInspect() {
	for _, e := range m.CreateHomeDirMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			m.t.Errorf("Expected call to InterfaceMock.CreateHomeDir with params: %#v", *e.params)
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.CreateHomeDirMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterCreateHomeDirCounter) < 1 {
		if m.CreateHomeDirMock.defaultExpectation.params == nil {
			m.t.Error("Expected call to InterfaceMock.CreateHomeDir")
		} else {
			m.t.Errorf("Expected call to InterfaceMock.CreateHomeDir with params: %#v", *m.CreateHomeDirMock.defaultExpectation.params)
		}
	}
	// if func was set then invocations count should be greater than zero
	if m.funcCreateHomeDir != nil && mm_atomic.LoadUint64(&m.afterCreateHomeDirCounter) < 1 {
		m.t.Error("Expected call to InterfaceMock.CreateHomeDir")
	}
}

type mInterfaceMockDeleteHomeDir struct {
	mock               *InterfaceMock
	defaultExpectation *InterfaceMockDeleteHomeDirExpectation
	expectations       []*InterfaceMockDeleteHomeDirExpectation

	callArgs []*InterfaceMockDeleteHomeDirParams
	mutex    sync.RWMutex
}

// InterfaceMockDeleteHomeDirExpectation specifies expectation struct of the Interface.DeleteHomeDir
type InterfaceMockDeleteHomeDirExpectation struct {
	mock    *InterfaceMock
	params  *InterfaceMockDeleteHomeDirParams
	results *InterfaceMockDeleteHomeDirResults
	Counter uint64
}

// InterfaceMockDeleteHomeDirParams contains parameters of the Interface.DeleteHomeDir
type InterfaceMockDeleteHomeDirParams struct {
	dir string
}

// InterfaceMockDeleteHomeDirResults contains results of the Interface.DeleteHomeDir
type InterfaceMockDeleteHomeDirResults struct {
	err error
}

// Expect sets up expected params for Interface.DeleteHomeDir
func (mmDeleteHomeDir *mInterfaceMockDeleteHomeDir) Expect(dir string) *mInterfaceMockDeleteHomeDir {
	if mmDeleteHomeDir.mock.funcDeleteHomeDir != nil {
		mmDeleteHomeDir.mock.t.Fatalf("InterfaceMock.DeleteHomeDir mock is already set by Set")
	}

	if mmDeleteHomeDir.defaultExpectation == nil {
		mmDeleteHomeDir.defaultExpectation = &InterfaceMockDeleteHomeDirExpectation{}
	}

	mmDeleteHomeDir.defaultExpectation.params = &InterfaceMockDeleteHomeDirParams{dir}
	for _, e := range mmDeleteHomeDir.expectations {
		if minimock.Equal(e.params, mmDeleteHomeDir.defaultExpectation.params) {
			mmDeleteHomeDir.mock.t.Fatalf("Expectation set by When has same params: %#v", *mmDeleteHomeDir.defaultExpectation.params)
		}
	}

	return mmDeleteHomeDir
}

// Inspect accepts an inspector function that has same arguments as the Interface.DeleteHomeDir
func (mmDeleteHomeDir *mInterfaceMockDeleteHomeDir) Inspect(f func(dir string)) *mInterfaceMockDeleteHomeDir {
	if mmDeleteHomeDir.mock.inspectFuncDeleteHomeDir != nil {
		mmDeleteHomeDir.mock.t.Fatalf("Inspect function is already set for InterfaceMock.DeleteHomeDir")
	}

	mmDeleteHomeDir.mock.inspectFuncDeleteHomeDir = f

	return mmDeleteHomeDir
}

// Return sets up results that will be returned by Interface.DeleteHomeDir
func (mmDeleteHomeDir *mInterfaceMockDeleteHomeDir) Return(err error) *InterfaceMock {
	if mmDeleteHomeDir.mock.funcDeleteHomeDir != nil {
		mmDeleteHomeDir.mock.t.Fatalf("InterfaceMock.DeleteHomeDir mock is already set by Set")
	}

	if mmDeleteHomeDir.defaultExpectation == nil {
		mmDeleteHomeDir.defaultExpectation = &InterfaceMockDeleteHomeDirExpectation{mock: mmDeleteHomeDir.mock}
	}
	mmDeleteHomeDir.defaultExpectation.results = &InterfaceMockDeleteHomeDirResults{err}
	return mmDeleteHomeDir.mock
}

//Set uses given function f to mock the Interface.DeleteHomeDir method
func (mmDeleteHomeDir *mInterfaceMockDeleteHomeDir) Set(f func(dir string) (err error)) *InterfaceMock {
	if mmDeleteHomeDir.defaultExpectation != nil {
		mmDeleteHomeDir.mock.t.Fatalf("Default expectation is already set for the Interface.DeleteHomeDir method")
	}

	if len(mmDeleteHomeDir.expectations) > 0 {
		mmDeleteHomeDir.mock.t.Fatalf("Some expectations are already set for the Interface.DeleteHomeDir method")
	}

	mmDeleteHomeDir.mock.funcDeleteHomeDir = f
	return mmDeleteHomeDir.mock
}

// When sets expectation for the Interface.DeleteHomeDir which will trigger the result defined by the following
// Then helper
func (mmDeleteHomeDir *mInterfaceMockDeleteHomeDir) When(dir string) *InterfaceMockDeleteHomeDirExpectation {
	if mmDeleteHomeDir.mock.funcDeleteHomeDir != nil {
		mmDeleteHomeDir.mock.t.Fatalf("InterfaceMock.DeleteHomeDir mock is already set by Set")
	}

	expectation := &InterfaceMockDeleteHomeDirExpectation{
		mock:   mmDeleteHomeDir.mock,
		params: &InterfaceMockDeleteHomeDirParams{dir},
	}
	mmDeleteHomeDir.expectations = append(mmDeleteHomeDir.expectations, expectation)
	return expectation
}

// Then sets up Interface.DeleteHomeDir return parameters for the expectation previously defined by the When method
func (e *InterfaceMockDeleteHomeDirExpectation) Then(err error) *InterfaceMock {
	e.results = &InterfaceMockDeleteHomeDirResults{err}
	return e.mock
}

// DeleteHomeDir implements system.Interface
func (mmDeleteHomeDir *InterfaceMock) DeleteHomeDir(dir string) (err error) {
	mm_atomic.AddUint64(&mmDeleteHomeDir.beforeDeleteHomeDirCounter, 1)
	defer mm_atomic.AddUint64(&mmDeleteHomeDir.afterDeleteHomeDirCounter, 1)

	if mmDeleteHomeDir.inspectFuncDeleteHomeDir != nil {
		mmDeleteHomeDir.inspectFuncDeleteHomeDir(dir)
	}

	mm_params := &InterfaceMockDeleteHomeDirParams{dir}

	// Record call args
	mmDeleteHomeDir.DeleteHomeDirMock.mutex.Lock()
	mmDeleteHomeDir.DeleteHomeDirMock.callArgs = append(mmDeleteHomeDir.DeleteHomeDirMock.callArgs, mm_params)
	mmDeleteHomeDir.DeleteHomeDirMock.mutex.Unlock()

	for _, e := range mmDeleteHomeDir.DeleteHomeDirMock.expectations {
		if minimock.Equal(e.params, mm_params) {
			mm_atomic.AddUint64(&e.Counter, 1)
			return e.results.err
		}
	}

	if mmDeleteHomeDir.DeleteHomeDirMock.defaultExpectation != nil {
		mm_atomic.AddUint64(&mmDeleteHomeDir.DeleteHomeDirMock.defaultExpectation.Counter, 1)
		mm_want := mmDeleteHomeDir.DeleteHomeDirMock.defaultExpectation.params
		mm_got := InterfaceMockDeleteHomeDirParams{dir}
		if mm_want != nil && !minimock.Equal(*mm_want, mm_got) {
			mmDeleteHomeDir.t.Errorf("InterfaceMock.DeleteHomeDir got unexpected parameters, want: %#v, got: %#v%s\n", *mm_want, mm_got, minimock.Diff(*mm_want, mm_got))
		}

		mm_results := mmDeleteHomeDir.DeleteHomeDirMock.defaultExpectation.results
		if mm_results == nil {
			mmDeleteHomeDir.t.Fatal("No results are set for the InterfaceMock.DeleteHomeDir")
		}
		return (*mm_results).err
	}
	if mmDeleteHomeDir.funcDeleteHomeDir != nil {
		return mmDeleteHomeDir.funcDeleteHomeDir(dir)
	}
	mmDeleteHomeDir.t.Fatalf("Unexpected call to InterfaceMock.DeleteHomeDir. %v", dir)
	return
}

// DeleteHomeDirAfterCounter returns a count of finished InterfaceMock.DeleteHomeDir invocations
func (mmDeleteHomeDir *InterfaceMock) DeleteHomeDirAfterCounter() uint64 {
	return mm_atomic.LoadUint64(&mmDeleteHomeDir.afterDeleteHomeDirCounter)
}

// DeleteHomeDirBeforeCounter returns a count of InterfaceMock.DeleteHomeDir invocations
func (mmDeleteHomeDir *InterfaceMock) DeleteHomeDirBeforeCounter() uint64 {
	return mm_atomic.LoadUint64(&mmDeleteHomeDir.beforeDeleteHomeDirCounter)
}

// Calls returns a list of arguments used in each call to InterfaceMock.DeleteHomeDir.
// The list is in the same order as the calls were made (i.e. recent calls have a higher index)
func (mmDeleteHomeDir *mInterfaceMockDeleteHomeDir) Calls() []*InterfaceMockDeleteHomeDirParams {
	mmDeleteHomeDir.mutex.RLock()

	argCopy := make([]*InterfaceMockDeleteHomeDirParams, len(mmDeleteHomeDir.callArgs))
	copy(argCopy, mmDeleteHomeDir.callArgs)

	mmDeleteHomeDir.mutex.RUnlock()

	return argCopy
}

// MinimockDeleteHomeDirDone returns true if the count of the DeleteHomeDir invocations corresponds
// the number of defined expectations
func (m *InterfaceMock) MinimockDeleteHomeDirDone() bool {
	for _, e := range m.DeleteHomeDirMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			return false
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.DeleteHomeDirMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterDeleteHomeDirCounter) < 1 {
		return false
	}
	// if func was set then invocations count should be greater than zero
	if m.funcDeleteHomeDir != nil && mm_atomic.LoadUint64(&m.afterDeleteHomeDirCounter) < 1 {
		return false
	}
	return true
}

// MinimockDeleteHomeDirInspect logs each unmet expectation
func (m *InterfaceMock) MinimockDeleteHomeDirInspect() {
	for _, e := range m.DeleteHomeDirMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			m.t.Errorf("Expected call to InterfaceMock.DeleteHomeDir with params: %#v", *e.params)
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.DeleteHomeDirMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterDeleteHomeDirCounter) < 1 {
		if m.DeleteHomeDirMock.defaultExpectation.params == nil {
			m.t.Error("Expected call to InterfaceMock.DeleteHomeDir")
		} else {
			m.t.Errorf("Expected call to InterfaceMock.DeleteHomeDir with params: %#v", *m.DeleteHomeDirMock.defaultExpectation.params)
		}
	}
	// if func was set then invocations count should be greater than zero
	if m.funcDeleteHomeDir != nil && mm_atomic.LoadUint64(&m.afterDeleteHomeDirCounter) < 1 {
		m.t.Error("Expected call to InterfaceMock.DeleteHomeDir")
	}
}

type mInterfaceMockPurgeUserLegacy struct {
	mock               *InterfaceMock
	defaultExpectation *InterfaceMockPurgeUserLegacyExpectation
	expectations       []*InterfaceMockPurgeUserLegacyExpectation

	callArgs []*InterfaceMockPurgeUserLegacyParams
	mutex    sync.RWMutex
}

// InterfaceMockPurgeUserLegacyExpectation specifies expectation struct of the Interface.PurgeUserLegacy
type InterfaceMockPurgeUserLegacyExpectation struct {
	mock    *InterfaceMock
	params  *InterfaceMockPurgeUserLegacyParams
	results *InterfaceMockPurgeUserLegacyResults
	Counter uint64
}

// InterfaceMockPurgeUserLegacyParams contains parameters of the Interface.PurgeUserLegacy
type InterfaceMockPurgeUserLegacyParams struct {
	username string
}

// InterfaceMockPurgeUserLegacyResults contains results of the Interface.PurgeUserLegacy
type InterfaceMockPurgeUserLegacyResults struct {
	err error
}

// Expect sets up expected params for Interface.PurgeUserLegacy
func (mmPurgeUserLegacy *mInterfaceMockPurgeUserLegacy) Expect(username string) *mInterfaceMockPurgeUserLegacy {
	if mmPurgeUserLegacy.mock.funcPurgeUserLegacy != nil {
		mmPurgeUserLegacy.mock.t.Fatalf("InterfaceMock.PurgeUserLegacy mock is already set by Set")
	}

	if mmPurgeUserLegacy.defaultExpectation == nil {
		mmPurgeUserLegacy.defaultExpectation = &InterfaceMockPurgeUserLegacyExpectation{}
	}

	mmPurgeUserLegacy.defaultExpectation.params = &InterfaceMockPurgeUserLegacyParams{username}
	for _, e := range mmPurgeUserLegacy.expectations {
		if minimock.Equal(e.params, mmPurgeUserLegacy.defaultExpectation.params) {
			mmPurgeUserLegacy.mock.t.Fatalf("Expectation set by When has same params: %#v", *mmPurgeUserLegacy.defaultExpectation.params)
		}
	}

	return mmPurgeUserLegacy
}

// Inspect accepts an inspector function that has same arguments as the Interface.PurgeUserLegacy
func (mmPurgeUserLegacy *mInterfaceMockPurgeUserLegacy) Inspect(f func(username string)) *mInterfaceMockPurgeUserLegacy {
	if mmPurgeUserLegacy.mock.inspectFuncPurgeUserLegacy != nil {
		mmPurgeUserLegacy.mock.t.Fatalf("Inspect function is already set for InterfaceMock.PurgeUserLegacy")
	}

	mmPurgeUserLegacy.mock.inspectFuncPurgeUserLegacy = f

	return mmPurgeUserLegacy
}

// Return sets up results that will be returned by Interface.PurgeUserLegacy
func (mmPurgeUserLegacy *mInterfaceMockPurgeUserLegacy) Return(err error) *InterfaceMock {
	if mmPurgeUserLegacy.mock.funcPurgeUserLegacy != nil {
		mmPurgeUserLegacy.mock.t.Fatalf("InterfaceMock.PurgeUserLegacy mock is already set by Set")
	}

	if mmPurgeUserLegacy.defaultExpectation == nil {
		mmPurgeUserLegacy.defaultExpectation = &InterfaceMockPurgeUserLegacyExpectation{mock: mmPurgeUserLegacy.mock}
	}
	mmPurgeUserLegacy.defaultExpectation.results = &InterfaceMockPurgeUserLegacyResults{err}
	return mmPurgeUserLegacy.mock
}

//Set uses given function f to mock the Interface.PurgeUserLegacy method
func (mmPurgeUserLegacy *mInterfaceMockPurgeUserLegacy) Set(f func(username string) (err error)) *InterfaceMock {
	if mmPurgeUserLegacy.defaultExpectation != nil {
		mmPurgeUserLegacy.mock.t.Fatalf("Default expectation is already set for the Interface.PurgeUserLegacy method")
	}

	if len(mmPurgeUserLegacy.expectations) > 0 {
		mmPurgeUserLegacy.mock.t.Fatalf("Some expectations are already set for the Interface.PurgeUserLegacy method")
	}

	mmPurgeUserLegacy.mock.funcPurgeUserLegacy = f
	return mmPurgeUserLegacy.mock
}

// When sets expectation for the Interface.PurgeUserLegacy which will trigger the result defined by the following
// Then helper
func (mmPurgeUserLegacy *mInterfaceMockPurgeUserLegacy) When(username string) *InterfaceMockPurgeUserLegacyExpectation {
	if mmPurgeUserLegacy.mock.funcPurgeUserLegacy != nil {
		mmPurgeUserLegacy.mock.t.Fatalf("InterfaceMock.PurgeUserLegacy mock is already set by Set")
	}

	expectation := &InterfaceMockPurgeUserLegacyExpectation{
		mock:   mmPurgeUserLegacy.mock,
		params: &InterfaceMockPurgeUserLegacyParams{username},
	}
	mmPurgeUserLegacy.expectations = append(mmPurgeUserLegacy.expectations, expectation)
	return expectation
}

// Then sets up Interface.PurgeUserLegacy return parameters for the expectation previously defined by the When method
func (e *InterfaceMockPurgeUserLegacyExpectation) Then(err error) *InterfaceMock {
	e.results = &InterfaceMockPurgeUserLegacyResults{err}
	return e.mock
}

// PurgeUserLegacy implements system.Interface
func (mmPurgeUserLegacy *InterfaceMock) PurgeUserLegacy(username string) (err error) {
	mm_atomic.AddUint64(&mmPurgeUserLegacy.beforePurgeUserLegacyCounter, 1)
	defer mm_atomic.AddUint64(&mmPurgeUserLegacy.afterPurgeUserLegacyCounter, 1)

	if mmPurgeUserLegacy.inspectFuncPurgeUserLegacy != nil {
		mmPurgeUserLegacy.inspectFuncPurgeUserLegacy(username)
	}

	mm_params := &InterfaceMockPurgeUserLegacyParams{username}

	// Record call args
	mmPurgeUserLegacy.PurgeUserLegacyMock.mutex.Lock()
	mmPurgeUserLegacy.PurgeUserLegacyMock.callArgs = append(mmPurgeUserLegacy.PurgeUserLegacyMock.callArgs, mm_params)
	mmPurgeUserLegacy.PurgeUserLegacyMock.mutex.Unlock()

	for _, e := range mmPurgeUserLegacy.PurgeUserLegacyMock.expectations {
		if minimock.Equal(e.params, mm_params) {
			mm_atomic.AddUint64(&e.Counter, 1)
			return e.results.err
		}
	}

	if mmPurgeUserLegacy.PurgeUserLegacyMock.defaultExpectation != nil {
		mm_atomic.AddUint64(&mmPurgeUserLegacy.PurgeUserLegacyMock.defaultExpectation.Counter, 1)
		mm_want := mmPurgeUserLegacy.PurgeUserLegacyMock.defaultExpectation.params
		mm_got := InterfaceMockPurgeUserLegacyParams{username}
		if mm_want != nil && !minimock.Equal(*mm_want, mm_got) {
			mmPurgeUserLegacy.t.Errorf("InterfaceMock.PurgeUserLegacy got unexpected parameters, want: %#v, got: %#v%s\n", *mm_want, mm_got, minimock.Diff(*mm_want, mm_got))
		}

		mm_results := mmPurgeUserLegacy.PurgeUserLegacyMock.defaultExpectation.results
		if mm_results == nil {
			mmPurgeUserLegacy.t.Fatal("No results are set for the InterfaceMock.PurgeUserLegacy")
		}
		return (*mm_results).err
	}
	if mmPurgeUserLegacy.funcPurgeUserLegacy != nil {
		return mmPurgeUserLegacy.funcPurgeUserLegacy(username)
	}
	mmPurgeUserLegacy.t.Fatalf("Unexpected call to InterfaceMock.PurgeUserLegacy. %v", username)
	return
}

// PurgeUserLegacyAfterCounter returns a count of finished InterfaceMock.PurgeUserLegacy invocations
func (mmPurgeUserLegacy *InterfaceMock) PurgeUserLegacyAfterCounter() uint64 {
	return mm_atomic.LoadUint64(&mmPurgeUserLegacy.afterPurgeUserLegacyCounter)
}

// PurgeUserLegacyBeforeCounter returns a count of InterfaceMock.PurgeUserLegacy invocations
func (mmPurgeUserLegacy *InterfaceMock) PurgeUserLegacyBeforeCounter() uint64 {
	return mm_atomic.LoadUint64(&mmPurgeUserLegacy.beforePurgeUserLegacyCounter)
}

// Calls returns a list of arguments used in each call to InterfaceMock.PurgeUserLegacy.
// The list is in the same order as the calls were made (i.e. recent calls have a higher index)
func (mmPurgeUserLegacy *mInterfaceMockPurgeUserLegacy) Calls() []*InterfaceMockPurgeUserLegacyParams {
	mmPurgeUserLegacy.mutex.RLock()

	argCopy := make([]*InterfaceMockPurgeUserLegacyParams, len(mmPurgeUserLegacy.callArgs))
	copy(argCopy, mmPurgeUserLegacy.callArgs)

	mmPurgeUserLegacy.mutex.RUnlock()

	return argCopy
}

// MinimockPurgeUserLegacyDone returns true if the count of the PurgeUserLegacy invocations corresponds
// the number of defined expectations
func (m *InterfaceMock) MinimockPurgeUserLegacyDone() bool {
	for _, e := range m.PurgeUserLegacyMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			return false
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.PurgeUserLegacyMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterPurgeUserLegacyCounter) < 1 {
		return false
	}
	// if func was set then invocations count should be greater than zero
	if m.funcPurgeUserLegacy != nil && mm_atomic.LoadUint64(&m.afterPurgeUserLegacyCounter) < 1 {
		return false
	}
	return true
}

// MinimockPurgeUserLegacyInspect logs each unmet expectation
func (m *InterfaceMock) MinimockPurgeUserLegacyInspect() {
	for _, e := range m.PurgeUserLegacyMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			m.t.Errorf("Expected call to InterfaceMock.PurgeUserLegacy with params: %#v", *e.params)
		}
	}

	// if default expectation was set then invocations count should be greater than zero
	if m.PurgeUserLegacyMock.defaultExpectation != nil && mm_atomic.LoadUint64(&m.afterPurgeUserLegacyCounter) < 1 {
		if m.PurgeUserLegacyMock.defaultExpectation.params == nil {
			m.t.Error("Expected call to InterfaceMock.PurgeUserLegacy")
		} else {
			m.t.Errorf("Expected call to InterfaceMock.PurgeUserLegacy with params: %#v", *m.PurgeUserLegacyMock.defaultExpectation.params)
		}
	}
	// if func was set then invocations count should be greater than zero
	if m.funcPurgeUserLegacy != nil && mm_atomic.LoadUint64(&m.afterPurgeUserLegacyCounter) < 1 {
		m.t.Error("Expected call to InterfaceMock.PurgeUserLegacy")
	}
}

// MinimockFinish checks that all mocked methods have been called the expected number of times
func (m *InterfaceMock) MinimockFinish() {
	if !m.minimockDone() {
		m.MinimockCreateHomeDirInspect()

		m.MinimockDeleteHomeDirInspect()

		m.MinimockPurgeUserLegacyInspect()
		m.t.FailNow()
	}
}

// MinimockWait waits for all mocked methods to be called the expected number of times
func (m *InterfaceMock) MinimockWait(timeout mm_time.Duration) {
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

func (m *InterfaceMock) minimockDone() bool {
	done := true
	return done &&
		m.MinimockCreateHomeDirDone() &&
		m.MinimockDeleteHomeDirDone() &&
		m.MinimockPurgeUserLegacyDone()
}
