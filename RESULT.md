# RESULT.md

Scenarios S1-S6, run in order on a freshly started app (`go run ./src`).
Everything below is the real captured output of [`scenarios.sh`](scenarios.sh) (`bash scenarios.sh`).
To reproduce: start the app, then run `bash scenarios.sh`. Transfer IDs and timestamps will differ.

| # | Expected | Actual |
| --- | --- | --- |
| S1 | 201, fee 5000, totalDebit 505000, receiveAmount 20600.00 | 201, as expected |
| S2 | Same transfer as S1, no second transfer | 201, same `transferId`, header `Idempotent-Replayed: true` |
| S3 | 409 | 409 `CONFLICT` |
| S4 | Create 201, cancel succeeds | 201, then 200 with `status` CANCELLED |
| S5 | 409 | 409 `CONFLICT` |
| S6 | 400 | 400 `VALIDATION_ERROR` |

### S1 - Create a transfer for C001, amount 500000

Request:
```http
POST /api/transfers
Content-Type: application/json
Idempotency-Key: 6f1d3c52-1a2b-4c3d-8e4f-000000000001

{"customerId":"C001","sendCurrency":"KRW","sendAmount":"500000","receiveCurrency":"PHP","recipientName":"Juan Dela Cruz"}
```

Status: 201

Response:
```json
{"transferId":"09edf414-d036-4675-8a09-64afcbf247aa","status":"REQUESTED","sendAmount":"500000","fee":"5000","totalDebit":"505000","receiveAmount":"20600.00","createdAt":"2026-10-26T03:55:35.836Z","updatedAt":"2026-10-26T03:55:35.836Z","customerId":"C001","sendCurrency":"KRW","receiveCurrency":"PHP","fxRate":"0.0412","recipientName":"Juan Dela Cruz"}
```

### S2 - Retry S1 with the exact same request (same Idempotency-Key)

Request:
```http
POST /api/transfers
Content-Type: application/json
Idempotency-Key: 6f1d3c52-1a2b-4c3d-8e4f-000000000001

{"customerId":"C001","sendCurrency":"KRW","sendAmount":"500000","receiveCurrency":"PHP","recipientName":"Juan Dela Cruz"}
```

Status: 201

Response header: `Idempotent-Replayed: true`

Response:
```json
{"transferId":"09edf414-d036-4675-8a09-64afcbf247aa","status":"REQUESTED","sendAmount":"500000","fee":"5000","totalDebit":"505000","receiveAmount":"20600.00","createdAt":"2026-10-26T03:55:35.836Z","updatedAt":"2026-10-26T03:55:35.836Z","customerId":"C001","sendCurrency":"KRW","receiveCurrency":"PHP","fxRate":"0.0412","recipientName":"Juan Dela Cruz"}
```

### S3 - Same Idempotency-Key as S1, but amount 600000

Request:
```http
POST /api/transfers
Content-Type: application/json
Idempotency-Key: 6f1d3c52-1a2b-4c3d-8e4f-000000000001

{"customerId":"C001","sendCurrency":"KRW","sendAmount":"600000","receiveCurrency":"PHP","recipientName":"Juan Dela Cruz"}
```

Status: 409

Response:
```json
{"error":"CONFLICT","message":"Idempotency-Key \"6f1d3c52-1a2b-4c3d-8e4f-000000000001\" was already used with different request data"}
```

### S4 - Create another transfer, then cancel it

#### S4a - create

Request:
```http
POST /api/transfers
Content-Type: application/json
Idempotency-Key: 6f1d3c52-1a2b-4c3d-8e4f-000000000004

{"customerId":"C002","sendCurrency":"KRW","sendAmount":"100000","receiveCurrency":"PHP","recipientName":"Maria Santos"}
```

Status: 201

Response:
```json
{"transferId":"239cad10-bbb6-434d-9780-3dccf605cf9e","status":"REQUESTED","sendAmount":"100000","fee":"3000","totalDebit":"103000","receiveAmount":"4120.00","createdAt":"2026-10-26T03:55:37.525Z","updatedAt":"2026-10-26T03:55:37.525Z","customerId":"C002","sendCurrency":"KRW","receiveCurrency":"PHP","fxRate":"0.0412","recipientName":"Maria Santos"}
```

#### S4b - cancel

Request:
```http
POST /api/transfers/239cad10-bbb6-434d-9780-3dccf605cf9e/cancel
```

Status: 200

Response:
```json
{"transferId":"239cad10-bbb6-434d-9780-3dccf605cf9e","status":"CANCELLED","sendAmount":"100000","fee":"3000","totalDebit":"103000","receiveAmount":"4120.00","createdAt":"2026-10-26T03:55:37.525Z","updatedAt":"2026-10-26T03:55:38.028Z","customerId":"C002","sendCurrency":"KRW","receiveCurrency":"PHP","fxRate":"0.0412","recipientName":"Maria Santos"}
```

### S5 - Cancel the same transfer again

Request:
```http
POST /api/transfers/239cad10-bbb6-434d-9780-3dccf605cf9e/cancel
```

Status: 409

Response:
```json
{"error":"CONFLICT","message":"transfer is CANCELLED and cannot move to CANCELLED"}
```

### S6 - Create with sendAmount = 5000

Request:
```http
POST /api/transfers
Content-Type: application/json
Idempotency-Key: 6f1d3c52-1a2b-4c3d-8e4f-000000000006

{"customerId":"C001","sendCurrency":"KRW","sendAmount":"5000","receiveCurrency":"PHP","recipientName":"Juan Dela Cruz"}
```

Status: 400

Response:
```json
{"error":"VALIDATION_ERROR","message":"sendAmount must be at least 10000"}
```

