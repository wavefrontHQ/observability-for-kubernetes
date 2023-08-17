# Top Level MonoRepo Structure for the Test Proxy

## Context

We decided to develop a test proxy written in go that was a drop in replacement for the wavefront proxy. This test proxy recorded all metrics/logs in memory and exposed an API for our end-to-end integration tests.

Since the test proxy is located in the Collector's directory, any test proxy changes that add/remove packages from the Collector's OSL dependencies, require us to create a new OSL request for the Collector. If we request a new OSL for the Collector, then we are also required to create a new OSL request for the Operator.

## Decision

We will move the test proxy and dependencies to a separate top level directory and use the repo structure below:

```
.
├── adr/
├── ci/
├── collector
│   ├── cmd
│   │   └── wavefront-collector
│   │       └── main.go
│   ├── internal
│   │   └── <collector packages>
│   ├── vendor
│   │   └── <collector dependencies>
│   ├── Dockerfile
│   ├── go.mod
│   ├── go.sum
│   ├── Makefile
│   └── open_source_licenses.txt
├── deploy/
├── docs/
├── operator/
├── scripts/
├── test-proxy
│   └── cmd
│   │   └── test-proxy
│   │       └── main.go
│   ├── internal
│   │   └── <test-proxy packages>
│   ├── vendor
│   │   └── <test-proxy dependencies>
│   ├── Dockerfile
│   ├── go.mod
│   ├── go.sum
│   └── Makefile
├── ci.Jenkinsfile
├── Makefile
└── README.md
```

This repo structure for the test proxy will have the following notable changes:

- The test proxy dependencies will be located in the `test-proxy/vendor/` directory and `test-proxy/go.mod` file.
- The test proxy dependencies will no longer be coupled with the Collector's dependencies.
- Any test proxy code changes that add/remove dependencies will no longer require us to create a new OSL request for the Collector.

## Other options considered

Stay with the current repo structure for the Collector:

```
.
├── adr/
├── ci/
├── collector
│   ├── cmd
│   │   |── wavefront-collector
│   │   |   └── main.go
│   │   └── test-proxy
│   │       └── main.go
│   ├── internal
│   │   └── <collector and test-proxy packages>
│   ├── vendor
│   │   └── <collector and test-proxy dependencies>
│   ├── Dockerfile
│   ├── Dockerfile.test-proxy
│   ├── go.mod
│   ├── go.sum
│   ├── Makefile
│   └── open_source_licenses.txt
├── deploy/
├── docs/
├── operator/
├── scripts/
├── ci.Jenkinsfile
├── Makefile
└── README.md
```

As we tried to add test proxy support for HTTP, we realized the following pain point and decided to move the test proxy to its own top-level directory:

- Merging test proxy code with new dependencies would require a new OSL request for the Collector, which would also require a new OSL request for the Operator.

## Status

[Implemented](https://github.com/wavefrontHQ/observability-for-kubernetes/pull/303)

## Consequences

We have to add the test proxy to our `dependabot.yaml`, so we stay up to date on test proxy dependencies.

Decoupling the test proxy from the collector is more code that we have to maintain. For example, we have to add more test proxy code when there are internal collector packages we no longer have access to, (e.g. `events` package from `collector/internal/events`).
