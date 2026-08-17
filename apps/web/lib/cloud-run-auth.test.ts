import assert from "node:assert/strict";
import test from "node:test";

import {
  clearCloudRunIdentityTokenCache,
  getCloudRunIdentityToken,
  identityTokenExpiry,
} from "./cloud-run-auth.ts";

function base64url(value: unknown): string {
  return Buffer.from(JSON.stringify(value), "utf8").toString("base64url");
}

function tokenWithExpiry(expSeconds: number): string {
  return `${base64url({ alg: "none", typ: "JWT" })}.${base64url({ exp: expSeconds })}.signature`;
}

test("identityTokenExpiry reads the JWT exp claim", () => {
  assert.equal(identityTokenExpiry(tokenWithExpiry(1_900_000_000)), 1_900_000_000_000);
  assert.equal(identityTokenExpiry("not-a-jwt"), null);
});

test("getCloudRunIdentityToken requests the metadata server with the correct audience", async () => {
  clearCloudRunIdentityTokenCache();
  const now = 1_800_000_000_000;
  const expectedToken = tokenWithExpiry(Math.floor(now / 1000) + 600);
  let calls = 0;

  const fetchImpl: typeof fetch = async (input, init) => {
    calls += 1;
    const url = new URL(String(input));
    assert.equal(url.hostname, "metadata.google.internal");
    assert.equal(url.searchParams.get("audience"), "https://rentstage-api.example.run.app");
    assert.equal(url.searchParams.get("format"), "full");
    assert.equal(new Headers(init?.headers).get("Metadata-Flavor"), "Google");
    return new Response(expectedToken, { status: 200 });
  };

  const first = await getCloudRunIdentityToken("https://rentstage-api.example.run.app", {
    fetchImpl,
    now: () => now,
  });
  const second = await getCloudRunIdentityToken("https://rentstage-api.example.run.app", {
    fetchImpl,
    now: () => now + 1_000,
  });

  assert.equal(first, expectedToken);
  assert.equal(second, expectedToken);
  assert.equal(calls, 1, "a non-expiring cached token should be reused");
});

test("tokens near expiry are refreshed instead of reused", async () => {
  clearCloudRunIdentityTokenCache();
  const now = 1_800_000_000_000;
  const firstToken = tokenWithExpiry(Math.floor(now / 1000) + 30);
  const secondToken = tokenWithExpiry(Math.floor(now / 1000) + 600);
  let calls = 0;

  const fetchImpl: typeof fetch = async () => {
    calls += 1;
    return new Response(calls === 1 ? firstToken : secondToken, { status: 200 });
  };

  assert.equal(
    await getCloudRunIdentityToken("https://rentstage-api.example.run.app", { fetchImpl, now: () => now }),
    firstToken,
  );
  assert.equal(
    await getCloudRunIdentityToken("https://rentstage-api.example.run.app", {
      fetchImpl,
      now: () => now + 1_000,
    }),
    secondToken,
  );
  assert.equal(calls, 2);
});

test("getCloudRunIdentityToken rejects metadata failures and empty audiences", async () => {
  clearCloudRunIdentityTokenCache();
  const fetchImpl: typeof fetch = async () => new Response("metadata unavailable", { status: 503 });

  await assert.rejects(
    () =>
      getCloudRunIdentityToken("https://rentstage-api.example.run.app", {
        fetchImpl,
        now: () => 1_800_000_000_000,
      }),
    /Unable to obtain a Cloud Run identity token \(503\): metadata unavailable/,
  );
  await assert.rejects(() => getCloudRunIdentityToken("   "), /API_AUDIENCE is required/);
  await assert.rejects(
    () => getCloudRunIdentityToken("http://rentstage-api.example.run.app"),
    /HTTPS origin/,
  );
  await assert.rejects(
    () => getCloudRunIdentityToken("https://rentstage-api.example.run.app/unexpected"),
    /without credentials, path, query, or fragment/,
  );
});
