# Remote HTTP clients

Record custom events and runtime metrics against an existing microscope HTTP endpoint.

Use these when a service **cannot** embed an in-process adaptor (separate repo, browser-only worker, legacy binary).

| Language | Package | README |
|----------|---------|--------|
| TypeScript / Node | `@opentoxic/microscope-client` | [typescript/README.md](typescript/README.md) |
| Ruby | `microscope_client` gem | [ruby/README.md](ruby/README.md) |
| Elixir | `microscope_client` | [elixir/README.md](elixir/README.md) |

**Tutorial:** [Custom events](../core/docs/tutorials/custom-events.md)

```ts
import { MicroscopeClient } from "@opentoxic/microscope-client";

const client = new MicroscopeClient({ baseUrl: "http://localhost:8080/microscope" });
await client.record("order_placed", { order_id: "ord_123" });
```

For in-process integration, use an [adaptor](../adaptor/) instead.
