package memdb

import (
	"fmt"
	"reflect"
)

type CustomTypeFieldIndexer struct {
	// Field represents the field of the object passed to actual table
	Field string
	// FromCustomType convert customTypeValue into []byte, used for writing and searching at index tree
	FromCustomType func(customTypeValue interface{}) ([]byte, error)
}

// FromObject used to evaluate values to put to index tree
func (c *CustomTypeFieldIndexer) FromObject(obj interface{}) (bool, []byte, error) {
	if c.FromCustomType == nil {
		panic("FromCustomType is mandatory field") // nolint:panic_check
	}
	v := reflect.ValueOf(obj)
	v = reflect.Indirect(v) // Dereference the pointer if any

	fv := v.FieldByName(c.Field)
	isPtr := fv.Kind() == reflect.Ptr
	fv = reflect.Indirect(fv)
	if !isPtr && !fv.IsValid() {
		return false, nil,
			fmt.Errorf("field '%s' for %#v is invalid %v ", c.Field, obj, isPtr)
	}

	if isPtr && !fv.IsValid() {
		val := ""
		return false, []byte(val), nil
	}

	val, err := c.FromCustomType(fv.Interface())
	if err != nil {
		return false, nil, err
	}
	if len(val) == 0 {
		return false, nil, nil
	}

	// Add the null character as a terminator
	val = append(val, '\x00')
	return true, val, nil
}

// FromArgs used to evaluate value for searching at index tree
func (c *CustomTypeFieldIndexer) FromArgs(args ...interface{}) ([]byte, error) {
	if c.FromCustomType == nil {
		panic("FromCustomType is mandatory field") // nolint:panic_check
	}
	if len(args) != 1 {
		return nil, fmt.Errorf("must provide only a single argument")
	}
	arg, err := c.FromCustomType(args[0])
	if err != nil {
		return nil, err
	}
	// Add the null character as a terminator
	arg = append(arg, '\x00')
	return arg, nil
}

type CustomTypeSliceFieldIndexer struct {
	// Field represents the field of the object passed to actual table
	Field string
	// FromCustomType convert customTypeValue into []byte, used for writing and searching at index tree
	FromCustomType func(customTypeValue interface{}) ([]byte, error)
}

func (c *CustomTypeSliceFieldIndexer) FromObject(obj interface{}) (bool, [][]byte, error) {
	if c.FromCustomType == nil {
		panic("FromCustomType is mandatory field") // nolint:panic_check
	}
	v := reflect.ValueOf(obj)
	v = reflect.Indirect(v) // Dereference the pointer if any

	fv := v.FieldByName(c.Field)
	if !fv.IsValid() {
		return false, nil,
			fmt.Errorf("field '%s' for %#v is invalid", c.Field, obj)
	}

	if fv.Kind() != reflect.Slice {
		return false, nil, fmt.Errorf("field '%s' is not a slice", c.Field)
	}

	length := fv.Len()
	vals := make([][]byte, 0, length)
	for i := 0; i < fv.Len(); i++ {
		val, err := c.FromCustomType(fv.Index(i).Interface())
		if err != nil {
			return false, nil, fmt.Errorf("field '%s' invalid evaluating:%w", c.Field, err)
		}
		if len(val) == 0 {
			continue
		}

		// Add the null character as a terminator
		val = append(val, '\x00')
		vals = append(vals, val)
	}
	if len(vals) == 0 {
		return false, nil, nil
	}
	return true, vals, nil
}

// FromArgs used to evaluate value for searching at index tree
func (c *CustomTypeSliceFieldIndexer) FromArgs(args ...interface{}) ([]byte, error) {
	if c.FromCustomType == nil {
		panic("FromCustomType is mandatory field") // nolint:panic_check
	}
	if len(args) != 1 {
		return nil, fmt.Errorf("must provide only a single argument")
	}
	arg, err := c.FromCustomType(args[0])
	if err != nil {
		return nil, err
	}
	// Add the null character as a terminator
	arg = append(arg, '\x00')
	return arg, nil
}
