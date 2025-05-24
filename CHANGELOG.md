# CHANGELOG

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Project restructuring for better maintainability
- Comprehensive documentation structure
- CI/CD workflows with GitHub Actions
- Development tools and scripts
- Better package organization

### Changed
- Reorganized code into logical packages (`pkg/`, `internal/`)
- Moved examples to dedicated `examples/` directory
- Improved testing structure with integration and benchmark tests

### Fixed
- Package import paths for better Go module compliance

## [1.0.0] - YYYY-MM-DD

### Added
- Initial release of goclient
- HTTP client wrapper with fluent API
- Circuit breaker pattern implementation
- Request/response streaming support
- Server-Sent Events (SSE) support
- Request and response interceptors
- Retry mechanism with configurable policies
- Connection pooling configuration
- Context-aware operations
- Comprehensive test coverage

### Features
- ✅ Simple API for GET, POST, PUT, DELETE, etc.
- 🧱 Clean abstraction over net/http
- 📦 JSON encoding/decoding helpers
- 🧾 Easy header, query params, and body handling
- 🔄 Built-in retry support
- ⚡ Circuit breaker pattern for resilience
- ⏱ Context-aware requests and interceptors
- 🔌 Extensible with interceptors
- 📥 Support for streaming responses and SSE
