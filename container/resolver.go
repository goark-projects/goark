package container

import (
	"context"
	"reflect"
	"sort"

	arkerrors "goark.dev/goark/errors"
)

// Get 按名称解析 Bean。
func (c *Container) Get(ctx context.Context, name string) (any, error) {
	if c == nil {
		return nil, arkerrors.New(arkerrors.CodeInvalidArgument, "bean container is nil")
	}
	if ctx == nil {
		return nil, arkerrors.New(arkerrors.CodeInvalidArgument, "context is nil")
	}
	if name == "" {
		return nil, arkerrors.New(arkerrors.CodeInvalidArgument, "bean name is empty")
	}
	ctx, state := ensureResolutionState(ctx)
	return c.resolve(ctx, state, name)
}

// GetByType 按类型解析 Bean；多候选时依次使用 qualifier、Primary、Priority 选择。
func (c *Container) GetByType(ctx context.Context, typ reflect.Type, options ...ResolveOption) (any, error) {
	if c == nil {
		return nil, arkerrors.New(arkerrors.CodeInvalidArgument, "bean container is nil")
	}
	if ctx == nil {
		return nil, arkerrors.New(arkerrors.CodeInvalidArgument, "context is nil")
	}
	if typ == nil {
		return nil, arkerrors.New(arkerrors.CodeInvalidArgument, "bean type is nil")
	}
	resolveOptions, err := newResolveOptions(options)
	if err != nil {
		return nil, err
	}
	name, err := c.selectByType(typ, resolveOptions)
	if err != nil {
		return nil, err
	}
	return c.Get(ctx, name)
}

// GetAllByType 按类型解析全部匹配 Bean，返回顺序与容器启动拓扑一致。
func (c *Container) GetAllByType(ctx context.Context, typ reflect.Type) ([]any, error) {
	if c == nil {
		return nil, arkerrors.New(arkerrors.CodeInvalidArgument, "bean container is nil")
	}
	if ctx == nil {
		return nil, arkerrors.New(arkerrors.CodeInvalidArgument, "context is nil")
	}
	if typ == nil {
		return nil, arkerrors.New(arkerrors.CodeInvalidArgument, "bean type is nil")
	}
	names := c.matchingNamesInStartupOrder(typ)
	values := make([]any, 0, len(names))
	for _, name := range names {
		value, err := c.Get(ctx, name)
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}

// InitializeSingletons 初始化所有非延迟单例 Bean。
func (c *Container) InitializeSingletons(ctx context.Context) error {
	if c == nil {
		return arkerrors.New(arkerrors.CodeInvalidArgument, "bean container is nil")
	}
	if ctx == nil {
		return arkerrors.New(arkerrors.CodeInvalidArgument, "context is nil")
	}
	names := make([]string, 0, len(c.definitions))
	if len(c.startupOrder) > 0 {
		names = append(names, c.startupOrder...)
	} else {
		for name, definition := range c.definitions {
			if definition.Scope == ScopeSingleton && !definition.Lazy {
				names = append(names, name)
			}
		}
		sort.Strings(names)
	}
	for _, name := range names {
		if _, err := c.Get(ctx, name); err != nil {
			return err
		}
	}
	return nil
}

func (c *Container) resolve(ctx context.Context, state *resolutionState, name string) (any, error) {
	if err := ctx.Err(); err != nil {
		return nil, arkerrors.Wrapf(arkerrors.CodeLifecycle, err, "resolve bean %q canceled", name)
	}
	definition, ok := c.definitions[name]
	if !ok {
		return nil, arkerrors.Newf(arkerrors.CodeNotFound, "bean %q not found", name)
	}
	if cycle, ok := state.enter(name); ok {
		if c.allowCircularReferences {
			if value, exists := c.getEarlySingleton(name); exists {
				return value, nil
			}
		}
		return nil, arkerrors.Newf(arkerrors.CodeCircularDependency, "circular dependency detected: %s", cycle)
	}
	defer state.exit(name)

	if definition.Scope == ScopeSingleton {
		return c.resolveSingleton(ctx, state, definition)
	}
	if err := c.resolveDependsOn(ctx, state, definition); err != nil {
		return nil, err
	}
	return c.create(ctx, definition)
}

func (c *Container) resolveDependsOn(ctx context.Context, state *resolutionState, definition Definition) error {
	for _, dependency := range definition.normalized().DependencyDescriptors {
		if dependency.Kind != DependencyKindDependsOn {
			continue
		}
		if _, err := c.resolve(ctx, state, dependency.Name); err != nil {
			return err
		}
	}
	return nil
}

func (c *Container) create(ctx context.Context, definition Definition) (value any, err error) {
	value, err = c.instantiate(ctx, definition)
	if err != nil {
		return nil, err
	}
	if err := c.populate(ctx, definition, value); err != nil {
		return nil, err
	}
	return value, nil
}

func (c *Container) instantiate(ctx context.Context, definition Definition) (value any, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			if recoveredErr, ok := recovered.(error); ok {
				err = arkerrors.Wrapf(arkerrors.CodeCreation, recoveredErr, "bean %q provider panicked", definition.Name)
				return
			}
			err = arkerrors.Newf(arkerrors.CodeCreation, "bean %q provider panicked: %v", definition.Name, recovered)
		}
	}()
	value, err = definition.Factory(ctx, c)
	if err != nil {
		return nil, arkerrors.Wrapf(arkerrors.CodeCreation, err, "failed to create bean %q", definition.Name)
	}
	return normalizeInstance(definition.Name, definition.Type, value)
}

func (c *Container) populate(ctx context.Context, definition Definition, value any) (err error) {
	if definition.DependencyInjector == nil {
		return nil
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			if recoveredErr, ok := recovered.(error); ok {
				err = arkerrors.Wrapf(arkerrors.CodeCreation, recoveredErr, "bean %q injector panicked", definition.Name)
				return
			}
			err = arkerrors.Newf(arkerrors.CodeCreation, "bean %q injector panicked: %v", definition.Name, recovered)
		}
	}()
	if err := definition.DependencyInjector(ctx, c, value); err != nil {
		return arkerrors.Wrapf(arkerrors.CodeCreation, err, "failed to populate bean %q", definition.Name)
	}
	return nil
}
