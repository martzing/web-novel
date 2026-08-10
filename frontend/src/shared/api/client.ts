/**
 * The HTTP transport every feature's `api.ts` is built on.
 *
 * One request helper owns the access token, the documented error envelope, and
 * one silent refresh-and-retry on 401 — so no screen has to think about token
 * expiry. It knows no endpoints and no domain types; those live with their
 * feature.
 */

const BASE = import.meta.env.VITE_API_BASE ?? "/api/v1";

const ACCESS_KEY = "mokchan.token";
const REFRESH_KEY = "mokchan.refresh";

export const tokenStore = {
  access: () => localStorage.getItem(ACCESS_KEY),
  refresh: () => localStorage.getItem(REFRESH_KEY),
  set(access: string, refresh?: string) {
    localStorage.setItem(ACCESS_KEY, access);
    if (refresh) localStorage.setItem(REFRESH_KEY, refresh);
  },
  clear() {
    localStorage.removeItem(ACCESS_KEY);
    localStorage.removeItem(REFRESH_KEY);
  },
};

/** ApiError carries the server's error code so screens can react to specifics. */
export class ApiError extends Error {
  readonly status: number;
  readonly code: string;

  constructor(status: number, code: string, message: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
  }
}

export interface RequestOptions {
  method?: string;
  body?: unknown;
  /** Required by the coin-mutating endpoints. */
  idempotencyKey?: string;
  auth?: boolean;
  /** Internal: prevents an endless refresh loop. */
  retry?: boolean;
}

export async function request<T>(path: string, opts: RequestOptions = {}): Promise<T> {
  const { method = "GET", body, idempotencyKey, auth = true, retry = true } = opts;

  const headers: Record<string, string> = { Accept: "application/json" };
  if (body !== undefined) headers["Content-Type"] = "application/json";
  if (idempotencyKey) headers["Idempotency-Key"] = idempotencyKey;

  const token = tokenStore.access();
  if (auth && token) headers.Authorization = `Bearer ${token}`;

  const res = await fetch(`${BASE}${path}`, {
    method,
    headers,
    body: body === undefined ? undefined : JSON.stringify(body),
  });

  // An expired access token is refreshed once, transparently.
  if (res.status === 401 && retry && auth && tokenStore.refresh()) {
    const refreshed = await tryRefresh();
    if (refreshed) return request<T>(path, { ...opts, retry: false });
  }

  if (res.status === 204) return undefined as T;

  const text = await res.text();
  const payload = text ? safeParse(text) : null;

  if (!res.ok) throw toApiError(res.status, payload);
  return payload as T;
}

function safeParse(text: string): unknown {
  try {
    return JSON.parse(text);
  } catch {
    return null;
  }
}

function toApiError(status: number, payload: unknown): ApiError {
  const envelope = payload as { error?: { code?: string; message?: string } } | null;
  return new ApiError(
    status,
    envelope?.error?.code ?? "UNKNOWN",
    envelope?.error?.message ?? `HTTP ${status}`,
  );
}

let refreshInFlight: Promise<boolean> | null = null;

/**
 * The refresh response, restated here rather than imported from `identity`.
 *
 * The transport cannot depend on a feature, and these two fields are the whole
 * contract it needs from `/auth/refresh`.
 */
interface RefreshResponse {
  token: string;
  refresh_token: string;
}

/** Refreshes the access token, coalescing concurrent callers into one request. */
function tryRefresh(): Promise<boolean> {
  if (refreshInFlight) return refreshInFlight;

  refreshInFlight = (async () => {
    try {
      const res = await fetch(`${BASE}/auth/refresh`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ refresh_token: tokenStore.refresh() }),
      });
      if (!res.ok) {
        tokenStore.clear();
        return false;
      }
      const body = (await res.json()) as RefreshResponse;
      tokenStore.set(body.token, body.refresh_token);
      return true;
    } catch {
      return false;
    } finally {
      refreshInFlight = null;
    }
  })();

  return refreshInFlight;
}

/**
 * Multipart upload.
 *
 * It cannot go through `request` — the body is FormData and the browser must
 * set its own boundary — but it still needs the bearer token and the same
 * error envelope.
 */
export async function upload<T>(path: string, field: string, file: File): Promise<T> {
  const form = new FormData();
  form.append(field, file);

  const headers: Record<string, string> = { Accept: "application/json" };
  const token = tokenStore.access();
  if (token) headers.Authorization = `Bearer ${token}`;

  const res = await fetch(`${BASE}${path}`, { method: "POST", headers, body: form });

  const text = await res.text();
  const payload = text ? safeParse(text) : null;
  if (!res.ok) throw toApiError(res.status, payload);
  return payload as T;
}

/** Builds a fresh idempotency key for a coin-mutating request. */
export function newIdempotencyKey(): string {
  return crypto.randomUUID();
}

/** Serialises a query string, dropping empty and undefined values. */
export function qs(params: Record<string, string | number | undefined>): string {
  const p = new URLSearchParams();
  for (const [k, v] of Object.entries(params)) {
    if (v !== undefined && v !== "") p.set(k, String(v));
  }
  const s = p.toString();
  return s ? `?${s}` : "";
}
