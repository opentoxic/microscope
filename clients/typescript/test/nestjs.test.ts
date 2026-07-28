import test from "node:test";
import assert from "node:assert/strict";
import { of } from "rxjs";
import { firstValueFrom } from "rxjs";

import { MICROSCOPE_CLIENT, MicroscopeInterceptor } from "../src/nestjs.ts";
import type { MicroscopeClient } from "../src/index.ts";

function fakeExecutionContext(method: string, url: string, statusCode: number): any {
  const request = { method, url };
  const response = { statusCode };
  return {
    switchToHttp: () => ({
      getRequest: () => request,
      getResponse: () => response,
    }),
  };
}

test("MicroscopeInterceptor records the request after the handler completes", async () => {
  const recorded: Array<[string, Record<string, unknown>]> = [];
  const fakeClient = {
    record: async (name: string, content: Record<string, unknown>) => {
      recorded.push([name, content]);
      return "entry-1";
    },
  } as unknown as MicroscopeClient;

  const interceptor = new MicroscopeInterceptor(fakeClient);
  const context = fakeExecutionContext("GET", "/health", 200);
  const handler = { handle: () => of("handler result") };

  const result = await firstValueFrom(interceptor.intercept(context, handler));

  assert.equal(result, "handler result");
  assert.equal(recorded.length, 1);
  const [name, content] = recorded[0];
  assert.equal(name, "http_request");
  assert.equal(content.method, "GET");
  assert.equal(content.path, "/health");
  assert.equal(content.status, 200);
  assert.equal(typeof content.duration_ms, "number");
});

test("MICROSCOPE_CLIENT token is a stable string", () => {
  assert.equal(MICROSCOPE_CLIENT, "MICROSCOPE_CLIENT");
});
