/**
 * Typed API client.
 *
 * A single request helper owns the access token, the documented error
 * envelope, and one silent refresh-and-retry on 401 — so no screen has to
 * think about token expiry.
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

interface RequestOptions {
  method?: string;
  body?: unknown;
  /** Required by the coin-mutating endpoints. */
  idempotencyKey?: string;
  auth?: boolean;
  /** Internal: prevents an endless refresh loop. */
  retry?: boolean;
}

async function request<T>(path: string, opts: RequestOptions = {}): Promise<T> {
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

  if (!res.ok) {
    const envelope = payload as { error?: { code?: string; message?: string } } | null;
    throw new ApiError(
      res.status,
      envelope?.error?.code ?? "UNKNOWN",
      envelope?.error?.message ?? `HTTP ${res.status}`,
    );
  }
  return payload as T;
}

function safeParse(text: string): unknown {
  try {
    return JSON.parse(text);
  } catch {
    return null;
  }
}

let refreshInFlight: Promise<boolean> | null = null;

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
      const body = (await res.json()) as AuthResponse;
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

/** Builds a fresh idempotency key for a coin-mutating request. */
export function newIdempotencyKey(): string {
  return crypto.randomUUID();
}

// ── Wire types ────────────────────────────────────────────────────────────

export interface Paged<T> {
  data: T[];
  next_cursor?: string;
}

export interface Genre {
  id: string;
  slug: string;
  name_th: string;
}

export interface NovelListItem {
  id: string;
  slug: string;
  title_th: string;
  title_cn?: string;
  author_name?: string;
  cover_url?: string;
  status: "ongoing" | "complete" | "hiatus";
  rating_avg: number;
  rating_count: number;
  followers_count: number;
  chapters_count: number;
  genres: Genre[];
}

export interface RankedNovel extends NovelListItem {
  rank: number;
  score: number;
}

export interface Arc {
  id: string;
  arc_no: number;
  name: string;
  from_chapter_no: number;
  to_chapter_no: number;
}

export interface NovelDetail extends NovelListItem {
  description?: string;
  arcs: Arc[];
  glossary_count: number;
  comments_count: number;
}

export interface ChapterListItem {
  id: string;
  chapter_no: number;
  title: string;
  price_coins: number;
  word_count: number;
  published_at?: string;
  arc_id?: string;
  unlocked: boolean;
}

export interface ChapterView {
  id: string;
  novel_id: string;
  novel_slug: string;
  novel_title_th: string;
  arc_no?: number;
  arc_name?: string;
  chapter_no: number;
  title: string;
  price_coins: number;
  word_count: number;
  locked: boolean;
  body_html: string | null;
  prev_id?: string;
  next_id?: string;
}

export interface GlossaryEntry {
  id: string;
  term_key: string;
  title_th: string;
  title_cn?: string;
  body: string;
  kind?: string;
}

export interface GlossaryGroup {
  id: string;
  name: string;
  sort_no: number;
  entries: GlossaryEntry[];
}

export interface User {
  id: string;
  username: string;
  email: string;
  display_name: string;
  avatar_url?: string;
  roles: string[];
  status: string;
}

export interface AuthResponse {
  user: User;
  token: string;
  refresh_token: string;
  expires_at: string;
}

export interface Prefs {
  theme: "light" | "sepia" | "dark";
  font: "loop" | "serif" | "sans";
  font_size: number;
  line_height: number;
  column_width: "narrow" | "normal" | "wide";
}

export interface Progress {
  novel_id: string;
  last_chapter_id?: string;
  last_chapter_no?: number;
  para_anchor: number;
  pct: number;
}

export interface ShelfItem {
  novel_id: string;
  slug: string;
  title_th: string;
  title_cn?: string;
  cover_url?: string;
  status: "reading" | "saved" | "done";
  chapters_count: number;
  last_chapter_no?: number;
  pct: number;
}

export interface ShelfCounts {
  reading: number;
  saved: number;
  done: number;
  total: number;
}

export interface Bookmark {
  id: string;
  novel_id: string;
  chapter_id: string;
  chapter_no: number;
  title: string;
  para_anchor: number;
  excerpt: string;
  note?: string;
  created_at: string;
}

export interface Wallet {
  balance: number;
  bonus_balance: number;
  bonus_expires_at?: string;
  total: number;
}

export interface LedgerEntry {
  id: string;
  delta: number;
  bonus_delta: number;
  kind: string;
  ref_type?: string;
  reason?: string;
  balance_after: number;
  created_at: string;
}

export interface CoinPack {
  id: string;
  coins: number;
  bonus_coins: number;
  price_satang: number;
  currency: string;
  is_best_value: boolean;
}

export interface PurchaseResult {
  purchase_id: string;
  status: string;
  amount_satang: number;
  mock_checkout_url: string;
}

export interface Receipt {
  ledger_id: string;
  coins_spent: number;
  balance_after: number;
  bonus_balance_after: number;
  replayed: boolean;
}

export interface CommentAuthor {
  id: string;
  display_name: string;
  avatar_url?: string;
  role: string;
}

