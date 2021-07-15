package model

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestSlice(t *testing.T) {
	t.Run("value slice by value marshaller", func(t *testing.T) {
		dc1 := test1Container{
			Public:    "public data",
			Sensitive: "secret data",
		}

		slice1 := arrayOfData{Items: []test1Container{dc1}}

		data, err := json.Marshal(OmitSensitive(slice1))
		require.NoError(t, err)
		var res map[string][]map[string]string
		_ = json.Unmarshal(data, &res)
		items := res["items"]

		require.Len(t, items, 1)
		assert.Equal(t, "public data", items[0]["public"])
		assert.Equal(t, "", items[0]["sens"])
	})

	t.Run("pointer slice by value marshaller", func(t *testing.T) {
		dc1 := &test1Container{
			Public:    "t1",
			Sensitive: "t2",
		}

		slice1 := arrayOfPointer{Items: []*test1Container{dc1}}

		data, err := json.Marshal(OmitSensitive(slice1))
		require.NoError(t, err)
		var res map[string][]map[string]string
		_ = json.Unmarshal(data, &res)
		items := res["items"]
		require.Len(t, items, 1)
		assert.Equal(t, "", items[0]["sens"])
		assert.Equal(t, "t1", items[0]["public"])
	})

	t.Run("value slice with pointer marshaller", func(t *testing.T) {
		dc1 := test2Container{
			Public:    "t1",
			Sensitive: "t2",
		}

		slice1 := arrayOfData{T2Items: []test2Container{dc1}}

		data, err := json.Marshal(OmitSensitive(slice1))
		require.NoError(t, err)
		var res map[string][]map[string]string
		_ = json.Unmarshal(data, &res)
		items := res["t2items"]
		require.Len(t, items, 1)
		assert.Equal(t, "", items[0]["sens"])
		assert.Equal(t, "t1", items[0]["public"])
	})

	t.Run("pointer slice with pointer marshaller", func(t *testing.T) {
		dc1 := &test2Container{
			Public:    "t1",
			Sensitive: "t2",
		}

		slice1 := arrayOfPointer{T2Items: []*test2Container{dc1}}

		data, err := json.Marshal(OmitSensitive(slice1))
		require.NoError(t, err)
		var res map[string][]map[string]string
		_ = json.Unmarshal(data, &res)
		items := res["t2items"]
		require.Len(t, items, 1)
		assert.Equal(t, "", items[0]["sens"])
		assert.Equal(t, "t1", items[0]["public"])
	})
}

type test1Container struct {
	Public    string `json:"public"`
	Sensitive string `json:"sens" sensitive:""`
}

func (dc test1Container) ObjType() string {
	return "test1Container"
}

func (dc test1Container) ObjId() string {
	return dc.Public
}

type test2Container struct {
	Public    string `json:"public"`
	Sensitive string `json:"sens" sensitive:""`
}

func (dc test2Container) ObjType() string {
	return "test2Container"
}

func (dc test2Container) ObjId() string {
	return dc.Public
}

type arrayOfData struct {
	Items   []test1Container `json:"items"`
	T2Items []test2Container `json:"t2items"`
}

func (dc arrayOfData) ObjType() string {
	return "arrayOfData"
}

func (dc arrayOfData) ObjId() string {
	return "1"
}

type arrayOfPointer struct {
	Items   []*test1Container `json:"items"`
	T2Items []*test2Container `json:"t2items"`
}

func (dc arrayOfPointer) ObjType() string {
	return "arrayOfPointer"
}

func (dc arrayOfPointer) ObjId() string {
	return "2"
}

