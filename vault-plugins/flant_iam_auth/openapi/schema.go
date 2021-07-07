package openapi

import (
	"fmt"
	"github.com/go-openapi/spec"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/validate"
	"github.com/go-openapi/validate/post"
	"github.com/hashicorp/go-multierror"
	"gopkg.in/yaml.v2"
)

// Validator interface
// need for hide implementation of openapi validation
type Validator interface {
	Validate(data interface{}) (interface{}, error)
}

type validatorOpenApi2 struct {
	validator *validate.SchemaValidator
}

func (v *validatorOpenApi2) Validate(data interface{}) (interface{}, error) {
	result := v.validator.Validate(data)
	if result.IsValid() {
		// Add default values from openAPISpec
		post.ApplyDefaults(result)
		return result.Data(), nil
	}

	var allErrs *multierror.Error
	allErrs = multierror.Append(allErrs, result.Errors...)

	return false, allErrs.ErrorOrNil()
}

func SchemaValidator(content string) (Validator, error) {
	byteContent := []byte(content)
	schema := new(spec.Schema)
	if err := yaml.Unmarshal(byteContent, schema); err != nil {
		return nil, fmt.Errorf("json unmarshal: %v", err)
	}

	err := spec.ExpandSchema(schema, schema, nil)
	if err != nil {
		return nil, fmt.Errorf("expand the schema: %v", err)
	}

	validator := validate.NewSchemaValidator(schema, nil, "", strfmt.Default)
	return &validatorOpenApi2{validator: validator}, nil
}
