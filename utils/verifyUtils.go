package utils

import "reflect"

func IsPointer(v interface{}) bool {
	typeOf := reflect.TypeOf(v)
	if typeOf == nil {
		return false
	}
	kind := typeOf.Kind()
	return kind == reflect.Ptr
}
