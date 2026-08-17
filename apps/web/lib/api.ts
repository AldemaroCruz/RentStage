export type ApiErrorPayload = {
  error?: string;
  message?: string;
  fields?: Record<string, string>;
  request_id?: string;
  [key: string]: unknown;
};

export class ApiError extends Error {
  status: number;
  code?: string;
  fields?: Record<string, string>;
  requestId?: string;
  payload: ApiErrorPayload;

  constructor(status: number, payload: ApiErrorPayload) {
    super(payload.message || "The request could not be completed.");
    this.name = "ApiError";
    this.status = status;
    this.code = typeof payload.error === "string" ? payload.error : undefined;
    this.fields = payload.fields;
    this.requestId = typeof payload.request_id === "string" ? payload.request_id : undefined;
    this.payload = payload;
  }
}

let csrfToken = "";
let csrfPromise: Promise<string> | null = null;

async function loadCSRFToken(): Promise<string> {
  if (csrfToken) return csrfToken;
  if (csrfPromise) return csrfPromise;
  csrfPromise = fetch("/api/backend/api/v1/auth/csrf", {
    method: "GET",
    cache: "no-store",
    credentials: "same-origin",
  })
    .then(async (response) => {
      if (!response.ok) throw new Error("Could not initialize request protection.");
      const payload = (await response.json()) as { csrf_token?: string };
      if (!payload.csrf_token) throw new Error("CSRF token was not returned.");
      csrfToken = payload.csrf_token;
      return csrfToken;
    })
    .finally(() => {
      csrfPromise = null;
    });
  return csrfPromise;
}

function unsafe(method: string): boolean {
  return !["GET", "HEAD", "OPTIONS"].includes(method.toUpperCase());
}

export function resetCSRFToken(): void {
  csrfToken = "";
  csrfPromise = null;
}

async function request<T>(path: string, options: RequestInit, retryCSRF: boolean): Promise<T> {
  const method = (options.method || "GET").toUpperCase();
  const headers = new Headers(options.headers);
  headers.set("Accept", "application/json");
  if (options.body !== undefined && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }
  if (unsafe(method)) {
    headers.set("X-CSRF-Token", await loadCSRFToken());
  }

  const response = await fetch(`/api/backend${path}`, {
    ...options,
    method,
    cache: "no-store",
    credentials: "same-origin",
    headers,
  });

  if (!response.ok) {
    let payload: ApiErrorPayload = {};
    try {
      payload = (await response.json()) as ApiErrorPayload;
    } catch {
      payload = { message: `Request failed with status ${response.status}.` };
    }
    if (payload.error === "csrf_validation_failed") {
      resetCSRFToken();
      if (retryCSRF && unsafe(method)) {
        return request<T>(path, options, false);
      }
    }
    const sessionProbe = path === "/api/v1/auth/me" || path === "/api/v1/auth/session";
    const publicEndpoint = path.startsWith("/api/v1/public/");
    if (response.status === 401 && !sessionProbe && !publicEndpoint && typeof window !== "undefined") {
      window.dispatchEvent(new CustomEvent("rentstage:unauthorized", { detail: { path } }));
    }
    throw new ApiError(response.status, payload);
  }

  if (response.status === 204) {
    return undefined as T;
  }
  return (await response.json()) as T;
}

export async function api<T>(path: string, options: RequestInit = {}): Promise<T> {
  return request<T>(path, options, true);
}
