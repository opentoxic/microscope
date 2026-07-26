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

## Runtime metrics

Periodically records event loop utilization and memory so the dashboard's metrics view has
something to show for Node services, the same way it does for Go. No-op outside Node (e.g. in
a browser bundle):

```ts
client.startRuntimeMetrics(15_000); // call once at startup
```

## Express

```ts
import express from "express";
import { microscopeMiddleware } from "@qobly/microscope-client/express";

const app = express();
app.use(microscopeMiddleware({ baseUrl: "http://localhost:8093/microscope" }));
```

## NestJS

```ts
import { APP_INTERCEPTOR } from "@nestjs/common";
import { MicroscopeModule, MicroscopeInterceptor } from "@qobly/microscope-client/nestjs";

@Module({
  imports: [MicroscopeModule.forRoot({ baseUrl: "http://localhost:8093/microscope" })],
  providers: [{ provide: APP_INTERCEPTOR, useClass: MicroscopeInterceptor }],
})
export class AppModule {}
```

## Testing

`test/client.test.ts` and `test/runtimeMetrics.test.ts` run directly against source:

```bash
node --test test/client.test.ts test/runtimeMetrics.test.ts
```

`test/express.test.ts` and `test/nestjs.test.ts` exercise code that imports its sibling module
by its compiled `.js` name (the correct specifier once built), so they need a build first:

```bash
npm install && npm run build
node --test test/
```
