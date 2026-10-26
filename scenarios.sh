#!/usr/bin/env bash
# Runs scenarios S1-S6 against a freshly started app and prints Markdown for RESULT.md.
#
# Usage:
#   1. Terminal A:  go run ./src
#   2. Terminal B:  bash scenarios.sh > RESULT_RAW.md
#
# Requires bash + curl (Git Bash on Windows works).

set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"

# Fixed keys so every run is reproducible. A real client generates a new UUID v4
# per transfer and reuses it for every retry of that transfer.
KEY_S1="6f1d3c52-1a2b-4c3d-8e4f-000000000001"
KEY_S4="6f1d3c52-1a2b-4c3d-8e4f-000000000004"
KEY_S6="6f1d3c52-1a2b-4c3d-8e4f-000000000006"

LAST_BODY=""

# Fail early with a clear message if the app cannot be reached.
if ! curl -s -m 5 -o /dev/null "$BASE_URL/api/transfers/ping"; then
  echo "ERROR: cannot reach the app at $BASE_URL" >&2
  echo "Start it first with 'go run ./src'. On Windows, run this script in Git Bash, not WSL." >&2
  exit 1
fi

# call TITLE(optional sub-heading) METHOD PATH [IDEMPOTENCY_KEY] [JSON_BODY]
call() {
  local title="$1" method="$2" path="$3" key="${4:-}" body="${5:-}"
  local headers_file body_file status
  headers_file="$(mktemp)"
  body_file="$(mktemp)"

  local args=(-s -X "$method" "$BASE_URL$path" -D "$headers_file" -o "$body_file" -w "%{http_code}")
  [[ -n "$key" ]] && args+=(-H "Idempotency-Key: $key")
  [[ -n "$body" ]] && args+=(-H "Content-Type: application/json" --data "$body")
  status="$(curl "${args[@]}")"
  LAST_BODY="$(cat "$body_file")"

  if [[ -n "$title" ]]; then
    echo "#### $title"
    echo
  fi
  echo "Request:"
  echo '```http'
  echo "$method $path"
  [[ -n "$body" ]] && echo "Content-Type: application/json"
  [[ -n "$key" ]] && echo "Idempotency-Key: $key"
  if [[ -n "$body" ]]; then
    echo
    echo "$body"
  fi
  echo '```'
  echo
  echo "Status: $status"
  echo
  if grep -qi '^Idempotent-Replayed:' "$headers_file"; then
    echo "Response header: \`$(grep -i '^Idempotent-Replayed:' "$headers_file" | tr -d '\r')\`"
    echo
  fi
  echo "Response:"
  echo '```json'
  echo "$LAST_BODY"
  echo '```'
  echo

  rm -f "$headers_file" "$body_file"
}

transfer_id_of_last_response() {
  echo "$LAST_BODY" | sed -n 's/.*"transferId":"\([^"]*\)".*/\1/p'
}

S1_BODY='{"customerId":"C001","sendCurrency":"KRW","sendAmount":"500000","receiveCurrency":"PHP","recipientName":"Juan Dela Cruz"}'
S3_BODY='{"customerId":"C001","sendCurrency":"KRW","sendAmount":"600000","receiveCurrency":"PHP","recipientName":"Juan Dela Cruz"}'
S4_BODY='{"customerId":"C002","sendCurrency":"KRW","sendAmount":"100000","receiveCurrency":"PHP","recipientName":"Maria Santos"}'
S6_BODY='{"customerId":"C001","sendCurrency":"KRW","sendAmount":"5000","receiveCurrency":"PHP","recipientName":"Juan Dela Cruz"}'

echo "### S1 - Create a transfer for C001, amount 500000"
echo
call "" POST /api/transfers "$KEY_S1" "$S1_BODY"

echo "### S2 - Retry S1 with the exact same request (same Idempotency-Key)"
echo
call "" POST /api/transfers "$KEY_S1" "$S1_BODY"

echo "### S3 - Same Idempotency-Key as S1, but amount 600000"
echo
call "" POST /api/transfers "$KEY_S1" "$S3_BODY"

echo "### S4 - Create another transfer, then cancel it"
echo
call "S4a - create" POST /api/transfers "$KEY_S4" "$S4_BODY"
S4_ID="$(transfer_id_of_last_response)"
call "S4b - cancel" POST "/api/transfers/$S4_ID/cancel"

echo "### S5 - Cancel the same transfer again"
echo
call "" POST "/api/transfers/$S4_ID/cancel"

echo "### S6 - Create with sendAmount = 5000"
echo
call "" POST /api/transfers "$KEY_S6" "$S6_BODY"
