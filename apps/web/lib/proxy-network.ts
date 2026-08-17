export function trustedClientAddress(
  forwardedFor: string | null,
  realIP: string | null,
  cloudRun: boolean,
): string | null {
  const forwarded = (forwardedFor || "")
    .split(",")
    .map((entry) => entry.trim())
    .filter(Boolean);

  if (cloudRun) {
    // Cloud Run appends the actual client and Google frontend addresses to an
    // incoming X-Forwarded-For chain. Ignore any client-supplied prefix and use
    // the second-to-last value. The API only trusts this header from the web SA.
    if (forwarded.length >= 2) return forwarded[forwarded.length - 2];
    return realIP?.trim() || null;
  }

  return forwarded[0] || realIP?.trim() || null;
}
