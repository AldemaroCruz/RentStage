import assert from "node:assert/strict";
import test from "node:test";

import {
  oppositeTheme,
  resolveTheme,
  storedTheme,
  THEME_STORAGE_KEY,
  themeBootstrapScript,
} from "./theme.ts";

test("storedTheme accepts only the two supported persistent values", () => {
  assert.equal(storedTheme("light"), "light");
  assert.equal(storedTheme("dark"), "dark");
  assert.equal(storedTheme("system"), null);
  assert.equal(storedTheme(null), null);
});

test("resolveTheme uses a stored choice before the operating-system preference", () => {
  assert.equal(resolveTheme("light", true), "light");
  assert.equal(resolveTheme("dark", false), "dark");
  assert.equal(resolveTheme(undefined, true), "dark");
  assert.equal(resolveTheme(undefined, false), "light");
});

test("oppositeTheme always produces a valid toggle target", () => {
  assert.equal(oppositeTheme("light"), "dark");
  assert.equal(oppositeTheme("dark"), "light");
});

test("the bootstrap script applies the same key before React hydrates", () => {
  const script = themeBootstrapScript();
  assert.ok(script.includes(THEME_STORAGE_KEY));
  assert.ok(script.includes("prefers-color-scheme: dark"));
  assert.ok(script.includes("document.documentElement.dataset.theme"));
  assert.equal(script.includes("innerHTML"), false);
});
