# API reference

The API server (`byteforge serve`) exposes JSON over HTTP plus one WebSocket endpoint. All request
and response bodies are JSON; timestamps are RFC 3339; durations are whole milliseconds under a
`*Ms` key (not Go's native nanosecond encoding).

## Collections

| Method   | Path                             | Description                                   |
| -------- | --------------------------------- | ---------------------------------------------- |
| `GET`    | `/api/collections`                | List all collections.                          |
| `POST`   | `/api/collections`                | Create a collection.                           |
| `GET`    | `/api/collections/{id}`           | Get one collection.                            |
| `PUT`    | `/api/collections/{id}`           | Replace a collection's contents.               |
| `DELETE` | `/api/collections/{id}`           | Delete a collection.                           |
| `GET`    | `/api/collections/{id}/export`    | Download the collection as a `.json` file.     |
| `POST`   | `/api/collections/{id}/run`       | Run the collection synchronously; returns a `Report`. Body: `{"environmentId": "...", "stopOnFailure": false}` (both optional). |
| `GET`    | `/api/collections/{id}/runs`      | List past runs, most recent first.             |

## Environments

| Method   | Path                       | Description                                                        |
| -------- | --------------------------- | -------------------------------------------------------------------- |
| `GET`    | `/api/environments`         | List all environments, **secrets masked**.                          |
| `POST`   | `/api/environments`         | Create an environment.                                               |
| `GET`    | `/api/environments/{id}`    | Get one environment, secrets masked.                                 |
| `PUT`    | `/api/environments/{id}`    | Replace an environment's variables.                                  |
| `DELETE` | `/api/environments/{id}`    | Delete an environment.                                                |

Secret values are never returned by any read endpoint — `Redacted()` runs on every environment
before it's serialized. The only place real secret values are used is inside the server, when
rendering a request template just before it's sent.

## Requests and history

| Method | Path                | Description                                                    |
| ------ | -------------------- | ---------------------------------------------------------------- |
| `POST` | `/api/requests/send` | Send one ad-hoc request outside of any collection. Body: `{"request": {...}, "environmentId": "..."}`. Returns a `StepResult` and records a history entry. |
| `GET`  | `/api/history`       | List the most recent ad-hoc sends.                              |

## Live test runs

```
GET /api/ws/collections/{id}/run?environmentId=...&stopOnFailure=true
```

Upgrades to a WebSocket and streams one JSON message per request as the collection runs:

```json
{"type": "step", "step": { "...": "a StepResult" }}
{"type": "step", "step": { "...": "a StepResult" }}
{"type": "done", "report": { "...": "the full Report" }}
```

or, if the run itself can't proceed (e.g. the collection or environment doesn't exist):

```json
{"type": "error", "error": "..."}
```

The finished run is persisted the same way a synchronous `POST .../run` is, so it shows up in
`GET /api/collections/{id}/runs` afterward.

## Health

`GET /healthz` — `{"status": "ok", "time": "..."}`. Used by the Docker `HEALTHCHECK` and by
`docker-compose`.

## Errors

Every non-2xx response is `{"error": "..."}`. `404` means the resource doesn't exist; `400` means
the request body failed validation (an unsupported HTTP method, a missing collection name, malformed
JSON); anything else is a `500` with the detail logged server-side rather than returned, so
internal errors never leak implementation details to a client.
