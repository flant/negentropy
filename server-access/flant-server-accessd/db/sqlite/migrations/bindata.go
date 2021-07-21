package migrations

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"strings"
)

func bindata_read(data []byte, name string) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("Read %q: %v", name, err)
	}

	var buf bytes.Buffer
	_, err = io.Copy(&buf, gz)
	gz.Close()

	if err != nil {
		return nil, fmt.Errorf("Read %q: %v", name, err)
	}

	return buf.Bytes(), nil
}

var __1_initialize_schema_down_sql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x01\x00\x00\xff\xff\x00\x00\x00\x00\x00\x00\x00\x00")

func _1_initialize_schema_down_sql() ([]byte, error) {
	return bindata_read(
		__1_initialize_schema_down_sql,
		"1_initialize_schema.down.sql",
	)
}

var __1_initialize_schema_up_sql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x94\x90\xc1\x0a\x82\x40\x10\x86\xef\xfb\x14\xff\xb1\xa0\x37\xe8\x64\x31\x84\x64\x16\xcb\x04\x79\x0a\xc9\xc1\x15\x34\xc5\xc1\xf7\x0f\xd3\xcc\x0d\x3c\xf4\x1d\x77\x3f\xbe\x9d\x9d\xbd\xa5\x80\x09\x1c\xec\x22\x42\xde\xd6\x5d\xa3\x66\x65\x00\xe0\x99\x56\x02\xa6\x1b\xe3\x62\xc3\x53\x60\x13\x1c\x29\xd9\xbc\xaf\xf2\x22\x03\xc2\x98\xe9\x40\x16\xf1\x99\x11\x5f\xa3\xc8\xac\xb7\xc6\x78\xb9\x4e\xa5\xf5\x6a\x23\xbf\xd1\x29\x31\xd4\xbb\xbe\x3e\xf2\x79\x64\xc2\x57\xf3\x3f\x54\x79\xd4\x3a\x1b\x60\x38\x75\x75\x25\x59\xd1\x7e\xc7\x9a\xe3\x07\xd4\x49\x59\xce\x7f\xb0\xac\xba\x54\x9d\x64\xf7\x26\x55\x5d\x56\xfb\x7d\xbd\x02\x00\x00\xff\xff\x37\x24\xde\x6d\x7e\x01\x00\x00")

func _1_initialize_schema_up_sql() ([]byte, error) {
	return bindata_read(
		__1_initialize_schema_up_sql,
		"1_initialize_schema.up.sql",
	)
}

// Asset loads and returns the asset for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func Asset(name string) ([]byte, error) {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[cannonicalName]; ok {
		return f()
	}
	return nil, fmt.Errorf("Asset %s not found", name)
}

// AssetNames returns the names of the assets.
func AssetNames() []string {
	names := make([]string, 0, len(_bindata))
	for name := range _bindata {
		names = append(names, name)
	}
	return names
}

// _bindata is a table, holding each asset generator, mapped to its name.
var _bindata = map[string]func() ([]byte, error){
	"1_initialize_schema.down.sql": _1_initialize_schema_down_sql,
	"1_initialize_schema.up.sql":   _1_initialize_schema_up_sql,
}

// AssetDir returns the file names below a certain
// directory embedded in the file by go-bindata.
// For example if you run go-bindata on data/... and data contains the
// following hierarchy:
//     data/
//       foo.txt
//       img/
//         a.png
//         b.png
// then AssetDir("data") would return []string{"foo.txt", "img"}
// AssetDir("data/img") would return []string{"a.png", "b.png"}
// AssetDir("foo.txt") and AssetDir("notexist") would return an error
// AssetDir("") will return []string{"data"}.
func AssetDir(name string) ([]string, error) {
	node := _bintree
	if len(name) != 0 {
		cannonicalName := strings.Replace(name, "\\", "/", -1)
		pathList := strings.Split(cannonicalName, "/")
		for _, p := range pathList {
			node = node.Children[p]
			if node == nil {
				return nil, fmt.Errorf("Asset %s not found", name)
			}
		}
	}
	if node.Func != nil {
		return nil, fmt.Errorf("Asset %s not found", name)
	}
	rv := make([]string, 0, len(node.Children))
	for name := range node.Children {
		rv = append(rv, name)
	}
	return rv, nil
}

type _bintree_t struct {
	Func     func() ([]byte, error)
	Children map[string]*_bintree_t
}

var _bintree = &_bintree_t{nil, map[string]*_bintree_t{
	"1_initialize_schema.down.sql": &_bintree_t{_1_initialize_schema_down_sql, map[string]*_bintree_t{}},
	"1_initialize_schema.up.sql":   &_bintree_t{_1_initialize_schema_up_sql, map[string]*_bintree_t{}},
}}
