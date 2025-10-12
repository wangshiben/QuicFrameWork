package utils

import "reflect"

func IsPointer(v interface{}) bool {
	kind := reflect.TypeOf(v).Kind()
	return kind == reflect.Ptr
}
