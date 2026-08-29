package mvc

import (
	"fmt"
	"net/url"
	"reflect"
	"strings"
)

// maxModelAttributeMapEntries 限制表单 map 自动增长规模，防止单请求过量分配。
const maxModelAttributeMapEntries = 256

type mappedModelAttributeValue struct {
	key    string
	values []string
}

func shouldBindMappedModelAttributeField(value reflect.Value, values url.Values, name string) bool {
	if name == "" || modelAttributeDerefType(value.Type()).Kind() != reflect.Map {
		return false
	}
	return hasBracketedModelAttributePrefix(values, name)
}

func hasBracketedModelAttributePrefix(values url.Values, name string) bool {
	prefix := name + "["
	for key := range values {
		if strings.HasPrefix(key, prefix) {
			return true
		}
	}
	return false
}

func (b modelAttributeBinder) bindMappedField(name string, field reflect.Value, values url.Values) error {
	mapType := modelAttributeDerefType(field.Type())
	entries, err := collectMappedModelAttributeValues(values, name)
	if err != nil {
		return err
	}
	out := reflect.MakeMapWithSize(mapType, len(entries))
	for _, entry := range entries {
		entryName := name + "[" + entry.key + "]"
		key, err := b.convertMapKey(entryName, entry.key, mapType.Key())
		if err != nil {
			return err
		}
		value, err := b.convertMapValue(entryName, entry.values, mapType.Elem())
		if err != nil {
			return err
		}
		out.SetMapIndex(key, value)
	}
	return setModelAttributeField(name, "", field, out.Interface())
}

func collectMappedModelAttributeValues(values url.Values, name string) ([]mappedModelAttributeValue, error) {
	prefix := name + "["
	entries := make([]mappedModelAttributeValue, 0)
	indexes := make(map[string]int)
	for key, list := range values {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		mapKey, err := parseModelAttributeMapKey(name, key)
		if err != nil {
			return nil, invalidParameterError(key, firstIndexedModelAttributeValue(list), "mapped model attribute", err)
		}
		if position, exists := indexes[mapKey]; exists {
			entries[position].values = append(entries[position].values, list...)
			continue
		}
		if len(entries) >= maxModelAttributeMapEntries {
			return nil, invalidParameterError(key, firstIndexedModelAttributeValue(list), "mapped model attribute", fmt.Errorf("map 属性数量超过上限 %d", maxModelAttributeMapEntries))
		}
		indexes[mapKey] = len(entries)
		entries = append(entries, mappedModelAttributeValue{
			key:    mapKey,
			values: append([]string(nil), list...),
		})
	}
	return entries, nil
}

func parseModelAttributeMapKey(name string, key string) (string, error) {
	offset := len(name)
	if len(key) <= offset+2 || key[offset] != '[' {
		return "", fmt.Errorf("缺少 map 键")
	}
	end := strings.IndexByte(key[offset+1:], ']')
	if end < 1 {
		return "", fmt.Errorf("缺少 map 键")
	}
	mapKey := key[offset+1 : offset+1+end]
	if suffix := key[offset+2+end:]; suffix != "" {
		return "", fmt.Errorf("map 属性暂不支持嵌套路径 %q", suffix)
	}
	return mapKey, nil
}

func (b modelAttributeBinder) convertMapKey(name string, raw string, targetType reflect.Type) (reflect.Value, error) {
	converted, err := b.convertString(name, raw, targetType)
	if err != nil {
		return reflect.Value{}, err
	}
	return modelAttributeFieldValue(targetType, converted)
}

func (b modelAttributeBinder) convertMapValue(name string, values []string, targetType reflect.Type) (reflect.Value, error) {
	if modelAttributeDerefType(targetType).Kind() == reflect.Slice {
		return b.convertMapSliceValue(name, values, targetType)
	}
	raw := firstIndexedModelAttributeValue(values)
	converted, err := b.convertString(name, raw, targetType)
	if err != nil {
		return reflect.Value{}, err
	}
	return modelAttributeFieldValue(targetType, converted)
}

func (b modelAttributeBinder) convertMapSliceValue(name string, values []string, targetType reflect.Type) (reflect.Value, error) {
	sliceType := modelAttributeDerefType(targetType)
	items := splitParamValues(values)
	slice := reflect.MakeSlice(sliceType, 0, len(items))
	for _, item := range items {
		converted, err := b.convertString(name, item, sliceType.Elem())
		if err != nil {
			return reflect.Value{}, err
		}
		value, err := modelAttributeFieldValue(sliceType.Elem(), converted)
		if err != nil {
			return reflect.Value{}, invalidParameterError(name, item, sliceType.String(), err)
		}
		slice = reflect.Append(slice, value)
	}
	return modelAttributeFieldValue(targetType, slice.Interface())
}
