package context

import (
	"reflect"

	"goark.dev/goark/container"
	coreenv "goark.dev/goark/core/env"
	arkerrors "goark.dev/goark/errors"
)

// ConfigurationContext 暴露配置注册期可用的核心上下文。
type ConfigurationContext interface {
	Environment() coreenv.Environment
	Registry() *container.Registry
}

type configurationContext struct {
	environment coreenv.Environment
	registry    *container.Registry
}

// NewConfigurationContext 创建配置注册上下文，供生成代码和测试使用。
func NewConfigurationContext(environment coreenv.Environment, registry *container.Registry) ConfigurationContext {
	return configurationContext{
		environment: environment,
		registry:    registry,
	}
}

func (c configurationContext) Environment() coreenv.Environment {
	return c.environment
}

func (c configurationContext) Registry() *container.Registry {
	return c.registry
}

// AnnotationMetadata 描述注解扫描得到的目标元数据。
type AnnotationMetadata struct {
	Name       string
	Type       reflect.Type
	Source     string
	Attributes map[string]any
}

// Attribute 返回指定注解属性。
func (m AnnotationMetadata) Attribute(name string) (any, bool) {
	if m.Attributes == nil || name == "" {
		return nil, false
	}
	value, ok := m.Attributes[name]
	return value, ok
}

// Condition 按环境与注解元数据决定目标是否参与注册。
type Condition interface {
	Matches(ctx ConfigurationContext, metadata AnnotationMetadata) (bool, error)
}

// ConditionFunc 将函数适配为 Condition。
type ConditionFunc func(ctx ConfigurationContext, metadata AnnotationMetadata) (bool, error)

// Matches 执行条件判断。
func (f ConditionFunc) Matches(ctx ConfigurationContext, metadata AnnotationMetadata) (bool, error) {
	if f == nil {
		return false, arkerrors.New(arkerrors.CodeInvalidArgument, "condition function is nil")
	}
	return f(ctx, metadata)
}

// ProfileCondition 使用 Spring 风格 profile 表达式判断是否匹配。
type ProfileCondition struct {
	Expression string
}

// Matches 判断 profile 表达式是否匹配当前环境。
func (c ProfileCondition) Matches(ctx ConfigurationContext, _ AnnotationMetadata) (bool, error) {
	if ctx == nil {
		return false, arkerrors.New(arkerrors.CodeInvalidArgument, "condition context is nil")
	}
	return coreenv.MatchProfileExpression(ctx.Environment(), c.Expression)
}
