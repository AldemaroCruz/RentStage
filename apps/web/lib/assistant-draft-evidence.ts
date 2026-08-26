import type { AssistantMessage } from "./types";

export type AssistantDraftEvidence = {
  kind: "PACKAGE" | "RESOURCE";
  kindLabel: "Paquete" | "Recurso";
  name: string;
};

type DraftMessage = Pick<
  AssistantMessage,
  "direction" | "sender_type" | "status" | "metadata"
>;

const maximumReferences = 5;
const maximumNameLength = 180;

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

export function assistantDraftEvidence(
  message: DraftMessage | undefined,
): AssistantDraftEvidence[] {
  if (!message) return [];
  if (message.direction !== "OUTBOUND") return [];
  if (message.sender_type !== "ASSISTANT") return [];
  if (message.status !== "DRAFT" && message.status !== "APPROVED") return [];

  const rawReferences = message.metadata.grounding_references;
  if (!Array.isArray(rawReferences)) return [];
  if (rawReferences.length > maximumReferences) return [];

  const result: AssistantDraftEvidence[] = [];
  const seen = new Set<string>();

  for (const rawReference of rawReferences) {
    if (!isRecord(rawReference)) return [];

    const kind = rawReference.kind;
    const rawName = rawReference.name;
    if (kind !== "PACKAGE" && kind !== "RESOURCE") return [];
    if (typeof rawName !== "string") return [];

    const name = rawName.trim();
    if (!name || Array.from(name).length > maximumNameLength) return [];

    const key = `${kind}\u0000${name.toLocaleLowerCase("es")}`;
    if (seen.has(key)) return [];
    seen.add(key);

    result.push({
      kind,
      kindLabel: kind === "PACKAGE" ? "Paquete" : "Recurso",
      name,
    });
  }

  return result;
}
