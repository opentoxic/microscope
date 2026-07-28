# Custom events

Mark business moments in the microscope timeline — order placed, payment captured, feature flag toggled.

## HTTP API

```http
POST /microscope/api/entries
Content-Type: application/json

{
  "name": "payment_charged",
  "content": {
    "order_id": "ord_123",
    "amount_cents": 4200,
    "currency": "USD"
  }
}
```

Response: `202 Accepted` with `{ "id": "…" }`.

Recording must be enabled and the `custom` signal type allowed in Settings.

## curl

```bash
curl -X POST http://127.0.0.1:8080/microscope/api/entries \
  -H "Content-Type: application/json" \
  -d '{"name":"user_registered","content":{"user_id":"42"}}'
```

## TypeScript client

```bash
npm install @opentoxic/microscope-client
```

```ts
import { MicroscopeClient } from "@opentoxic/microscope-client";

const client = new MicroscopeClient({
  baseUrl: "http://127.0.0.1:8080/microscope",
});

await client.record("payment_charged", {
  order_id: "ord_123",
  amount_cents: 4200,
});
```

See [`clients/typescript`](../../../clients/typescript/README.md) for Express and NestJS middleware.

## Go (in-process)

If the adaptor is embedded, record through the hub:

```go
hub := ms.Hub()
hub.Record(ctx, microscope.Entry{
    Type: microscope.TypeCustom,
    Tags: []string{"custom:payment_charged"},
    Content: map[string]any{
        "name": "payment_charged",
        "order_id": "ord_123",
    },
})
```

## PHP (in-process)

```php
$hub->record(EntryType::CUSTOM, [
    'name' => 'payment_charged',
    'order_id' => 'ord_123',
]);
```

## Naming conventions

- Use `snake_case` names — they appear as tags `custom:your_name`.
- Keep `content` JSON-serializable; large blobs respect `MICROSCOPE_MAX_BODY_BYTES`.
- Enable **Custom** under Settings → Signals if entries are rejected with 409.

## See also

- [OpenAPI schema](../../api/openapi.yaml) — `POST /api/entries`
- [Remote clients](../../../clients/)