export interface Comment {
  id: string;
  chapter_id: string;
  parent_id?: string;
  body: string;
  is_spoiler_hidden: boolean;
  likes_count: number;
  liked: boolean;
  is_translator: boolean;
  created_at: string;
  author: CommentAuthor;
  replies: Comment[];
}

export interface Review {
  id: string;
  novel_id: string;
  rating: number;
  body?: string;
  created_at: string;
  author: CommentAuthor;
}

export interface WriterNovel {
  id: string;
  slug: string;
  title_th: string;
  title_cn?: string;
  author_name?: string;
  description?: string;
  cover_url?: string;
  status: string;
  genre_ids: string[];
}

export interface WriterChapter {
  id: string;
  novel_id: string;
  arc_id?: string;
  chapter_no: number;
  title: string;
  body_source: string;
  body_html?: string;
  price_coins: number;
  word_count: number;
  status: "draft" | "scheduled" | "published";
  scheduled_at?: string;
  published_at?: string;
  updated_at?: string;
}

export interface Stats {
  reads: number;
  followers: number;
  coins_earned: number;
  reads_trend_pct: number;
  coins_trend_pct: number;
  period_from: string;
  period_to: string;
  series: { day: string; reads: number; coins_earned: number; followers: number }[];
  top_chapters: { chapter_id: string; chapter_no: number; title: string; reads: number; coins_earned: number }[];
}

export interface Notification {
  id: string;
  kind: string;
  payload: Record<string, unknown>;
  read: boolean;
  created_at: string;
}

// ── Endpoints ─────────────────────────────────────────────────────────────

function qs(params: Record<string, string | number | undefined>): string {
  const p = new URLSearchParams();
  for (const [k, v] of Object.entries(params)) {
    if (v !== undefined && v !== "") p.set(k, String(v));
  }
  const s = p.toString();
  return s ? `?${s}` : "";
}

