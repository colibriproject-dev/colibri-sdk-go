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
- Emits HTTP server and client metrics (`http.server.request.duration`, `http.client.request.duration`) via `otelfiber` / `otelhttp`
- Emits database metrics (`db.client.operation.duration`) via `otelsql`
- Emits Go runtime metrics (heap, GC, goroutines) via `opentelemetry-contrib/instrumentation/runtime`
- Enriches every resource with `service.name`, `service.version`, and `service.instance.id`

A disabled signal gets a noop provider, so instrumented code keeps working and simply
reports nothing.

> **Note:** `OTEL_EXPORTER_OTLP_ENDPOINT` should be the base endpoint without signal-specific paths. The SDK appends `/v1/traces` and `/v1/metrics` automatically.

### SDK component metrics

With metrics enabled, the SDK modules report their own metrics. Every attribute comes from
a bounded set: identifiers such as `correlationId`, `messageId`, `userId`, `tenantId`,
storage keys and request paths are recorded on spans only.

| Metric                        | Type             | Unit          | Attributes                          | Module                   |
|-------------------------------|------------------|---------------|-------------------------------------|--------------------------|
| `messaging.published`         | counter          | `{message}`   | `topic`, `result`                   | messaging                |
| `messaging.consumed`          | counter          | `{message}`   | `queue`, `action`, `result`         | messaging                |
| `messaging.process.duration`  | histogram        | `s`           | `queue`, `action`, `result`         | messaging                |
| `messaging.rejected`          | counter          | `{message}`   | `queue`, `action`, `reason`         | messaging                |
| `messaging.in_flight`         | observable gauge | `{message}`   | `queue`                             | messaging                |
| `db.client.connections.*`     | pool metrics     | —             | `db.system`, `pool.name`, …         | cacheDB, via `redisotel` |
| `db.sql.connections.*`        | pool metrics     | —             | `db.instance`, `db.system.name`     | sqlDB, via `otelsql`     |
| `storage.operation`           | counter          | `{operation}` | `operation`, `result`               | storage                  |
| `storage.operation.duration`  | histogram        | `s`           | `operation`, `result`               | storage                  |
| `storage.transferred`         | histogram        | `By`          | `operation`                         | storage                  |
| `http.server.panic.recovered` | counter          | `{panic}`     | `http.request.method`, `http.route` | restserver               |

- `result` is `success`, `error` or `panic` (`panic` for consumed messages only); `reason` is `error` or `panic`.
- `action` is set by the application on `Publish`, so it must come from a fixed set of event names — never an identifier.
- `messaging.rejected` counts messages nacked without requeue. The SDK leaves them to the
  broker dead-letter handling (SQS redrive policy, Pub/Sub dead-letter topic, RabbitMQ
  DLX), so whether one actually reached a DLQ is reported by the broker, not by the SDK.

To assert on metrics in a test, `monitoringtest.Install(t)` swaps in an in-memory reader
for the duration of the test:

```go
recorder := monitoringtest.Install(t)
// ... exercise the code ...
published := recorder.Metric(t, "messaging.published")
monitoringtest.AssertShape(t, published, "{message}", "topic", "result")
```

### Custom metrics

Attributes are passed as an `Attrs` value. Build it once and reuse it: it caches its
provider representation, which is what keeps recording free of allocations.

```go
import (
    "github.com/colibriproject-dev/colibri-sdk-go/pkg/base/monitoring"
    monitoringbase "github.com/colibriproject-dev/colibri-sdk-go/pkg/base/monitoring/colibri-monitoring-base"
)

// Build the attributes once, outside the hot path.
var usersRoute = monitoringbase.NewAttrs("route", "/api/users")

// Counter — monotonically increasing
requests := monitoring.Counter("app.requests", "Total HTTP requests", "1")
requests.AddAttrs(ctx, 1, usersRoute)

// Histogram — value distribution
duration := monitoring.Histogram("app.request.duration", "Request duration", "ms")
duration.RecordAttrs(ctx, float64(elapsed.Milliseconds()), usersRoute)

// Gauge — current value, pushed by the caller
activeConns := monitoring.Gauge("app.connections.active", "Active connections", "1")
activeConns.RecordAttrs(ctx, float64(count), monitoringbase.Attrs{})
```

### Observable gauges

For values that are sampled rather than pushed — pool size, queue depth, cache entries —
register a callback invoked on every collection:

```go
registration := monitoring.ObservableGauge(
    "app.db.connections.open", "Open database connections", "1",
    func(ctx context.Context) []monitoringbase.Observation {
        stats := db.Stats()
        return []monitoringbase.Observation{
            {Value: float64(stats.InUse), Attributes: monitoringbase.NewAttrs("state", "in_use")},
            {Value: float64(stats.Idle), Attributes: monitoringbase.NewAttrs("state", "idle")},
        }
    },
)
defer registration.Unregister()
```

Register a given name once: each call registers its own callback, and the caller owns its
lifetime through the returned `Registration`.

### Migrating from map attributes

`Add` and `Record` taking a `map[string]string` still work and are deprecated. They convert
the map to attributes on every call; the `Attrs` variants do it once.

```go
// Before
requests.Add(ctx, 1, map[string]string{"route": "/api/users"})

// After — build once, reuse
var usersRoute = monitoringbase.NewAttrs("route", "/api/users")
requests.AddAttrs(ctx, 1, usersRoute)

// Or, to migrate mechanically from an existing map
requests.AddAttrs(ctx, 1, monitoringbase.AttrsFromMap(attributes))
```

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

