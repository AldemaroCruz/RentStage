import assert from "node:assert/strict";
import test from "node:test";

import { assistantDraftEvidence } from "./assistant-draft-evidence.ts";

function draft(groundingReferences: unknown) {
  return {
    direction: "OUTBOUND" as const,
    sender_type: "ASSISTANT" as const,
    status: "DRAFT" as const,
    metadata: { grounding_references: groundingReferences },
  };
}

test("returns validated package and resource evidence", () => {
  assert.deepEqual(
    assistantDraftEvidence(
      draft([
        { kind: "PACKAGE", name: "  Paquete Fiesta 100 personas  " },
        { kind: "RESOURCE", name: "JBL PRX815W" },
      ]),
    ),
    [
      {
        kind: "PACKAGE",
        kindLabel: "Paquete",
        name: "Paquete Fiesta 100 personas",
      },
      {
        kind: "RESOURCE",
        kindLabel: "Recurso",
        name: "JBL PRX815W",
      },
    ],
  );
});

test("accepts approved assistant drafts and an empty evidence list", () => {
  assert.deepEqual(
    assistantDraftEvidence({
      ...draft([]),
      status: "APPROVED",
    }),
    [],
  );
});

test("rejects malformed or untrusted evidence as a whole", () => {
  const invalidValues: unknown[] = [
    undefined,
    null,
    "PACKAGE:Paquete Fiesta",
    [{ kind: "INTERNAL", name: "Paquete Fiesta" }],
    [{ kind: "PACKAGE", name: 42 }],
    [{ kind: "PACKAGE", name: "   " }],
    [{ kind: "PACKAGE", name: "a".repeat(181) }],
    [null],
    [
      { kind: "PACKAGE", name: "Paquete Fiesta" },
      { kind: "PACKAGE", name: "paquete fiesta" },
    ],
    Array.from({ length: 6 }, () => ({
      kind: "PACKAGE",
      name: "Paquete Fiesta",
    })),
  ];

  for (const value of invalidValues) {
    assert.deepEqual(assistantDraftEvidence(draft(value)), []);
  }
});

test("ignores messages that are not pending assistant drafts", () => {
  const references = [{ kind: "PACKAGE", name: "Paquete Fiesta" }];

  assert.deepEqual(assistantDraftEvidence(undefined), []);
  assert.deepEqual(
    assistantDraftEvidence({ ...draft(references), direction: "INBOUND" }),
    [],
  );
  assert.deepEqual(
    assistantDraftEvidence({ ...draft(references), sender_type: "USER" }),
    [],
  );
  assert.deepEqual(
    assistantDraftEvidence({ ...draft(references), status: "SENT" }),
    [],
  );
});
