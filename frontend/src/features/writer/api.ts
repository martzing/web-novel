import { qs, request, upload } from "@/shared/api/client";
import type {
  CoverStyle,
  GlossaryEntry,
  GlossaryGroup,
  NovelStatus,
  Paged,
  RelationKind,
  ReleaseSchedule,
} from "@/shared/api/types";

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

export const writerApi = {
  // The writer lists below ask for the server's maximum page size rather than
  // letting the smaller default apply. None of these screens has a "load more",
  // so anything past the first page is simply unreachable — a translator with
  // 21 works could not open the 21st at all. Past the maximum this still
  // truncates; that needs real cursor paging in the UI.
  listWriterNovels: (limit = 100) => request<Paged<WriterNovel>>(`/writer/novels${qs({ limit })}`),
  createWriterNovel: (body: WriterNovelPatch) =>
    request<WriterNovel>("/writer/novels", { method: "POST", body }),
  updateWriterNovel: (id: string, body: WriterNovelPatch) =>
    request<WriterNovel>(`/writer/novels/${id}`, { method: "PATCH", body }),
  uploadCover: (id: string, file: File) =>
    upload<{ cover_url: string }>(`/writer/novels/${id}/cover`, "cover", file),

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

  listEarnings: (cursor?: string) => request<EarningsPage>(`/writer/earnings${qs({ cursor })}`),
  requestPayout: (amountSatang: number) =>
    request<Payout>("/writer/payouts", { method: "POST", body: { amount_satang: amountSatang } }),
};
