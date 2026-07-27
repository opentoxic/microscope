import test from "node:test";
import assert from "node:assert/strict";
import { existsSync } from "node:fs";
import { createRequire } from "node:module";

// Exercises the built dist/ output (requires `npm run build` to have run first),
// guarding against the SDK regressing to ESM-only and breaking CommonJS consumers.
const require = createRequire(import.meta.url);

test("cjs build is requireable and exposes MicroscopeClient", { skip: !existsSync("dist/cjs/index.js") }, () => {
  const { MicroscopeClient } = require("../dist/cjs/index.js");
  assert.equal(typeof MicroscopeClient, "function");
});

test("cjs express integration is requireable", { skip: !existsSync("dist/cjs/express.js") }, () => {
  const { microscopeMiddleware } = require("../dist/cjs/express.js");
  assert.equal(typeof microscopeMiddleware, "function");
});

test("esm build is importable and exposes MicroscopeClient", { skip: !existsSync("dist/esm/index.js") }, async () => {
  const { MicroscopeClient } = await import("../dist/esm/index.js");
  assert.equal(typeof MicroscopeClient, "function");
});

test("dist/cjs is marked commonjs so it isn't shadowed by the package's \"type\": \"module\"", {
  skip: !existsSync("dist/cjs/package.json"),
}, () => {
  const pkg = require("../dist/cjs/package.json");
  assert.equal(pkg.type, "commonjs");
});
