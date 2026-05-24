[![Reliability Rating](https://sonarcloud.io/api/project_badges/measure?project=colibri-project-dev_colibri-sdk-go&metric=reliability_rating)](https://sonarcloud.io/summary/new_code?id=colibri-project-dev_colibri-sdk-go)
[![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=colibri-project-dev_colibri-sdk-go&metric=alert_status)](https://sonarcloud.io/summary/new_code?id=colibri-project-dev_colibri-sdk-go)
[![Lines of Code](https://sonarcloud.io/api/project_badges/measure?project=colibri-project-dev_colibri-sdk-go&metric=ncloc)](https://sonarcloud.io/summary/new_code?id=colibri-project-dev_colibri-sdk-go)
[![Coverage](https://sonarcloud.io/api/project_badges/measure?project=colibri-project-dev_colibri-sdk-go&metric=coverage)](https://sonarcloud.io/summary/new_code?id=colibri-project-dev_colibri-sdk-go)
[![Maintainability Rating](https://sonarcloud.io/api/project_badges/measure?project=colibri-project-dev_colibri-sdk-go&metric=sqale_rating)](https://sonarcloud.io/summary/new_code?id=colibri-project-dev_colibri-sdk-go)

Disponível em: [Inglês](README.md) | [Português](README.pt-BR.md)

# colibri-sdk-go

Uma biblioteca abrangente para desenvolvimento de aplicações Go com suporte para diversos serviços e funcionalidades.

## Sumário

* [Introdução](#introdução)
* [Status do Projeto](#status-do-projeto)
* [Funcionalidades](#funcionalidades)
* [Instalação](#instalação)
* [Uso](#uso)
* [Contribuições](#contribuições)
* [Licença](#licença)

## Introdução

O `colibri-sdk-go` é um conjunto de ferramentas e bibliotecas projetado para facilitar o desenvolvimento de aplicações Go robustas e escaláveis. O SDK fornece abstrações e implementações para diversos serviços e funcionalidades comuns, permitindo que os desenvolvedores se concentrem na lógica de negócios de suas aplicações.

## Status do Projeto

Em desenvolvimento ativo.

## Funcionalidades

O `colibri-sdk-go` oferece as seguintes funcionalidades:

### Base
- **cloud**: Integrações com serviços de nuvem
- **config**: Gerenciamento de configurações para diferentes ambientes
- **logging**: Sistema de logging flexível e extensível
- **monitoring**: Integração com ferramentas de monitoramento e observabilidade
- **observer**: Implementação do padrão Observer para graceful shutdown
- **security**: Funcionalidades relacionadas à segurança
- **test**: Utilitários para testes
- **transaction**: Gerenciamento de transações
- **types**: Tipos comuns utilizados em toda a biblioteca
- **validator**: Utilitários para validação de dados

### Banco de Dados
- **Cache**: Integração com bancos de dados de cache (como Redis)
- **SQL**: Acesso e gerenciamento de bancos de dados SQL

### Web
- **Cliente REST**: Cliente para consumo de APIs REST
- **Servidor REST**: Servidor para criação de APIs REST

### Outros
- **Mensageria**: Serviços de mensageria
- **Armazenamento**: Serviços de armazenamento
- **Injeção de Dependência**: Sistema de injeção de dependência

## Instalação

Para instalar o `colibri-sdk-go`, utilize o comando go get:

```bash
go get github.com/colibriproject-dev/colibri-sdk-go
```

## Uso

Para inicializar o SDK em sua aplicação:

```go
package main

import (
    "github.com/colibriproject-dev/colibri-sdk-go"
)

func main() {
    // Inicializa o SDK
    colibri.InitializeApp()

    // Sua aplicação aqui
}
```

## Observabilidade

O SDK exporta traces e métricas OpenTelemetry via OTLP HTTP. Configure as variáveis de ambiente abaixo para habilitá-lo:

| Variável | Obrigatória | Descrição |
|---|---|---|
| `OTEL_EXPORTER_OTLP_ENDPOINT` | Sim | Endpoint do coletor OTLP — aceita `host:porta` ou URL completa (ex: `http://localhost:4318`) |
| `OTEL_EXPORTER_OTLP_HEADERS` | Não | Headers no formato `chave=valor` separados por vírgula (ex: `api-key=secret,x-env=prod`) |
| `OTEL_EXPORTER_OTLP_METRICS_ENDPOINT` | Não | Substitui o endpoint apenas para o sinal de métricas. Padrão: `OTEL_EXPORTER_OTLP_ENDPOINT` |
| `OTEL_SERVICE_NAME` | Não | Nome do serviço reportado ao backend. Padrão: valor de `APP_NAME` |

Quando `OTEL_EXPORTER_OTLP_ENDPOINT` está configurado, o SDK automaticamente:
- Exporta **traces** e **métricas** para o coletor OTLP configurado
- Emite métricas de servidor e cliente HTTP (`http.server.request.duration`, `http.client.request.duration`) via `otelfiber` / `otelhttp`
- Emite métricas de banco de dados (`db.client.operation.duration`) via `otelsql`
- Emite métricas de runtime do Go (heap, GC, goroutines) via `opentelemetry-contrib/instrumentation/runtime`
- Enriquece cada resource com `service.name`, `service.version` e `service.instance.id`

> **Atenção:** `OTEL_EXPORTER_OTLP_ENDPOINT` deve ser o endpoint base sem o caminho específico do sinal. O SDK adiciona automaticamente `/v1/traces` e `/v1/metrics`.

### Métricas customizadas

```go
import "github.com/colibriproject-dev/colibri-sdk-go/pkg/base/monitoring"

// Counter — incremento monotônico
requests := monitoring.Counter("app.requests", "Total de requisições HTTP", "1")
requests.Add(ctx, 1, map[string]string{"route": "/api/users"})

// Histogram — distribuição de valores
duration := monitoring.Histogram("app.request.duration", "Duração das requisições", "ms")
duration.Record(ctx, elapsed.Milliseconds(), map[string]string{"status": "200"})

// Gauge — valor corrente
activeConns := monitoring.Gauge("app.connections.active", "Conexões ativas", "1")
activeConns.Record(ctx, float64(count), nil)
```

## Contribuições

Contribuições são bem-vindas! Por favor, leia o [Código de Conduta](CODE_OF_CONDUCT.md) antes de contribuir.

Para contribuir:
1. Faça um fork do repositório
2. Crie uma branch para sua feature (`git checkout -b feature/amazing-feature`)
3. Faça commit de suas mudanças (`git commit -m 'Add some amazing feature'`)
4. Faça push para a branch (`git push origin feature/amazing-feature`)
5. Abra um Pull Request

## Licença

Este projeto está licenciado sob a licença Apache 2.0 - veja o arquivo [LICENSE](LICENSE) para mais detalhes.


