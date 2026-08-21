package container

import (
	"reflect"
	"sort"
	"sync"

	arkerrors "github.com/goark-projects/goark/errors"
)

// Container 是不可变定义、并发安全实例缓存的 Bean 容器。
type Container struct {
	definitions             map[string]Definition
	singletonOrder          []string
	startupOrder            []string
	allowCircularReferences bool

	typeIndexMu sync.RWMutex
	typeIndex   map[reflect.Type][]string

	singletonMu     sync.Mutex
	singletons      map[string]any
	earlySingletons map[string]any
	inFlight        map[string]*singletonCall
}

// New 基于注册表快照创建容器。
func New(registry *Registry, options ...ContainerOption) (*Container, error) {
	if registry == nil {
		return nil, arkerrors.New(arkerrors.CodeInvalidArgument, "bean registry is nil")
	}
	definitions := registry.Definitions()
	containerOptions := newContainerOptions(options)

	c := &Container{
		definitions:             make(map[string]Definition, len(definitions)),
		allowCircularReferences: containerOptions.allowCircularReferences,
		typeIndex:               make(map[reflect.Type][]string),
		singletons:              make(map[string]any),
		earlySingletons:         make(map[string]any),
		inFlight:                make(map[string]*singletonCall),
	}
	for _, definition := range definitions {
		if err := c.addDefinition(definition); err != nil {
			return nil, err
		}
	}
	c.rebuildTypeIndex(definitions)
	graph, err := c.buildDependencyGraph()
	if err != nil {
		return nil, err
	}
	if err := c.validateDependencyGraph(graph); err != nil {
		return nil, err
	}
	c.singletonOrder = c.computeStartupOrder(graph)
	c.startupOrder = c.filterEagerSingletons(c.singletonOrder)
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

func (c *Container) filterEagerSingletons(names []string) []string {
	filtered := make([]string, 0, len(names))
	for _, name := range names {
		definition := c.definitions[name].normalized()
		if definition.Scope == ScopeSingleton && !definition.Lazy {
			filtered = append(filtered, name)
		}
	}
	return filtered
}
