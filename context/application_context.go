package context

import (
	"sync"

	"goark.dev/goark/container"
	coreenv "goark.dev/goark/core/env"
	"goark.dev/goark/event"
	"goark.dev/goark/lifecycle"
)

// ApplicationContext 是 Goark 核心运行时上下文。
type ApplicationContext struct {
	mu                      sync.RWMutex
	registry                *container.Registry
	env                     coreenv.ConfigurableEnvironment
	configurations          map[string]Configuration
	events                  *event.Bus
	container               *container.Container
	lifecycle               *lifecycle.Manager
	allowCircularReferences bool
	refreshing              bool
	refreshed               bool
	closing                 bool
	closed                  bool
}

// New 创建应用上下文。
func New(options ...Option) (*ApplicationContext, error) {
	env, err := coreenv.NewStandardEnvironment()
	if err != nil {
		return nil, err
	}
	app := &ApplicationContext{
		registry:       container.NewRegistry(),
		env:            env,
		configurations: make(map[string]Configuration),
		events:         event.NewBus(),
	}
	for _, option := range options {
		if option == nil {
			continue
		}
		if err := option(app); err != nil {
			return nil, err
		}
	}
	return app, nil
}

// Environment 返回配置环境。
func (a *ApplicationContext) Environment() coreenv.Environment {
	if a == nil {
		return nil
	}
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.env
}

// ConfigurableEnvironment 返回可配置环境。
func (a *ApplicationContext) ConfigurableEnvironment() coreenv.ConfigurableEnvironment {
	if a == nil {
		return nil
	}
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.env
}

// Events 返回同步事件总线。
func (a *ApplicationContext) Events() *event.Bus {
	if a == nil {
		return nil
	}
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.events
}
