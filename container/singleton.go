package container

import (
	"context"
	"sort"

	arkerrors "github.com/goark-projects/goark/errors"
)

type singletonCall struct {
	done  chan struct{}
	value any
	err   error
}

func (c *Container) resolveSingleton(ctx context.Context, state *resolutionState, definition Definition) (any, error) {
	c.singletonMu.Lock()
	if value, ok := c.singletons[definition.Name]; ok {
		c.singletonMu.Unlock()
		return value, nil
	}
	if call, ok := c.inFlight[definition.Name]; ok {
		c.singletonMu.Unlock()
		select {
		case <-call.done:
			return call.value, call.err
		case <-ctx.Done():
			return nil, arkerrors.Wrapf(arkerrors.CodeLifecycle, ctx.Err(), "resolve bean %q canceled", definition.Name)
		}
	}

	call := &singletonCall{done: make(chan struct{})}
	c.inFlight[definition.Name] = call
	c.singletonMu.Unlock()

	var value any
	err := c.resolveDependsOn(ctx, state, definition)
	if err == nil {
		value, err = c.instantiate(ctx, definition)
		if err == nil {
			if c.allowCircularReferences && definition.DependencyInjector != nil {
				c.exposeEarlySingleton(definition.Name, value)
			}
			err = c.populate(ctx, definition, value)
		}
	}

	c.singletonMu.Lock()
	if err == nil {
		c.singletons[definition.Name] = value
	}
	delete(c.earlySingletons, definition.Name)
	delete(c.inFlight, definition.Name)
	call.value = value
	call.err = err
	close(call.done)
	c.singletonMu.Unlock()
	return value, err
}

func (c *Container) exposeEarlySingleton(name string, value any) {
	c.singletonMu.Lock()
	defer c.singletonMu.Unlock()
	if _, exists := c.singletons[name]; exists {
		return
	}
	c.earlySingletons[name] = value
}

func (c *Container) getEarlySingleton(name string) (any, bool) {
	c.singletonMu.Lock()
	defer c.singletonMu.Unlock()
	value, exists := c.earlySingletons[name]
	return value, exists
}

// SingletonInstances 返回已创建单例实例的快照。
func (c *Container) SingletonInstances() map[string]any {
	if c == nil {
		return nil
	}
	c.singletonMu.Lock()
	defer c.singletonMu.Unlock()
	instances := make(map[string]any, len(c.singletons))
	for name, instance := range c.singletons {
		instances[name] = instance
	}
	return instances
}

// SingletonNamesInStartupOrder 返回已创建单例名称快照，顺序与依赖启动顺序一致。
func (c *Container) SingletonNamesInStartupOrder() []string {
	if c == nil {
		return nil
	}
	c.singletonMu.Lock()
	defer c.singletonMu.Unlock()
	names := make([]string, 0, len(c.singletons))
	seen := make(map[string]struct{}, len(c.singletons))
	for _, name := range c.singletonOrder {
		if _, exists := c.singletons[name]; !exists {
			continue
		}
		names = append(names, name)
		seen[name] = struct{}{}
	}
	remaining := make([]string, 0)
	for name := range c.singletons {
		if _, exists := seen[name]; exists {
			continue
		}
		remaining = append(remaining, name)
	}
	sort.Strings(remaining)
	names = append(names, remaining...)
	return names
}
