package util

import (
	"strings"
	"unicode"
)

// IsBlank 判断字符串是否为空白。
func IsBlank(value string) bool {
	return strings.TrimSpace(value) == ""
}

// DefaultIfBlank 返回非空白字符串，否则返回默认值。
func DefaultIfBlank(value string, fallback string) string {
	if IsBlank(value) {
		return fallback
	}
	return value
}

// SplitComma 按英文逗号切分并去除首尾空白。
func SplitComma(value string) []string {
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		out = append(out, part)
	}
	return out
}

// ToKebab 将标识符转换为 kebab-case。
func ToKebab(name string) string {
	return splitWords(name, "-")
}

// ToSnake 将标识符转换为 snake_case。
func ToSnake(name string) string {
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
