package mvc

import (
	"fmt"
	"net/url"
	"reflect"
	"strconv"
	"strings"
)

// maxModelAttributeCollectionAutoGrow 对齐 Spring DataBinder 默认自动增长上限，避免大索引触发过量分配。
const maxModelAttributeCollectionAutoGrow = 256

type indexedModelAttributeValue struct {
	direct []string
	nested url.Values
}

func shouldBindIndexedModelAttributeField(value reflect.Value, values url.Values, name string) bool {
	if name == "" || modelAttributeDerefType(value.Type()).Kind() != reflect.Slice {
		return false
	}
	return hasIndexedModelAttributePrefix(values, name)
}

func hasIndexedModelAttributePrefix(values url.Values, name string) bool {
	prefix := name + "["
	for key := range values {
		if strings.HasPrefix(key, prefix) {
			return true
		}
	}
	return false
}

func (b modelAttributeBinder) bindIndexedSliceField(name string, field reflect.Value, values url.Values) error {
	sliceType := modelAttributeDerefType(field.Type())
	bindings, maxIndex, err := collectIndexedModelAttributeValues(values, name)
	if err != nil || maxIndex < 0 {
		return err
	}
	slice := reflect.MakeSlice(sliceType, maxIndex+1, maxIndex+1)
	for i := 0; i <= maxIndex; i++ {
		binding := bindings[i]
		if binding == nil {
			continue
		}
		elementName := fmt.Sprintf("%s[%d]", name, i)
		element := slice.Index(i)
		if len(binding.direct) > 0 {
			if err := b.setIndexedElement(elementName, element, binding.direct); err != nil {
				return err
			}
		}
		if len(binding.nested) > 0 {
			if err := b.bindIndexedElement(elementName, element, binding.nested); err != nil {
				return err
			}
		}
	}
	return setModelAttributeField(name, "", field, slice.Interface())
}

func collectIndexedModelAttributeValues(values url.Values, name string) (map[int]*indexedModelAttributeValue, int, error) {
	prefix := name + "["
	bindings := make(map[int]*indexedModelAttributeValue)
	maxIndex := -1
	for key, list := range values {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		index, suffix, err := parseModelAttributeIndex(name, key)
		if err != nil {
			return nil, -1, invalidParameterError(key, firstIndexedModelAttributeValue(list), "indexed model attribute", err)
		}
		binding := bindings[index]
		if binding == nil {
			binding = &indexedModelAttributeValue{}
			bindings[index] = binding
		}
		switch {
		case suffix == "":
			binding.direct = append(binding.direct, list...)
		case strings.HasPrefix(suffix, ".") && len(suffix) > 1:
			if binding.nested == nil {
				binding.nested = make(url.Values)
			}
			binding.nested[suffix[1:]] = append(binding.nested[suffix[1:]], list...)
		default:
			return nil, -1, invalidParameterError(key, firstIndexedModelAttributeValue(list), "indexed model attribute", fmt.Errorf("非法索引属性路径 %q", key))
		}
		if index > maxIndex {
			maxIndex = index
		}
	}
	return bindings, maxIndex, nil
}

func parseModelAttributeIndex(name string, key string) (int, string, error) {
	offset := len(name)
	if len(key) <= offset+2 || key[offset] != '[' {
		return 0, "", fmt.Errorf("缺少集合索引")
	}
	end := strings.IndexByte(key[offset+1:], ']')
	if end < 1 {
		return 0, "", fmt.Errorf("缺少集合索引")
	}
	rawIndex := key[offset+1 : offset+1+end]
	for _, ch := range rawIndex {
		if ch < '0' || ch > '9' {
			return 0, "", fmt.Errorf("集合索引 %q 非法", rawIndex)
		}
	}
	index, err := strconv.ParseUint(rawIndex, 10, 31)
	if err != nil {
		return 0, "", fmt.Errorf("集合索引 %q 非法: %w", rawIndex, err)
	}
	if index >= maxModelAttributeCollectionAutoGrow {
		return 0, "", fmt.Errorf("集合索引 %d 超过自动增长上限 %d", index, maxModelAttributeCollectionAutoGrow)
	}
	return int(index), key[offset+2+end:], nil
}

func (b modelAttributeBinder) setIndexedElement(name string, element reflect.Value, values []string) error {
	if modelAttributeDerefType(element.Type()).Kind() == reflect.Slice {
		return b.setSliceField(name, element, values)
	}
	raw := values[0]
	converted, err := b.convertString(name, raw, element.Type())
	if err != nil {
		return err
	}
	return setModelAttributeField(name, raw, element, converted)
}

func (b modelAttributeBinder) bindIndexedElement(name string, element reflect.Value, values url.Values) error {
	targetType := modelAttributeDerefType(element.Type())
	if targetType.Kind() != reflect.Struct || isScalarModelAttributeStruct(targetType) {
		return invalidParameterError(name, "", "struct", fmt.Errorf("索引嵌套属性需要结构体元素"))
	}
	nested := indirectModelAttributeStruct(element)
	if !nested.IsValid() {
		return invalidParameterError(name, "", "struct", fmt.Errorf("索引嵌套属性需要可设置元素"))
	}
	return b.bindStruct(nested, values, "")
}

func firstIndexedModelAttributeValue(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}
