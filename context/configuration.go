package context

import (
	stdcontext "context"
	"sort"

	"github.com/goark-projects/goark/container"
	coreenv "github.com/goark-projects/goark/core/env"
	"github.com/goark-projects/goark/core/lang"
	"github.com/goark-projects/goark/core/util"
	arkerrors "github.com/goark-projects/goark/errors"
)

// Configuration 是 Goark 应用的配置装配单元，对齐 Spring 的 Configuration 语义。
type Configuration interface {
	lang.Ordered
	Name() string
	Register(ctx stdcontext.Context, registry *container.Registry) error
}

// ContextAwareConfiguration 可以在注册 Bean 时访问环境和注册表。
type ContextAwareConfiguration interface {
	RegisterWithContext(ctx stdcontext.Context, config ConfigurationContext) error
}

// EnvironmentConfigurer 允许配置单元在 Bean 注册前调整环境。
type EnvironmentConfigurer interface {
	ConfigureEnvironment(ctx stdcontext.Context, env coreenv.ConfigurableEnvironment) error
}

// ConfigurationDescriptor 描述已注册配置单元的只读元数据。
type ConfigurationDescriptor struct {
	Name                  string
	Order                 int
	Priority              bool
	ConfiguresEnvironment bool
}

// Configurations 返回已注册配置单元的只读快照，顺序与 Refresh 装配顺序一致。
func (a *ApplicationContext) Configurations() []ConfigurationDescriptor {
	if a == nil {
		return nil
	}
	a.mu.RLock()
	configurations := make([]Configuration, 0, len(a.configurations))
	for _, configuration := range a.configurations {
		configurations = append(configurations, configuration)
	}
	a.mu.RUnlock()
	sortConfigurations(configurations)

	descriptors := make([]ConfigurationDescriptor, 0, len(configurations))
	for _, configuration := range configurations {
		_, configuresEnvironment := configuration.(EnvironmentConfigurer)
		descriptors = append(descriptors, ConfigurationDescriptor{
			Name:                  configuration.Name(),
			Order:                 configuration.Order(),
			Priority:              util.IsPriorityOrdered(configuration),
			ConfiguresEnvironment: configuresEnvironment,
		})
	}
	return descriptors
}

// RegisterConfiguration 注册应用配置单元，必须在 Refresh 前调用。
func (a *ApplicationContext) RegisterConfiguration(configuration Configuration) error {
	if a == nil {
		return arkerrors.New(arkerrors.CodeInvalidArgument, "application context is nil")
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.closed || a.closing {
		return arkerrors.New(arkerrors.CodeClosed, "application context is closed")
	}
	if a.refreshed || a.refreshing {
		return arkerrors.New(arkerrors.CodeConflict, "application context has already been refreshed")
	}
	return a.registerConfigurationLocked(configuration)
}

func (a *ApplicationContext) registerConfigurationLocked(configuration Configuration) error {
	if err := validateConfiguration(configuration); err != nil {
		return err
	}
	name := configuration.Name()
	if _, exists := a.configurations[name]; exists {
		return arkerrors.Newf(arkerrors.CodeAlreadyExists, "configuration %q already exists", name)
	}
	a.configurations[name] = configuration
	return nil
}

func validateConfiguration(configuration Configuration) error {
	if util.IsNil(configuration) {
		return arkerrors.New(arkerrors.CodeInvalidArgument, "configuration is nil")
	}
	if util.IsBlank(configuration.Name()) {
		return arkerrors.New(arkerrors.CodeInvalidArgument, "configuration name is empty")
	}
	return nil
}

func sortConfigurations(configurations []Configuration) {
	sort.SliceStable(configurations, func(i, j int) bool {
		leftPriority := util.IsPriorityOrdered(configurations[i])
		rightPriority := util.IsPriorityOrdered(configurations[j])
		if leftPriority != rightPriority {
			return leftPriority
		}
		if configurations[i].Order() == configurations[j].Order() {
			return configurations[i].Name() < configurations[j].Name()
		}
		return configurations[i].Order() < configurations[j].Order()
	})
}

func applyConfigurations(ctx stdcontext.Context, env coreenv.ConfigurableEnvironment, registry *container.Registry, configurations []Configuration) error {
	sortConfigurations(configurations)
	for _, configuration := range configurations {
		configurer, ok := configuration.(EnvironmentConfigurer)
		if !ok {
			continue
		}
		if err := ctx.Err(); err != nil {
			return arkerrors.Wrap(arkerrors.CodeLifecycle, err, "configuration environment configuration canceled")
		}
		if err := configurer.ConfigureEnvironment(ctx, env); err != nil {
			return arkerrors.Wrapf(arkerrors.CodeCreation, err, "configuration %q environment configuration failed", configuration.Name())
		}
	}
	for _, configuration := range configurations {
		if err := ctx.Err(); err != nil {
			return arkerrors.Wrap(arkerrors.CodeLifecycle, err, "configuration registration canceled")
		}
		registrationContext := NewConfigurationContext(env, registry)
		if contextAware, ok := configuration.(ContextAwareConfiguration); ok {
			if err := contextAware.RegisterWithContext(ctx, registrationContext); err != nil {
				return arkerrors.Wrapf(arkerrors.CodeCreation, err, "configuration %q registration failed", configuration.Name())
			}
			continue
		}
		if err := configuration.Register(ctx, registry); err != nil {
			return arkerrors.Wrapf(arkerrors.CodeCreation, err, "configuration %q registration failed", configuration.Name())
		}
	}
	return nil
}
