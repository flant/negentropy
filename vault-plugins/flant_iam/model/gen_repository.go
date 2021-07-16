// +build ignore

package main

import (
	"flag"
	"log"
	"os"
	"strings"
	"text/template"
)

type data struct {
	Type       string
	ParentType string
	IDsuffix   string
	Var        string
	ID         string
	IDType     string
	TypeName   string
}

func main() {
	var d data
	flag.StringVar(&d.Type, "type", "", "Type to generate for, e.g. User")
	flag.StringVar(&d.ParentType, "parentType", "", "Parent type, e.g. Tenant.")
	flag.StringVar(&d.IDsuffix, "IDsuffix", "UUID", "The suffix of the ID field, e.g Name")
	flag.Parse()

	d.Var = strings.ToLower(d.Type)
	d.ID = d.Var + d.IDsuffix
	d.IDType = d.Type + d.IDsuffix
	d.TypeName = d.Type + "Type"

	funcMap := template.FuncMap{
		"ToLower": strings.ToLower,
	}

	t := template.Must(template.New("repo").Funcs(funcMap).Parse(repoTemplate))

	filename := strings.ToLower(d.Type) + "_repository.generated.go"
	out, err := os.Create(filename)
	if err != nil {
		log.Fatalf("cannot create file %q: %v", filename, err)
	}
	defer out.Close()

	t.Execute(out, d)
}

var repoTemplate = `// DO NOT EDIT
// This file was generated automatically with 
// 		go run gen_repository.go -type {{.Type}} {{- if .ParentType }}-parentType {{.ParentType}}{{end}}
// 

package model

import (
	"encoding/json"
	"github.com/flant/negentropy/vault-plugins/shared/io"
)

type {{.IDType}} = string 

const {{.TypeName}} = "{{ .Var }}" // also, memdb schema name

func (u *{{.Type}}) ObjType() string {
	return {{.TypeName}}
}

func (u *{{.Type}}) ObjId() string {
	return u.{{.IDsuffix}}
}

type {{.Type}}Repository struct {
	db *io.MemoryStoreTxn // called "db" not to provoke transaction semantics
}

func New{{.Type}}Repository(tx *io.MemoryStoreTxn) *{{.Type}}Repository {
	return &{{.Type}}Repository{db: tx}
}

func (r *{{.Type}}Repository) save({{ .Var }} *{{.Type}}) error {
	return r.db.Insert({{.Type}}Type, {{ .Var }})
}

func (r *{{.Type}}Repository) Create({{ .Var }} *{{.Type}}) error {
	return r.save({{.Var}})
}

func (r *{{.Type}}Repository) GetRawByID(id {{.IDType}}) (interface{}, error) {
	raw, err := r.db.First({{.Type}}Type, PK, id)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, ErrNotFound
	}
	return raw, nil
}

func (r *{{.Type}}Repository) GetByID(id {{.IDType}}) (*{{.Type}}, error) {
	raw, err := r.GetRawByID(id)
	if raw == nil {
		return nil, err
	}
	return raw.(*{{.Type}}), err
}

func (r *{{.Type}}Repository) Update({{ .Var }} *{{.Type}}) error {
	_, err := r.GetByID({{.Var}}.{{.IDsuffix}})
	if err != nil {
		return err
	}
	return r.save({{.Var}})
}

func (r *{{.Type}}Repository) Delete(id {{.IDType}}) error {
	{{ .Var }}, err := r.GetByID(id)
	if err != nil {
		return err
	}
	return r.db.Delete({{.Type}}Type, {{ .Var }})
}

func (r *{{.Type}}Repository) List({{- if .ParentType }}{{.ParentType | ToLower }}UUID {{.ParentType}}UUID{{end}}) ([]*{{.Type}}, error) {
	{{if .ParentType }}
	iter, err := r.db.Get({{.TypeName}}, {{.ParentType}}ForeignPK, {{.ParentType | ToLower }}UUID)
	{{else}}
	iter, err := r.db.Get({{.TypeName}}, PK)
	{{end}}
	if err != nil {
		return nil, err
	}

	list := []*{{.Type}}{}
	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		obj := raw.(*{{.Type}})
		list = append(list, obj)
	}
	return list, nil
}

func (r *{{.Type}}Repository) ListIDs({{- if .ParentType }}{{.ParentType | ToLower }}ID {{.ParentType}}UUID{{end}}) ([]{{.IDType}}, error) {
	objs, err := r.List({{- if .ParentType }}{{.ParentType | ToLower }}ID{{end}})
	if err != nil {
		return nil, err
	}
	ids := make([]{{.IDType}}, len(objs))
	for i := range objs {
		ids[i] = objs[i].ObjId()
	}
	return ids, nil
}

func (r *{{.Type}}Repository) Iter(action func(*{{.Type}}) (bool, error)) error {
	iter, err := r.db.Get({{.TypeName}}, PK)
	if err != nil {
		return err
	}

	for {
		raw := iter.Next()
		if raw == nil {
			break
		}
		obj := raw.(*{{.Type}})
		next, err := action(obj)
		if err != nil {
			return err
		}

		if !next {
			break
		}
	}

	return nil
}

func (r *{{.Type}}Repository) Sync(objID string, data []byte) error {
	if data == nil {
		return r.Delete(objID)
	}

	{{.Var}} := &{{.Type}}{}
	err := json.Unmarshal(data, {{.Var}})
	if err != nil {
		return err
	}

	return r.save({{.Var}})
}
`
