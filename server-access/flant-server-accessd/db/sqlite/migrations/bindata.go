// Code generated for package main by go-bindata DO NOT EDIT. (@generated)
// sources:
// 1_initialize_schema.down.sql
// 1_initialize_schema.up.sql
// bindata.go
package migrations

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func bindataRead(data []byte, name string) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("Read %q: %v", name, err)
	}

	var buf bytes.Buffer
	_, err = io.Copy(&buf, gz)
	clErr := gz.Close()

	if err != nil {
		return nil, fmt.Errorf("Read %q: %v", name, err)
	}
	if clErr != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

type asset struct {
	bytes []byte
	info  os.FileInfo
}

type bindataFileInfo struct {
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
}

// Name return file name
func (fi bindataFileInfo) Name() string {
	return fi.name
}

// Size return file size
func (fi bindataFileInfo) Size() int64 {
	return fi.size
}

// Mode return file mode
func (fi bindataFileInfo) Mode() os.FileMode {
	return fi.mode
}

// Mode return file modify time
func (fi bindataFileInfo) ModTime() time.Time {
	return fi.modTime
}

// IsDir return file whether a directory
func (fi bindataFileInfo) IsDir() bool {
	return fi.mode&os.ModeDir != 0
}

// Sys return file is sys mode
func (fi bindataFileInfo) Sys() interface{} {
	return nil
}

var __1_initialize_schemaDownSql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x01\x00\x00\xff\xff\x00\x00\x00\x00\x00\x00\x00\x00")

func _1_initialize_schemaDownSqlBytes() ([]byte, error) {
	return bindataRead(
		__1_initialize_schemaDownSql,
		"1_initialize_schema.down.sql",
	)
}

func _1_initialize_schemaDownSql() (*asset, error) {
	bytes, err := _1_initialize_schemaDownSqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "1_initialize_schema.down.sql", size: 0, mode: os.FileMode(420), modTime: time.Unix(1627385334, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var __1_initialize_schemaUpSql = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x94\x90\xc1\x4a\x03\x31\x14\x45\xf7\xf9\x8a\x4b\x57\x2d\xf8\x07\x5d\x8d\xfa\x94\xe0\x98\x4a\x1a\x61\xba\x2a\xa1\x09\x93\xc0\xb4\x13\x12\xf2\xff\xe2\x4c\x1d\x13\x41\xa5\x77\x99\x77\x38\x79\xef\x3e\x48\x6a\x14\x41\x35\xf7\x2d\xa1\x8f\x63\x0e\x89\xad\x19\x00\x5c\xf4\xd9\x42\x51\xa7\xf0\x26\xf9\x6b\x23\x0f\x78\xa1\xc3\xdd\x34\xea\xbd\x01\xb8\x50\xf4\x4c\x12\x62\xa7\x20\xde\xdb\x96\x6d\xb6\x8c\x55\xba\x9c\x6c\xac\x6c\xd7\xfc\x94\x2e\x8a\xd9\x9e\x3f\xed\xd7\x7c\x7d\xb2\xa4\x46\xfb\x1b\x50\x7b\x1a\x53\xb1\xc0\xfc\xea\xc6\xb3\x35\x3e\x7e\xaf\x55\xa6\x16\x24\x67\x87\xa1\xbc\xe0\x77\xd4\xe9\xe4\xac\x39\x06\x9d\xd2\x7f\x68\x88\xfe\x72\xf2\x41\x0f\x7f\x59\xcb\x6a\xb9\x78\xa4\x0e\xfc\x69\x1a\x52\xc7\xf7\x6a\x8f\xd5\xd4\xf4\x31\x7b\xb3\xc2\x4e\xcc\xbd\xaf\xb3\x37\x9b\xed\x47\x00\x00\x00\xff\xff\x9e\x0b\xe8\x2b\xdf\x01\x00\x00")

func _1_initialize_schemaUpSqlBytes() ([]byte, error) {
	return bindataRead(
		__1_initialize_schemaUpSql,
		"1_initialize_schema.up.sql",
	)
}

func _1_initialize_schemaUpSql() (*asset, error) {
	bytes, err := _1_initialize_schemaUpSqlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "1_initialize_schema.up.sql", size: 479, mode: os.FileMode(420), modTime: time.Unix(1628595107, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _bindataGo = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x01\x00\x00\xff\xff\x00\x00\x00\x00\x00\x00\x00\x00")

func bindataGoBytes() ([]byte, error) {
	return bindataRead(
		_bindataGo,
		"bindata.go",
	)
}

func bindataGo() (*asset, error) {
	bytes, err := bindataGoBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "bindata.go", size: 0, mode: os.FileMode(420), modTime: time.Unix(1628596015, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

// Asset loads and returns the asset for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func Asset(name string) ([]byte, error) {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[cannonicalName]; ok {
		a, err := f()
		if err != nil {
			return nil, fmt.Errorf("Asset %s can't read by error: %v", name, err)
		}
		return a.bytes, nil
	}
	return nil, fmt.Errorf("Asset %s not found", name)
}

// MustAsset is like Asset but panics when Asset would return an error.
// It simplifies safe initialization of global variables.
func MustAsset(name string) []byte {
	a, err := Asset(name)
	if err != nil {
		panic("asset: Asset(" + name + "): " + err.Error())
	}

	return a
}

// AssetInfo loads and returns the asset info for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func AssetInfo(name string) (os.FileInfo, error) {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[cannonicalName]; ok {
		a, err := f()
		if err != nil {
			return nil, fmt.Errorf("AssetInfo %s can't read by error: %v", name, err)
		}
		return a.info, nil
	}
	return nil, fmt.Errorf("AssetInfo %s not found", name)
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
var _bindata = map[string]func() (*asset, error){
	"1_initialize_schema.down.sql": _1_initialize_schemaDownSql,
	"1_initialize_schema.up.sql":   _1_initialize_schemaUpSql,
	"bindata.go":                   bindataGo,
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
	for childName := range node.Children {
		rv = append(rv, childName)
	}
	return rv, nil
}

type bintree struct {
	Func     func() (*asset, error)
	Children map[string]*bintree
}

var _bintree = &bintree{nil, map[string]*bintree{
	"1_initialize_schema.down.sql": &bintree{_1_initialize_schemaDownSql, map[string]*bintree{}},
	"1_initialize_schema.up.sql":   &bintree{_1_initialize_schemaUpSql, map[string]*bintree{}},
	"bindata.go":                   &bintree{bindataGo, map[string]*bintree{}},
}}

// RestoreAsset restores an asset under the given directory
func RestoreAsset(dir, name string) error {
	data, err := Asset(name)
	if err != nil {
		return err
	}
	info, err := AssetInfo(name)
	if err != nil {
		return err
	}
	err = os.MkdirAll(_filePath(dir, filepath.Dir(name)), os.FileMode(0755))
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(_filePath(dir, name), data, info.Mode())
	if err != nil {
		return err
	}
	err = os.Chtimes(_filePath(dir, name), info.ModTime(), info.ModTime())
	if err != nil {
		return err
	}
	return nil
}

// RestoreAssets restores an asset under the given directory recursively
func RestoreAssets(dir, name string) error {
	children, err := AssetDir(name)
	// File
	if err != nil {
		return RestoreAsset(dir, name)
	}
	// Dir
	for _, child := range children {
		err = RestoreAssets(dir, filepath.Join(name, child))
		if err != nil {
			return err
		}
	}
	return nil
}

func _filePath(dir, name string) string {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	return filepath.Join(append([]string{dir}, strings.Split(cannonicalName, "/")...)...)
}
