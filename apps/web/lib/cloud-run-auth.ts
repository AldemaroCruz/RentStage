type FetchLike = typeof fetch;

type IdentityTokenDependencies = {
  fetchImpl?: FetchLike;
  now?: () => number;
};

type CachedToken = {
  token: string;
  expiresAt: number;
};

const metadataIdentityEndpoint =
  "http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/identity";
const refreshSkewMs = 60_000;
const fallbackLifetimeMs = 5 * 60_000;
const metadataTimeoutMs = 3_000;
const tokenCache = new Map<string, CachedToken>();

function decodeBase64Url(value: string): string {
  const normalized = value.replace(/-/g, "+").replace(/_/g, "/");
  const padded = normalized.padEnd(Math.ceil(normalized.length / 4) * 4, "=");
  return Buffer.from(padded, "base64").toString("utf8");
}

export function identityTokenExpiry(token: string): number | null {
  const parts = token.split(".");
  if (parts.length < 2) return null;

  try {
    const payload = JSON.parse(decodeBase64Url(parts[1])) as { exp?: unknown };
    if (typeof payload.exp !== "number" || !Number.isFinite(payload.exp)) return null;
    return payload.exp * 1000;
  } catch {
    return null;
  }
}

export function clearCloudRunIdentityTokenCache(): void {
  tokenCache.clear();
}

export async function getCloudRunIdentityToken(
  audience: string,
  dependencies: IdentityTokenDependencies = {},
): Promise<string> {
  const normalizedAudience = audience.trim();
  if (!normalizedAudience) {
    throw new Error("API_AUDIENCE is required when Cloud Run service authentication is enabled.");
  }
  let audienceURL: URL;
  try {
    audienceURL = new URL(normalizedAudience);
  } catch {
    throw new Error("API_AUDIENCE must be an absolute HTTPS Cloud Run service URL.");
  }
  if (
    audienceURL.protocol !== "https:" ||
    audienceURL.username ||
    audienceURL.password ||
    audienceURL.search ||
    audienceURL.hash ||
    (audienceURL.pathname !== "/" && audienceURL.pathname !== "")
  ) {
    throw new Error("API_AUDIENCE must be an HTTPS origin without credentials, path, query, or fragment.");
  }

  const now = dependencies.now?.() ?? Date.now();
  const cached = tokenCache.get(normalizedAudience);
  if (cached && cached.expiresAt - now > refreshSkewMs) {
    return cached.token;
  }

  const fetchImpl = dependencies.fetchImpl ?? fetch;
  const metadataURL = new URL(metadataIdentityEndpoint);
  metadataURL.searchParams.set("audience", normalizedAudience);
  metadataURL.searchParams.set("format", "full");

  const response = await fetchImpl(metadataURL, {
    method: "GET",
    headers: { "Metadata-Flavor": "Google" },
    cache: "no-store",
    signal: AbortSignal.timeout(metadataTimeoutMs),
  });

  if (!response.ok) {
    const detail = (await response.text()).trim();
    throw new Error(
      `Unable to obtain a Cloud Run identity token (${response.status})${detail ? `: ${detail}` : "."}`,
    );
  }

  const token = (await response.text()).trim();
  if (!token) {
    throw new Error("The Cloud Run metadata server returned an empty identity token.");
  }

  const expiresAt = identityTokenExpiry(token) ?? now + fallbackLifetimeMs;
  tokenCache.set(normalizedAudience, { token, expiresAt });
  return token;
}
