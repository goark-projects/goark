package container

import (
	"reflect"
	"sort"
	"strings"
	"sync"

	arkerrors "github.com/goark-projects/goark/errors"
)

// Container 是不可变定义、并发安全实例缓存的 Bean 容器。
type Container struct {
	definitions map[string]Definition

	typeIndexMu sync.RWMutex
	typeIndex   map[reflect.Type][]string

	singletonMu sync.Mutex
	singletons  map[string]any
	inFlight    map[string]*singletonCall
}

// New 基于注册表快照创建容器。
func New(registry *Registry) (*Container, error) {
	if registry == nil {
		return nil, arkerrors.New(arkerrors.CodeInvalidArgument, "bean registry is nil")
	}
	definitions := registry.Definitions()

	c := &Container{
		definitions: make(map[string]Definition, len(definitions)),
		typeIndex:   make(map[reflect.Type][]string),
		singletons:  make(map[string]any),
		inFlight:    make(map[string]*singletonCall),
	}
	for _, definition := range definitions {
		if err := c.addDefinition(definition); err != nil {
			return nil, err
		}
	}
	c.rebuildTypeIndex(definitions)
	if err := c.validateDependencies(definitions); err != nil {
		return nil, err
	}
	return c, nil
}

// Names 返回已注册 Bean 名称。
func (c *Container) Names() []string {
	if c == nil {
		return nil
	}
	names := make([]string, 0, len(c.definitions))
	for name := range c.definitions {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (c *Container) addDefinition(definition Definition) error {
	definition = definition.normalized()
	if err := validateDefinition(definition); err != nil {
		return err
	}
	if _, exists := c.definitions[definition.Name]; exists {
		return arkerrors.Newf(arkerrors.CodeAlreadyExists, "bean %q already exists", definition.Name)
	}
	c.definitions[definition.Name] = definition.clone()
	return nil
}

func (c *Container) validateDependencies(definitions []Definition) error {
	for _, definition := range definitions {
		normalized := definition.normalized()
		for _, dependency := range normalized.DependsOn {
			if _, exists := c.definitions[dependency]; !exists {
				return arkerrors.Newf(arkerrors.CodeNotFound, "bean %q depends on missing bean %q", normalized.Name, dependency)
			}
		}
	}
	return c.validateDependencyCycles(definitions)
}

type dependencyVisitState uint8

const (
	dependencyVisiting dependencyVisitState = iota + 1
	dependencyVisited
)

func (c *Container) validateDependencyCycles(definitions []Definition) error {
	states := make(map[string]dependencyVisitState, len(definitions))
	path := make([]string, 0, len(definitions))
	var visit func(string) error
	visit = func(name string) error {
		switch states[name] {
		case dependencyVisiting:
			return arkerrors.Newf(arkerrors.CodeCircularDependency, "circular depends-on detected: %s", dependencyCycle(path, name))
		case dependencyVisited:
			return nil
		}
		definition, exists := c.definitions[name]
		if !exists {
			return nil
		}
		states[name] = dependencyVisiting
		path = append(path, name)
		for _, dependency := range definition.normalized().DependsOn {
			if err := visit(dependency); err != nil {
				return err
			}
		}
		path = path[:len(path)-1]
		states[name] = dependencyVisited
		return nil
	}
	for _, definition := range definitions {
		if err := visit(definition.normalized().Name); err != nil {
			return err
		}
	}
	return nil
}

func dependencyCycle(path []string, target string) string {
	index := 0
	for i, name := range path {
		if name == target {
			index = i
			break
		}
	}
	cycle := append([]string(nil), path[index:]...)
	cycle = append(cycle, target)
	return strings.Join(cycle, " -> ")
}
