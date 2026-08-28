package env

import (
	"os"
	"reflect"
	"sort"
	"strings"
	"sync"

	"goark.dev/goark/core/convert"
	"goark.dev/goark/core/util"
	arkerrors "goark.dev/goark/errors"
)

const (
	// SystemPropertiesPropertySourceName 对齐 Spring 的 systemProperties 源名称。
	SystemPropertiesPropertySourceName = "systemProperties"
	// SystemEnvironmentPropertySourceName 对齐 Spring 的 systemEnvironment 源名称。
	SystemEnvironmentPropertySourceName = "systemEnvironment"
)

// Environment 表示应用运行环境。
type Environment interface {
	PropertyResolver
	ActiveProfiles() []string
	DefaultProfiles() []string
	AcceptsProfiles(profiles ...string) bool
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

// EnvironmentCapable 表示持有 Environment 的对象。
type EnvironmentCapable interface {
	Environment() Environment
}

// StandardEnvironment 是 Goark 默认环境实现。
type StandardEnvironment struct {
	mu              sync.RWMutex
	propertySources *MutablePropertySources
	resolver        *PropertySourcesPropertyResolver
	activeProfiles  []string
	defaultProfiles []string
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

// NewEnvironment 创建标准环境。
func NewEnvironment() (*StandardEnvironment, error) {
	return NewStandardEnvironment()
}

// MustNewStandardEnvironment 创建标准环境，失败时 panic。
func MustNewStandardEnvironment() *StandardEnvironment {
	environment, err := NewStandardEnvironment()
	if err != nil {
		panic(err)
	}
	return environment
}

func (e *StandardEnvironment) PropertySources() *MutablePropertySources {
	if e == nil {
		return nil
	}
	return e.propertySources
}

func (e *StandardEnvironment) ContainsProperty(key string) bool {
	return e != nil && e.resolver.ContainsProperty(key)
}

func (e *StandardEnvironment) GetProperty(key string) (string, bool) {
	if e == nil {
		return "", false
	}
	return e.resolver.GetProperty(key)
}

func (e *StandardEnvironment) GetPropertyOrDefault(key string, defaultValue string) string {
	if e == nil {
		return defaultValue
	}
	return e.resolver.GetPropertyOrDefault(key, defaultValue)
}

func (e *StandardEnvironment) GetRequiredProperty(key string) (string, error) {
	if e == nil {
		return "", arkerrors.New(arkerrors.CodeInvalidArgument, "environment is nil")
	}
	return e.resolver.GetRequiredProperty(key)
}

func (e *StandardEnvironment) GetPropertyAs(key string, targetType reflect.Type) (any, bool, error) {
	if e == nil {
		return nil, false, arkerrors.New(arkerrors.CodeInvalidArgument, "environment is nil")
	}
	return e.resolver.GetPropertyAs(key, targetType)
}

func (e *StandardEnvironment) GetRequiredPropertyAs(key string, targetType reflect.Type) (any, error) {
	if e == nil {
		return nil, arkerrors.New(arkerrors.CodeInvalidArgument, "environment is nil")
	}
	return e.resolver.GetRequiredPropertyAs(key, targetType)
}

func (e *StandardEnvironment) ResolvePlaceholders(text string) (string, error) {
	if e == nil {
		return text, nil
	}
	return e.resolver.ResolvePlaceholders(text)
}

func (e *StandardEnvironment) ResolveRequiredPlaceholders(text string) (string, error) {
	if e == nil {
		return "", arkerrors.New(arkerrors.CodeInvalidArgument, "environment is nil")
	}
	return e.resolver.ResolveRequiredPlaceholders(text)
}

func (e *StandardEnvironment) ConversionService() *convert.Service {
	if e == nil {
		return nil
	}
	return e.resolver.ConversionService()
}

func (e *StandardEnvironment) SetConversionService(service *convert.Service) error {
	if e == nil {
		return arkerrors.New(arkerrors.CodeInvalidArgument, "environment is nil")
	}
	return e.resolver.SetConversionService(service)
}

func (e *StandardEnvironment) SetRequiredProperties(keys ...string) {
	if e == nil {
		return
	}
	e.resolver.SetRequiredProperties(keys...)
}

func (e *StandardEnvironment) ValidateRequiredProperties() error {
	if e == nil {
		return arkerrors.New(arkerrors.CodeInvalidArgument, "environment is nil")
	}
	return e.resolver.ValidateRequiredProperties()
}

func (e *StandardEnvironment) ActiveProfiles() []string {
	if e == nil {
		return nil
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	return append([]string(nil), e.activeProfiles...)
}

func (e *StandardEnvironment) DefaultProfiles() []string {
	if e == nil {
		return nil
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	return append([]string(nil), e.defaultProfiles...)
}

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
	systemProperties, err := NewMapPropertySource(SystemPropertiesPropertySourceName, map[string]any{})
	if err != nil {
		return err
	}
	systemEnvironment, err := NewSystemEnvironmentPropertySource(SystemEnvironmentPropertySourceName, os.Environ())
	if err != nil {
		return err
	}
	if err := e.propertySources.AddLast(systemProperties); err != nil {
		return err
	}
	return e.propertySources.AddLast(systemEnvironment)
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

func validateProfiles(profiles []string) ([]string, error) {
	cleaned := make([]string, 0, len(profiles))
	for _, profile := range profiles {
		profile = strings.TrimSpace(profile)
		if profile == "" {
			return nil, arkerrors.New(arkerrors.CodeInvalidArgument, "profile is empty")
		}
		if strings.HasPrefix(profile, "!") {
			return nil, arkerrors.Newf(arkerrors.CodeInvalidArgument, "profile %q must not start with !", profile)
		}
		cleaned = append(cleaned, profile)
	}
	sort.Strings(cleaned)
	return util.UniqueStrings(cleaned), nil
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
