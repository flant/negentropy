package config

import (
	"encoding/json"
	"fmt"

	"github.com/go-openapi/spec"
	"github.com/go-openapi/swag"
)

var Schemas = map[string]string{
	"AuthdConfig/v1": `
type: object
additionalProperties: false
minProperties: 4
properties:
  apiVersion:
    title: apiVersion
    description: |
      API domain and version
    type: string
  kind:
    title: kind
    description: |
      Config kind
    type: string
  jwtPath:
    type: string
  defaultSocketDirectory:
    description: |
      A path where all server sockets are created.
    type: string
  servers:
    type: array
    additionalItems: false
    minItems: 1
    items:
      type: object
      additionalProperties: false
      required:
      - domain
      - type
      properties:
        type:
          description: |
            Vault server type
          type: string
          enum:
          - Auth
          - RootSource
          - Test
        domain:
          description: |
            Vault server address
          type: string
        allowRedirects:
          description: |
            Vault server address
          type: array
          items:
            type: string
`,
	"AuthdSocketConfig/v1": `
type: object
additionalProperties: false
minProperties: 4
properties:
  apiVersion:
    title: apiVersion
    description: |
      API domain and version
    type: string
  kind:
    title: kind
    description: |
      Config kind
    type: string
  path:
    type: string
  user:
    type: string
  group:
    type: string
  mode:
    type: number
  allowedRoles:
    type: array
    additionalItems: false
    minItems: 1
    items:
      type: object
      additionalProperties: false
      required:
      - role
      properties:
        role:
          type: string
`,
}

var SchemasCache = map[string]*spec.Schema{}

// GetSchema returns loaded schema.
func GetSchema(name string) *spec.Schema {
	if s, ok := SchemasCache[name]; ok {
		return s
	}
	if _, ok := Schemas[name]; !ok {
		return nil
	}

	// ignore error because load is guaranteed by tests
	SchemasCache[name], _ = LoadSchema(name)
	return SchemasCache[name]
}

// LoadSchema returns spec.Schema object loaded from yaml in Schemas map.
func LoadSchema(name string) (*spec.Schema, error) {
	yml, err := swag.BytesToYAMLDoc([]byte(Schemas[name]))
	if err != nil {
		return nil, fmt.Errorf("yaml unmarshal: %v", err)
	}
	d, err := swag.YAMLToJSON(yml)
	if err != nil {
		return nil, fmt.Errorf("yaml to json: %v", err)
	}

	s := new(spec.Schema)

	if err := json.Unmarshal(d, s); err != nil {
		return nil, fmt.Errorf("json unmarshal: %v", err)
	}

	err = spec.ExpandSchema(s, s, nil /*new(noopResCache)*/)
	if err != nil {
		return nil, fmt.Errorf("expand schema: %v", err)
	}

	return s, nil
}
