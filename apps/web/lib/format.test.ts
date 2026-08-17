import assert from "node:assert/strict";
import test from "node:test";

import {
  formatQuoteNumber,
  formatReservationNumber,
  invoiceStatusLabel,
  pricingUnitLabel,
  quoteRequestStatusTone,
  quoteStatusLabel,
  reservationStatusLabel,
} from "./format.ts";

test("display numbers are padded consistently", () => {
  assert.equal(formatQuoteNumber(42), "QT-000042");
  assert.equal(formatReservationNumber(7), "RS-000007");
});

test("known statuses are translated and unknown values remain visible", () => {
  assert.equal(quoteStatusLabel("ACCEPTED"), "Aceptada");
  assert.equal(reservationStatusLabel("CHECKED_OUT"), "Entregada");
  assert.equal(invoiceStatusLabel("PARTIALLY_PAID"), "Pago parcial");
  assert.equal(invoiceStatusLabel("CUSTOM_STATE"), "CUSTOM_STATE");
});

test("pricing and request tone helpers use safe fallbacks", () => {
  assert.equal(pricingUnitLabel("DAY"), "día");
  assert.equal(pricingUnitLabel("WEEK"), "week");
  assert.equal(quoteRequestStatusTone("IN_REVIEW"), "review");
  assert.equal(quoteRequestStatusTone("UNKNOWN"), "closed");
});
