package config

import (
	"reflect"
	"strings"
	"unicode"
)

func fieldKey(field reflect.StructField) (string, bool) {
	for _, tagName := range []string{"goark", "config"} {
		tag := field.Tag.Get(tagName)
		if tag == "-" {
			return "", true
		}
		if tag != "" {
			return strings.TrimSpace(strings.Split(tag, ",")[0]), false
		}
	}
	return toKebab(field.Name), false
}

func fieldCandidates(prefix string, field reflect.StructField) []string {
	key, skip := fieldKey(field)
	if skip {
		return nil
	}
	base := []string{key}
	if !hasExplicitTag(field) {
		base = append(base, toSnake(field.Name), strings.ToLower(field.Name))
	}
	seen := make(map[string]struct{}, len(base))
	candidates := make([]string, 0, len(base))
	for _, item := range base {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		full := joinKey(prefix, item)
		if _, exists := seen[full]; exists {
			continue
		}
		seen[full] = struct{}{}
		candidates = append(candidates, full)
	}
	return candidates
}

func hasExplicitTag(field reflect.StructField) bool {
	if field.Tag.Get("goark") != "" {
		return true
	}
	return field.Tag.Get("config") != ""
}

func cleanPrefix(prefix string) string {
	return strings.Trim(strings.TrimSpace(prefix), ".")
}

func joinKey(prefix string, key string) string {
	prefix = cleanPrefix(prefix)
	key = cleanPrefix(key)
	if prefix == "" {
		return key
	}
	if key == "" {
		return prefix
	}
	return prefix + "." + key
}

func toKebab(name string) string {
	return splitWords(name, "-")
}

func toSnake(name string) string {
	return splitWords(name, "_")
}

func splitWords(name string, separator string) string {
	if name == "" {
		return ""
	}
	runes := []rune(name)
	var builder strings.Builder
	for i, r := range runes {
		if isWordBoundary(runes, i) {
			builder.WriteString(separator)
		}
		builder.WriteRune(unicode.ToLower(r))
	}
	return builder.String()
}

func isWordBoundary(runes []rune, index int) bool {
	if index == 0 || index >= len(runes) {
		return false
	}
	current := runes[index]
	if !unicode.IsUpper(current) {
		return false
	}
	previous := runes[index-1]
	if unicode.IsLower(previous) || unicode.IsDigit(previous) {
		return true
	}
	if unicode.IsUpper(previous) && index+1 < len(runes) {
		return unicode.IsLower(runes[index+1])
	}
	return false
}
