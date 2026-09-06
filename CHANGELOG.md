# Changelog

English | [中文](CHANGELOG.zh-CN.md)

All notable changes to Goark Core are recorded here.

## [Unreleased]

No unreleased changes.

## [0.0.1] - 2026-09-06

### Added

- Explicit bean definitions, scopes, aliases, dependency graphs, ordered startup,
  and lifecycle-safe application contexts.
- Configuration environments, property sources, conversion, placeholders,
  configuration-property contracts, and safe GaEL expression evaluation.
- Synchronous ordered events and structured framework errors.
- Go-native Web and MVC contracts for routing, binding, validation, advice,
  filters, interceptors, static resources, views, streaming, WebSocket, and HTTP
  clients.
- Cross-platform CI with Go 1.26 tests, vet, and race gates.

### Changed

- Aligned all used `golang.org/x` modules with their latest stable releases.

### Fixed

- Empty lifecycle transitions no longer fail.
- Bean order is honored during startup traversal.
- Transport-neutral Servlet requests and missing-cookie semantics are preserved.
- Conditional route selection retains the negotiated response media type.

[Unreleased]: https://github.com/goark-projects/goark/compare/v0.0.1...HEAD
[0.0.1]: https://github.com/goark-projects/goark/releases/tag/v0.0.1
