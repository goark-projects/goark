package container

import (
	"reflect"
	"sort"
	"sync"

	arkerrors "github.com/goark-projects/goark/errors"
)

// Container 是不可变定义、并发安全实例缓存的 Bean 容器。
type Container struct {
	definitions map[string]Definition
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
	if err := c.validateDependencies(definitions); err != nil {
		return nil, err
	}
	for typ := range c.typeIndex {
		sort.Strings(c.typeIndex[typ])
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
	if err := validateDefinition(definition); err != nil {
		return err
	}
	if _, exists := c.definitions[definition.Name]; exists {
		return arkerrors.Newf(arkerrors.CodeAlreadyExists, "bean %q already exists", definition.Name)
	}
	c.definitions[definition.Name] = definition.clone()
	c.typeIndex[definition.Type] = append(c.typeIndex[definition.Type], definition.Name)
	return nil
}

func (c *Container) validateDependencies(definitions []Definition) error {
	for _, definition := range definitions {
		for _, dependency := range definition.Dependencies {
			if _, exists := c.definitions[dependency]; !exists {
				return arkerrors.Newf(arkerrors.CodeNotFound, "bean %q depends on missing bean %q", definition.Name, dependency)
			}
		}
	}
	return nil
}
