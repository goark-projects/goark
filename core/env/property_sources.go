package env

import (
	"sort"
	"sync"

	"github.com/goark-projects/goark/core/util"
	arkerrors "github.com/goark-projects/goark/errors"
)

// PropertySources 表示只读配置源集合。
type PropertySources interface {
	Contains(name string) bool
	Get(name string) (PropertySource, bool)
	Names() []string
	Snapshot() []PropertySource
}

// MutablePropertySources 保存有序配置源集合，越靠前优先级越高。
type MutablePropertySources struct {
	mu      sync.RWMutex
	sources []PropertySource
}

// NewMutablePropertySources 创建可变配置源集合。
func NewMutablePropertySources(sources ...PropertySource) (*MutablePropertySources, error) {
	out := &MutablePropertySources{}
	for _, source := range sources {
		if err := out.AddLast(source); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// Contains 判断指定名称的配置源是否存在。
func (s *MutablePropertySources) Contains(name string) bool {
	_, ok := s.Get(name)
	return ok
}

// Get 返回指定名称的配置源。
func (s *MutablePropertySources) Get(name string) (PropertySource, bool) {
	if s == nil || name == "" {
		return nil, false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, source := range s.sources {
		if source.Name() == name {
			return source, true
		}
	}
	return nil, false
}

// Names 返回配置源名称，顺序即查找优先级。
func (s *MutablePropertySources) Names() []string {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	names := make([]string, 0, len(s.sources))
	for _, source := range s.sources {
		names = append(names, source.Name())
	}
	return names
}

// Snapshot 返回配置源快照。
func (s *MutablePropertySources) Snapshot() []PropertySource {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]PropertySource(nil), s.sources...)
}

// AddFirst 将配置源加入最高优先级。
func (s *MutablePropertySources) AddFirst(source PropertySource) error {
	if err := validatePropertySource(source); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.containsLocked(source.Name()) {
		return arkerrors.Newf(arkerrors.CodeAlreadyExists, "property source %q already exists", source.Name())
	}
	s.sources = append([]PropertySource{source}, s.sources...)
	return nil
}

// AddLast 将配置源加入最低优先级。
func (s *MutablePropertySources) AddLast(source PropertySource) error {
	if err := validatePropertySource(source); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.containsLocked(source.Name()) {
		return arkerrors.Newf(arkerrors.CodeAlreadyExists, "property source %q already exists", source.Name())
	}
	s.sources = append(s.sources, source)
	return nil
}

// AddBefore 将配置源加入到指定配置源之前。
func (s *MutablePropertySources) AddBefore(relativeName string, source PropertySource) error {
	return s.addRelative(relativeName, source, 0)
}

// AddAfter 将配置源加入到指定配置源之后。
func (s *MutablePropertySources) AddAfter(relativeName string, source PropertySource) error {
	return s.addRelative(relativeName, source, 1)
}

// Replace 替换指定配置源。
func (s *MutablePropertySources) Replace(name string, source PropertySource) error {
	if err := validatePropertySource(source); err != nil {
		return err
	}
	if util.IsBlank(name) {
		return arkerrors.New(arkerrors.CodeInvalidArgument, "property source name is empty")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	index := s.indexOfLocked(name)
	if index < 0 {
		return arkerrors.Newf(arkerrors.CodeNotFound, "property source %q not found", name)
	}
	if source.Name() != name && s.containsLocked(source.Name()) {
		return arkerrors.Newf(arkerrors.CodeAlreadyExists, "property source %q already exists", source.Name())
	}
	s.sources[index] = source
	return nil
}

// Remove 删除指定配置源。
func (s *MutablePropertySources) Remove(name string) (PropertySource, bool) {
	if s == nil || name == "" {
		return nil, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	index := s.indexOfLocked(name)
	if index < 0 {
		return nil, false
	}
	removed := s.sources[index]
	s.sources = append(s.sources[:index], s.sources[index+1:]...)
	return removed, true
}

// PropertyNames 返回所有可枚举配置源中的属性名，去重后排序。
func (s *MutablePropertySources) PropertyNames() []string {
	if s == nil {
		return nil
	}
	seen := make(map[string]struct{})
	for _, source := range s.Snapshot() {
		enumerable, ok := source.(EnumerablePropertySource)
		if !ok {
			continue
		}
		for _, name := range enumerable.PropertyNames() {
			seen[name] = struct{}{}
		}
	}
	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (s *MutablePropertySources) addRelative(relativeName string, source PropertySource, offset int) error {
	if err := validatePropertySource(source); err != nil {
		return err
	}
	if util.IsBlank(relativeName) {
		return arkerrors.New(arkerrors.CodeInvalidArgument, "relative property source name is empty")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.containsLocked(source.Name()) {
		return arkerrors.Newf(arkerrors.CodeAlreadyExists, "property source %q already exists", source.Name())
	}
	index := s.indexOfLocked(relativeName)
	if index < 0 {
		return arkerrors.Newf(arkerrors.CodeNotFound, "relative property source %q not found", relativeName)
	}
	insert := index + offset
	s.sources = append(s.sources, nil)
	copy(s.sources[insert+1:], s.sources[insert:])
	s.sources[insert] = source
	return nil
}

func (s *MutablePropertySources) containsLocked(name string) bool {
	return s.indexOfLocked(name) >= 0
}

func (s *MutablePropertySources) indexOfLocked(name string) int {
	for i, source := range s.sources {
		if source.Name() == name {
			return i
		}
	}
	return -1
}

func validatePropertySource(source PropertySource) error {
	if source == nil {
		return arkerrors.New(arkerrors.CodeInvalidArgument, "property source is nil")
	}
	if util.IsBlank(source.Name()) {
		return arkerrors.New(arkerrors.CodeInvalidArgument, "property source name is empty")
	}
	return nil
}
