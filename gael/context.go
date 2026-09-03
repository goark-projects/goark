// Package gael 提供 Goark Expression Language 的解析与安全求值能力。
package gael

import (
	"context"

	arkerrors "goark.dev/goark/errors"
)

// PropertyResolver 提供表达式读取配置属性所需的最小契约。
type PropertyResolver interface {
	ContainsProperty(key string) bool
	GetProperty(key string) (string, bool)
}

// Function 是显式注册到表达式上下文中的安全函数。
type Function func(context.Context, ...any) (any, error)

// EvaluationContext 提供表达式求值所需的数据边界。
type EvaluationContext interface {
	Properties() PropertyResolver
	Variable(name string) (any, bool)
	Function(name string) (Function, bool)
}

// StandardEvaluationContext 是并发只读的默认求值上下文。
type StandardEvaluationContext struct {
	properties PropertyResolver
	variables  map[string]any
	functions  map[string]Function
}

// ContextOption 配置默认求值上下文。
type ContextOption func(*StandardEvaluationContext) error

// NewEvaluationContext 创建默认求值上下文。
func NewEvaluationContext(properties PropertyResolver, options ...ContextOption) (*StandardEvaluationContext, error) {
	if properties == nil {
		return nil, arkerrors.New(arkerrors.CodeInvalidArgument, "GaEL property resolver is nil")
	}
	evaluationContext := &StandardEvaluationContext{
		properties: properties,
		variables:  make(map[string]any),
		functions:  make(map[string]Function),
	}
	for _, option := range options {
		if option == nil {
			continue
		}
		if err := option(evaluationContext); err != nil {
			return nil, err
		}
	}
	return evaluationContext, nil
}

// WithVariable 注册只读变量。
func WithVariable(name string, value any) ContextOption {
	return func(evaluationContext *StandardEvaluationContext) error {
		if name == "" {
			return arkerrors.New(arkerrors.CodeInvalidArgument, "GaEL variable name is empty")
		}
		evaluationContext.variables[name] = value
		return nil
	}
}

// WithFunction 注册可由表达式调用的白名单函数。
func WithFunction(name string, function Function) ContextOption {
	return func(evaluationContext *StandardEvaluationContext) error {
		if name == "" {
			return arkerrors.New(arkerrors.CodeInvalidArgument, "GaEL function name is empty")
		}
		if function == nil {
			return arkerrors.Newf(arkerrors.CodeInvalidArgument, "GaEL function %q is nil", name)
		}
		evaluationContext.functions[name] = function
		return nil
	}
}

func (c *StandardEvaluationContext) Properties() PropertyResolver {
	if c == nil {
		return nil
	}
	return c.properties
}

func (c *StandardEvaluationContext) Variable(name string) (any, bool) {
	if c == nil {
		return nil, false
	}
	value, ok := c.variables[name]
	return value, ok
}

func (c *StandardEvaluationContext) Function(name string) (Function, bool) {
	if c == nil {
		return nil, false
	}
	function, ok := c.functions[name]
	return function, ok
}
