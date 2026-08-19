package reflectx

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

// TypeOf 返回泛型类型 T 对应的 reflect.Type。
func TypeOf[T any]() reflect.Type {
	var zero *T
	return reflect.TypeOf(zero).Elem()
}
