package env

import "strings"

// SystemEnvironmentPropertySource 表示系统环境变量配置源。
type SystemEnvironmentPropertySource struct {
	*MapPropertySource
	canonical map[string]string
}

// NewSystemEnvironmentPropertySource 创建系统环境变量配置源。
func NewSystemEnvironmentPropertySource(name string, environ []string) (*SystemEnvironmentPropertySource, error) {
	values := make(map[string]any, len(environ))
	canonical := make(map[string]string, len(environ))
	for _, item := range environ {
		key, value, ok := strings.Cut(item, "=")
		if !ok || key == "" {
			continue
		}
		values[key] = value
		canonical[canonicalEnvironmentKey(key)] = key
	}
	source, err := NewMapPropertySource(name, values)
	if err != nil {
		return nil, err
	}
	return &SystemEnvironmentPropertySource{
		MapPropertySource: source,
		canonical:         canonical,
	}, nil
}

func (s *SystemEnvironmentPropertySource) ContainsProperty(name string) bool {
	if s == nil || name == "" {
		return false
	}
	if s.MapPropertySource.ContainsProperty(name) {
		return true
	}
	_, ok := s.canonical[canonicalEnvironmentKey(name)]
	return ok
}

func (s *SystemEnvironmentPropertySource) GetProperty(name string) (any, bool) {
	if s == nil || name == "" {
		return nil, false
	}
	if value, ok := s.MapPropertySource.GetProperty(name); ok {
		return value, true
	}
	actual, ok := s.canonical[canonicalEnvironmentKey(name)]
	if !ok {
		return nil, false
	}
	return s.MapPropertySource.GetProperty(actual)
}

func canonicalEnvironmentKey(name string) string {
	replacer := strings.NewReplacer(".", "_", "-", "_")
	return strings.ToUpper(replacer.Replace(name))
}
