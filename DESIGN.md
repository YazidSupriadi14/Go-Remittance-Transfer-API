# DESIGN.md

## Q1. Tables, indexes and constraints

Two tables. `customers` is given. `transfers` has one row per transfer: amounts, status, idempotency key, request hash and timestamps.

Each constraint and the problem it stops:

- `UNIQUE (idempotency_key)`: a retry, or two parallel requests, creating a second transfer.
- `FOREIGN KEY customer_id`: a transfer for a customer that does not exist.
- `CHECK status IN (...)`: a status outside the state machine.
- `CHECK` on currencies (`KRW`, `PHP`), `send_amount >= 10000` and `fee >= 3000`: invalid data saved because of a bug in the code.
- `CHECK total_debit = send_amount + fee`: an inconsistent total.
- `CHECK length(recipient_name) BETWEEN 1 AND 100`: SQLite does not enforce `VARCHAR(100)`, so I check it myself.

Indexes: the primary key serves lookups by `transfer_id`; the UNIQUE index makes the retry lookup fast. `(customer_id, created_at)` is prepared for the Part 2 daily limit query ("sum for one customer on one day").

## Q2. Idempotency

The client sends an `Idempotency-Key` header: a UUID it creates once per transfer and reuses on every retry. No key means `400`.

The key and a SHA-256 hash of the business fields are stored in the same `transfers` row. On create:

- key exists, same hash: return the original transfer (`201`, header `Idempotent-Replayed: true`);
- key exists, different hash: `409`;
- key is new: insert.

It is enforced in two places. Application code handles the normal path. The database `UNIQUE` constraint is the real guarantee.

Two identical requests at the same moment: both may find no row and both try to INSERT. The database accepts one. The other gets a unique violation on `idempotency_key`. My code catches only that error, reads the winner's row, compares the hash and returns the same transfer. Result: one transfer, same response for both. A test sends 20 parallel requests to check this.

## Q3. Money

**Code:** `int64` in the smallest unit. KRW in won (it has no minor unit). PHP in centavos, so `20600.00` is `2060000`. No floats anywhere. The FX rate is the fraction 412 / 10000.

**API:** every amount is a JSON string (`"5000"`, `"20600.00"`), so no client parses it as a float. `receiveAmount` always has exactly 2 decimals.

**Database:** `INTEGER` columns (`send_amount`, `fee`, `total_debit`, `receive_amount_minor`). `fx_rate` is stored as text, for audit.

**Rounding** happens once, in `NewQuote`, when the transfer is created:

- fee = round_half_up(send / 100), minimum 3000;
- receive = round_half_up(send × 412 / 100) centavos.

Integer HALF_UP is `(a + b/2) / b`.

**Why there:** the price is fixed when the customer creates the transfer. It is stored and never recalculated. Reads only format stored integers, so a transfer always shows the same numbers.

## Q4 and Q5

Not answered: Part 2 is not implemented (see README.md).
