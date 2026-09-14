# Relay

[![Stability: Alpha](https://img.shields.io/badge/stability-alpha-f59e0b)](https://docs.harvor.io/reference/stability-labels)

**Open-source event routing for your infrastructure.**

Relay is a lightweight, self-hosted event ingestion and routing service from [Harvor](https://harvor.io).

Applications publish events to Relay, and Relay handles getting them where they need to go. Configure your sources, routes, and destinations without coupling producers to the systems consuming their events.

See [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for an overview of how Relay is put together.

> [!WARNING]
> **Alpha**
>
> Relay carries Harvor's [Alpha stability label](https://docs.harvor.io/reference/stability-labels): early and experimental, not yet ready for production use. Not all planned features are implemented yet, and APIs, configuration formats, and behavior may still change as we learn.
>
> Contributions, feedback, and ideas are welcome.

## Why Relay?

Event-driven systems often require applications to know too much about their infrastructure.

A service shouldn't need to know whether an event ultimately goes to a webhook, SQS queue, RabbitMQ exchange, Kafka topic, or something else. It should publish the event and move on.

Relay provides a boundary between event producers and event infrastructure.

```text
                         ┌──▶ Webhook
                         │
HTTP ──┐                 ├──▶ SQS
       │                 │
       ├──▶ Relay ───────┼──▶ RabbitMQ
       │                 │
gRPC ──┘                 ├──▶ Kafka
                         │
                         └──▶ ...
```

Producers send events to Relay through a supported source.

Relay determines where those events should go based on your configuration and delivers them to the appropriate destinations.

## Goals

Relay aims to be:

- **Simple** — easy to run, configure, and understand.
- **Infrastructure agnostic** — producers don't need to know how events are ultimately delivered.
- **Self-hostable** — run Relay in your own infrastructure.
- **Declarative** — define event infrastructure alongside your application using configuration files.
- **Extensible** — support different sources and destinations without changing your applications.
- **Production focused** — provide the controls needed to operate event routing reliably.

## Planned Features

Relay is still early in development. The planned feature set includes:

### Event ingestion

Receive events through multiple source protocols, including:

- HTTP
- gRPC

### Routing

Configure how events flow from sources to one or more destinations.

```text
source → route → destination
```

Routes will allow event producers to remain independent from the infrastructure and consumers behind them.

### Destinations

Planned destinations include:

- Webhooks
- Amazon SQS
- RabbitMQ
- Apache Kafka
- Additional destinations over time

### Infrastructure as Code

Provision Relay configuration declaratively using YAML.

For example:

```yaml
sources:
  - name: orders
    type: http

destinations:
  - name: order-processing
    type: sqs
    queue: orders

routes:
  - source: orders
    destination: order-processing
```

> [!NOTE]
> The configuration format above is illustrative and is not yet a stable API.

Relay configuration is intended to be version controlled, reviewed, and deployed alongside the applications that depend on it.

### Traffic controls

Planned operational controls include:

- Rate limiting
- Retry policies
- Delivery timeouts
- Failure handling
- Additional delivery and reliability controls as Relay evolves

## Example

An application might publish:

```json
{
  "type": "order.created",
  "data": {
    "order_id": "12345"
  }
}
```

Relay could then route that event to multiple destinations:

```text
order.created
      │
      ├──▶ SQS → Order Processing
      │
      ├──▶ Kafka → Analytics
      │
      └──▶ Webhook → External Integration
```

The producer only needs to know how to publish the event to Relay.

## Status

Relay is currently in early development.

The initial focus is establishing the core event ingestion, routing, configuration, and delivery model before expanding the supported integrations and operational features.

## Running locally

Relay is a single Go binary. The entrypoint is [cmd/relay](cmd/relay), and `relay serve` starts the HTTP API.

```sh
make run     # runs `relay serve` on http://localhost:8080
```

```sh
curl -i http://localhost:8080/api/v1/healthz
```

OpenAPI documentation is served at <http://localhost:8080/docs/api>.

See [CONTRIBUTING.md](CONTRIBUTING.md) for the full local development guide — the Docker Compose workflow, project layout, configuration, tests, and the checks CI enforces.

## Contributing

Relay is open source and contributions are welcome.

Bug reports, feature ideas, documentation improvements, and code contributions are all valuable as the project takes shape.

General Harvor contribution guidelines live in the [Harvor community repository](https://github.com/harvor-io/community/blob/main/CONTRIBUTING.md). For working in this repository specifically — running Relay locally, project structure, and required checks — see [CONTRIBUTING.md](CONTRIBUTING.md).

## Harvor

Relay is part of **Harvor**, a collection of open-source, production-focused infrastructure building blocks designed to help developers spend less time rebuilding common infrastructure.

**Infrastructure you can trust.**

Learn more at [harvor.io](https://harvor.io).
