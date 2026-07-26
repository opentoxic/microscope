# @qobly/microscope-client

Thin HTTP client for the [microscope](https://github.com/qobly/microscope) observability API.

## Install

```bash
npm install @qobly/microscope-client
```

## Usage

```ts
import { MicroscopeClient } from "@qobly/microscope-client";

const client = new MicroscopeClient({ baseUrl: "http://localhost:8093/microscope" });

await client.record("payment_charged", { amount: 4200 });
const entries = await client.listEntries({ type: "custom", limit: 20 });
```
