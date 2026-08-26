import assert from "node:assert/strict";
import test from "node:test";

import { assistantDraftProvenance } from "./assistant-draft-provenance.ts";

function draft(metadata: Record<string, unknown>) {
  return {
    direction: "OUTBOUND" as const,
    sender_type: "ASSISTANT" as const,
    status: "DRAFT" as const,
    metadata,
  };
}

test("labels Vertex AI drafts with their normalized model", () => {
  assert.deepEqual(
    assistantDraftProvenance(
      draft({
        engine: " VERTEX_AI ",
        model: " gemini-2.5-flash ",
        used_fallback: false,
      }),
    ),
    {
      label: "Vertex AI · gemini-2.5-flash",
      description:
        "Borrador generado por IA y pendiente de revisión humana antes de publicarse.",
      tone: "ai",
    },
  );
});

test("keeps the Vertex label useful when model metadata is absent", () => {
  assert.equal(
    assistantDraftProvenance(
      draft({ engine: "VERTEX_AI", model: "   " }),
    )?.label,
    "Vertex AI",
  );

  assert.equal(
    assistantDraftProvenance(
      draft({ engine: "VERTEX_AI", model: 42 }),
    )?.label,
    "Vertex AI",
  );
});

test("gives fallback precedence over provider metadata", () => {
  const result = assistantDraftProvenance(
    draft({
      engine: "VERTEX_AI",
      model: "gemini-2.5-flash",
      used_fallback: "true",
    }),
  );

  assert.equal(result?.label, "Fallback seguro");
  assert.equal(result?.tone, "fallback");
});

test("labels primary deterministic drafts without calling them fallback", () => {
  const result = assistantDraftProvenance(
    draft({
      engine: "WEB_CHAT_RULES",
      model: "DETERMINISTIC_V1",
      used_fallback: false,
    }),
  );

  assert.equal(result?.label, "Reglas determinísticas");
  assert.equal(result?.tone, "rules");
});

test("does not expose unknown or malformed engine metadata", () => {
  assert.equal(
    assistantDraftProvenance(draft({ engine: "UNTRUSTED_VALUE" })),
    null,
  );
  assert.equal(
    assistantDraftProvenance(draft({ engine: { raw: "VERTEX_AI" } })),
    null,
  );
});

test("ignores messages that are not pending assistant drafts", () => {
  assert.equal(
    assistantDraftProvenance({
      ...draft({ engine: "VERTEX_AI" }),
      direction: "INBOUND",
    }),
    null,
  );
  assert.equal(
    assistantDraftProvenance({
      ...draft({ engine: "VERTEX_AI" }),
      sender_type: "USER",
    }),
    null,
  );
  assert.equal(
    assistantDraftProvenance({
      ...draft({ engine: "VERTEX_AI" }),
      status: "SENT",
    }),
    null,
  );
});
