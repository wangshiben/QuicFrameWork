package utils

import "reflect"

func IsPointer(v interface{}) bool {
	return reflect.TypeOf(v).Kind() == reflect.Ptr
}
