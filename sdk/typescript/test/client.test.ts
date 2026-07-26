import test from "node:test";
import assert from "node:assert/strict";

import { MicroscopeClient } from "../src/index.ts";

interface Call {
  url: string;
  init?: RequestInit;
}

function fakeFetch(response: unknown, ok = true, status = 200) {
  const calls: Call[] = [];
  const fetchImpl = (async (url: string, init?: RequestInit) => {
    calls.push({ url, init });
    return {
      ok,
      status,
      json: async () => response,
    } as Response;
  }) as typeof fetch;
  return { fetchImpl, calls };
}

test("record() posts name and content, returns id", async () => {
  const { fetchImpl, calls } = fakeFetch({ id: "entry-1" });
  const client = new MicroscopeClient({ baseUrl: "http://localhost:8093/microscope/", fetchImpl });

  const id = await client.record("payment_charged", { amount: 4200 });

  assert.equal(id, "entry-1");
  assert.equal(calls.length, 1);
  assert.equal(calls[0].url, "http://localhost:8093/microscope/api/entries");
  assert.equal(calls[0].init?.method, "POST");
  assert.deepEqual(JSON.parse(calls[0].init?.body as string), {
    name: "payment_charged",
    content: { amount: 4200 },
  });
});

test("record() defaults content to an empty object", async () => {
  const { fetchImpl, calls } = fakeFetch({ id: "entry-2" });
  const client = new MicroscopeClient({ baseUrl: "http://localhost:8093/microscope", fetchImpl });

  await client.record("no_content_event");

  assert.deepEqual(JSON.parse(calls[0].init?.body as string), {
    name: "no_content_event",
    content: {},
  });
});

test("record() throws on a non-ok response", async () => {
  const { fetchImpl } = fakeFetch({}, false, 500);
  const client = new MicroscopeClient({ baseUrl: "http://localhost:8093/microscope", fetchImpl });

  await assert.rejects(() => client.record("boom"), /status 500/);
});

test("listEntries() builds query params and skips undefined values", async () => {
  const { fetchImpl, calls } = fakeFetch({ entries: [], total: 0 });
  const client = new MicroscopeClient({ baseUrl: "http://localhost:8093/microscope", fetchImpl });

  await client.listEntries({ type: "custom", limit: 20 });

  assert.equal(calls[0].url, "http://localhost:8093/microscope/api/entries?type=custom&limit=20");
});

test("getEntry() builds the correct url", async () => {
  const { fetchImpl, calls } = fakeFetch({ id: "entry-1" });
  const client = new MicroscopeClient({ baseUrl: "http://localhost:8093/microscope", fetchImpl });

  const entry = await client.getEntry("entry-1");

  assert.equal(calls[0].url, "http://localhost:8093/microscope/api/entries/entry-1");
  assert.deepEqual(entry, { id: "entry-1" });
});

test("getEntry() throws on a non-ok response", async () => {
  const { fetchImpl } = fakeFetch({}, false, 404);
  const client = new MicroscopeClient({ baseUrl: "http://localhost:8093/microscope", fetchImpl });

  await assert.rejects(() => client.getEntry("missing"), /status 404/);
});
