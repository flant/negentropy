package model

import (
	"encoding/json"
	"reflect"
	"testing"
)

type sensitiveType struct {
	String    string
	Slice     []string
	StructPtr *sensitiveType

	SensitiveString    string         `sensitive:""`
	SensitiveSlice     []string       `sensitive:""`
	SensitiveStructPtr *sensitiveType `sensitive:""`
}

func Test_OmitSensitive(t *testing.T) {
	flipflop := func(in sensitiveType) sensitiveType {
		b, err := json.Marshal(in)
		if err != nil {
			t.Fatalf("cannot marshal: %v", err)
		}

		var restored sensitiveType

		err = json.Unmarshal(b, &restored)
		if err != nil {
			t.Fatalf("cannot unmarshal: %v", err)
		}

		return restored
	}

	initial := sensitiveType{
		String:    "str",
		Slice:     []string{"s", "l", "i", "c", "e"},
		StructPtr: &sensitiveType{String: "substr"},

		SensitiveString:    "sense-str",
		SensitiveSlice:     []string{"s", "e", "n", "s", "e", "-", "s", "l", "i", "c", "e"},
		SensitiveStructPtr: &sensitiveType{String: "sense-substr"},
	}

	{
		restored := flipflop(initial)

		if !reflect.DeepEqual(initial, restored) {
			t.Fatalf("structs not equal: was=%v, got=%v", initial, restored)
		}
	}

	{

		cleaned := OmitSensitive(initial).(sensitiveType)
		restored := flipflop(cleaned)

		initial.SensitiveString = ""
		initial.SensitiveSlice = nil
		initial.SensitiveStructPtr = nil

		if !reflect.DeepEqual(initial, restored) {
			t.Fatalf("structs not equal: was=%v, got=%v", initial, restored)
		}
	}
}
