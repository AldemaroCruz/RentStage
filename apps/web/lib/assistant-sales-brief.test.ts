import assert from "node:assert/strict";
import test from "node:test";

import { assistantDraftSalesBrief } from "./assistant-sales-brief.ts";

function draft(salesBrief: unknown) {
  return {
    direction: "OUTBOUND" as const,
    sender_type: "ASSISTANT" as const,
    status: "DRAFT" as const,
    metadata: { sales_brief: salesBrief },
  };
}

test("returns a bounded reviewer-facing sales brief", () => {
  assert.deepEqual(
    assistantDraftSalesBrief(
      draft({
        signals: [
          { kind: "EVENT_TYPE", value: " evento corporativo " },
          { kind: "LOCATION", value: "San Salvador" },
          { kind: "GUEST_COUNT", value: "90 personas" },
        ],
        missing_fields: ["EVENT_DATE", "BUDGET"],
        next_question: " ¿Qué fecha tiene prevista para el evento? ",
      }),
    ),
    {
      signals: [
        {
          kind: "EVENT_TYPE",
          label: "Tipo de evento",
          value: "evento corporativo",
        },
        { kind: "LOCATION", label: "Ubicación", value: "San Salvador" },
        { kind: "GUEST_COUNT", label: "Asistentes", value: "90 personas" },
      ],
      missingFields: [
        { kind: "EVENT_DATE", label: "Fecha exacta" },
        { kind: "BUDGET", label: "Presupuesto" },
      ],
      nextQuestion: "¿Qué fecha tiene prevista para el evento?",
    },
  );
});

test("returns undefined for an empty sales brief", () => {
  assert.equal(
    assistantDraftSalesBrief(
      draft({ signals: [], missing_fields: [], next_question: "" }),
    ),
    undefined,
  );
});

test("rejects malformed or oversized sales brief metadata", () => {
  const invalidValues = [
    null,
    {},
    { signals: "invalid", missing_fields: [], next_question: "" },
    {
      signals: [{ kind: "PHONE", value: "7000-0000" }],
      missing_fields: [],
      next_question: "",
    },
    {
      signals: [
        { kind: "LOCATION", value: "San Salvador" },
        { kind: "LOCATION", value: "Santa Ana" },
      ],
      missing_fields: [],
      next_question: "",
    },
    {
      signals: [{ kind: "LOCATION", value: "x".repeat(181) }],
      missing_fields: [],
      next_question: "",
    },
    {
      signals: [],
      missing_fields: ["EVENT_DATE"],
      next_question: "",
    },
    {
      signals: [],
      missing_fields: [],
      next_question: "¿Qué fecha tiene prevista?",
    },
  ];

  for (const value of invalidValues) {
    assert.equal(assistantDraftSalesBrief(draft(value)), undefined);
  }
});

test("ignores messages that are not pending assistant drafts", () => {
  const brief = {
    signals: [{ kind: "LOCATION", value: "San Salvador" }],
    missing_fields: [],
    next_question: "",
  };

  assert.equal(assistantDraftSalesBrief(undefined), undefined);
  assert.equal(
    assistantDraftSalesBrief({ ...draft(brief), direction: "INBOUND" }),
    undefined,
  );
  assert.equal(
    assistantDraftSalesBrief({ ...draft(brief), sender_type: "USER" }),
    undefined,
  );
  assert.equal(
    assistantDraftSalesBrief({ ...draft(brief), status: "SENT" }),
    undefined,
  );
});
