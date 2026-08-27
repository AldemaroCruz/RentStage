import type { AssistantMessage } from "./types";

export type AssistantSalesSignalKind =
  | "EVENT_TYPE"
  | "EVENT_DATE"
  | "LOCATION"
  | "GUEST_COUNT"
  | "BUDGET";

export type AssistantSalesSignal = {
  kind: AssistantSalesSignalKind;
  label: string;
  value: string;
};

export type AssistantSalesBrief = {
  signals: AssistantSalesSignal[];
  missingFields: Array<{
    kind: AssistantSalesSignalKind;
    label: string;
  }>;
  nextQuestion: string;
};

type DraftMessage = Pick<
  AssistantMessage,
  "direction" | "sender_type" | "status" | "metadata"
>;

const maximumItems = 5;
const maximumSignalLength = 180;
const maximumQuestionLength = 300;

const signalLabels: Record<AssistantSalesSignalKind, string> = {
  EVENT_TYPE: "Tipo de evento",
  EVENT_DATE: "Fecha indicada",
  LOCATION: "Ubicación",
  GUEST_COUNT: "Asistentes",
  BUDGET: "Presupuesto",
};

const missingLabels: Record<AssistantSalesSignalKind, string> = {
  EVENT_TYPE: "Tipo de evento",
  EVENT_DATE: "Fecha exacta",
  LOCATION: "Ubicación",
  GUEST_COUNT: "Cantidad de asistentes",
  BUDGET: "Presupuesto",
};

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function isSignalKind(value: unknown): value is AssistantSalesSignalKind {
  return typeof value === "string" && value in signalLabels;
}

export function assistantDraftSalesBrief(
  message: DraftMessage | undefined,
): AssistantSalesBrief | undefined {
  if (!message) return undefined;
  if (message.direction !== "OUTBOUND") return undefined;
  if (message.sender_type !== "ASSISTANT") return undefined;
  if (message.status !== "DRAFT" && message.status !== "APPROVED") {
    return undefined;
  }

  const rawBrief = message.metadata.sales_brief;
  if (!isRecord(rawBrief)) return undefined;

  const rawSignals = rawBrief.signals;
  const rawMissingFields = rawBrief.missing_fields;
  const rawNextQuestion = rawBrief.next_question;
  if (!Array.isArray(rawSignals) || !Array.isArray(rawMissingFields)) {
    return undefined;
  }
  if (typeof rawNextQuestion !== "string") return undefined;
  if (rawSignals.length > maximumItems || rawMissingFields.length > maximumItems) {
    return undefined;
  }

  const signals: AssistantSalesSignal[] = [];
  const seenSignals = new Set<AssistantSalesSignalKind>();
  for (const rawSignal of rawSignals) {
    if (!isRecord(rawSignal)) return undefined;
    if (!isSignalKind(rawSignal.kind)) return undefined;
    if (typeof rawSignal.value !== "string") return undefined;

    const value = rawSignal.value.trim();
    if (!value || Array.from(value).length > maximumSignalLength) {
      return undefined;
    }
    if (seenSignals.has(rawSignal.kind)) return undefined;
    seenSignals.add(rawSignal.kind);

    signals.push({
      kind: rawSignal.kind,
      label: signalLabels[rawSignal.kind],
      value,
    });
  }

  const missingFields: AssistantSalesBrief["missingFields"] = [];
  const seenMissing = new Set<AssistantSalesSignalKind>();
  for (const rawField of rawMissingFields) {
    if (!isSignalKind(rawField)) return undefined;
    if (seenMissing.has(rawField)) return undefined;
    seenMissing.add(rawField);
    missingFields.push({ kind: rawField, label: missingLabels[rawField] });
  }

  const nextQuestion = rawNextQuestion.trim();
  if (Array.from(nextQuestion).length > maximumQuestionLength) return undefined;
  if (missingFields.length > 0 && !nextQuestion) return undefined;
  if (missingFields.length === 0 && nextQuestion) return undefined;
  if (signals.length === 0 && missingFields.length === 0) return undefined;

  return { signals, missingFields, nextQuestion };
}
