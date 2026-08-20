import assert from "node:assert/strict";
import test from "node:test";
import { DEFAULT_SUPPORT_EMAIL, isPlaceholderLegalContact, legalContactEmail } from "./legal.ts";

test("legal contact uses an explicit public address", () => {
  assert.equal(legalContactEmail(" Privacy@Example.com "), "privacy@example.com");
  assert.equal(isPlaceholderLegalContact("privacy@example.com"), false);
});

test("legal contact fails visibly to the local placeholder", () => {
  assert.equal(legalContactEmail(""), DEFAULT_SUPPORT_EMAIL);
  assert.equal(isPlaceholderLegalContact(DEFAULT_SUPPORT_EMAIL), true);
});
