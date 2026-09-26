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

O SDK exporta traces e métricas OpenTelemetry. Traces e métricas são sinais independentes:
**as métricas vêm habilitadas por padrão** e podem ser coletadas em `/metrics` sem nenhuma
configuração, enquanto os traces precisam de um coletor OTLP.

| Variável | Obrigatória | Descrição |
|---|---|---|
| `OTEL_EXPORTER_OTLP_ENDPOINT` | Não | Endpoint do coletor OTLP — aceita `host:porta` ou URL completa (ex: `http://localhost:4318`). Habilita traces e o exportador OTLP de métricas |
| `OTEL_EXPORTER_OTLP_HEADERS` | Não | Headers no formato `chave=valor` separados por vírgula (ex: `api-key=secret,x-env=prod`) |
| `OTEL_EXPORTER_OTLP_METRICS_ENDPOINT` | Não | Substitui o endpoint apenas para o sinal de métricas. Padrão: `OTEL_EXPORTER_OTLP_ENDPOINT` |
| `OTEL_SERVICE_NAME` | Não | Nome do serviço reportado ao backend. Padrão: valor de `APP_NAME` |
| `OTEL_TRACES_ENABLED` | Não | Desliga os traces. Padrão `true` — os traces ainda exigem o endpoint do coletor |
| `OTEL_METRICS_ENABLED` | Não | Desliga as métricas, nos dois readers. Padrão `true` |
| `OTEL_METRICS_PROMETHEUS_ENABLED` | Não | Expõe as métricas em `/metrics` pelo registry do Prometheus. Padrão `true` |

### Combinações de sinais

| Configuração | Traces | `/metrics` | Métricas OTLP |
|---|---|---|---|
| Nada configurado (padrão) | desligado | **ligado** | desligado |
| `OTEL_EXPORTER_OTLP_ENDPOINT` configurado | ligado | ligado | ligado |
| Endpoint configurado, `OTEL_METRICS_PROMETHEUS_ENABLED=false` | ligado | desligado | ligado |
| Endpoint configurado, `OTEL_TRACES_ENABLED=false` | desligado | ligado | ligado |
| `OTEL_METRICS_ENABLED=false` e sem endpoint | desligado | desligado | desligado |

Com pelo menos um sinal habilitado, o SDK automaticamente:
- Emite métricas de servidor e cliente HTTP (`http.server.request.duration`, `http.client.request.duration`) via `otelfiber` / `otelhttp`
- Emite métricas de banco de dados (`db.client.operation.duration`) via `otelsql`
- Emite métricas de runtime do Go (heap, GC, goroutines) via `opentelemetry-contrib/instrumentation/runtime`
- Enriquece cada resource com `service.name`, `service.version` e `service.instance.id`

Um sinal desligado recebe um provider noop, então o código instrumentado continua válido e
apenas não reporta nada.

> **Atenção:** `OTEL_EXPORTER_OTLP_ENDPOINT` deve ser o endpoint base sem o caminho específico do sinal. O SDK adiciona automaticamente `/v1/traces` e `/v1/metrics`.

### Métricas dos componentes do SDK

Com as métricas habilitadas, os módulos do SDK reportam as próprias métricas. Todo atributo
vem de um conjunto limitado: identificadores como `correlationId`, `messageId`, `userId`,
`tenantId`, chaves do storage e paths de requisição são registrados apenas nos spans.

