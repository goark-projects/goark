package util_test

import (
	"reflect"
	"testing"

	"goark.dev/goark/core/util"
)

type orderedValue struct {
	name     string
	order    int
	priority bool
}

func (v orderedValue) Order() int {
	return v.order
}

func (v orderedValue) PriorityOrdered() {
}

func TestSortByOrder_whenValuesHavePriority_shouldSortPriorityGroupFirst(t *testing.T) {
	values := []any{
		orderedOnly{name: "normal-first", order: 1},
		orderedValue{name: "priority", order: 100},
		orderedOnly{name: "normal-zero", order: 0},
	}

	util.SortByOrder(values)

	names := make([]string, 0, len(values))
	for _, value := range values {
		switch typed := value.(type) {
		case orderedOnly:
			names = append(names, typed.name)
		case orderedValue:
			names = append(names, typed.name)
		}
	}
	expected := []string{"priority", "normal-zero", "normal-first"}
	if !reflect.DeepEqual(names, expected) {
		t.Fatalf("unexpected order: %#v", names)
	}
}

type orderedOnly struct {
	name  string
	order int
}

func (v orderedOnly) Order() int {
	return v.order
}
