package flash

import (
	"net/url"
	"strings"
	"time"
)

// Map 表示 Spring FlashMap 风格的一次性重定向属性集合。
type Map struct {
	targetPath   string
	targetParams url.Values
	attributes   map[string]any
	expiresAt    time.Time
}

// NewMap 创建空 FlashMap。
func NewMap() *Map {
	return &Map{attributes: make(map[string]any)}
}

// AddAttribute 添加 Flash 属性。
func (m *Map) AddAttribute(name string, value any) *Map {
	name = strings.TrimSpace(name)
	if m == nil || name == "" {
		return m
	}
	if m.attributes == nil {
		m.attributes = make(map[string]any, 1)
	}
	m.attributes[name] = value
	return m
}

// AddAllAttributes 添加多个 Flash 属性。
func (m *Map) AddAllAttributes(values map[string]any) *Map {
	if m == nil || len(values) == 0 {
		return m
	}
	for name, value := range values {
		m.AddAttribute(name, value)
	}
	return m
}

// Attribute 返回 Flash 属性值。
func (m *Map) Attribute(name string) (any, bool) {
	if m == nil || len(m.attributes) == 0 {
		return nil, false
	}
	value, ok := m.attributes[name]
	return value, ok
}

// Len 返回 Flash 属性数量。
func (m *Map) Len() int {
	if m == nil {
		return 0
	}
	return len(m.attributes)
}

// Values 返回 Flash 属性副本。
func (m *Map) Values() map[string]any {
	if m == nil || len(m.attributes) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(m.attributes))
	for name, value := range m.attributes {
		out[name] = value
	}
	return out
}

// TargetPath 返回目标请求路径。
func (m *Map) TargetPath() string {
	if m == nil {
		return ""
	}
	return m.targetPath
}

// TargetParams 返回目标请求参数副本。
func (m *Map) TargetParams() url.Values {
	if m == nil || len(m.targetParams) == 0 {
		return url.Values{}
	}
	return cloneValues(m.targetParams)
}

// ExpiresAt 返回过期时间；零值表示尚未进入过期倒计时。
func (m *Map) ExpiresAt() time.Time {
	if m == nil {
		return time.Time{}
	}
	return m.expiresAt
}

// SetTargetLocation 从重定向 Location 设置目标请求匹配条件。
func (m *Map) SetTargetLocation(location string) bool {
	path, params, ok := parseTargetLocation(location)
	if m == nil || !ok {
		return false
	}
	m.targetPath = path
	m.targetParams = params
	return true
}

func (m *Map) clone() Map {
	if m == nil {
		return Map{}
	}
	return Map{
		targetPath:   m.targetPath,
		targetParams: cloneValues(m.targetParams),
		attributes:   cloneAttributes(m.attributes),
		expiresAt:    m.expiresAt,
	}
}

func (m *Map) startExpiration(now time.Time, timeout time.Duration) Map {
	copied := m.clone()
	if timeout > 0 {
		copied.expiresAt = now.Add(timeout)
	}
	return copied
}

func (m Map) expired(now time.Time) bool {
	return !m.expiresAt.IsZero() && !now.Before(m.expiresAt)
}

func cloneAttributes(values map[string]any) map[string]any {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]any, len(values))
	for name, value := range values {
		out[name] = value
	}
	return out
}

func cloneValues(values url.Values) url.Values {
	if len(values) == 0 {
		return nil
	}
	out := make(url.Values, len(values))
	for name, list := range values {
		out[name] = append([]string(nil), list...)
	}
	return out
}

func parseTargetLocation(location string) (string, url.Values, bool) {
	location = strings.TrimSpace(location)
	if location == "" {
		return "", nil, false
	}
	parsed, err := url.Parse(location)
	if err != nil || parsed.Path == "" {
		return "", nil, false
	}
	return parsed.Path, cloneValues(parsed.Query()), true
}
