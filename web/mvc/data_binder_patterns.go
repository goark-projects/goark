package mvc

import "strings"

func normalizeBinderFieldPatterns(fields []string, foldCase bool) []string {
	if len(fields) == 0 {
		return nil
	}
	out := make([]string, 0, len(fields))
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}
		if foldCase {
			field = strings.ToLower(field)
		}
		out = append(out, field)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func matchesBinderFieldPattern(patterns []string, name string) bool {
	for _, pattern := range patterns {
		if binderFieldPatternMatches(pattern, name) {
			return true
		}
	}
	return false
}

func binderFieldPatternMatches(pattern string, name string) bool {
	if pattern == "*" {
		return true
	}
	if !strings.Contains(pattern, "*") {
		return pattern == name
	}
	return matchBinderFieldWildcard(pattern, name)
}

func matchBinderFieldWildcard(pattern string, name string) bool {
	patternIndex, nameIndex := 0, 0
	starIndex, matchIndex := -1, 0
	for nameIndex < len(name) {
		if patternIndex < len(pattern) && pattern[patternIndex] == '*' {
			starIndex = patternIndex
			matchIndex = nameIndex
			patternIndex++
			continue
		}
		if patternIndex < len(pattern) && pattern[patternIndex] == name[nameIndex] {
			patternIndex++
			nameIndex++
			continue
		}
		if starIndex >= 0 {
			patternIndex = starIndex + 1
			matchIndex++
			nameIndex = matchIndex
			continue
		}
		return false
	}
	for patternIndex < len(pattern) && pattern[patternIndex] == '*' {
		patternIndex++
	}
	return patternIndex == len(pattern)
}
