import { getCloudRunIdentityToken } from "@/lib/cloud-run-auth";
import { trustedClientAddress } from "@/lib/proxy-network";
import { NextRequest } from "next/server";

export const dynamic = "force-dynamic";
export const runtime = "nodejs";

type RouteContext = {
  params: Promise<{ path: string[] }>;
};

async function proxy(request: NextRequest, context: RouteContext): Promise<Response> {
  const apiBase = process.env.API_INTERNAL_URL || "http://localhost:8080";
  const { path } = await context.params;

  const target = new URL(`${apiBase}/${path.join("/")}`);
  target.search = request.nextUrl.search;

  const headers = new Headers();
  headers.set("Accept", request.headers.get("Accept") || "application/json");

  for (const name of [
    "Cookie",
    "Content-Type",
    "X-CSRF-Token",
    "X-Request-ID",
    "X-RentStage-Quote-Token",
    "Origin",
    "User-Agent",
  ]) {
    const value = request.headers.get(name);
    if (value) headers.set(name, value);
  }

  const cloudRunIdentityEnabled = process.env.CLOUD_RUN_IDENTITY_TOKEN_ENABLED === "true";
  const clientAddress = trustedClientAddress(
    request.headers.get("X-Forwarded-For"),
    request.headers.get("X-Real-IP"),
    cloudRunIdentityEnabled,
  );
  if (clientAddress) headers.set("X-Forwarded-For", clientAddress);

  let body: ArrayBuffer | undefined;
  if (!["GET", "HEAD"].includes(request.method)) {
    const buffer = await request.arrayBuffer();
    if (buffer.byteLength > 0) body = buffer;
  }

  try {
    if (cloudRunIdentityEnabled) {
      const audience = process.env.API_AUDIENCE || apiBase;
      const identityToken = await getCloudRunIdentityToken(audience);
      // Cloud Run validates this header before forwarding the request. Using
      // X-Serverless-Authorization leaves Authorization free for future app APIs.
      headers.set("X-Serverless-Authorization", `Bearer ${identityToken}`);
    }

    const response = await fetch(target, {
      method: request.method,
      headers,
      body,
      cache: "no-store",
      redirect: "manual",
    });

    const responseHeaders = new Headers();
    responseHeaders.set(
      "Content-Type",
      response.headers.get("Content-Type") || "application/json; charset=utf-8",
    );
    const upstreamRequestID = response.headers.get("X-Request-ID");
    if (upstreamRequestID) responseHeaders.set("X-Request-ID", upstreamRequestID);
    for (const name of [
      "Retry-After",
      "Cache-Control",
      "Content-Language",
      "Pragma",
      "Referrer-Policy",
      "X-Content-Type-Options",
      "X-Robots-Tag",
      "Vary",
    ]) {
      const value = response.headers.get(name);
      if (value) responseHeaders.set(name, value);
    }

    const cookieHeaders = response.headers as Headers & { getSetCookie?: () => string[] };
    const setCookies = cookieHeaders.getSetCookie?.() || [];
    if (setCookies.length > 0) {
      for (const cookie of setCookies) responseHeaders.append("Set-Cookie", cookie);
    } else {
      const cookie = response.headers.get("Set-Cookie");
      if (cookie) responseHeaders.append("Set-Cookie", cookie);
    }

    return new Response(response.body, {
      status: response.status,
      headers: responseHeaders,
    });
  } catch (error) {
    const payload: Record<string, string> = {
      error: "upstream_unavailable",
      message: "RentStage API is not available.",
    };
    if (process.env.NODE_ENV !== "production") {
      payload.detail = error instanceof Error ? error.message : "Unknown proxy error";
    }
    return Response.json(payload, {
      status: 502,
      headers: { "Cache-Control": "no-store" },
    });
  }
}

export { proxy as GET, proxy as POST, proxy as PATCH, proxy as DELETE, proxy as PUT };
