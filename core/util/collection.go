package util

import "sort"

// CopyMap 复制 map，返回值可安全修改。
func CopyMap[K comparable, V any](source map[K]V) map[K]V {
	if source == nil {
		return nil
	}
	copied := make(map[K]V, len(source))
	for key, value := range source {
		copied[key] = value
	}
	return copied
}

// SortedKeys 返回按字典序排序的字符串键。
func SortedKeys[V any](source map[string]V) []string {
	if source == nil {
		return nil
	}
	keys := make([]string, 0, len(source))
	for key := range source {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// UniqueStrings 去重字符串并保持首次出现顺序。
func UniqueStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
