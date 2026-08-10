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

/**
 * Cover upload is multipart, so it cannot go through `request` — but it still
 * needs the bearer token and the same error envelope.
 */
async function uploadCover(novelId: string, file: File): Promise<{ cover_url: string }> {
  const form = new FormData();
  form.append("cover", file);

  const headers: Record<string, string> = { Accept: "application/json" };
  const token = tokenStore.access();
  if (token) headers.Authorization = `Bearer ${token}`;

  const res = await fetch(`${BASE}/writer/novels/${novelId}/cover`, {
    method: "POST",
    headers,
    body: form,
  });

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
  return payload as { cover_url: string };
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

/** Cover template styles, mirroring the CHECK on novels.cover_style. */
export type CoverStyle = "image" | "ink" | "seal" | "brush" | "plain";

/** ซ่อนจากหน้าร้าน is only ever seen by the novel's own translator. */
export type NovelStatus = "ongoing" | "complete" | "hiatus" | "hidden";

export interface NovelListItem {
  id: string;
  slug: string;
  title_th: string;
  title_cn?: string;
  author_name?: string;
  cover_url?: string;
  status: NovelStatus;
  rating_avg: number;
  rating_count: number;
  followers_count: number;
  /** บทที่แปลแล้ว. */
  chapters_count: number;
  /** บทในต้นฉบับ; 0 when the translator has not entered one. */
  source_chapters_count: number;
  cover_style: CoverStyle;
  cover_color?: string;
  cover_text?: string;
  series_id?: string;
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
  sell_by_arc: boolean;
  tips_enabled: boolean;
  /** รอบปล่อยบทใหม่ — informational only. */
  release_schedule?: ReleaseSchedule;
  early_access_hours: number;
  /**
   * The novel's default pricing, so the detail page can state the deal in one
   * line. An individual chapter may still override it, which is why the ToC
   * rows carry their own price tags too.
   */
  price_per_chapter: number;
  free_until_chapter: number;
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
  /**
   * Why the body is withheld. "paywall" wants a purchase; "early_access" wants
   * an auto-unlock subscription, and no amount of coins will open it yet.
   */
  locked_reason?: "paywall" | "early_access";
  tips_enabled?: boolean;
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
  source_chapters_count: number;
  last_chapter_no?: number;
  /** The chapter the continue button opens; absent before the first read. */
  last_chapter_id?: string;
  cover_style: CoverStyle;
  cover_color?: string;
  cover_text?: string;
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
  status: NovelStatus;
  /**
   * Genre ids as **numbers**, and the one id in this API that is not a string.
   *
   * `encoding/json`'s `,string` option is silently ignored on slices, so
   * `[]int64` on the Go side emits and demands JSON numbers while every scalar
   * id — including `Genre.id` from `/genres` — travels as a string. Typing this
   * `string[]` (as it was) made the compiler agree with a payload the server
   * rejects.
   */
  genre_ids: number[];

  series_id?: string;
  series_position: number;
  series_note?: string;

  chapters_count: number;
  source_chapters_count: number;

  price_per_chapter: number;
  free_until_chapter: number;
  sell_by_arc: boolean;
  tips_enabled: boolean;
  early_access_hours: number;
  release_schedule?: ReleaseSchedule;

  cover_style: CoverStyle;
  cover_color?: string;
  cover_text?: string;
}

/** รอบปล่อยบทใหม่ — display-only metadata; it does not drive the scheduler. */
export type ReleaseSchedule = "irregular" | "daily" | "weekly" | "biweekly" | "monthly";

/**
 * A novel patch. Every settings field is optional so an omitted key means
 * "leave it alone" — and because they are sent explicitly, `false` and `0`
 * still apply. Sending `series_id: null` removes the novel from its series.
 */
export interface WriterNovelPatch {
  slug?: string;
  title_th?: string;
  title_cn?: string;
  author_name?: string;
  description?: string;
  status?: NovelStatus;
  /**
   * Numbers, not strings — see `WriterNovel.genre_ids`.
   *
   * Omit the key to leave the novel's genres alone; the server replaces them
   * only when the field is present. An empty array is therefore "remove every
   * genre", not "no change".
   */
  genre_ids?: number[];
  series_id?: string | null;
  series_position?: number;
  series_note?: string;
  source_chapters_count?: number;
  price_per_chapter?: number;
  free_until_chapter?: number;
  sell_by_arc?: boolean;
  tips_enabled?: boolean;
  early_access_hours?: number;
  release_schedule?: ReleaseSchedule;
  cover_style?: CoverStyle;
  cover_color?: string;
  cover_text?: string;
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
  series: { day: string; reads: number; coins_earned: number; followers: number; completions: number }[];
  top_chapters: { chapter_id: string; chapter_no: number; title: string; reads: number; coins_earned: number }[];
  /** อ่านจบต่อบท — completions ÷ reads over the window. */
  completion_rate_pct: number;
}

/** One credited unlock or tip in the translator's earnings ledger. */
export interface Earning {
  id: string;
  chapter_id: string;
  gross_coins: number;
  /** Gross minus the platform fee — what the translator actually keeps. */
  net_coins: number;
  created_at: string;
}

export interface EarningsPage {
  data: Earning[];
  next_cursor?: string;
  /** What is still withdrawable once pending payouts are subtracted. */
  available_satang: number;
}

export interface Payout {
  id: string;
  amount_satang: number;
  status: "requested" | "approved" | "paid" | "rejected";
  requested_at: string;
}

export interface Notification {
  id: string;
  kind: string;
  payload: Record<string, unknown>;
  read: boolean;
  created_at: string;
}

/** One book's slot in a series' reading order. */
export interface SeriesBook extends NovelListItem {
  position: number;
  note?: string;
  /** The book's ภาค, loaded with the series rather than per book. */
  arcs: Arc[];
}

export interface SeriesDetail {
  id: string;
  slug: string;
  title: string;
  description?: string;
  cover_url?: string;
  books: SeriesBook[];
  /** Summed across the visible books. */
  chapters_count: number;
  source_chapters_count: number;
  arcs_count: number;
}

/**
 * How much of a series the reader follows.
 *
 * Three-valued rather than a boolean because ติดตามทั้งชุด fans out over
 * per-novel follows: a reader can follow some books and not others, and a book
 * joining later does not follow itself.
 */
export interface SeriesFollow {
  state: "none" | "partial" | "all";
  total: number;
  following: number;
}

/** The five kinds of เรื่องเกี่ยวเนื่อง. */
export type RelationKind = "sequel" | "prequel" | "spinoff" | "side_story" | "same_world";

export interface RelatedNovel extends NovelListItem {
  kind: RelationKind;
  kind_label: string;
  note?: string;
}

/** A quote for buying a whole arc, before the reader commits. */
export interface ArcBundle {
  arc_id: string;
  novel_id: string;
  arc_no: number;
  name: string;
  chapter_count: number;
  gross: number;
  discount_percent: number;
  discount: number;
  total: number;
  chapters: { chapter_id: string; chapter_no: number; list_price: number; coins: number }[];
}

/** One auto-unlock opt-in. */
export interface AutoUnlock {
  novel_id: string;
  novel_title_th?: string;
  novel_slug?: string;
  active: boolean;
  max_coins_per_chapter: number;
}

/** A series as its owner manages it. */
export interface WriterSeries {
  id: string;
  slug: string;
  title: string;
  description?: string;
  cover_url?: string;
  book_count: number;
}

export interface WriterSeriesBook {
  novel_id: string;
  position: number;
  note?: string;
  slug: string;
  title_th: string;
  cover_url?: string;
  cover_style: CoverStyle;
  cover_color?: string;
  cover_text?: string;
  status: NovelStatus;
  chapters_count: number;
  source_chapters_count: number;
}

export interface WriterRelation {
  novel_id: string;
  related_novel_id: string;
  kind: RelationKind;
  kind_label: string;
  note?: string;
  sort_no: number;
  /** Declared on the other novel; shown here but not editable. */
  mirrored: boolean;
  slug: string;
  title_th: string;
  cover_url?: string;
  cover_style: CoverStyle;
  cover_color?: string;
  cover_text?: string;
  novel_status: NovelStatus;
}

export interface WriterArc {
  id: string;
  novel_id: string;
  arc_no: number;
  name: string;
  from_chapter_no: number;
  to_chapter_no: number;
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
  getSeries: (idOrSlug: string) => request<SeriesDetail>(`/series/${encodeURIComponent(idOrSlug)}`),
  listRelated: (novelId: string) => request<Paged<RelatedNovel>>(`/novels/${novelId}/related`),

  // Reading
  getChapter: (id: string) => request<ChapterView>(`/chapters/${id}`),
  // `completed` marks a read that reached the end of the chapter; it is what
  // the writer's อ่านจบต่อบท KPI counts.
  readEvent: (id: string, completed = false) =>
    request<void>(`/chapters/${id}/read-event`, { method: "POST", body: { completed } }),
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

  // ติดตามทั้งชุด — a fan-out over the per-novel follows above, which is why
  // the state is none / partial / all rather than a boolean.
  seriesFollowState: (seriesId: string) =>
    request<SeriesFollow>(`/series/${encodeURIComponent(seriesId)}/follow`),
  followSeries: (seriesId: string) =>
    request<SeriesFollow>(`/series/${encodeURIComponent(seriesId)}/follow`, { method: "POST" }),
  unfollowSeries: (seriesId: string) =>
    request<SeriesFollow>(`/series/${encodeURIComponent(seriesId)}/follow`, { method: "DELETE" }),

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
  quoteArcBundle: (arcId: string) => request<ArcBundle>(`/arcs/${arcId}/bundle`),
  unlockArc: (arcId: string, key: string) =>
    request<Receipt>(`/arcs/${arcId}/unlock`, { method: "POST", idempotencyKey: key }),
  tipChapter: (chapterId: string, coins: number, key: string) =>
    request<Receipt>(`/chapters/${chapterId}/tip`, {
      method: "POST",
      body: { coins },
      idempotencyKey: key,
    }),

  // Auto-unlock subscriptions
  listAutoUnlock: () => request<Paged<AutoUnlock>>("/me/auto-unlock"),
  setAutoUnlock: (novelId: string, body: { active: boolean; max_coins_per_chapter: number }) =>
    request<AutoUnlock>(`/me/auto-unlock/${novelId}`, { method: "PUT", body }),
  removeAutoUnlock: (novelId: string) =>
    request<void>(`/me/auto-unlock/${novelId}`, { method: "DELETE" }),

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
  // The writer lists below ask for the server's maximum page size rather than
  // letting the smaller default apply. None of these screens has a "load more",
  // so anything past the first page is simply unreachable — a translator with
  // 21 works could not open the 21st at all. Past the maximum this still
  // truncates; that needs real cursor paging in the UI.
  listWriterNovels: (limit = 100) =>
    request<Paged<WriterNovel>>(`/writer/novels${qs({ limit })}`),
  createWriterNovel: (body: WriterNovelPatch) =>
    request<WriterNovel>("/writer/novels", { method: "POST", body }),
  updateWriterNovel: (id: string, body: WriterNovelPatch) =>
    request<WriterNovel>(`/writer/novels/${id}`, { method: "PATCH", body }),
  uploadCover: (id: string, file: File) => uploadCover(id, file),

  // Series and related works
  listWriterSeries: () => request<Paged<WriterSeries>>("/writer/series"),
  createWriterSeries: (body: { title: string; description?: string }) =>
    request<WriterSeries>("/writer/series", { method: "POST", body }),
  updateWriterSeries: (id: string, body: { title?: string; description?: string }) =>
    request<WriterSeries>(`/writer/series/${id}`, { method: "PATCH", body }),
  deleteWriterSeries: (id: string) => request<void>(`/writer/series/${id}`, { method: "DELETE" }),
  listWriterSeriesBooks: (id: string) =>
    request<Paged<WriterSeriesBook>>(`/writer/series/${id}/books`),
  reorderWriterSeries: (id: string, novelIds: string[]) =>
    request<Paged<WriterSeriesBook>>(`/writer/series/${id}/order`, {
      method: "PUT",
      body: { novel_ids: novelIds },
    }),
  setSeriesNote: (novelId: string, note: string) =>
    request<void>(`/writer/novels/${novelId}/series-note`, { method: "PUT", body: { note } }),

  listWriterRelations: (novelId: string) =>
    request<Paged<WriterRelation>>(`/writer/novels/${novelId}/relations`),
  linkNovels: (
    novelId: string,
    body: { related_novel_id: string; kind: RelationKind; note?: string; sort_no?: number },
  ) => request<WriterRelation>(`/writer/novels/${novelId}/relations`, { method: "POST", body }),
  unlinkNovels: (novelId: string, relatedNovelId: string) =>
    request<void>(`/writer/novels/${novelId}/relations/${relatedNovelId}`, { method: "DELETE" }),

  listWriterArcs: (novelId: string) => request<Paged<WriterArc>>(`/writer/novels/${novelId}/arcs`),
  createWriterArc: (
    novelId: string,
    body: { arc_no: number; name: string; from_chapter_no: number; to_chapter_no: number },
  ) => request<WriterArc>(`/writer/novels/${novelId}/arcs`, { method: "POST", body }),
  updateWriterArc: (
    id: string,
    body: { arc_no: number; name: string; from_chapter_no: number; to_chapter_no: number },
  ) => request<WriterArc>(`/writer/arcs/${id}`, { method: "PATCH", body }),
  // Newest-first, and the editor's chapter rail searches this list client-side,
  // so a short page would put older chapters out of reach entirely.
  listWriterChapters: (novelId: string, limit = 500) =>
    request<Paged<WriterChapter>>(`/writer/novels/${novelId}/chapters${qs({ limit })}`),
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
  createGlossaryGroup: (novelId: string, name: string, sortNo = 0) =>
    request<GlossaryGroup>(`/writer/novels/${novelId}/glossary`, {
      method: "POST",
      body: { name, sort_no: sortNo },
    }),
  createGlossaryEntry: (
    novelId: string,
    body: { group_id: string; term_key: string; title_th: string; title_cn?: string; body: string; kind?: string },
  ) => request<GlossaryEntry>(`/writer/novels/${novelId}/glossary`, { method: "POST", body }),
  updateGlossaryEntry: (
    id: string,
    body: { term_key?: string; title_th?: string; title_cn?: string; body?: string; kind?: string },
  ) => request<GlossaryEntry>(`/writer/glossary-entries/${id}`, { method: "PATCH", body }),
  // Deleting a term bumps the novel's glossary_rev, so the re-render worker
  // rewrites every chapter that bound it; the marker survives as plain text.
  deleteGlossaryEntry: (id: string) =>
    request<void>(`/writer/glossary-entries/${id}`, { method: "DELETE" }),
  // Refused with 409 GROUP_NOT_EMPTY while the group still holds terms.
  deleteGlossaryGroup: (id: string) =>
    request<void>(`/writer/glossary-groups/${id}`, { method: "DELETE" }),

  getStats: (novelId: string, period = "14d") =>
    request<Stats>(`/writer/stats/novels/${novelId}${qs({ period })}`),

  // Earnings and payouts
  listEarnings: (cursor?: string) => request<EarningsPage>(`/writer/earnings${qs({ cursor })}`),
  requestPayout: (amountSatang: number) =>
    request<Payout>("/writer/payouts", { method: "POST", body: { amount_satang: amountSatang } }),

  // Notifications
  listNotifications: () => request<Paged<Notification>>("/me/notifications"),
  unreadCount: () => request<{ unread: number }>("/me/notifications/unread-count"),
  markNotificationsRead: (ids: number[] = []) =>
    request<{ unread: number }>("/me/notifications/read", { method: "POST", body: { ids } }),
};
