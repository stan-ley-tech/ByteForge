# ByteForge

[![CI](https://github.com/stan-ley-tech/ByteForge/actions/workflows/ci.yml/badge.svg)](https://github.com/stan-ley-tech/ByteForge/actions/workflows/ci.yml)
[![API Tests](https://github.com/stan-ley-tech/ByteForge/actions/workflows/api-tests.yml/badge.svg)](https://github.com/stan-ley-tech/ByteForge/actions/workflows/api-tests.yml)

ByteForge is an API testing and debugging tool for developers, built around the idea that a test
suite for HTTP APIs should be able to run the same way in a GUI while you're building it and in
CI once you trust it. It's a Go engine (HTTP client, request chaining, an assertion language, a
JSONPath evaluator) with two front ends on top: a React web UI for building and exploring
requests, and a CLI for running them headlessly.

```
GET https://api.example.com/users/{{user_id}}
```
```
status == 200
response.body.id exists
response.body.email == "test@example.com"
response.time < 500ms
```
```
✓ Status code
✓ User ID exists
✓ Email matches
✓ Response under 500ms

4/4 PASSED
```

## Features

- **Request builder** — GET/POST/PUT/PATCH/DELETE/HEAD/OPTIONS, headers, query params,
  JSON/XML/form/raw bodies, bearer/basic/API-key auth.
- **Environments** — named variable sets (`Development`, `Staging`, `Production`, ...) resolved
  into requests via `{{VARIABLE}}` templating. Secrets are marked per-variable and are never
  written to exported collections, logs, or WebSocket output in plaintext.
- **Collections & request history** — save requests into named, ordered collections; every ad-hoc
  send is recorded to a local history.
- **Response viewer** — status, timing, headers, and JSON-formatted body.
- **Assertions** — a small, readable expression language (`status == 200`,
  `response.body.id exists`, `response.time < 500ms`) evaluated with a real JSONPath subset, not
  string matching. See [docs/assertions.md](docs/assertions.md).
- **Request chaining** — extract a value from one response (`access_token`) and reference it in
  the next request (`Authorization: Bearer {{access_token}}`), so a login → profile → assert flow
  is a normal, orderly collection, not a special mode.
- **Automated test suites** — run a whole collection and get a structured report: pass/fail per
  request, per-assertion messages, timings, status codes.
- **Live test output** — a run streams progress over a WebSocket as each request completes,
  instead of blocking on the whole collection.
- **CLI + CI gate** — `byteforge run` / `test` / `export` operate on a collection JSON file
  directly, no server required, so the same tests that run in the UI run in a GitHub Actions job
  (see [`.github/workflows/api-tests.yml`](.github/workflows/api-tests.yml)) and fail the build on
  a failed assertion.

## Quick start

### Docker

```bash
docker compose up --build
```

Then open `http://localhost:8080`. Data persists in the `byteforge-data` volume (a single SQLite
file).

### Local development

Requires Go 1.25+ and Node 20+.

```bash
# backend + API, on :8080
go run ./cmd/byteforge serve

# frontend dev server, on :5173, proxying /api to :8080
cd web && npm install && npm run dev
```

### CLI

```bash
go build -o byteforge ./cmd/byteforge

./byteforge run    tests/fixtures/sample-collection.json --env tests/fixtures/sample-environment.json
./byteforge test   tests/fixtures/sample-collection.json --env tests/fixtures/sample-environment.json
./byteforge export tests/fixtures/sample-collection.json
```

- `run` prints the report and always exits `0` — for exploring results locally.
- `test` prints the same report and exits `1` if any assertion failed — the CI gate.
- `export` validates a collection and re-serializes it as normalized, pretty-printed JSON.
- `--var KEY=VALUE` (repeatable) sets or overrides an environment variable from the command line,
  which is how a CI job injects real secrets without ever writing them to a file:

  ```bash
  ./byteforge test collection.json --var API_KEY="$PROD_API_KEY"
  ```

## Project structure

```
byteforge/
├── cmd/byteforge/     entrypoint — dispatches to cli/
├── internal/
│   ├── httpclient/    pooled, retrying, streaming HTTP client
│   ├── collections/   request/collection model + JSON codec
│   ├── environments/  variables + {{template}} rendering, secret redaction
│   ├── assertions/    assertion language: parser + evaluator
│   ├── jsonpath/      the JSONPath subset assertions/extraction use
│   ├── runner/        execution engine: chaining, concurrency, reports
│   ├── storage/       SQLite persistence
│   └── api/           REST API + WebSocket live run output
├── cli/               run / test / export / serve commands
├── web/               React + TypeScript UI (Vite), with its own unit + E2E tests
├── tests/fixtures/    sample collection + environment used in docs, CI, and manual testing
├── docs/              architecture, assertion language, and API reference
├── Dockerfile, docker-compose.yml
└── .github/workflows/ CI, plus a CLI-gated API test run on every push
```

## Testing

```bash
# Go: unit + HTTP integration tests (httptest-backed, no network dependency)
go test ./...

# Frontend: component tests (Vitest + Testing Library)
cd web && npm test

# End-to-end: real browser against the real Go backend
cd web && npx playwright install chromium && npm run e2e
```

Coverage by layer:

- **`internal/httpclient`** — retries, backoff, timeouts, cancellation, response truncation,
  streaming, all against a real `httptest.Server`.
- **`internal/assertions`** / **`internal/jsonpath`** — the parser, the evaluator, and the path
  language, including edge cases (missing fields, out-of-range indices, top-level arrays).
- **`internal/runner`** — request chaining end to end (a real login → extract → authenticated
  request flow), `StopOnFailure`, context cancellation mid-chain, bounded concurrency.
- **`internal/api`** — every REST endpoint plus the secret-redaction contract, over `httptest`.
- **`cli/`** — `run`/`test` exit codes, `--var` overrides, `export`'s normalization.
- **`web/`** — component behavior (including a regression test pinning the JSON contract between
  the Go assertion engine and the UI's pass/fail rendering) and a full browser flow: build a
  request, send it, add assertions, save, run the collection, read the report.

## Engineering notes

A few decisions worth calling out, covered in more depth in [docs/architecture.md](docs/architecture.md):

- **No CGO.** The SQLite driver (`modernc.org/sqlite`) is pure Go, so the binary cross-compiles
  without a C toolchain and the Docker image needs nothing beyond `alpine`.
- **Assertions have no `and`/`or`.** Each line is one independent check. A test report that can
  point at exactly which condition failed is worth more than a terser but ambiguous expression
  language.
- **Extracted chain variables beat environment variables of the same name.** A freshly-extracted
  `access_token` always wins over a stale one already sitting in the environment.
- **Secrets are redacted at the boundary, not scattered through the codebase.**
  `Environment.Redacted()` and `ExportSafe()` are the only two places a secret value gets masked
  or dropped; every API response and export goes through one of them.

## License

[MIT](LICENSE)
