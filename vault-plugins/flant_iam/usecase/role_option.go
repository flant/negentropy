package usecase

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/getkin/kin-openapi/openapi3"
)

// checkRoleOptionSchema check is schemaJson valid openApi specification
func checkRoleOptionSchema(schemaJson string) error {
	_, err := buildSchema(schemaJson)
	return err
}

func buildSchema(schemaJson string) (*openapi3.Schema, error) {
	if schemaJson == "" {
		return &openapi3.Schema{}, nil
	}
	var schema openapi3.Schema
	err := json.Unmarshal([]byte(schemaJson), &schema)
	if err != nil {
		return nil, fmt.Errorf("option_schema unmarshalling: %w", err)
	}
	err = schema.Validate(context.TODO())
	if err != nil {
		return nil, fmt.Errorf("option_schema validation: %w", err)
	}
	return &schema, nil

}

// checkBackwardsCompatibility checks that new one doesn't demands new required options and doesn't change old types and formats
func checkBackwardsCompatibility(oldSchemaJson, newSchemaJson string) error {
	if oldSchemaJson == newSchemaJson {
		return nil
	}
	oldSchema, err := buildSchema(oldSchemaJson)
	if err != nil {
		return err
	}
	newSchema, err := buildSchema(newSchemaJson)
	if err != nil {
		return err
	}
	if err = checkRequired(oldSchema.Required, newSchema.Required); err != nil {
		return fmt.Errorf("check role option shema backwords compatibility: %w", err)
	}
	if err = checkPropertyTypes(oldSchema.Properties, newSchema.Properties); err != nil {
		return fmt.Errorf("check role option shema backwords compatibility: %w", err)
	}
	return nil
}

func checkRequired(oldRequireds []string, newRequireds []string) error {
	if len(newRequireds) > len(oldRequireds) {
		return fmt.Errorf("new schema has more required properties then old one")
	}
	for _, newRequired := range newRequireds {
		exists := false
		for _, oldRequired := range oldRequireds {
			if newRequired == oldRequired {
				exists = true
				break
			}
		}
		if !exists {
			return fmt.Errorf("property %q in new shema required is new, it is forbidden", newRequired)
		}
	}
	return nil
}

func checkPropertyTypes(oldProperties openapi3.Schemas, newProperties openapi3.Schemas) error {
	if len(oldProperties) > len(newProperties) {
		return fmt.Errorf("new schema has less described properties then old one")
	}

	for name, oldProperty := range oldProperties {
		newProperty, exists := newProperties[name]
		if !exists {
			return fmt.Errorf("property %q in not presented in new schema, it is forbidden", name)
		}
		if newProperty.Value.Type != oldProperty.Value.Type {
			return fmt.Errorf("property %q has changed type in new schema, it is forbidden", name)
		}
		if newProperty.Value.Format != oldProperty.Value.Format {
			return fmt.Errorf("property %q has changed format in new schema, it is forbidden", name)
		}
		if newProperty.Value.Pattern != oldProperty.Value.Pattern {
			return fmt.Errorf("property %q has changed patter in new schema, it is forbidden", name)
		}
	}
	return nil
}
