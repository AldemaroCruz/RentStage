import assert from "node:assert/strict";
import test from "node:test";

import { safeInternalPath } from "./navigation.ts";

test("safeInternalPath accepts application-local paths", () => {
  assert.equal(safeInternalPath(" /quotes/123?tab=portal#history "), "/quotes/123?tab=portal#history");
  assert.equal(safeInternalPath("/"), "/");
});

test("safeInternalPath rejects external and protocol-relative redirects", () => {
  assert.equal(safeInternalPath("https://evil.example"), null);
  assert.equal(safeInternalPath("//evil.example/path"), null);
  assert.equal(safeInternalPath("javascript:alert(1)"), null);
  assert.equal(safeInternalPath(undefined), null);
  assert.equal(safeInternalPath("/\\evil.example/path"), null);
});
