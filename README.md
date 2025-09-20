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

