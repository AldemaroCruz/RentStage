import assert from "node:assert/strict";
import test from "node:test";

import {
  authEmulatorEnabled,
  LOCAL_DEMO_CREDENTIALS,
  loginDefaults,
} from "./runtime-config.ts";

test("local authentication remains the default for Docker Compose builds", () => {
  assert.equal(authEmulatorEnabled(undefined), true);
  assert.equal(authEmulatorEnabled("true"), true);
  assert.equal(authEmulatorEnabled(" TRUE "), true);
});

test("non-local builds fail closed and never enable demo credentials", () => {
  assert.equal(authEmulatorEnabled("false"), false);
  assert.equal(authEmulatorEnabled("FALSE"), false);
  assert.equal(authEmulatorEnabled("staging"), false);
  assert.equal(authEmulatorEnabled(""), false);
});

test("the documented local account is centralized in runtime configuration", () => {
  assert.deepEqual(LOCAL_DEMO_CREDENTIALS, {
    email: "owner@rentstage.local",
    password: "RentStage123!",
  });
  assert.deepEqual(loginDefaults("true"), LOCAL_DEMO_CREDENTIALS);
  assert.deepEqual(loginDefaults("false"), { email: "", password: "" });
});
