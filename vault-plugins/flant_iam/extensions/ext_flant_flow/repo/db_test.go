package repo

import (
	"testing"
)

func Test_MergeSchema(t *testing.T) {
	_, err := GetSchema()
	if err != nil {
		t.Fatalf("merged schema is invalid: %v", err)
	}
}
