package mvc

import (
	"net/url"
	"sort"
	"strings"

	arkweb "goark.dev/arkarta/web"
)

const (
	attributeDataBinder       = "goark.dev/goark/web/mvc.dataBinder"
	defaultFieldMarkerPrefix  = "_"
	defaultFieldDefaultPrefix = "!"
)

type preparedModelAttributeValues struct {
	values           url.Values
	fieldMarkers     map[string]struct{}
	suppressedFields []string
}

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

// SetFieldMarkerPrefix 设置字段 marker 前缀；空字符串表示禁用 marker 处理。
func (b *DataBinder) SetFieldMarkerPrefix(prefix string) error {
	if b == nil {
		return ErrNilDataBinder
	}
	b.fieldMarkerPrefix = strings.TrimSpace(prefix)
	return nil
}

// FieldMarkerPrefix 返回字段 marker 前缀。
func (b *DataBinder) FieldMarkerPrefix() string {
	if b == nil {
		return ""
	}
	return b.fieldMarkerPrefix
}

// SetFieldDefaultPrefix 设置字段默认值前缀；空字符串表示禁用默认值处理。
func (b *DataBinder) SetFieldDefaultPrefix(prefix string) error {
	if b == nil {
		return ErrNilDataBinder
	}
	b.fieldDefaultPrefix = strings.TrimSpace(prefix)
	return nil
}

// FieldDefaultPrefix 返回字段默认值前缀。
func (b *DataBinder) FieldDefaultPrefix() string {
	if b == nil {
		return ""
	}
	return b.fieldDefaultPrefix
}

func (b *DataBinder) inheritFieldRules(parent *DataBinder) {
	if b == nil || parent == nil {
		return
	}
	b.allowedFields = parent.AllowedFields()
	b.disallowedFields = parent.DisallowedFields()
	b.fieldMarkerPrefix = parent.FieldMarkerPrefix()
	b.fieldDefaultPrefix = parent.FieldDefaultPrefix()
}

func (b *DataBinder) prepareModelAttributeValues(values url.Values) preparedModelAttributeValues {
	if b == nil {
		return preparedModelAttributeValues{values: values}
	}
	values = b.applyFieldDefaults(values)
	values, markers := b.extractFieldMarkers(values)
	values = b.adaptEmptyArrayIndices(values)
	var suppressed []string
	values, suppressed = b.filterModelAttributeValues(values)
	markers, suppressed = b.filterFieldMarkers(markers, suppressed)
	suppressed = sortedSuppressedFields(suppressed)
	return preparedModelAttributeValues{
		values:           values,
		fieldMarkers:     markers,
		suppressedFields: suppressed,
	}
}

func (b *DataBinder) applyFieldDefaults(values url.Values) url.Values {
	prefix := b.FieldDefaultPrefix()
	if prefix == "" || len(values) == 0 {
		return values
	}
	var out url.Values
	for name, list := range values {
		field, ok := strings.CutPrefix(name, prefix)
		if !ok {
			continue
		}
		out = ensureClonedModelAttributeValues(out, values)
		delete(out, name)
		if field == "" {
			continue
		}
		if _, exists := out[field]; !exists {
			out[field] = append([]string(nil), list...)
		}
	}
	if out == nil {
		return values
	}
	return out
}

func (b *DataBinder) extractFieldMarkers(values url.Values) (url.Values, map[string]struct{}) {
	prefix := b.FieldMarkerPrefix()
	if prefix == "" || len(values) == 0 {
		return values, nil
	}
	var out url.Values
	var markers map[string]struct{}
	for name := range values {
		field, ok := strings.CutPrefix(name, prefix)
		if !ok {
			continue
		}
		out = ensureClonedModelAttributeValues(out, values)
		delete(out, name)
		if field == "" {
			continue
		}
		if _, exists := out[field]; exists {
			continue
		}
		if markers == nil {
			markers = make(map[string]struct{})
		}
		markers[field] = struct{}{}
	}
	if out == nil {
		return values, markers
	}
	return out, markers
}

func (b *DataBinder) adaptEmptyArrayIndices(values url.Values) url.Values {
	if len(values) == 0 {
		return values
	}
	var out url.Values
	for name, list := range values {
		field, ok := strings.CutSuffix(name, "[]")
		if !ok {
			continue
		}
		out = ensureClonedModelAttributeValues(out, values)
		delete(out, name)
		if field == "" {
			continue
		}
		if _, exists := out[field]; !exists {
			out[field] = append([]string(nil), list...)
		}
	}
	if out == nil {
		return values
	}
	return out
}

func (b *DataBinder) filterModelAttributeValues(values url.Values) (url.Values, []string) {
	if b == nil || len(values) == 0 || !b.hasFieldRules() {
		return values, nil
	}
	out := make(url.Values, len(values))
	suppressed := make([]string, 0)
	for name, list := range values {
		if name == "" || len(list) == 0 {
			continue
		}
		if !b.isFieldAllowed(name) {
			suppressed = append(suppressed, name)
			continue
		}
		out[name] = append([]string(nil), list...)
	}
	return out, suppressed
}

func (b *DataBinder) filterFieldMarkers(markers map[string]struct{}, suppressed []string) (map[string]struct{}, []string) {
	if len(markers) == 0 || !b.hasFieldRules() {
		return markers, suppressed
	}
	out := make(map[string]struct{}, len(markers))
	for name := range markers {
		if name != "" && b.isFieldAllowed(name) {
			out[name] = struct{}{}
			continue
		}
		suppressed = append(suppressed, name)
	}
	if len(out) == 0 {
		return nil, suppressed
	}
	return out, suppressed
}

func sortedSuppressedFields(fields []string) []string {
	if len(fields) == 0 {
		return nil
	}
	sort.Strings(fields)
	out := fields[:0]
	for _, field := range fields {
		if field == "" {
			continue
		}
		if len(out) == 0 || out[len(out)-1] != field {
			out = append(out, field)
		}
	}
	if len(out) == 0 {
		return nil
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

func ensureClonedModelAttributeValues(out url.Values, values url.Values) url.Values {
	if out != nil {
		return out
	}
	return url.Values(cloneStringValuesMap(values))
}

func modelAttributeValuesForCurrentBinder(ctx *arkweb.Context, values url.Values) preparedModelAttributeValues {
	binder, ok := dataBinderFromContext(ctx)
	if !ok {
		binder = newDataBinder(nil)
	}
	return binder.prepareModelAttributeValues(values)
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
