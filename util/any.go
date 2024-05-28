package util

import (
	"reflect"
	"unsafe"
)

func ShallowCopy(dst, src any) {
	dstVal := reflect.ValueOf(dst).Elem()
	srcVal := reflect.ValueOf(src).Elem()

	for i := 0; i < dstVal.NumField(); i++ {
		field := dstVal.Field(i)
		if field.CanSet() {
			field.Set(srcVal.Field(i))
		} else {
			// Using unsafe to set unexported fields
			srcField := srcVal.Field(i)
			dstField := field
			reflect.NewAt(dstField.Type(), unsafe.Pointer(dstField.UnsafeAddr())).Elem().Set(srcField)
		}
	}
}

func ShallowCopyPub(dst, src any) {
	dstVal := reflect.ValueOf(dst).Elem()
	srcVal := reflect.ValueOf(src).Elem()

	for i := 0; i < dstVal.NumField(); i++ {
		field := dstVal.Field(i)
		if field.CanSet() {
			field.Set(srcVal.Field(i))
		}
	}
}