export const api = {
  // Auth
  register: (body: { username: string; email: string; password: string; display_name?: string }) =>
    request<AuthResponse>("/auth/register", { method: "POST", body, auth: false }),
  login: (body: { email: string; password: string }) =>
    request<AuthResponse>("/auth/login", { method: "POST", body, auth: false }),
  logout: (refresh_token: string | null) =>
    request<void>("/auth/logout", { method: "POST", body: { refresh_token } }),
  me: () => request<User>("/auth/me"),
  updateMe: (body: { display_name?: string; avatar_url?: string }) =>
    request<User>("/users/me", { method: "PATCH", body }),

  // Prefs
  getPrefs: () => request<Prefs>("/users/me/prefs"),
  setPrefs: (body: Prefs) => request<Prefs>("/users/me/prefs", { method: "PUT", body }),
  getGenrePrefs: () => request<Paged<{ genre_id: string; weight: number }>>("/users/me/genre-prefs"),
  setGenrePrefs: (genres: { genre_id: string; weight: number }[]) =>
    request<Paged<{ genre_id: string; weight: number }>>("/users/me/genre-prefs", {
      method: "PUT",
      body: { genres },
    }),

  // Catalog
  listGenres: () => request<Paged<Genre>>("/genres", { auth: false }),
  listNovels: (params: { q?: string; genre?: string; sort?: string; cursor?: string; limit?: number } = {}) =>
    request<Paged<NovelListItem>>(`/novels${qs(params)}`),
  getNovel: (idOrSlug: string) => request<NovelDetail>(`/novels/${encodeURIComponent(idOrSlug)}`),
  listChapters: (novelId: string, limit = 500) =>
    request<Paged<ChapterListItem>>(`/novels/${novelId}/chapters${qs({ limit })}`),
  listArcs: (novelId: string) => request<Paged<Arc>>(`/novels/${novelId}/arcs`),
  getGlossary: (novelId: string) => request<Paged<GlossaryGroup>>(`/novels/${novelId}/glossary`),
  weeklyRanking: (limit = 5) => request<Paged<RankedNovel>>(`/ranking/weekly${qs({ limit })}`),

  // Reading
  getChapter: (id: string) => request<ChapterView>(`/chapters/${id}`),
  readEvent: (id: string) => request<void>(`/chapters/${id}/read-event`, { method: "POST", body: {} }),
  getProgress: (novelId: string) => request<Progress>(`/me/progress/${novelId}`),
  saveProgress: (novelId: string, body: Partial<Progress>) =>
    request<Progress>(`/me/progress/${novelId}`, { method: "PUT", body }),

  // Library
  getShelf: (tab?: string) =>
    request<Paged<ShelfItem> & { counts: ShelfCounts }>(`/me/library${qs({ tab })}`),
  setShelfStatus: (novelId: string, status: string) =>
    request<{ novel_id: string; status: string }>(`/me/library/${novelId}`, {
      method: "PUT",
      body: { status },
    }),
  removeFromShelf: (novelId: string) => request<void>(`/me/library/${novelId}`, { method: "DELETE" }),

  listBookmarks: (novelId?: string) =>
    request<Paged<Bookmark>>(`/me/bookmarks${qs({ novel_id: novelId })}`),
  createBookmark: (body: {
    novel_id: string;
    chapter_id: string;
    para_anchor: number;
    excerpt: string;
    note?: string;
  }) => request<Bookmark>("/me/bookmarks", { method: "POST", body }),
  deleteBookmark: (id: string) => request<void>(`/me/bookmarks/${id}`, { method: "DELETE" }),

  isFollowing: (novelId: string) => request<{ following: boolean }>(`/me/follows/${novelId}`),
  follow: (novelId: string) =>
    request<{ following: boolean }>(`/me/follows/${novelId}`, { method: "POST" }),
  unfollow: (novelId: string) =>
    request<{ following: boolean }>(`/me/follows/${novelId}`, { method: "DELETE" }),

  // Coins
  getWallet: () => request<Wallet>("/me/wallet"),
  getLedger: (cursor?: string) => request<Paged<LedgerEntry>>(`/me/wallet/ledger${qs({ cursor })}`),
  listPacks: () => request<Paged<CoinPack>>("/coin-packs", { auth: false }),
  createPurchase: (packId: string) =>
    request<PurchaseResult>("/purchases", {
      method: "POST",
      body: { pack_id: packId },
      idempotencyKey: newIdempotencyKey(),
    }),
  completePurchase: (purchaseId: string, key: string) =>
    request<Receipt>(`/purchases/${purchaseId}/mock-complete`, {
      method: "POST",
      idempotencyKey: key,
    }),
  failPurchase: (purchaseId: string) =>
    request<{ status: string }>(`/purchases/${purchaseId}/mock-fail`, { method: "POST" }),
  unlockChapter: (chapterId: string, key: string) =>
    request<Receipt>(`/chapters/${chapterId}/unlock`, { method: "POST", idempotencyKey: key }),

  // Social
  listComments: (chapterId: string, sort = "popular") =>
    request<Paged<Comment>>(`/chapters/${chapterId}/comments${qs({ sort })}`),
  createComment: (chapterId: string, body: { body: string; parent_id?: string; is_spoiler_hidden?: boolean }) =>
    request<Comment>(`/chapters/${chapterId}/comments`, { method: "POST", body }),
  likeComment: (id: string) =>
    request<{ likes_count: number; liked: boolean }>(`/comments/${id}/like`, { method: "POST" }),
  unlikeComment: (id: string) =>
    request<{ likes_count: number; liked: boolean }>(`/comments/${id}/like`, { method: "DELETE" }),
  deleteComment: (id: string) => request<void>(`/comments/${id}`, { method: "DELETE" }),

  listReviews: (novelId: string) =>
    request<Paged<Review> & { my_review?: Review }>(`/novels/${novelId}/reviews`),
  upsertReview: (novelId: string, body: { rating: number; body?: string }) =>
    request<Review>(`/novels/${novelId}/reviews`, { method: "POST", body }),

  // Writer
  listWriterNovels: () => request<Paged<WriterNovel>>("/writer/novels"),
  createWriterNovel: (body: Partial<WriterNovel>) =>
    request<WriterNovel>("/writer/novels", { method: "POST", body }),
  updateWriterNovel: (id: string, body: Partial<WriterNovel>) =>
    request<WriterNovel>(`/writer/novels/${id}`, { method: "PATCH", body }),
  listWriterChapters: (novelId: string) =>
    request<Paged<WriterChapter>>(`/writer/novels/${novelId}/chapters`),
  getWriterChapter: (id: string) => request<WriterChapter>(`/writer/chapters/${id}`),
  createWriterChapter: (
    novelId: string,
    body: { chapter_no: number; title: string; body_source: string; price_coins: number },
  ) => request<WriterChapter>(`/writer/novels/${novelId}/chapters`, { method: "POST", body }),
  saveWriterChapter: (
    id: string,
    body: { chapter_no: number; title: string; body_source: string; price_coins: number },
  ) => request<WriterChapter>(`/writer/chapters/${id}`, { method: "PUT", body }),
  publishChapter: (id: string, scheduledAt?: string) =>
    request<WriterChapter>(`/writer/chapters/${id}/publish`, {
      method: "POST",
      body: scheduledAt ? { scheduled_at: scheduledAt } : {},
    }),
  unpublishChapter: (id: string) =>
    request<WriterChapter>(`/writer/chapters/${id}/unpublish`, { method: "POST" }),
  listWriterGlossary: (novelId: string) =>
    request<Paged<GlossaryGroup>>(`/writer/novels/${novelId}/glossary`),
  getStats: (novelId: string, period = "14d") =>
    request<Stats>(`/writer/stats/novels/${novelId}${qs({ period })}`),

  // Notifications
  listNotifications: () => request<Paged<Notification>>("/me/notifications"),
  unreadCount: () => request<{ unread: number }>("/me/notifications/unread-count"),
  markNotificationsRead: (ids: number[] = []) =>
    request<{ unread: number }>("/me/notifications/read", { method: "POST", body: { ids } }),
};
