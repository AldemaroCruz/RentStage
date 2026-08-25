import assert from "node:assert/strict";
import test from "node:test";

import {
  classifyPublicWebChatFailure,
  pendingPublicWebChatMessage,
} from "./public-web-chat.ts";

test("creates a stable client message identity from normalized content", () => {
  let calls = 0;

  const pending = pendingPublicWebChatMessage(
    null,
    "  Necesito información  ",
    () => {
      calls += 1;
      return "message-1";
    },
  );

  assert.deepEqual(pending, {
    body: "Necesito información",
    clientMessageId: "message-1",
  });
  assert.equal(calls, 1);
});

test("reuses the same identity when retrying the same message", () => {
  const current = {
    body: "Necesito información",
    clientMessageId: "message-1",
  };

  let calls = 0;
  const pending = pendingPublicWebChatMessage(
    current,
    " Necesito información ",
    () => {
      calls += 1;
      return "message-2";
    },
  );

  assert.equal(pending, current);
  assert.equal(calls, 0);
});

test("creates a new identity when the visitor changes the message", () => {
  const pending = pendingPublicWebChatMessage(
    {
      body: "Mensaje anterior",
      clientMessageId: "message-1",
    },
    "Mensaje corregido",
    () => "message-2",
  );

  assert.deepEqual(pending, {
    body: "Mensaje corregido",
    clientMessageId: "message-2",
  });
});

test("rejects an empty normalized message", () => {
  const pending = pendingPublicWebChatMessage(
    null,
    "   ",
    () => "unused",
  );

  assert.equal(pending, null);
});

test("classifies unavailable sessions as terminal failures", () => {
  assert.equal(classifyPublicWebChatFailure(404), "terminal");
  assert.equal(classifyPublicWebChatFailure(410), "terminal");
});

test("classifies rate limits separately from temporary failures", () => {
  assert.equal(classifyPublicWebChatFailure(429), "rate_limited");
  assert.equal(classifyPublicWebChatFailure(500), "temporary");
  assert.equal(classifyPublicWebChatFailure(undefined), "temporary");
});