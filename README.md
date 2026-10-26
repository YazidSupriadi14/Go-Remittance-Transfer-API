# Finshot Remittance Transfer API

this is a implementation Backend for sending money from Korea (KRW) to the Philippines (PHP).
Part 1 is implemented. Part 2 is not (see [What I did not implement](#what-i-did-not-implement)).

## How to run

```bash
go run ./src
```

Run it from the repository root. The API listens on `http://localhost:8080`.

- Requires **Go 1.26 or newer**. The first run downloads two Go modules, so it needs internet once.
- Nothing else to install. The database is SQLite in memory (pure Go driver), created from `schema.sql` on every start, so each start is a clean state.

Optional flags (not needed for the scenarios):

| Flag | Env var | Default | Meaning |
| --- | --- | --- | --- |
| `-port` | `PORT` | `8080` | HTTP port |
| `-db` | `DB_PATH` | empty = in memory | Use a SQLite file instead, e.g. `-db finshot.db`, to look at the data with a viewer. The file is recreated on every start. |
| `-schema` | `SCHEMA_PATH` | `schema.sql` | Path to the schema file |

## Stack

- Go 1.26, standard library `net/http` (no web framework)
- SQLite via `modernc.org/sqlite` (pure Go, no CGO)
- `github.com/google/uuid` for transfer IDs

## How a client retries a create request safely

Every `POST /api/transfers` must have an **`Idempotency-Key`** header.

- **What to send:** a UUID v4 (any string of 1-64 characters is accepted).
- **How to generate it:** the client creates a new key **once per new transfer**, before the first attempt, and keeps it.
- **Retry** (timeout, dropped connection, 5xx, user tapped twice): send the exact same request **with the same key**. The server returns the original transfer and does not create a new one.
- **New transfer**, or the user changed any field: create a **new key**.

| Request | Response |
| --- | --- |
| New key | `201`, new transfer |
| Same key, same body | `201`, the original transfer, plus header `Idempotent-Replayed: true` |
| Same key, different body | `409 CONFLICT` |
| No key | `400 VALIDATION_ERROR` |

Example (run it twice; the second call returns the same `transferId`):

```bash
curl -i -X POST http://localhost:8080/api/transfers \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: 6f1d3c52-1a2b-4c3d-8e4f-000000000001" \
  -d '{"customerId":"C001","sendCurrency":"KRW","sendAmount":"500000","receiveCurrency":"PHP","recipientName":"Juan Dela Cruz"}'
```

Design details: [DESIGN.md](DESIGN.md) Q2.

## API

| Method | Path | Success | Errors |
| --- | --- | --- | --- |
| `POST` | `/api/transfers` | `201` | `400`, `409` |
| `GET` | `/api/transfers/{transferId}` | `200` | `404` |
| `POST` | `/api/transfers/{transferId}/cancel` | `200` | `404`, `409` |

Every non-2xx response is `{"error": "...", "message": "..."}`.

## Scenarios and tests

- **S1-S6:** start the app, then in a second terminal run `bash scenarios.sh` (bash + curl; on Windows use Git Bash, not WSL, because WSL cannot reach the app on `localhost`). The captured output is in [RESULT.md](RESULT.md).
- **Tests:** `go test ./...`. Output is in [test-output.txt](test-output.txt). One test checks the fee and rounding rules; one sends 20 identical create requests in parallel and checks that only one transfer exists.
- **Postman:** import `postman/finshot.postman_collection.json` (S1-S6 plus extra checks, with assertions).

## Project structure

```
src/main.go         wires everything together and starts the server
src/httpapi/        HTTP layer: routes, JSON in/out, error -> status code
src/transfer/       business rules: validation, money, states, idempotency (no HTTP, no SQL)
src/storage/        SQLite implementation of the interfaces in transfer/ports.go
schema.sql          tables, constraints, indexes, seed data
```

## AI tools used

- I used Claude Code (Anthropic) to analyse the brief, design the solution, and write most of the code, tests, scripts and the first drafts of README.md and DESIGN.md.
- I ran the app myself with Postman and checked the data in DBeaver; problems found there (a Postman retry bug, SQLite locking while DBeaver was open) were fixed together with the AI.
- I reviewed the code and can explain and change every part of it.

## What I did not implement

- **Part 2:** partner client with retries, the `SENDING` flow, daily limit, partner callback, scenarios S7-S9, DESIGN.md Q4-Q5.
- Things the brief says not to build: authentication, list or search endpoints, Docker, CI, Swagger, admin API, metrics, health checks.
- Idempotency keys never expire. A key stays bound to its transfer for as long as the data exists.

## Assumptions

| # | Unclear point | What I did |
| --- | --- | --- |
| 1 | Status code for a retried create | `201` with the original body, plus header `Idempotent-Replayed: true`, so the client gets the same answer as the first call |
| 2 | Missing `Idempotency-Key` | `400 VALIDATION_ERROR`, because without it a retry cannot be detected |
| 3 | Scope of the key | Global: one key = one transfer. The same key from another customer is "different data", so `409` |
| 4 | Retry after the transfer was cancelled | Returns the original transfer with its current status (`CANCELLED`); no new transfer |
| 5 | Status code for a successful cancel | `200` with the updated transfer |
| 6 | `sendAmount` sent as a JSON number (`500000`) | Accepted and validated with the same rules as the string form |
| 7 | Upper limit for `sendAmount` | Max 1,000,000,000,000 KRW, so integer math can never overflow |
| 8 | FX base | `receiveAmount` is calculated from the full `sendAmount`, not from `sendAmount - fee` (matches the example: 500000 -> 20600.00) |
| 9 | `recipientName` spaces | Leading and trailing spaces are trimmed. Length is counted in characters, not bytes |
| 10 | Format of `createdAt` | ISO-8601 in UTC with milliseconds, e.g. `2026-10-26T02:56:17.637Z` |
| 11 | Extra response fields | Allowed by the brief. Added `updatedAt`, `customerId`, `sendCurrency`, `receiveCurrency`, `fxRate`, `recipientName` |
| 12 | Unknown route or wrong HTTP method | `404 NOT_FOUND` in the required error shape (not `405`) |
| 13 | Unexpected server error | `500` with `"error": "INTERNAL_ERROR"`, same shape as other errors |
| 14 | Source location | Code lives in `/src` as the brief asks, so the run command is `go run ./src` |
