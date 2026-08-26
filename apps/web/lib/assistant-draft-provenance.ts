import type { AssistantMessage } from "./types";

export type AssistantDraftProvenance = {
  label: string;
  description: string;
  tone: "ai" | "rules" | "fallback";
};

type DraftMessage = Pick<
  AssistantMessage,
  "direction" | "sender_type" | "status" | "metadata"
>;

function metadataText(value: unknown): string | undefined {
  if (typeof value !== "string") return undefined;
  const normalized = value.trim();
  return normalized || undefined;
}

function metadataFlag(value: unknown): boolean {
  return value === true || value === "true";
}

export function assistantDraftProvenance(
  message: DraftMessage,
): AssistantDraftProvenance | null {
  if (message.direction !== "OUTBOUND") return null;
  if (message.sender_type !== "ASSISTANT") return null;
  if (message.status !== "DRAFT") return null;

  const engine = metadataText(message.metadata.engine);
  const model = metadataText(message.metadata.model);
  const usedFallback = metadataFlag(message.metadata.used_fallback);

  if (usedFallback) {
    return {
      label: "Fallback seguro",
      description:
        "El proveedor principal no produjo un borrador válido; se utilizaron reglas determinísticas.",
      tone: "fallback",
    };
  }

  if (engine === "VERTEX_AI") {
    return {
      label: model ? `Vertex AI · ${model}` : "Vertex AI",
      description:
        "Borrador generado por IA y pendiente de revisión humana antes de publicarse.",
      tone: "ai",
    };
  }

  if (engine === "WEB_CHAT_RULES") {
    return {
      label: "Reglas determinísticas",
      description:
        "Borrador local generado mediante reglas y pendiente de revisión humana.",
      tone: "rules",
    };
  }

  return null;
}
