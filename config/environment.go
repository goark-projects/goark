package config

import (
	"sort"
	"sync"

	arkerrors "github.com/goark-projects/goark/errors"
)

// PropertySource 表示一个配置源，越靠前优先级越高。
type PropertySource interface {
	Name() string
	Lookup(key string) (string, bool)
	Keys() []string
}

// MapSource 是基于内存映射的不可变配置源。
type MapSource struct {
	name   string
	values map[string]string
}

// NewMapSource 创建内存配置源，并复制输入映射。
func NewMapSource(name string, values map[string]string) (*MapSource, error) {
	if name == "" {
		return nil, arkerrors.New(arkerrors.CodeInvalidArgument, "property source name is empty")
	}
	copied := make(map[string]string, len(values))
	for key, value := range values {
		if key == "" {
			return nil, arkerrors.New(arkerrors.CodeInvalidArgument, "property key is empty")
		}
		copied[key] = value
	}
	return &MapSource{name: name, values: copied}, nil
}

func (s *MapSource) Name() string {
	if s == nil {
		return ""
	}
	return s.name
}

func (s *MapSource) Lookup(key string) (string, bool) {
	if s == nil {
		return "", false
	}
	value, ok := s.values[key]
	return value, ok
}

func (s *MapSource) Keys() []string {
	if s == nil {
		return nil
	}
	keys := make([]string, 0, len(s.values))
	for key := range s.values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// Environment 保存有序配置源集合。
type Environment struct {
	mu      sync.RWMutex
	sources []PropertySource
}

// NewEnvironment 创建空配置环境。
func NewEnvironment(sources ...PropertySource) (*Environment, error) {
	env := &Environment{}
	for _, source := range sources {
		if err := env.AddLast(source); err != nil {
			return nil, err
		}
	}
	return env, nil
}

// AddFirst 将配置源加入最高优先级。
func (e *Environment) AddFirst(source PropertySource) error {
	if e == nil {
		return arkerrors.New(arkerrors.CodeInvalidArgument, "environment is nil")
	}
	if err := validateSource(source); err != nil {
		return err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	e.sources = append([]PropertySource{source}, e.sources...)
	return nil
}

// AddLast 将配置源加入最低优先级。
func (e *Environment) AddLast(source PropertySource) error {
	if e == nil {
		return arkerrors.New(arkerrors.CodeInvalidArgument, "environment is nil")
	}
	if err := validateSource(source); err != nil {
		return err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	e.sources = append(e.sources, source)
	return nil
}

// Get 按优先级查找配置值。
func (e *Environment) Get(key string) (string, bool) {
	if e == nil || key == "" {
		return "", false
	}
	e.mu.RLock()
	sources := append([]PropertySource(nil), e.sources...)
	e.mu.RUnlock()
	for _, source := range sources {
		if value, ok := source.Lookup(key); ok {
			return value, true
		}
	}
	return "", false
}

// GetOrDefault 返回配置值，缺失时返回默认值。
func (e *Environment) GetOrDefault(key string, fallback string) string {
	if value, ok := e.Get(key); ok {
		return value
	}
	return fallback
}

// Require 返回必需配置值，缺失时报错。
func (e *Environment) Require(key string) (string, error) {
	if value, ok := e.Get(key); ok {
		return value, nil
	}
	return "", arkerrors.Newf(arkerrors.CodeNotFound, "required property %q not found", key)
}

// Keys 返回所有配置键，去重后按字典序排序。
func (e *Environment) Keys() []string {
	if e == nil {
		return nil
	}
	e.mu.RLock()
	sources := append([]PropertySource(nil), e.sources...)
	e.mu.RUnlock()

	seen := make(map[string]struct{})
	for _, source := range sources {
		for _, key := range source.Keys() {
			seen[key] = struct{}{}
		}
	}
	keys := make([]string, 0, len(seen))
	for key := range seen {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// Snapshot 返回按当前优先级折叠后的配置快照。
func (e *Environment) Snapshot() map[string]string {
	snapshot := make(map[string]string)
	if e == nil {
		return snapshot
	}
	e.mu.RLock()
	sources := append([]PropertySource(nil), e.sources...)
	e.mu.RUnlock()

	for i := len(sources) - 1; i >= 0; i-- {
		source := sources[i]
		for _, key := range source.Keys() {
			if value, ok := source.Lookup(key); ok {
				snapshot[key] = value
			}
		}
	}
	return snapshot
}

func validateSource(source PropertySource) error {
	if source == nil {
		return arkerrors.New(arkerrors.CodeInvalidArgument, "property source is nil")
	}
	if source.Name() == "" {
		return arkerrors.New(arkerrors.CodeInvalidArgument, "property source name is empty")
	}
	return nil
}
