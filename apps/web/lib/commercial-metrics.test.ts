import assert from "node:assert/strict";
import test from "node:test";
import { customerSourceLabel, metricBarPercent, responseTimeLabel } from "./commercial-metrics.ts";

test("metric bars keep visible positive values inside their bounds", () => {
  assert.equal(metricBarPercent(0, 10), 0);
  assert.equal(metricBarPercent(1, 100), 4);
  assert.equal(metricBarPercent(5, 10), 50);
  assert.equal(metricBarPercent(20, 10), 100);
  assert.equal(metricBarPercent(Number.NaN, 10), 0);
});

test("response time labels disclose missing samples and format useful durations", () => {
  assert.equal(responseTimeLabel(0, 0), "Sin muestra");
  assert.equal(responseTimeLabel(0.4, 2), "< 1 min");
  assert.equal(responseTimeLabel(12.6, 3), "13 min");
  assert.equal(responseTimeLabel(120, 1), "2 h");
  assert.equal(responseTimeLabel(81, 1), "1 h 21 min");
});

test("customer sources use commercial labels and retain unknown providers", () => {
  assert.equal(customerSourceLabel("WEB"), "Catálogo web");
  assert.equal(customerSourceLabel("WHATSAPP"), "WhatsApp");
  assert.equal(customerSourceLabel("MANUAL"), "Registro manual");
  assert.equal(customerSourceLabel("IMPORT"), "Importación");
  assert.equal(customerSourceLabel("PARTNER"), "PARTNER");
});
