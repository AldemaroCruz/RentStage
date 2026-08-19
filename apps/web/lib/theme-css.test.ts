import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";

const stylesheet = readFileSync(new URL("../app/globals.css", import.meta.url), "utf8");

test("the dark theme owns every high-contrast internal surface", () => {
  assert.ok(stylesheet.includes("RentStage v0.15.4 — complete internal dark-surface contract"));

  for (const selector of [
    ".operation-metric-card",
    ".quote-metric-strip article",
    ".calendar-day",
    ".package-card-grid",
    ".data-table th",
    ".architecture-note",
  ]) {
    assert.ok(stylesheet.includes(selector), `missing dark-surface selector: ${selector}`);
  }
});

test("compound search controls keep their inner input transparent", () => {
  assert.match(
    stylesheet,
    /:root\[data-theme="dark"\] :is\(\.search-box input, \.input-prefix input\)\s*\{[^}]*background:\s*transparent;/s,
  );
});

test("printing a dark session restores light document tokens", () => {
  assert.match(
    stylesheet,
    /@media print\s*\{\s*:root\[data-theme="dark"\]\s*\{[^}]*--surface:\s*#fff;[^}]*color-scheme:\s*light;/s,
  );
});
