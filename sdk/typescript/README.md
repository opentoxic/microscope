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
