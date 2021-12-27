package log

import (
	"context"
	"testing"
)

// No assertions, just print some messages.
func Test_GetLogger(t *testing.T) {
	parentCtx := context.Background()

	logger := GetLogger(parentCtx)
	if logger == nil {
		t.Fatalf("logger should not be nil")
	}

	logger.Errorf("azaza")

	// fields
	subCtx := WithFields(parentCtx, map[string]interface{}{"component": "main", "app": "test"})
	GetLogger(subCtx).Errorf("from sub")

	subSubCtx := WithFields(subCtx, map[string]interface{}{"component": "sub"})
	GetLogger(subSubCtx).Errorf("from subsub")
}
