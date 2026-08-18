import assert from "node:assert/strict";
import test from "node:test";
import { demoReadiness } from "./demo-readiness.ts";

test("a complete commercial scenario is fully ready", () => {
  const result = demoReadiness({
    activeResourceCount: 5,
    quoteCount: 3,
    acceptedQuoteCount: 1,
    activeReservationCount: 1,
    issuedTotal: 299,
    collectedTotal: 150,
    dteProviderMode: "MOCK",
    dteEnvironment: "TEST",
  });

  assert.equal(result.readyCount, 5);
  assert.equal(result.totalCount, 5);
  assert.equal(result.percent, 100);
  assert.deepEqual(Object.values(result.steps), [true, true, true, true, true]);
});

test("an empty workspace reports no ready demo steps", () => {
  const result = demoReadiness({
    activeResourceCount: 0,
    quoteCount: 0,
    acceptedQuoteCount: 0,
    activeReservationCount: 0,
    issuedTotal: 0,
    collectedTotal: 0,
  });

  assert.equal(result.readyCount, 0);
  assert.equal(result.percent, 0);
});

test("a production DTE provider is not presented as a safe demo boundary", () => {
  const result = demoReadiness({
    activeResourceCount: 1,
    quoteCount: 1,
    acceptedQuoteCount: 1,
    activeReservationCount: 1,
    issuedTotal: 100,
    collectedTotal: 50,
    dteProviderMode: "MH_HTTP",
    dteEnvironment: "PRODUCTION",
  });

  assert.equal(result.steps["fiscal-boundary"], false);
  assert.equal(result.readyCount, 4);
  assert.equal(result.percent, 80);
});

test("billing requires both an issued amount and a collected amount", () => {
  const result = demoReadiness({
    activeResourceCount: 1,
    quoteCount: 1,
    acceptedQuoteCount: 1,
    activeReservationCount: 1,
    issuedTotal: 299,
    collectedTotal: 0,
    dteProviderMode: "MOCK",
    dteEnvironment: "TEST",
  });

  assert.equal(result.steps.billing, false);
  assert.equal(result.readyCount, 4);
});
