# PureGo API Testing Guide (Screenshot Evidence Edition)

This guide is designed for proving advanced features with screenshots.

Base URL:

```text
http://localhost:8080
```

## 1. What to Capture (Checklist)

Take screenshots for these items in this order:

1. Login response with access token
2. Idempotency miss/hit on same request key
3. FIFO transfer request + DB before/after proof
4. Worker pool processing log line
5. Webhook receiver hit (request body + signature header)
6. Admin dashboard cache MISS then HIT

## 2. Prerequisites

1. Start services (Postgres, Redis).
2. Run API server:

```bash
go run .
```

3. Keep one terminal open for server logs (for worker/webhook screenshots).
4. Use Postman or curl.
5. Optional: open webhook capture URL at https://webhook.site and copy your unique URL.

## 3. Auth Setup (Needed for Protected Routes)

### 3.1 Signup

Method:

```text
POST /signup
```

URL:

```text
http://localhost:8080/signup
```

Headers:

```text
Content-Type: application/json
```

Body:

```json
{
  "email": "admin1@example.com",
  "password": "SecurePassword123!",
  "role": "admin"
}
```

### 3.2 Verify email (if your flow requires it)

Method:

```text
GET /verify-email?token=...
```

### 3.3 Login

Method:

```text
POST /login
```

URL:

```text
http://localhost:8080/login
```

Headers:

```text
Content-Type: application/json
```

Body:

```json
{
  "email": "admin1@example.com",
  "password": "SecurePassword123!"
}
```

Expected:

```json
{
  "access_token": "...",
  "refresh_token": "..."
}
```

Screenshot A:

- Capture full login response showing access token.

Use this header for protected APIs:

```text
Authorization: Bearer <access_token>
```

## 4. Feature 1: Idempotency Key (Proof)

### 4.1 First request (MISS)

Method:

```text
POST /products
```

URL:

```text
http://localhost:8080/products
```

Headers:

```text
Content-Type: application/json
Authorization: Bearer <access_token>
X-Idempotency-Key: create-prod-001
```

Body:

```json
{
  "name": "Idempotency Demo Product",
  "price": 199.99,
  "quantity": 80
}
```

Expected:

- Status: 201
- Response header: X-Idempotency-Hit: false
- Response header: X-Idempotency-Key: create-prod-001

### 4.2 Second request with same key (HIT)

Send exactly the same request again.

Expected:

- Status: same as first call
- Same response body
- Response header: X-Idempotency-Hit: true

Screenshot B:

- Put first and second responses side by side (headers visible), highlight X-Idempotency-Hit false then true.

DB proof query:

```sql
SELECT id_key, user_id, response_code
FROM idempotency_keys
WHERE id_key = 'create-prod-001';
```

Screenshot C:

- Query result showing stored idempotency key record.

## 5. Feature 2: FIFO + Stock Transfer (Proof)

The transfer endpoint uses FIFO deduction from stock_batches ordered by expiry_date ascending.

### 5.1 Seed transfer test data

Run in PostgreSQL (adjust IDs if they already exist):

```sql
-- Ensure product exists
INSERT INTO products (id, name, price, quantity)
VALUES (1001, 'FIFO Demo Product', 120, 100)
ON CONFLICT (id) DO NOTHING;

-- Ensure warehouses exist
INSERT INTO warehouses (id, name)
VALUES (1, 'Warehouse A'), (2, 'Warehouse B')
ON CONFLICT (id) DO NOTHING;

-- Clean existing demo batches
DELETE FROM stock_batches WHERE product_id = 1001;

-- Batch 1 (oldest)
INSERT INTO stock_batches
  (product_id, warehouse_id, batch_number, initial_quantity, current_quantity, expiry_date)
VALUES
  (1001, 1, 'BATCH-OLD', 30, 30, NOW() + interval '10 days');

-- Batch 2 (newer)
INSERT INTO stock_batches
  (product_id, warehouse_id, batch_number, initial_quantity, current_quantity, expiry_date)
VALUES
  (1001, 1, 'BATCH-NEW', 40, 40, NOW() + interval '40 days');
```

Before snapshot query:

```sql
SELECT id, batch_number, warehouse_id, current_quantity, expiry_date
FROM stock_batches
WHERE product_id = 1001
ORDER BY expiry_date ASC;
```

Screenshot D:

- Show both source batches and quantities before transfer.

### 5.2 Transfer stock

Method:

```text
POST /stock/transfer
```

URL:

```text
http://localhost:8080/stock/transfer
```

Headers:

```text
Content-Type: application/json
Authorization: Bearer <access_token>
X-Idempotency-Key: transfer-001
```

Body:

```json
{
  "product_id": 1001,
  "from_warehouse_id": 1,
  "to_warehouse_id": 2,
  "quantity": 35,
  "reason": "FIFO proof transfer"
}
```

Expected:

```json
{
  "message": "Transfer successful"
}
```

### 5.3 FIFO verification query

```sql
SELECT id, batch_number, warehouse_id, current_quantity, expiry_date
FROM stock_batches
WHERE product_id = 1001
ORDER BY warehouse_id, expiry_date ASC;
```

Expected behavior:

- Oldest batch reduced first.
- New warehouse receives INTERNAL-TRANSFER batch.

Screenshot E:

- Capture before/after batch quantities proving FIFO order.

Audit proof query:

