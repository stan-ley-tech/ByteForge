# Architecture

ByteForge is a Go engine with two front ends — a REST/WebSocket API behind a web UI, and a CLI —
that both sit on the same internal packages. Nothing collection-related, request-related, or
assertion-related is implemented twice.

```
                        ┌─────────────┐
                        │   web/ (UI) │
                        └──────┬──────┘
                               │ HTTP + WebSocket
                        ┌──────▼──────┐        ┌─────────────┐
                        │ internal/api│        │     cli/    │
                        └──────┬──────┘        └──────┬──────┘
                               │                       │
                 ┌─────────────┼───────────────────────┘
                 │             │
          ┌──────▼─────┐ ┌─────▼──────┐
          │internal/   │ │internal/   │
          │storage     │ │runner      │
          │(SQLite)    │ │            │
          └────────────┘ └──────┬─────┘
                                 │
        ┌───────────────┬───────┼────────────────┬────────────────┐
        │               │       │                │                │
┌───────▼──────┐ ┌───────▼────┐ ┌▼────────────┐ ┌─▼────────────┐ ┌─▼───────────┐
│collections   │ │environments│ │ assertions   │ │ httpclient   │ │  jsonpath   │
│(request/coll.│ │(vars +     │ │(expression   │ │(pooled,      │ │ (path       │
│ model, codec)│ │ templating)│ │ parser/eval) │ │ retrying,    │ │  evaluator) │
└──────────────┘ └────────────┘ └──────────────┘ │ streaming)   │ └─────────────┘
                                                   └──────────────┘
```

## Package responsibilities

- **`internal/httpclient`** — the only package that talks to the network. Wraps `net/http` with a
  shared connection pool, configurable timeouts, exponential-backoff retries with jitter, and a
  `Stream` path for responses too large to buffer.
- **`internal/collections`** — the request/collection domain model plus its JSON codec. A
  collection never stores resolved secrets: auth and body fields hold `{{VARIABLE}}` references.
- **`internal/environments`** — named variable sets and the `{{name}}` template renderer.
  `Redacted()` and `ExportSafe()` are the two places secret values get masked or stripped —
  everything downstream of them is safe to log or write to a file.
- **`internal/assertions`** — parses and evaluates the assertion language (see
  [assertions.md](assertions.md)).
- **`internal/jsonpath`** — the small JSONPath subset assertions and extraction use to reach into
  a decoded JSON body.
- **`internal/runner`** — the execution engine. `Run` walks a collection's requests in order,
  rendering templates, sending each one, evaluating its assertions, and threading any extracted
  variables into the requests that follow (request chaining). `RunConcurrent` fires a batch of
  independent requests through a bounded worker pool instead.
- **`internal/storage`** — SQLite persistence (via `modernc.org/sqlite`, a pure-Go driver — no
  CGO, so the binary cross-compiles trivially and the Docker image needs no C toolchain).
  Collections, environments, run history, and ad-hoc request history each get a table.
- **`internal/api`** — a REST API plus a WebSocket endpoint that streams a run's `StepResult`s as
  they complete, instead of making the client wait for the whole collection to finish.
- **`cli/`** — `serve`, `run`, `test`, and `export`, all built on the same runner and collection
  codec the API uses. `run` and `test` differ only in exit code: `test` fails the process (and
  therefore the CI job) if any assertion failed.

## Request chaining

A chain is just sequential execution with shared state: `Run` keeps a `map[string]string` of
variables extracted so far, seeded empty, and passes it to every step. A step's `Extract` rules
run after its assertions and write into that map; the next step's template rendering reads from
it (with extracted variables taking precedence over the environment, so a freshly-extracted token
always wins over a stale one sitting in the environment). Nothing about this requires the steps to
be requests to the same host — chaining across services works the same way.

## Concurrency

Three different concurrency concerns show up in three different places, deliberately not
unified into one abstraction:

1. **Connection pooling** (`internal/httpclient`): one `http.Transport` per `Client`, reused
   across every request so TCP/TLS setup isn't repeated.
2. **Request cancellation and timeouts**: every request carries a `context.Context`; `Run`
   derives a per-request timeout from it when configured, and cancelling the parent context
   (Ctrl-C on the CLI, a closed WebSocket, a cancelled HTTP request to the API) stops the run
   after the in-flight request rather than leaking it.
3. **Concurrent execution** (`Runner.RunConcurrent`): a semaphore-bounded worker pool for firing
   independent requests in parallel — used for smoke-testing a batch of unrelated endpoints, as
   opposed to a chain where later requests depend on earlier ones and therefore can't run in
   parallel with them.

## Why SQLite, why WebSockets

SQLite because ByteForge is a local-first tool: one file, no separate database process, and
`modernc.org/sqlite`'s pure-Go implementation means the same binary that runs on a developer's
laptop also builds cleanly for the `FROM alpine` runtime stage without CGO. WebSockets because a
test run's whole point is showing progress as it happens — polling an endpoint for partial results
would mean either an under-engineered "check back in N seconds" loop or reinventing what a
persistent connection already gives you for free.
