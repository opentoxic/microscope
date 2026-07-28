import test from "node:test";
import assert from "node:assert/strict";
import { EventEmitter } from "node:events";

import { microscopeMiddleware } from "../src/express.ts";

function fakeReqRes(method: string, path: string, statusCode: number) {
  const req = { method, path } as any;
  const res = Object.assign(new EventEmitter(), { statusCode }) as any;
  return { req, res };
}

test("microscopeMiddleware records the request once the response finishes", async () => {
  const calls: { url: string; init?: RequestInit }[] = [];
  const originalFetch = globalThis.fetch;
  globalThis.fetch = (async (url: string, init?: RequestInit) => {
    calls.push({ url, init });
    return { ok: true, status: 200, json: async () => ({ id: "entry-1" }) } as Response;
  }) as typeof fetch;

  try {
    const middleware = microscopeMiddleware({ baseUrl: "http://localhost:8093/microscope" });
    const { req, res } = fakeReqRes("GET", "/health", 200);

    let nextCalled = false;
    middleware(req, res, () => {
      nextCalled = true;
    });
    assert.equal(nextCalled, true);

    res.emit("finish");
    await new Promise((resolve) => setTimeout(resolve, 20));

    assert.equal(calls.length, 1);
    const body = JSON.parse(calls[0].init?.body as string);
    assert.equal(body.name, "http_request");
    assert.equal(body.content.method, "GET");
    assert.equal(body.content.path, "/health");
    assert.equal(body.content.status, 200);
    assert.equal(typeof body.content.duration_ms, "number");
  } finally {
    globalThis.fetch = originalFetch;
  }
});