```sql
SELECT action, entity_name, entity_id, old_value, new_value, created_at
FROM audit_logs
WHERE action = 'TRANSFER'
ORDER BY created_at DESC
LIMIT 1;
```

Screenshot F:

- Transfer audit log row.

## 6. Feature 3: Worker Pool (Proof)

Worker pool starts with 5 workers and processes queued jobs.

For screenshot evidence, capture server logs while a queued job is processed.

Expected log format:

```text
Worker <id> processing job: <job_type>
```

Screenshot G:

- Terminal log showing worker processing at least one job.

## 7. Feature 4: Webhook (Proof)

Webhook sending is async through worker job type WEBHOOK_SEND and includes HMAC header X-Hub-Signature.

### 7.1 Prepare webhook receiver

Use webhook.site and copy your unique URL, example:

```text
https://webhook.site/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
```

### 7.2 Register webhook in DB

```sql
INSERT INTO webhooks (url, secret, event_type, is_active)
VALUES (
  'https://webhook.site/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx',
  'demo-secret-123',
  'stock.low',
  true
);
```

Screenshot H:

- webhooks table row with event_type stock.low and is_active true.

### 7.3 Trigger webhook event

Current code note:

- Webhook dispatch function exists and queues WEBHOOK_SEND jobs.
- If your branch already routes a stock-low flow to DispatchWebHook, trigger that flow and proceed.
- If not, trigger your implemented webhook path the same way you wired it in your branch.

Expected receiver payload format:

```json
{
  "event": "stock.low",
  "data": {
    "product_id": 1001,
    "current_qty": 9,
    "message": "Warning: Stock is running low!"
  }
}
```

Expected webhook header:

```text
X-Hub-Signature: <sha256_hmac_hex>
```

Screenshot I:

- webhook.site request body and request headers (show X-Hub-Signature).

Screenshot J:

- API server terminal showing worker processed WEBHOOK_SEND.

## 8. Feature 5: Dashboard Cache (MISS/HIT)

### 8.1 Product dashboard cache

Method:

```text
GET /admin/dashboard
```

URL:

```text
http://localhost:8080/admin/dashboard
```

Headers:

```text
Authorization: Bearer <access_token>
```

Call twice.

Expected:

- First response header: X-Cache: MISS
- Second response header: X-Cache: HIT

Screenshot K:

- Two consecutive calls showing MISS then HIT.

### 8.2 Warehouse dashboard cache

Method:

```text
GET /admin/warehouse-dashboard
```

URL:

```text
http://localhost:8080/admin/warehouse-dashboard
```

Expected response fields:

```json
{
  "total_products": 2,
  "global_stock_count": 35,
  "warehouse_balances": [
    {
      "warehouse_name": "Warehouse A",
      "total_current": 35,
      "total_initial": 70
    }
  ]
}
```

Screenshot L:

- Warehouse dashboard response with X-Cache header.

## 9. Feature 6: Product Search Cache (MISS/HIT)

Method:

```text
GET /products/search?q=fifo
```

URL:

```text
http://localhost:8080/products/search?q=fifo
```

Call twice.

Expected:

- First response header: X-Cache: MISS
- Second response header: X-Cache: HIT

Screenshot M:

- First and second search responses with X-Cache header.

## 10. Feature 7: Soft Delete + Audit Trail

### 10.1 Soft delete request

Method:

```text
DELETE /products/{id}
```

Example URL:

```text
http://localhost:8080/products/1001
```

Headers:

```text
Authorization: Bearer <access_token>
X-Idempotency-Key: delete-1001
```

Expected:

- Status 204 No Content

### 10.2 Verify product hidden from listing

Method:

```text
GET /products?limit=10&page=1
```

Expected:

- Deleted product does not appear.

### 10.3 Verify deleted_at in DB

```sql
SELECT id, name, deleted_at
FROM products
WHERE id = 1001;
```

Screenshot N:

- DELETE response + DB row showing deleted_at timestamp.

Audit query:

```sql
SELECT action, entity_name, entity_id, old_value, new_value, created_at
FROM audit_logs
WHERE action = 'DELETE'
ORDER BY created_at DESC
LIMIT 1;
```

Screenshot O:

- Latest DELETE audit record.

## 11. Feature 8: RBAC + Rate Limit

### 11.1 RBAC (admin-only endpoint)

Method:

```text
GET /admin/dashboard
```

Test:

- Login with non-admin user token.
- Call endpoint.

Expected:

- Status 403
- Message: Forbidden: You do not have the required permissions

Screenshot P:

- 403 response proving RBAC enforcement.

### 11.2 Rate limiting

Send multiple rapid requests from same client (for example, spam GET /products quickly).

Expected:

- Status 429 on excess calls
- Message: Too Many Request - Slow Down!

Screenshot Q:

- Burst calls and at least one 429 response.

## 12. Optional Postman Variables

```json
{
  "base_url": "http://localhost:8080",
  "access_token": "",
  "idempotency_key": "req-{{$timestamp}}",
  "product_id": "1001"
}
```

## 13. Final Screenshot Pack (Submission Order)

1. Login token response
2. Idempotency miss/hit headers
3. Idempotency DB row
4. FIFO before/after batches
5. Transfer audit log
6. Worker processing log
7. Webhook DB registration
8. webhook.site payload + signature
9. Dashboard MISS/HIT
10. Search cache MISS/HIT
11. Soft delete + deleted_at proof
12. RBAC 403 proof
13. Rate limit 429 proof
