package mvc

import (
	"net/url"
	"strings"

	arkweb "goark.dev/arkarta/web"
)

const attributeDataBinder = "goark.dev/goark/web/mvc.dataBinder"

// SetAllowedFields 设置允许绑定的字段模式；未设置时默认允许全部字段。
func (b *DataBinder) SetAllowedFields(fields ...string) error {
	if b == nil {
		return ErrNilDataBinder
	}
	b.allowedFields = normalizeBinderFieldPatterns(fields, false)
	return nil
}

// AllowedFields 返回允许绑定字段模式快照。
func (b *DataBinder) AllowedFields() []string {
	if b == nil || len(b.allowedFields) == 0 {
		return nil
	}
	return append([]string(nil), b.allowedFields...)
}

// SetDisallowedFields 设置拒绝绑定的字段模式；拒绝规则优先于允许规则。
func (b *DataBinder) SetDisallowedFields(fields ...string) error {
	if b == nil {
		return ErrNilDataBinder
	}
	b.disallowedFields = normalizeBinderFieldPatterns(fields, true)
	return nil
}

// DisallowedFields 返回拒绝绑定字段模式快照。
func (b *DataBinder) DisallowedFields() []string {
	if b == nil || len(b.disallowedFields) == 0 {
		return nil
	}
	return append([]string(nil), b.disallowedFields...)
}

func (b *DataBinder) inheritFieldRules(parent *DataBinder) {
	if b == nil || parent == nil {
		return
	}
	b.allowedFields = parent.AllowedFields()
	b.disallowedFields = parent.DisallowedFields()
}

func (b *DataBinder) filterModelAttributeValues(values url.Values) url.Values {
	if b == nil || len(values) == 0 || !b.hasFieldRules() {
		return values
	}
	out := make(url.Values, len(values))
	for name, list := range values {
		if name == "" || len(list) == 0 || !b.isFieldAllowed(name) {
			continue
		}
		out[name] = append([]string(nil), list...)
	}
	return out
}

func (b *DataBinder) hasFieldRules() bool {
	return b != nil && (len(b.allowedFields) > 0 || len(b.disallowedFields) > 0)
}

func (b *DataBinder) isFieldAllowed(name string) bool {
	if b == nil {
		return true
	}
	if len(b.allowedFields) > 0 && !matchesBinderFieldPattern(b.allowedFields, name) {
		return false
	}
	if len(b.disallowedFields) == 0 {
		return true
	}
	return !matchesBinderFieldPattern(b.disallowedFields, strings.ToLower(name))
}

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

func modelAttributeValuesForCurrentBinder(ctx *arkweb.Context, values url.Values) url.Values {
	binder, ok := dataBinderFromContext(ctx)
	if !ok {
		return values
	}
	return binder.filterModelAttributeValues(values)
}

func dataBinderFromContext(ctx *arkweb.Context) (*DataBinder, bool) {
	if ctx == nil || ctx.Request() == nil {
		return nil, false
	}
	value, ok := ctx.Request().Attribute(attributeDataBinder)
	if !ok {
		return nil, false
	}
	binder, ok := value.(*DataBinder)
	if !ok || binder == nil {
		return nil, false
	}
	return binder, true
}
