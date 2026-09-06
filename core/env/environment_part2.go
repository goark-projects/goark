package env

import (
	"reflect"
	"sort"
	"strings"

	"goark.dev/goark/core/convert"
	"goark.dev/goark/core/util"
	arkerrors "goark.dev/goark/errors"
)

func (e *StandardEnvironment) AcceptsProfiles(profiles ...string) bool {
	if e == nil || len(profiles) == 0 {
		return false
	}
	active := e.activeProfileSet()
	for _, profile := range profiles {
		profile = strings.TrimSpace(profile)
		if profile == "" {
			continue
		}
		negated := strings.HasPrefix(profile, "!")
		if negated {
			profile = strings.TrimSpace(strings.TrimPrefix(profile, "!"))
		}
		if profile == "" {
			continue
		}
		_, matched := active[profile]
		if negated && !matched {
			return true
		}
		if !negated && matched {
			return true
		}
	}
	return false
}

// NewStandardEnvironment 创建标准环境。
func NewStandardEnvironment() (*StandardEnvironment, error) {
	propertySources, err := NewMutablePropertySources()
	if err != nil {
		return nil, err
	}
	resolver, err := NewPropertySourcesPropertyResolver(propertySources)
	if err != nil {
		return nil, err
	}
	environment := &StandardEnvironment{
		propertySources: propertySources,
		resolver:        resolver,
		defaultProfiles: []string{"default"},
	}
	if err := environment.customizePropertySources(); err != nil {
		return nil, err
	}
	return environment, nil
}

func validateProfiles(profiles []string) ([]string, error) {
	cleaned := make([]string, 0, len(profiles))
	for _, profile := range profiles {
		profile = strings.TrimSpace(profile)
		if profile == "" {
			return nil, arkerrors.New(arkerrors.CodeInvalidArgument, "profile is empty")
		}
		if strings.HasPrefix(profile, "!") {
			return nil, arkerrors.Newf(
				arkerrors.CodeInvalidArgument,
				"profile %q must not start with !",
				profile,
			)
		}
		cleaned = append(cleaned, profile)
	}
	sort.Strings(cleaned)
	return util.UniqueStrings(cleaned), nil
}

func (e *StandardEnvironment) activeProfileSet() map[string]struct{} {
	e.mu.RLock()
	defer e.mu.RUnlock()
	profiles := e.activeProfiles
	if len(profiles) == 0 {
		profiles = e.defaultProfiles
	}
	out := make(map[string]struct{}, len(profiles))
	for _, profile := range profiles {
		out[profile] = struct{}{}
	}
	return out
}

func (e *StandardEnvironment) SetDefaultProfiles(profiles ...string) error {
	cleaned, err := validateProfiles(profiles)
	if err != nil {
		return err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	e.defaultProfiles = cleaned
	return nil
}

func (e *StandardEnvironment) GetRequiredPropertyAs(
	key string,
	targetType reflect.Type,
) (any, error) {
	if e == nil {
		return nil, arkerrors.New(arkerrors.CodeInvalidArgument, "environment is nil")
	}
	return e.resolver.GetRequiredPropertyAs(key, targetType)
}

// MustNewStandardEnvironment 创建标准环境，失败时 panic。
func MustNewStandardEnvironment() *StandardEnvironment {
	environment, err := NewStandardEnvironment()
	if err != nil {
		panic(err)
	}
	return environment
}

func (e *StandardEnvironment) DefaultProfiles() []string {
	if e == nil {
		return nil
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	return append([]string(nil), e.defaultProfiles...)
}

// PropertyNames 返回所有可枚举配置源中的属性名。
func (e *StandardEnvironment) PropertyNames() []string {
	if e == nil || e.propertySources == nil {
		return nil
	}
	return e.propertySources.PropertyNames()
}

const (
	// SystemPropertiesPropertySourceName 对齐 Spring 的 systemProperties 源名称。
	SystemPropertiesPropertySourceName = "systemProperties"
	// SystemEnvironmentPropertySourceName 对齐 Spring 的 systemEnvironment 源名称。
	SystemEnvironmentPropertySourceName = "systemEnvironment"
)

func (e *StandardEnvironment) GetProperty(key string) (string, bool) {
	if e == nil {
		return "", false
	}
	return e.resolver.GetProperty(key)
}

func (e *StandardEnvironment) GetRequiredProperty(key string) (string, error) {
	if e == nil {
		return "", arkerrors.New(arkerrors.CodeInvalidArgument, "environment is nil")
	}
	return e.resolver.GetRequiredProperty(key)
}

func (e *StandardEnvironment) ResolveRequiredPlaceholders(text string) (string, error) {
	if e == nil {
		return "", arkerrors.New(arkerrors.CodeInvalidArgument, "environment is nil")
	}
	return e.resolver.ResolveRequiredPlaceholders(text)
}

func (e *StandardEnvironment) SetConversionService(service *convert.Service) error {
	if e == nil {
		return arkerrors.New(arkerrors.CodeInvalidArgument, "environment is nil")
	}
	return e.resolver.SetConversionService(service)
}

func (e *StandardEnvironment) ValidateRequiredProperties() error {
	if e == nil {
		return arkerrors.New(arkerrors.CodeInvalidArgument, "environment is nil")
	}
	return e.resolver.ValidateRequiredProperties()
}

// NewEnvironment 创建标准环境。
func NewEnvironment() (*StandardEnvironment, error) {
	return NewStandardEnvironment()
}
