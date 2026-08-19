# goark

![goark](assets/goark-readme-logo.png)

`goark` is the core repository for the Goark ecosystem: a Go-native application framework intended to provide a Spring-like engineering model while staying close to Go's simplicity, explicit dependency boundaries, and concurrency primitives.

The project is in its initial public bootstrap stage. This repository currently defines the root module and project conventions; framework APIs will be added here as the core contracts stabilize.

## Goals

- Provide a production-oriented Go framework for service applications.
- Keep core contracts small, explicit, and testable.
- Separate runtime bootstrap concerns into [`goark-projects/boot`](https://github.com/goark-projects/boot).
- Favor Go-native design over Java-style reflection-heavy abstractions.
- Build clear extension points for configuration, lifecycle, web, data, messaging, and observability modules.

## Module

```bash
go get github.com/goark-projects/goark
```

## Repository Status

This repository is an early skeleton. Public APIs should be treated as unstable until the first tagged release.

## Development

Requirements:

- Go 1.25 or later
- Git

Useful commands:

```bash
go mod tidy
go list -m
```

## Repository Layout

```text
.
├── assets/      # README and brand assets
├── go.mod       # Go module definition
├── LICENSE      # Apache License 2.0
└── README.md    # Project overview
```

## Related Repositories

- [`goark-projects/goark`](https://github.com/goark-projects/goark): core framework contracts.
- [`goark-projects/boot`](https://github.com/goark-projects/boot): application bootstrap and convention layer.

## License

`goark` is released under the Apache License 2.0. See [LICENSE](LICENSE) for details.
