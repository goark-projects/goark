package lang

import "reflect"

// TypeOf 返回泛型类型 T 对应的 reflect.Type。
func TypeOf[T any]() reflect.Type {
	var zero *T
	return reflect.TypeOf(zero).Elem()
}

// Zero 返回泛型类型 T 的零值。
func Zero[T any]() T {
	var zero T
	return zero
}
