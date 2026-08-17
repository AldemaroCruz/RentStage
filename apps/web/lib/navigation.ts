export function safeInternalPath(value: string | null | undefined): string | null {
  if (!value) return null;
  const candidate = value.trim();
  if (!candidate.startsWith("/") || candidate.startsWith("//")) return null;
  try {
    const parsed = new URL(candidate, "http://rentstage.local");
    if (parsed.origin !== "http://rentstage.local") return null;
    return `${parsed.pathname}${parsed.search}${parsed.hash}`;
  } catch {
    return null;
  }
}
