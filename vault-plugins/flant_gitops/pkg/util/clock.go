package util

import "time"

type Clock interface {
	Now() time.Time
	Since(time.Time) time.Duration
}

func NewSystemClock() *SystemClock {
	return &SystemClock{}
}

type SystemClock struct{}

func (c *SystemClock) Now() time.Time {
	return time.Now()
}

func (c *SystemClock) Since(t time.Time) time.Duration {
	return time.Since(t)
}

// NewMockedClock provides mock for test purposes
func NewMockedClock(nowTime time.Time) (Clock, *MockClock) {
	mock := &MockClock{NowTime: nowTime}
	return mock, mock
}

type MockClock struct {
	NowTime time.Time
}

func (c *MockClock) Now() time.Time {
	return c.NowTime
}

func (c *MockClock) Since(t time.Time) time.Duration {
	return c.NowTime.Sub(t)
}

func (c *MockClock) SetNowTime(t time.Time) {
	c.NowTime = t
}
