package server

import (
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/sirupsen/logrus"
)

// StructuredLogger is a simple, but powerful implementation of a custom structured
// logger backed on logrus. It is adapted from https://github.com/go-chi/chi
// 'logging' example.

func NewStructuredLogger(socketPath string) func(next http.Handler) http.Handler {
	name := filepath.Base(socketPath)
	name = strings.TrimSuffix(name, ".yaml")
	return middleware.RequestLogger(&StructuredLogger{
		name,
	})
}

type StructuredLogger struct {
	Name string
}

func (l *StructuredLogger) NewLogEntry(r *http.Request) middleware.LogEntry {
	entry := &StructuredLoggerEntry{
		Name: l.Name,
		Fields: map[string]string{
			"uri":    r.RequestURI,
			"method": r.Method,
		},
	}
	return entry
}

type StructuredLoggerEntry struct {
	Name   string
	Fields map[string]string
}

func (l *StructuredLoggerEntry) Write(status, bytes int, _ http.Header, elapsed time.Duration, extra interface{}) {
	logrus.Infof("%s: %s %s %d",
		l.Name,
		l.Fields["method"],
		l.Fields["uri"],
		status,
	)
}

// This will log panics to log
func (l *StructuredLoggerEntry) Panic(v interface{}, stack []byte) {
	logrus.Infof("%s: %s %s panic: %v\n%s",
		l.Name,
		l.Fields["method"],
		l.Fields["uri"],
		v,
		string(stack),
	)
}
