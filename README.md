# goark

![goark](assets/goark-readme-logo.png)

`goark` is the core repository for the Goark ecosystem: a Go-native application framework intended to provide a Spring-like engineering model while staying close to Go's simplicity, explicit dependency boundaries, and concurrency primitives.

This module provides the core runtime contracts: bean registration, dependency resolution, configuration environment, application context, lifecycle hooks, synchronous events, and framework error types.

## Goals

- Provide a production-oriented Go framework for service applications.
- Keep core contracts small, explicit, and testable.
- Separate runtime bootstrap concerns into [`goark-projects/boot`](https://github.com/goark-projects/boot).
- Favor Go-native design over Java-style reflection-heavy abstractions.
- Use compile-time generated registration code instead of classpath scanning.
- Build clear extension points for configuration, lifecycle, web, data, messaging, and observability extensions.

## Core Packages

```text
.
├── core/        # lang, util, resource, convert, env contracts
├── container/   # Bean definitions, registry, resolver, scopes
├── context/     # ApplicationContext, Configuration, refresh lifecycle
├── errors/      # Framework error codes and wrappers
├── event/       # Synchronous ordered event bus
├── lifecycle/   # Start, Stop, Close lifecycle manager
└── internal/    # Internal helpers
```

## Design Boundary

Go does not have Java-style runtime class metadata or classpath scanning. Goark keeps that responsibility out of the core runtime.

- `goark`: accepts explicit bean definitions and runs the application context.
- `boot`: provides startup conventions, default config discovery, profile layering, and configuration assembly on top of core contracts.
- `cli`: discovers source metadata and generates deterministic registration code.

Generated code should prefer `context.Configuration` / `goark.Configuration` as the stable assembly target, then call normal Go APIs such as `goark.Register`, `goark.RegisterInstance`, and `container.NewDefinition` inside the configuration.

`core/env` follows the Spring Framework `core.env` boundary: `Environment`, `PropertyResolver`, `PropertySource`, `PropertySources`, `ConfigurableEnvironment`, and explicit file-backed `PropertySource` loading. Spring Boot style configuration discovery, profile layering, and binding belongs in `boot`, not in the core module.

## Minimal Usage

```go
package main

import (
	"context"

	"github.com/goark-projects/goark"
	"github.com/goark-projects/goark/container"
)

type UserRepository struct{}

type UserService struct {
	Repository *UserRepository
}

func main() {
	ctx := context.Background()
	app := goark.MustNew()

	_ = goark.Register(app, "userRepository", func(context.Context, container.Resolver) (*UserRepository, error) {
		return &UserRepository{}, nil
	})
	_ = goark.Register(app, "userService", func(ctx context.Context, resolver container.Resolver) (*UserService, error) {
		repository, err := goark.Get[*UserRepository](ctx, resolver, "userRepository")
		if err != nil {
			return nil, err
		}
		return &UserService{Repository: repository}, nil
	}, container.WithDependencies("userRepository"))

	if err := app.Start(ctx); err != nil {
		panic(err)
	}
	defer func() {
		if err := app.Close(ctx); err != nil {
			panic(err)
		}
	}()

	_ = goark.MustGet[*UserService](ctx, app, "userService")
}
```

## Configuration Assembly

CLI-generated code should target `goark.Configuration` instead of emitting unrelated registration functions.

```go
package main

import (
	"context"

	"github.com/goark-projects/goark"
	"github.com/goark-projects/goark/container"
)

type UserConfiguration struct{}

func (UserConfiguration) Name() string {
	return "user"
}

func (UserConfiguration) Order() int {
	return 100
}

func (UserConfiguration) Register(ctx context.Context, registry *container.Registry) error {
	_ = ctx
	if err := container.RegisterInstance[*UserRepository](registry, "userRepository", &UserRepository{}); err != nil {
		return err
	}
	return container.Register[*UserService](registry, "userService", func(ctx context.Context, resolver container.Resolver) (*UserService, error) {
		repository, err := container.Get[*UserRepository](ctx, resolver, "userRepository")
		if err != nil {
			return nil, err
		}
		return &UserService{Repository: repository}, nil
	}, container.WithDependencies("userRepository"))
}

func run(ctx context.Context) error {
	app := goark.MustNew(goark.WithConfiguration(UserConfiguration{}))
	if err := app.Start(ctx); err != nil {
		return err
	}
	defer app.Close(ctx)

	_ = goark.MustGet[*UserService](ctx, app, "userService")
	return nil
}
```

## Installation

```bash
go get github.com/goark-projects/goark
```

## Repository Status

This repository is in active early development. Public APIs should be treated as unstable until the first tagged release.

## Development

Requirements:

- Go 1.25 or later
- Git

Useful commands:

```bash
go mod tidy
go test ./...
```

## Repository Layout

```text
.
├── assets/       # README and brand assets
├── container/    # Bean container core
├── context/      # Application context
├── core/         # Spring-core style Go contracts
├── errors/       # Framework errors
├── event/        # Event bus
├── lifecycle/    # Lifecycle manager
├── go.mod        # Go module definition
├── LICENSE       # Apache License 2.0
└── README.md     # Project overview
```

## Related Repositories

- [`goark-projects/goark`](https://github.com/goark-projects/goark): core framework contracts.
- [`goark-projects/boot`](https://github.com/goark-projects/boot): application bootstrap and convention layer.
- [`goark-projects/cli`](https://github.com/goark-projects/cli): scaffolding and compile-time code generation.

## License

`goark` is released under the Apache License 2.0. See [LICENSE](LICENSE) for details.
