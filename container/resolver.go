package container

import (
	"context"
	"reflect"
	"sort"

	arkerrors "github.com/goark-projects/goark/errors"
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

// InitializeSingletons 初始化所有非延迟单例 Bean。
func (c *Container) InitializeSingletons(ctx context.Context) error {
	if c == nil {
		return arkerrors.New(arkerrors.CodeInvalidArgument, "bean container is nil")
	}
	if ctx == nil {
		return arkerrors.New(arkerrors.CodeInvalidArgument, "context is nil")
	}
	names := make([]string, 0, len(c.definitions))
	for name, definition := range c.definitions {
		if definition.Scope == ScopeSingleton && !definition.Lazy {
			names = append(names, name)
		}
	}
	sort.Strings(names)
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
		return nil, arkerrors.Newf(arkerrors.CodeCircularDependency, "circular dependency detected: %s", cycle)
	}
	defer state.exit(name)

	if definition.Scope == ScopeSingleton {
		return c.resolveSingleton(ctx, definition)
	}
	return c.create(ctx, definition)
}

func (c *Container) create(ctx context.Context, definition Definition) (value any, err error) {
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
