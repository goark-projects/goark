package mvc

import "strings"

// Model 表示 MVC 视图模型。
type Model struct {
	attributes map[string]any
}

// NewModel 创建 MVC 视图模型。
func NewModel() Model {
	return Model{attributes: make(map[string]any)}
}

// AddAttribute 添加模型属性。
func (m Model) AddAttribute(name string, value any) Model {
	name = strings.TrimSpace(name)
	if name == "" {
		return m
	}
	if m.attributes == nil {
		m.attributes = make(map[string]any, 1)
	}
	m.attributes[name] = value
	return m
}

// AddAttributeValue 添加模型属性，并按值类型推导属性名。
func (m Model) AddAttributeValue(value any) Model {
	name, ok := inferModelAttributeName(value)
	if !ok {
		return m
	}
	return m.AddAttribute(name, value)
}

// AddAllAttributes 添加多个模型属性。
func (m Model) AddAllAttributes(attributes map[string]any) Model {
	if len(attributes) == 0 {
		return m
	}
	if m.attributes == nil {
		m.attributes = make(map[string]any, len(attributes))
	}
	for name, value := range attributes {
		name = strings.TrimSpace(name)
		if name != "" {
			m.attributes[name] = value
		}
	}
	return m
}

// Attribute 返回模型属性。
func (m Model) Attribute(name string) (any, bool) {
	if len(m.attributes) == 0 {
		return nil, false
	}
	value, ok := m.attributes[name]
	return value, ok
}

// Len 返回模型属性数量。
func (m Model) Len() int {
	return len(m.attributes)
}

// Values 返回模型属性副本。
func (m Model) Values() map[string]any {
	if len(m.attributes) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(m.attributes))
	for name, value := range m.attributes {
		out[name] = value
	}
	return out
}
