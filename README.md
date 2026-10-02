[![Reliability Rating](https://sonarcloud.io/api/project_badges/measure?project=colibri-project-dev_colibri-sdk-go&metric=reliability_rating)](https://sonarcloud.io/summary/new_code?id=colibri-project-dev_colibri-sdk-go)
[![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=colibri-project-dev_colibri-sdk-go&metric=alert_status)](https://sonarcloud.io/summary/new_code?id=colibri-project-dev_colibri-sdk-go)
[![Lines of Code](https://sonarcloud.io/api/project_badges/measure?project=colibri-project-dev_colibri-sdk-go&metric=ncloc)](https://sonarcloud.io/summary/new_code?id=colibri-project-dev_colibri-sdk-go)
[![Coverage](https://sonarcloud.io/api/project_badges/measure?project=colibri-project-dev_colibri-sdk-go&metric=coverage)](https://sonarcloud.io/summary/new_code?id=colibri-project-dev_colibri-sdk-go)
[![Maintainability Rating](https://sonarcloud.io/api/project_badges/measure?project=colibri-project-dev_colibri-sdk-go&metric=sqale_rating)](https://sonarcloud.io/summary/new_code?id=colibri-project-dev_colibri-sdk-go)

Available languages: [English](README.md) | [Português](README.pt-BR.md)

# colibri-sdk-go

A comprehensive library for building Go applications with support for multiple services and features.

## Table of Contents

* [Introduction](#introduction)
* [Project Status](#project-status)
* [Features](#features)
* [Installation](#installation)
* [Usage](#usage)
* [Contributing](#contributing)
* [License](#license)

## Introduction

`colibri-sdk-go` is a set of tools and libraries designed to make it easier to develop robust and scalable Go applications. The SDK provides abstractions and implementations for a variety of common services and features, allowing developers to focus on their application’s business logic.

## Project Status

Actively under development.

## Features

`colibri-sdk-go` offers the following features:

### Base
- **cloud**: Cloud service integrations
- **config**: Configuration management for different environments
- **logging**: Flexible and extensible logging system
- **monitoring**: Integration with monitoring and observability tools
- **observer**: Observer pattern implementation for graceful shutdown
- **security**: Security-related functionality
- **test**: Testing utilities
- **transaction**: Transaction management
- **types**: Common types used across the library
- **validator**: Data validation utilities

### Database
- **Cache**: Integration with cache databases (such as Redis)
- **SQL**: Access and management of SQL databases

### Web
- **REST Client**: Client for consuming REST APIs
- **REST Server**: Server for building REST APIs

### Other
- **Messaging**: Messaging services
- **Storage**: Storage services
- **Dependency Injection**: Dependency injection system

## Installation

To install `colibri-sdk-go`, use go get:

```bash
go get github.com/colibriproject-dev/colibri-sdk-go
```

## Usage

To initialize the SDK in your application:

```go
package main

import (
    "github.com/colibriproject-dev/colibri-sdk-go"
)

func main() {
    // Initialize the SDK
    colibri.InitializeApp()

    // Your application here
}
```

## Observability

The SDK exports OpenTelemetry traces and metrics. Traces and metrics are independent
signals: **metrics are enabled by default** and scrapable on `/metrics` with no
configuration, while traces need an OTLP collector.

| Variable                              | Required | Description                                                                                                                           |
|---------------------------------------|----------|---------------------------------------------------------------------------------------------------------------------------------------|
| `OTEL_EXPORTER_OTLP_ENDPOINT`         | No       | OTLP collector endpoint — accepts `host:port` or full URL (e.g. `http://localhost:4318`). Enables traces and the OTLP metric exporter |
| `OTEL_EXPORTER_OTLP_HEADERS`          | No       | Comma-separated `key=value` headers (e.g. `api-key=secret,x-env=prod`)                                                                |
| `OTEL_EXPORTER_OTLP_METRICS_ENDPOINT` | No       | Override endpoint for the metrics signal only. Defaults to `OTEL_EXPORTER_OTLP_ENDPOINT`                                              |
| `OTEL_SERVICE_NAME`                   | No       | Service name reported to the backend. Defaults to `APP_NAME`                                                                          |
| `OTEL_TRACES_ENABLED`                 | No       | Kill switch for traces. Default `true` — traces still require a collector endpoint                                                    |
| `OTEL_METRICS_ENABLED`                | No       | Kill switch for metrics, covering both readers. Default `true`                                                                        |
| `OTEL_METRICS_PROMETHEUS_ENABLED`     | No       | Exposes metrics on `/metrics` through the Prometheus registry. Default `true`                                                         |

### Signal combinations

| Configuration                                         | Traces | `/metrics` | OTLP metrics |
|-------------------------------------------------------|--------|------------|--------------|
| Nothing set (default)                                 | off    | **on**     | off          |
| `OTEL_EXPORTER_OTLP_ENDPOINT` set                     | on     | on         | on           |
| Endpoint set, `OTEL_METRICS_PROMETHEUS_ENABLED=false` | on     | off        | on           |
| Endpoint set, `OTEL_TRACES_ENABLED=false`             | off    | on         | on           |
| `OTEL_METRICS_ENABLED=false` and no endpoint          | off    | off        | off          |

With at least one signal enabled the SDK automatically:
- Emits HTTP server metrics (`http.server.request.duration`, …) from its own middleware, and HTTP client metrics (`http.client.request.duration`) via `otelhttp`
- Emits SQL call and connection pool metrics (`db.sql.*`) via `otelsql`, and Redis pool metrics (`db.client.connections.*`) via `redisotel`
- Emits messaging, storage and recovered-panic metrics from the SDK modules
- Emits Go runtime metrics (memory, GC, goroutines) via `opentelemetry-contrib/instrumentation/runtime`
- Enriches every resource with `service.name`, `service.version`, and `service.instance.id`

A disabled signal gets a noop provider, so instrumented code keeps working and simply
reports nothing.

> **Note:** `OTEL_EXPORTER_OTLP_ENDPOINT` should be the base endpoint without signal-specific paths. The SDK appends `/v1/traces` and `/v1/metrics` automatically.

### Metrics

The metrics the SDK emits, the naming, unit and cardinality conventions, and how to record
business metrics — building `Attrs` once, observable gauges, testing with
`monitoringtest` — are documented in
[docs/observability/metrics.md](docs/observability/metrics.md).

### Dashboards and alerts

Grafana dashboards (HTTP, messaging, data stores, Go runtime) and Prometheus alert rules for
the metrics above ship in [observability/](observability/README.md), versioned with the code
that emits the metrics. CI fails when one of them queries a metric the SDK does not emit.

## Contributing

Contributions are welcome! Please read the [Code of Conduct](CODE_OF_CONDUCT.md) before contributing.

To contribute:
1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the Apache 2.0 License - see the [LICENSE](LICENSE) file for details.

