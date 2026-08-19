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

test("the final administrative pass owns remaining fixed light surfaces", () => {
  const marker = "RentStage v0.15.5 — final administrative dark-surface pass";
  const finalPass = stylesheet.slice(stylesheet.indexOf(marker));

  assert.ok(finalPass.length > marker.length, "missing final administrative dark-theme contract");

  for (const selector of [
    ".switch-row.prominent",
    ".public-admin-package-list article",
    ".public-admin-resource-list article",
    ".quote-portal-security-card",
    ".audit-event-card",
  ]) {
    assert.ok(finalPass.includes(selector), `missing final dark-surface selector: ${selector}`);
  }

  assert.match(finalPass, /background:\s*var\(--surface\);/);
});

test("published catalog entries keep a subtle themed highlight", () => {
  assert.match(
    stylesheet,
    /:root\[data-theme="dark"\] :is\(\s*\.public-admin-package-list article\.published,\s*\.public-admin-resource-list article\.published\s*\)\s*\{[^}]*background:\s*linear-gradient\(90deg, var\(--surface-tint\), var\(--surface\) 42%\);/s,
  );
});

test("DTE provider modes use semantic dark gradients", () => {
  assert.match(
    stylesheet,
    /:root\[data-theme="dark"\] \.dte-provider-banner\.mock\s*\{[^}]*background:\s*linear-gradient\(135deg, var\(--surface-tint\), var\(--surface\)\);/s,
  );
  assert.match(
    stylesheet,
    /:root\[data-theme="dark"\] \.dte-provider-banner\.mh_http\s*\{[^}]*background:\s*linear-gradient\(135deg, var\(--blue-light\), var\(--surface\)\);/s,
  );
});

test("audit markers and the sticky billing action bar follow theme tokens", () => {
  assert.match(
    stylesheet,
    /:root\[data-theme="dark"\] \.audit-marker\s*\{[^}]*border-color:\s*var\(--surface\);/s,
  );
  assert.match(
    stylesheet,
    /:root\[data-theme="dark"\] \.billing-settings-actions\s*\{[^}]*background:\s*color-mix\(in srgb, var\(--surface\) 94%, transparent\);/s,
  );
});

test("commercial metrics use themed surfaces and a responsive layout", () => {
  const marker = "RentStage v0.16.0 — tenant-scoped commercial metrics";
  const metricsStyles = stylesheet.slice(stylesheet.indexOf(marker));

  assert.ok(metricsStyles.length > marker.length, "missing commercial metrics styles");

  for (const selector of [
    ".commercial-kpi-grid",
    ".commercial-main-grid",
    ".commercial-month-chart",
    ".commercial-detail-grid",
  ]) {
    assert.ok(metricsStyles.includes(selector), `missing metrics selector: ${selector}`);
  }

  assert.match(metricsStyles, /background:\s*var\(--surface\);/);
  assert.doesNotMatch(metricsStyles, /background:\s*(?:white|#fff(?:fff)?)\s*;/i);
  assert.match(metricsStyles, /@media \(max-width: 620px\)/);
});
