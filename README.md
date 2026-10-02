# Qall Registry

**Content-addressed storage and distribution for Qall computations.**

Qall Registry is the storage and distribution component of the [Qall](https://github.com/qall-project/qall) framework.

It provides a provider-agnostic registry for storing, versioning and distributing computation objects such as task payloads, computation graph nodes and reusable workers.

## What does it do?

The registry treats computations as immutable, content-addressed objects.

```text
Task Payload
     │
     ▼
  Graph Node
     │
     ▼
  Workflow
```

Objects are identified by their content, enabling deterministic versioning, deduplication and reuse.

The registry deliberately separates **computation** from **infrastructure**: resource bindings and execution state are handled outside the registry.

## Architecture

The repository contains:

- **Client**: client library used by Qall components
- **Server**: registry service implementation
- **OpenAPI / Protobuf**: service interface definitions

## Role in Qall project

```text
                Qall SDK / CLI (local)
                       │
                       ▼
                Qall Registry (local/remote)
                       │
          ┌────────────┼────────────┐
          ▼            ▼            ▼
       Workflows     Tasks       Workers
                       │
                       ▼
                 Qall Daemon
```

The registry is designed to be usable independently of a any specific compute/cloud provider.

## Project status

Qall Registry is an **early-stage component of the Qall project** and its APIs are under active development.

See the main [Qall repository](https://github.com/qall-project/qall) for the overall project and usage examples.