| Métrica                       | Tipo             | Unidade       | Atributos                           | Módulo                   |
|-------------------------------|------------------|---------------|-------------------------------------|--------------------------|
| `messaging.published`         | counter          | `{message}`   | `topic`, `result`                   | messaging                |
| `messaging.consumed`          | counter          | `{message}`   | `queue`, `action`, `result`         | messaging                |
| `messaging.process.duration`  | histogram        | `s`           | `queue`, `action`, `result`         | messaging                |
| `messaging.rejected`          | counter          | `{message}`   | `queue`, `action`, `reason`         | messaging                |
| `messaging.in_flight`         | observable gauge | `{message}`   | `queue`                             | messaging                |
| `db.client.connections.*`     | métricas do pool | —             | `db.system`, `pool.name`, …         | cacheDB, via `redisotel` |
| `db.sql.connections.*`        | métricas do pool | —             | `db.instance`, `db.system.name`     | sqlDB, via `otelsql`     |
| `storage.operation`           | counter          | `{operation}` | `operation`, `result`               | storage                  |
| `storage.operation.duration`  | histogram        | `s`           | `operation`, `result`               | storage                  |
| `storage.transferred`         | histogram        | `By`          | `operation`                         | storage                  |
| `http.server.panic.recovered` | counter          | `{panic}`     | `http.request.method`, `http.route` | restserver               |

- `result` é `success`, `error` ou `panic` (`panic` apenas para mensagens consumidas); `reason` é `error` ou `panic`.
- `action` é definido pela aplicação no `Publish`, então deve vir de um conjunto fixo de nomes de evento — nunca um identificador.
- `messaging.rejected` conta as mensagens com nack sem requeue. O SDK as deixa para o
  tratamento de dead-letter do broker (redrive policy do SQS, dead-letter topic do Pub/Sub,
  DLX do RabbitMQ), então se uma delas chegou de fato a uma DLQ é o broker que reporta, não o SDK.

Para verificar métricas em um teste, `monitoringtest.Install(t)` instala um reader em
memória durante o teste:

```go
recorder := monitoringtest.Install(t)
// ... exercita o código ...
published := recorder.Metric(t, "messaging.published")
monitoringtest.AssertShape(t, published, "{message}", "topic", "result")
```

### Métricas customizadas

Os atributos são passados como um valor `Attrs`. Construa uma vez e reutilize: ele guarda a
representação do provider em cache, e é isso que mantém a gravação sem alocações.

```go
import (
    "github.com/colibriproject-dev/colibri-sdk-go/pkg/base/monitoring"
    monitoringbase "github.com/colibriproject-dev/colibri-sdk-go/pkg/base/monitoring/colibri-monitoring-base"
)

// Construa os atributos uma vez, fora do caminho quente.
var usersRoute = monitoringbase.NewAttrs("route", "/api/users")

// Counter — incremento monotônico
requests := monitoring.Counter("app.requests", "Total de requisições HTTP", "1")
requests.AddAttrs(ctx, 1, usersRoute)

// Histogram — distribuição de valores
duration := monitoring.Histogram("app.request.duration", "Duração das requisições", "ms")
duration.RecordAttrs(ctx, float64(elapsed.Milliseconds()), usersRoute)

// Gauge — valor corrente, enviado pelo chamador
activeConns := monitoring.Gauge("app.connections.active", "Conexões ativas", "1")
activeConns.RecordAttrs(ctx, float64(count), monitoringbase.Attrs{})
```

### Gauges observáveis

Para valores amostrados em vez de enviados — tamanho de pool, profundidade de fila,
entradas em cache — registre um callback chamado a cada coleta:

```go
registration := monitoring.ObservableGauge(
    "app.db.connections.open", "Conexões abertas com o banco", "1",
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

Registre cada nome uma única vez: cada chamada registra o próprio callback, e o ciclo de
vida fica com o chamador, pelo `Registration` retornado.

### Migrando dos atributos em map

`Add` e `Record` recebendo `map[string]string` continuam funcionando e estão marcados como
deprecated. Eles convertem o map em atributos a cada chamada; as variantes com `Attrs`
fazem isso uma vez só.

```go
// Antes
requests.Add(ctx, 1, map[string]string{"route": "/api/users"})

// Depois — construa uma vez e reutilize
var usersRoute = monitoringbase.NewAttrs("route", "/api/users")
requests.AddAttrs(ctx, 1, usersRoute)

// Ou, para migrar mecanicamente a partir de um map existente
requests.AddAttrs(ctx, 1, monitoringbase.AttrsFromMap(attributes))
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


