package model

import (
	"reflect"
)

type Marshaller interface {
	Marshal(bool) ([]byte, error)
	Unmarshal([]byte) error
}

type SensitiveMarshaller interface {
	Marshal(bool) ([]byte, error)
}

// OmitSensitive makes shallow copy and omits fields with "sensitive" tag regardless its value.
// To make deeper filtration of sensitive fields, implement Marshall(includeSensitive bool) method
// on nested structs or support recursive walk in this function.
func OmitSensitive(obj interface{}) interface{} {
	const key = "sensitive"

	t := reflect.TypeOf(obj)
	src := reflect.ValueOf(obj)
	dst := reflect.New(t).Elem()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		kind := field.Type.Kind()
		switch kind {
		case reflect.Map:
			// relfecting map for embedded extensions
			m := src.Field(i)
			iter := m.MapRange()
			for iter.Next() {
				if _, ok := iter.Value().Interface().(SensitiveMarshaller); ok {
					newValue := OmitSensitive(iter.Value().Elem().Interface())
					p := reflect.New(reflect.TypeOf(newValue))
					p.Elem().Set(reflect.ValueOf(newValue))
					m.SetMapIndex(iter.Key(), p)
				}
			}
			fallthrough

		default:
			_, isSensitive := field.Tag.Lookup(key)
			if isSensitive {
				continue
			}
			dst.Field(i).Set(src.Field(i))
		}
	}

	return dst.Interface()
}
