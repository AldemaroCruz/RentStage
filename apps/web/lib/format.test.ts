import assert from "node:assert/strict";
import test from "node:test";

import {
  assignmentStateLabel,
  customerSourceLabel,
  depositStatusLabel,
  formatCurrency,
  formatDate,
  formatDateTime,
  formatLongDate,
  formatQuoteNumber,
  formatReservationNumber,
  formatTime,
  invoiceStatusLabel,
  invoiceStatusTone,
  monthLabel,
  operationAlertLabel,
  paymentMethodLabel,
  pricingUnitLabel,
  quoteRequestStatusLabel,
  quoteRequestStatusTone,
  quoteStatusLabel,
  reservationSourceLabel,
  reservationStatusLabel,
  returnConditionLabel,
  toLocalDateInput,
  toLocalDateTimeInput,
  warehouseActivityLabel,
} from "./format.ts";

const timestamp = "2026-08-19T15:30:00.000Z";

test("currency values use USD by default and support an explicit currency", () => {
  assert.equal(formatCurrency(1_234.5), "$1,234.50");
  assert.equal(formatCurrency(0), "$0.00");
  assert.equal(
    formatCurrency(42, "EUR"),
    new Intl.NumberFormat("en-US", {
      style: "currency",
      currency: "EUR",
      maximumFractionDigits: 2,
    }).format(42),
  );
});

test("date and time helpers format valid values and reject absent or invalid values", () => {
  const date = new Date(timestamp);

  assert.equal(
    formatDate(timestamp),
    new Intl.DateTimeFormat("es-SV", {
      year: "numeric",
      month: "short",
      day: "2-digit",
    }).format(date),
  );
  assert.equal(
    formatDateTime(timestamp),
    new Intl.DateTimeFormat("es-SV", {
      year: "numeric",
      month: "short",
      day: "2-digit",
      hour: "2-digit",
      minute: "2-digit",
    }).format(date),
  );
  assert.equal(
    formatTime(timestamp),
    new Intl.DateTimeFormat("es-SV", {
      hour: "2-digit",
      minute: "2-digit",
    }).format(date),
  );
  assert.equal(formatDate(), "—");
  assert.equal(formatDate("not-a-date"), "—");
  assert.equal(formatDateTime(), "—");
  assert.equal(formatDateTime("not-a-date"), "—");
  assert.equal(formatTime(), "—");
  assert.equal(formatTime("not-a-date"), "—");
});

test("long and local date helpers accept strings and Date objects safely", () => {
  const date = new Date(timestamp);
  const expectedLongDate = new Intl.DateTimeFormat("es-SV", {
    weekday: "long",
    year: "numeric",
    month: "long",
    day: "numeric",
  }).format(date);
  const localDate = new Date(date.getTime() - date.getTimezoneOffset() * 60_000);

  assert.equal(formatLongDate(timestamp), expectedLongDate);
  assert.equal(formatLongDate(date), expectedLongDate);
  assert.equal(formatLongDate(), "—");
  assert.equal(formatLongDate("not-a-date"), "—");
  assert.equal(toLocalDateTimeInput(timestamp), localDate.toISOString().slice(0, 16));
  assert.equal(toLocalDateTimeInput(), "");
  assert.equal(toLocalDateTimeInput("not-a-date"), "");
  assert.equal(toLocalDateInput(timestamp), localDate.toISOString().slice(0, 10));
  assert.equal(toLocalDateInput(date), localDate.toISOString().slice(0, 10));
  assert.equal(toLocalDateInput(), "");
  assert.equal(toLocalDateInput("not-a-date"), "");
});

test("display numbers are padded consistently", () => {
  assert.equal(formatQuoteNumber(42), "QT-000042");
  assert.equal(formatReservationNumber(7), "RS-000007");
});

test("commercial labels translate known values and use their documented fallbacks", () => {
  const cases: Array<[string, string, string, (value: string) => string]> = [
    ["DAY", "día", "custom_state", pricingUnitLabel],
    ["ACCEPTED", "Aceptada", "CUSTOM_STATE", quoteStatusLabel],
    ["WHATSAPP", "WhatsApp", "CUSTOM_STATE", customerSourceLabel],
    ["CHECKED_OUT", "Entregada", "CUSTOM_STATE", reservationStatusLabel],
    ["RELEASED", "Liberada", "CUSTOM_STATE", assignmentStateLabel],
    ["ASSET_RETURNED", "Unidad devuelta", "CUSTOM_STATE", warehouseActivityLabel],
    ["AI_AGENT", "Agente AI", "CUSTOM_STATE", reservationSourceLabel],
    ["PREPARATION_INCOMPLETE", "Inventario incompleto", "CUSTOM_STATE", operationAlertLabel],
    ["IN_REVIEW", "En revisión", "CUSTOM_STATE", quoteRequestStatusLabel],
    ["PARTIALLY_PAID", "Pago parcial", "CUSTOM_STATE", invoiceStatusLabel],
    ["BANK_TRANSFER", "Transferencia bancaria", "CUSTOM_STATE", paymentMethodLabel],
    ["PARTIALLY_SETTLED", "Liquidación parcial", "CUSTOM_STATE", depositStatusLabel],
  ];

  for (const [input, expected, fallback, formatter] of cases) {
    assert.equal(formatter(input), expected);
    assert.equal(formatter("CUSTOM_STATE"), fallback);
  }
});

test("optional return conditions and visual tones use safe fallbacks", () => {
  assert.equal(returnConditionLabel(), "—");
  assert.equal(returnConditionLabel("MAINTENANCE_REQUIRED"), "Requiere mantenimiento");
  assert.equal(returnConditionLabel("CUSTOM_STATE"), "CUSTOM_STATE");
  assert.equal(quoteRequestStatusTone("IN_REVIEW"), "review");
  assert.equal(quoteRequestStatusTone("UNKNOWN"), "closed");
  assert.equal(invoiceStatusTone("OVERDUE"), "overdue");
  assert.equal(invoiceStatusTone("UNKNOWN"), "draft");
});

test("month labels are localized while invalid input remains visible", () => {
  const expected = new Intl.DateTimeFormat("es-SV", {
    month: "short",
    year: "2-digit",
  }).format(new Date("2026-08-01T00:00:00"));

  assert.equal(monthLabel("2026-08"), expected);
  assert.equal(monthLabel("not-a-month"), "not-a-month");
});
