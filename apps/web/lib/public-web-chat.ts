export type PendingPublicWebChatMessage = {
  body: string;
  clientMessageId: string;
};

export type PublicWebChatFailure =
  | "terminal"
  | "rate_limited"
  | "temporary";

export function pendingPublicWebChatMessage(
  current: PendingPublicWebChatMessage | null,
  rawBody: string,
  createID: () => string,
): PendingPublicWebChatMessage | null {
  const body = rawBody.trim();
  if (!body) return null;

  if (current?.body === body) {
    return current;
  }

  return {
    body,
    clientMessageId: createID(),
  };
}

export function classifyPublicWebChatFailure(
  status: number | undefined,
): PublicWebChatFailure {
  if (status === 404 || status === 410) {
    return "terminal";
  }

  if (status === 429) {
    return "rate_limited";
  }

  return "temporary";
}

export const PUBLIC_WEB_CHAT_POLL_INTERVAL_MS = 4_000;
export const PUBLIC_WEB_CHAT_MAX_POLL_INTERVAL_MS = 30_000;

export function publicWebChatPollDelay(
  consecutiveFailures: number,
): number {
  const failures = Number.isFinite(consecutiveFailures)
    ? Math.min(3, Math.max(0, Math.floor(consecutiveFailures)))
    : 0;

  return Math.min(
    PUBLIC_WEB_CHAT_POLL_INTERVAL_MS * 2 ** failures,
    PUBLIC_WEB_CHAT_MAX_POLL_INTERVAL_MS,
  );
}