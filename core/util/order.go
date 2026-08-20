package util

import (
	"sort"

	"github.com/goark-projects/goark/core/lang"
)

// OrderOf 返回对象声明的排序值；未实现 Ordered 时返回最低优先级。
func OrderOf(value any) int {
	if ordered, ok := value.(lang.Ordered); ok {
		return ordered.Order()
	}
	return lang.LowestPrecedence
}

// IsPriorityOrdered 判断对象是否声明为优先排序对象。
func IsPriorityOrdered(value any) bool {
	_, ok := value.(lang.PriorityOrdered)
	return ok
}

// SortByOrder 按 Ordered 顺序稳定排序。
func SortByOrder[T any](values []T) {
	sort.SliceStable(values, func(i, j int) bool {
		leftPriority := IsPriorityOrdered(any(values[i]))
		rightPriority := IsPriorityOrdered(any(values[j]))
		if leftPriority != rightPriority {
			return leftPriority
		}
		left := OrderOf(any(values[i]))
		right := OrderOf(any(values[j]))
		return left < right
	})
}
