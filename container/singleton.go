package container

import (
	"context"

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
		value, err = c.create(ctx, definition)
	}

	c.singletonMu.Lock()
	if err == nil {
		c.singletons[definition.Name] = value
	}
	delete(c.inFlight, definition.Name)
	call.value = value
	call.err = err
	close(call.done)
	c.singletonMu.Unlock()
	return value, err
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
