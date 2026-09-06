package env

import (
	"os"
	"reflect"
	"sort"
	"sync"

	"goark.dev/goark/core/convert"
	"goark.dev/goark/core/util"
	arkerrors "goark.dev/goark/errors"
)

func (e *StandardEnvironment) Merge(parent ConfigurableEnvironment) error {
	if e == nil {
		return arkerrors.New(arkerrors.CodeInvalidArgument, "environment is nil")
	}
	if parent == nil {
		return nil
	}
	parentSources := parent.PropertySources().Snapshot()
	for _, source := range parentSources {
		if e.propertySources.Contains(source.Name()) {
			continue
		}
		if err := e.propertySources.AddLast(source); err != nil {
			return err
		}
	}
	for _, profile := range parent.ActiveProfiles() {
		if err := e.AddActiveProfile(profile); err != nil {
			return err
		}
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, profile := range parent.DefaultProfiles() {
		if !containsString(e.defaultProfiles, profile) {
			e.defaultProfiles = append(e.defaultProfiles, profile)
		}
	}
	sort.Strings(e.defaultProfiles)
	e.defaultProfiles = util.UniqueStrings(e.defaultProfiles)
	return nil
}

func (e *StandardEnvironment) customizePropertySources() error {
	systemProperties, err := NewMapPropertySource(
		SystemPropertiesPropertySourceName,
		map[string]any{},
	)
	if err != nil {
		return err
	}
	systemEnvironment, err := NewSystemEnvironmentPropertySource(
		SystemEnvironmentPropertySourceName,
		os.Environ(),
	)
	if err != nil {
		return err
	}
	if err := e.propertySources.AddLast(systemProperties); err != nil {
		return err
	}
	return e.propertySources.AddLast(systemEnvironment)
}

func (e *StandardEnvironment) AddActiveProfile(profile string) error {
	cleaned, err := validateProfiles([]string{profile})
	if err != nil {
		return err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, existing := range e.activeProfiles {
		if existing == cleaned[0] {
			return nil
		}
	}
	e.activeProfiles = append(e.activeProfiles, cleaned[0])
	sort.Strings(e.activeProfiles)
	return nil
}

// ConfigurableEnvironment 表示可配置运行环境。
type ConfigurableEnvironment interface {
	Environment
	ConfigurablePropertyResolver
	PropertySources() *MutablePropertySources
	SetActiveProfiles(profiles ...string) error
	AddActiveProfile(profile string) error
	SetDefaultProfiles(profiles ...string) error
	Merge(parent ConfigurableEnvironment) error
}

func (e *StandardEnvironment) SetActiveProfiles(profiles ...string) error {
	cleaned, err := validateProfiles(profiles)
	if err != nil {
		return err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	e.activeProfiles = cleaned
	return nil
}

func (e *StandardEnvironment) GetPropertyAs(
	key string,
	targetType reflect.Type,
) (any, bool, error) {
	if e == nil {
		return nil, false, arkerrors.New(arkerrors.CodeInvalidArgument, "environment is nil")
	}
	return e.resolver.GetPropertyAs(key, targetType)
}

// StandardEnvironment 是 Goark 默认环境实现。
type StandardEnvironment struct {
	mu              sync.RWMutex
	propertySources *MutablePropertySources
	resolver        *PropertySourcesPropertyResolver
	activeProfiles  []string
	defaultProfiles []string
}

func (e *StandardEnvironment) ActiveProfiles() []string {
	if e == nil {
		return nil
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	return append([]string(nil), e.activeProfiles...)
}

// Environment 表示应用运行环境。
type Environment interface {
	PropertyResolver
	ActiveProfiles() []string
	DefaultProfiles() []string
	AcceptsProfiles(profiles ...string) bool
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
func (e *StandardEnvironment) PropertySources() *MutablePropertySources {
	if e == nil {
		return nil
	}
	return e.propertySources
}

func (e *StandardEnvironment) GetPropertyOrDefault(key string, defaultValue string) string {
	if e == nil {
		return defaultValue
	}
	return e.resolver.GetPropertyOrDefault(key, defaultValue)
}

func (e *StandardEnvironment) ResolvePlaceholders(text string) (string, error) {
	if e == nil {
		return text, nil
	}
	return e.resolver.ResolvePlaceholders(text)
}

func (e *StandardEnvironment) ConversionService() *convert.Service {
	if e == nil {
		return nil
	}
	return e.resolver.ConversionService()
}

func (e *StandardEnvironment) SetRequiredProperties(keys ...string) {
	if e == nil {
		return
	}
	e.resolver.SetRequiredProperties(keys...)
}

// EnvironmentCapable 表示持有 Environment 的对象。
type EnvironmentCapable interface {
	Environment() Environment
}

func (e *StandardEnvironment) ContainsProperty(key string) bool {
	return e != nil && e.resolver.ContainsProperty(key)
}
