package model

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/hashicorp/vault/sdk/helper/jsonutil"
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
			Public:    "t1",
			Sensitive: "t2",
		}

		slice1 := arrayOfData{Items: []test1Container{dc1}}

		data, err := slice1.Marshal(false)
		require.NoError(t, err)
		var res map[string][]map[string]string
		_ = json.Unmarshal(data, &res)
		items := res["items"]
		require.Len(t, items, 1)
		assert.Equal(t, "", items[0]["sens"])
		assert.Equal(t, "t1", items[0]["public"])
	})

	t.Run("pointer slice by value marshaller", func(t *testing.T) {
		dc1 := &test1Container{
			Public:    "t1",
			Sensitive: "t2",
		}

		slice1 := arrayOfPointer{Items: []*test1Container{dc1}}

		data, err := slice1.Marshal(false)
		require.NoError(t, err)
		var res map[string][]map[string]string
		_ = json.Unmarshal(data, &res)
		items := res["items"]
		require.Len(t, items, 1)
		assert.Equal(t, "", items[0]["sens"])
		assert.Equal(t, "t1", items[0]["public"])
	})

	t.Run("value slice with pointer marshaler", func(t *testing.T) {
		dc1 := test2Container{
			Public:    "t1",
			Sensitive: "t2",
		}

		slice1 := arrayOfData{T2Items: []test2Container{dc1}}

		data, err := slice1.Marshal(false)
		require.NoError(t, err)
		var res map[string][]map[string]string
		_ = json.Unmarshal(data, &res)
		items := res["t2items"]
		require.Len(t, items, 1)
		assert.Equal(t, "", items[0]["sens"])
		assert.Equal(t, "t1", items[0]["public"])
	})

	t.Run("pointer slice with pointer marshaler", func(t *testing.T) {
		dc1 := &test2Container{
			Public:    "t1",
			Sensitive: "t2",
		}

		slice1 := arrayOfPointer{T2Items: []*test2Container{dc1}}

		data, err := slice1.Marshal(false)
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

func (dc test1Container) Marshal(includeSensitive bool) ([]byte, error) {
	obj := dc
	if !includeSensitive {
		dc := OmitSensitive(dc).(test1Container)
		obj = dc
	}
	return jsonutil.EncodeJSON(obj)
}

type test2Container struct {
	Public    string `json:"public"`
	Sensitive string `json:"sens" sensitive:""`
}

func (dc *test2Container) Marshal(includeSensitive bool) ([]byte, error) {
	obj := dc
	if !includeSensitive {
		dc := OmitSensitive(dc).(test2Container)
		obj = &dc
	}
	return jsonutil.EncodeJSON(obj)
}

type arrayOfData struct {
	Items   []test1Container `json:"items"`
	T2Items []test2Container `json:"t2items"`
}

func (a arrayOfData) Marshal(includeSensitive bool) ([]byte, error) {
	obj := a
	if !includeSensitive {
		a := OmitSensitive(a).(arrayOfData)
		obj = a
	}
	return jsonutil.EncodeJSON(obj)
}

type arrayOfPointer struct {
	Items   []*test1Container `json:"items"`
	T2Items []*test2Container `json:"t2items"`
}

func (a arrayOfPointer) Marshal(includeSensitive bool) ([]byte, error) {
	obj := a
	if !includeSensitive {
		a := OmitSensitive(a).(arrayOfPointer)
		obj = a
	}
	return jsonutil.EncodeJSON(obj)
}
