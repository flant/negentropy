package model

import (
	"reflect"
)

// OmitSensitive makes shallow copy and omits fields with "sensitive" tag regardless its value.
// To make deeper filtration of sensitive fields, implement Marshall(includeSensitive bool) method
// on nested structs or support recursive walk in this function.
func OmitSensitive(obj interface{}) interface{} {
	const key = "sensitive"

	t := reflect.TypeOf(obj)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	src := reflect.ValueOf(obj)
	if src.Kind() == reflect.Ptr {
		src = src.Elem()
	}
	dst := reflect.New(t).Elem()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		// skip self sensitive values
		_, isSensitive := field.Tag.Lookup(key)
		if isSensitive {
			continue
		}

		// next code check nested structs sensitive data
		kind := field.Type.Kind()
		switch kind {
		case reflect.Slice:
			omitSlice(src.Field(i))
			dst.Field(i).Set(src.Field(i))

		case reflect.Map:
			// relfecting map for embedded extensions
			omitMap(src.Field(i))
			dst.Field(i).Set(src.Field(i))

		default:
			dst.Field(i).Set(src.Field(i))
		}
	}

	return dst.Interface()
}

func omitMap(f reflect.Value) {
	iter := f.MapRange()
	for iter.Next() {
		if _, ok := iter.Value().Interface().(sensitiveObjectHolder); ok {
			newValue := OmitSensitive(iter.Value().Elem().Interface())
			p := reflect.New(reflect.TypeOf(newValue))
			p.Elem().Set(reflect.ValueOf(newValue))
			f.SetMapIndex(iter.Key(), p)
		}
	}
}

func omitSlice(f reflect.Value) {
	for si := 0; si < f.Len(); si++ {
		value := f.Index(si)

		_, isValueM := value.Interface().(sensitiveObjectHolder)
		_, isPointerM := value.Addr().Interface().(sensitiveObjectHolder)

		if isValueM || isPointerM {
			if value.Kind() == reflect.Ptr {
				value = value.Elem()
			}
			newValue := OmitSensitive(value.Interface())
			newValueType := reflect.TypeOf(newValue)
			newValueR := reflect.ValueOf(newValue)
			p := reflect.New(newValueType)
			if newValueType.Kind() == reflect.Struct {
				p = newValueR
			} else {
				p.Elem().Set(newValueR)
			}
			value.Set(p)
		}
	}
}

type sensitiveObjectHolder interface {
	ObjType() string
	ObjId() string
}
