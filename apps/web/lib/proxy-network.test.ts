import assert from "node:assert/strict";
import test from "node:test";

import { trustedClientAddress } from "./proxy-network.ts";

test("Cloud Run uses the infrastructure-appended client address", () => {
  assert.equal(
    trustedClientAddress("198.51.100.250, 203.0.113.25, 35.191.0.1", null, true),
    "203.0.113.25",
  );
  assert.equal(trustedClientAddress("203.0.113.25, 35.191.0.1", null, true), "203.0.113.25");
});

test("Cloud Run never trusts a one-element client-supplied forwarded chain", () => {
  assert.equal(trustedClientAddress("198.51.100.250", "203.0.113.25", true), "203.0.113.25");
  assert.equal(trustedClientAddress("198.51.100.250", null, true), null);
});

test("localhost keeps the existing first-address behavior", () => {
  assert.equal(trustedClientAddress("127.0.0.1, 10.0.0.2", null, false), "127.0.0.1");
  assert.equal(trustedClientAddress(null, "127.0.0.1", false), "127.0.0.1");
});
