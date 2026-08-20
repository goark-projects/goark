package util

import "reflect"

// IsNil 判断接口值或其内部动态值是否为 nil。
func IsNil(value any) bool {
	if value == nil {
		return true
	}
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}

// DerefType 返回指针链最终指向的类型。
func DerefType(typ reflect.Type) reflect.Type {
	for typ != nil && typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	return typ
}

// TypeAssignable 判断实际类型是否可赋给期望类型。
func TypeAssignable(actual reflect.Type, expected reflect.Type) bool {
	if actual == nil || expected == nil {
		return false
	}
	if actual.AssignableTo(expected) {
		return true
	}
	return expected.Kind() == reflect.Interface && actual.Implements(expected)
}
