-- Finshot Remittance Transfer API - database schema (SQLite).
-- The application runs this file once at startup on a fresh in-memory database.

PRAGMA foreign_keys = ON;

-- Given by the assignment. Every submission starts from the same customers.
CREATE TABLE customers (
  customer_id VARCHAR(10)  PRIMARY KEY,
  name        VARCHAR(100) NOT NULL
);

INSERT INTO customers (customer_id, name) VALUES
  ('C001', 'Kim'), ('C002', 'Lee'), ('C003', 'Park');

-- One row per transfer. Money is stored as integers in the smallest unit:
--   KRW amounts in won (KRW has no minor unit),
--   PHP amount in centavos (1 PHP = 100 centavos), so 20600.00 PHP is stored as 2060000.
CREATE TABLE transfers (
  transfer_id          VARCHAR(36)  NOT NULL PRIMARY KEY,
  customer_id          VARCHAR(10)  NOT NULL REFERENCES customers (customer_id),

  -- Idempotency: one key can only ever create one transfer (see the UNIQUE constraint below).
  idempotency_key      VARCHAR(64)  NOT NULL CHECK (length(idempotency_key) BETWEEN 1 AND 64),
  -- SHA-256 of the business fields, used to detect a key reused with different data.
  request_hash         CHAR(64)     NOT NULL CHECK (length(request_hash) = 64),

  send_currency        CHAR(3)      NOT NULL CHECK (send_currency = 'KRW'),
  send_amount          INTEGER      NOT NULL CHECK (send_amount >= 10000),
  fee                  INTEGER      NOT NULL CHECK (fee >= 3000),
  total_debit          INTEGER      NOT NULL,

  receive_currency     CHAR(3)      NOT NULL CHECK (receive_currency = 'PHP'),
  receive_amount_minor INTEGER      NOT NULL CHECK (receive_amount_minor > 0),
  -- The FX rate used for this transfer, kept as text so it is never a float.
  fx_rate              VARCHAR(16)  NOT NULL,

  recipient_name       VARCHAR(100) NOT NULL
                       CHECK (length(trim(recipient_name)) BETWEEN 1 AND 100),

  status               VARCHAR(16)  NOT NULL
                       CHECK (status IN ('REQUESTED', 'SENDING', 'COMPLETED', 'FAILED', 'CANCELLED')),

  -- ISO-8601 UTC with milliseconds, e.g. 2026-10-26T03:15:00.123Z
  created_at           TEXT         NOT NULL,
  updated_at           TEXT         NOT NULL,

  CONSTRAINT uq_transfers_idempotency_key UNIQUE (idempotency_key),
  CONSTRAINT ck_transfers_total_debit     CHECK (total_debit = send_amount + fee)
);

-- Prepared for the Part 2 daily limit: "sum of sendAmount for one customer on one day".
CREATE INDEX idx_transfers_customer_created ON transfers (customer_id, created_at);
