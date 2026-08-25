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