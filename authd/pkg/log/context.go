package log

import (
	"context"
	"fmt"
	"path"
	"runtime"
	"strings"

	"github.com/sirupsen/logrus"
)

// Store log fields in context.
type (
	contextLoggerKey struct{}
)

// TODO can we left only one logger and use logrus hooks?
var (
	stdEntry = logrus.NewEntry(logrus.StandardLogger())
	dbg      = NewDebugLogger()
	dbgEntry = logrus.NewEntry(dbg)
)

// WithFields creates a new logger with merged fields if
// there is already a logger in context.
func WithFields(ctx context.Context, fields map[string]interface{}) context.Context {
	// Get logger from context and repack it with new fields.
	return context.WithValue(ctx, contextLoggerKey{}, GetLogger(ctx).WithFields(fields))
}

// WithLogger returns a new context with the provided logger. Use in
// combination with logger.WithField(s) for great effect.
func WithLogger(ctx context.Context, logger *logrus.Entry) context.Context {
	return context.WithValue(ctx, contextLoggerKey{}, logger)
}

func WithStandardLogger(ctx context.Context) context.Context {
	return context.WithValue(ctx, contextLoggerKey{}, stdEntry)
}

// WithDebugLogger returns a new context with the debug logger.
func WithDebugLogger(ctx context.Context) context.Context {
	return context.WithValue(ctx, contextLoggerKey{}, dbgEntry)
}

// GetLogger retrieves the current logger from the context. If no logger is
// available, the standard logger is returned.
func GetLogger(ctx context.Context) *logrus.Entry {
	logger := ctx.Value(contextLoggerKey{})
	if logger == nil {
		return stdEntry
	}
	return logger.(*logrus.Entry)
}

// Fields return fields from logger in context.
func Fields(ctx context.Context) map[string]interface{} {
	ctxLogger := GetLogger(ctx)
	return ctxLogger.Data
}

func NewDebugLogger() *logrus.Logger {
	logger := logrus.New()
	logrus.StandardLogger()
	logger.SetOutput(logrus.StandardLogger().Out)
	formatter := logrus.StandardLogger().Formatter
	if v, ok := formatter.(*logrus.TextFormatter); ok {
		v.CallerPrettyfier = DebugCallerPrettyfier
	}
	logger.SetFormatter(formatter)
	logger.SetReportCaller(true)
	logger.SetLevel(logrus.DebugLevel)
	return logger
}

func DebugLogger() *logrus.Logger {
	return dbg
}

func DebugCallerPrettyfier(f *runtime.Frame) (string, string) {
	dir, filename := path.Split(f.File)
	dir = strings.TrimSuffix(dir, "/")
	_, dir = path.Split(dir)
	filename = fmt.Sprintf("%s:%d", path.Join(dir, filename), f.Line)

	fn := f.Function
	idx := strings.LastIndex(fn, "/")
	if idx >= 0 {
		fn = fn[idx+1:]
	}

	return fn, filename
}
