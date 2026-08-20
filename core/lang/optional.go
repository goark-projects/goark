package lang

// Optional 表示一个可能存在的值。
type Optional[T any] struct {
	value   T
	present bool
}

// Some 创建存在值的 Optional。
func Some[T any](value T) Optional[T] {
	return Optional[T]{value: value, present: true}
}

// None 创建空 Optional。
func None[T any]() Optional[T] {
	return Optional[T]{}
}

// OfPtr 将指针转换为 Optional。
func OfPtr[T any](value *T) Optional[T] {
	if value == nil {
		return None[T]()
	}
	return Some(*value)
}

// Present 返回值是否存在。
func (o Optional[T]) Present() bool {
	return o.present
}

// Empty 返回值是否不存在。
func (o Optional[T]) Empty() bool {
	return !o.present
}

// Value 返回值及存在标记。
func (o Optional[T]) Value() (T, bool) {
	return o.value, o.present
}

// OrElse 返回存在值或默认值。
func (o Optional[T]) OrElse(fallback T) T {
	if o.present {
		return o.value
	}
	return fallback
}

// OrElseGet 返回存在值或通过函数延迟计算默认值。
func (o Optional[T]) OrElseGet(fallback func() T) T {
	if o.present {
		return o.value
	}
	if fallback == nil {
		var zero T
		return zero
	}
	return fallback()
}

// Ptr 返回值指针；值不存在时返回 nil。
func (o Optional[T]) Ptr() *T {
	if !o.present {
		return nil
	}
	value := o.value
	return &value
}
