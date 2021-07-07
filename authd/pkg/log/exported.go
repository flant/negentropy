package log

import "context"

// These helpers return functions because getCaller in logrus can't skip frames.

func Errorf(ctx context.Context) func(format string, args ...interface{}) {
	return GetLogger(ctx).Errorf
}

func Infof(ctx context.Context) func(format string, args ...interface{}) {
	return GetLogger(ctx).Infof
}

func Debugf(ctx context.Context) func(format string, args ...interface{}) {
	return GetLogger(ctx).Debugf
}
