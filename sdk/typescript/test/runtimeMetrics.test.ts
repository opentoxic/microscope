import test from "node:test";
import assert from "node:assert/strict";

import { isNodeRuntime, sampleRuntimeMetrics } from "../src/runtimeMetrics.ts";

test("isNodeRuntime() is true under node", () => {
  assert.equal(isNodeRuntime(), true);
});

test("sampleRuntimeMetrics() returns the expected shape", () => {
  const metrics = sampleRuntimeMetrics();

  assert.equal(metrics.name, "node.runtime");
  assert.equal(metrics.language, "node");
  assert.equal(metrics.unit, "%");
  assert.equal(typeof metrics.value, "number");
  assert.ok(metrics.value >= 0 && metrics.value <= 100);
  assert.equal(typeof metrics.memory_mb, "number");
  assert.ok((metrics.memory_mb as number) > 0);
  assert.equal(typeof metrics.rss_mb, "number");
  assert.ok((metrics.rss_mb as number) > 0);
});
