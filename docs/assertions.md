# Assertion language

Every request in a collection can carry a list of assertions, one per line. Each line is checked
independently after the response comes back, so a request with three assertions produces three
pass/fail results, not one.

```
status == 200
response.body.id exists
response.body.email == "test@example.com"
response.time < 500ms
```

## Grammar

Each line is `<operand> <operator> [<value>]`. There's no `and`/`or`, no parentheses, no operator
precedence — one line, one check. That's deliberate: a test report should be able to say exactly
which condition failed, and a flat list of checks makes that trivial.

### Operands (left-hand side)

| Operand                     | Resolves to                                           |
| ---------------------------- | ------------------------------------------------------ |
| `status` / `response.status` | The HTTP status code, as a number.                     |
| `response.time`              | Round-trip duration.                                   |
| `response.header.<Name>`     | The first value of that response header, if present.   |
| `response.body`              | The whole decoded JSON body.                           |
| `response.body.<path>`       | A value inside the JSON body — see JSONPath below.      |
| `body.<path>`                | Shorthand for `response.body.<path>`.                   |

### Operators

| Operator          | Meaning                                                          |
| ------------------ | ----------------------------------------------------------------- |
| `==`, `!=`          | Equality, with numeric types coerced (`200` matches a JSON `200`). |
| `<`, `<=`, `>`, `>=` | Numeric comparison. Works for `response.time` against a duration literal like `500ms`, and for numeric body fields. |
| `contains`          | Substring match for strings; membership check for arrays (used with a wildcard path, e.g. `response.body.items[*].id contains 5`). |
| `exists` / `not exists` | Whether the operand resolved to anything at all. No right-hand value. |

### Values (right-hand side)

- Strings: double-quoted — `"test@example.com"`.
- Numbers: bare — `200`, `3.5`.
- Booleans: `true` / `false`.
- Durations: a number with a unit — `500ms`, `2s` — parsed with Go's `time.ParseDuration`.

## Body paths (JSONPath subset)

`response.body.<path>` is evaluated by ByteForge's own small JSONPath implementation
(`internal/jsonpath`), which supports:

- Dot-separated field access: `data.user.id`
- A single numeric or wildcard index per segment: `items[0].id`, `items[*].id`
- An optional leading `$.`: `$.data.id` is the same as `data.id`
- A response body that's a bare JSON array at the top level, with no dot needed before the
  bracket: `response.body[0].id`, `response.body[*].userId contains 7`

A wildcard index (`[*]`) collects one value per array element into a list, skipping elements
where the rest of the path doesn't resolve — which is what makes `contains` over a wildcard path
a useful "is this value present anywhere in the list" check.

What it does **not** support: recursive descent (`..`), filter expressions (`[?(@.id==1)]`), or
multiple indices chained in one segment. If you need more than that, extract the value with a
chained request instead (see [request chaining](../README.md#request-chaining)) and assert on it
directly.

## Reading results

Evaluating an assertion produces a `Result` with `Passed` and a human-readable `Message`:

```
✓ 200 == 200
✓ response.body.id exists
✗ got 404, expected == 200
```

A step's overall `passed` is `true` only if every assertion on it passed — one failed assertion
fails the whole request, which is what makes `byteforge test` a reliable CI gate
(`internal/runner.Report.AllPassed`).
