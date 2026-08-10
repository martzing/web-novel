import { qs, request } from "@/shared/api/client";
import type {
  CoverStyle,
  GlossaryGroup,
  NovelStatus,
  Paged,
  RelationKind,
  ReleaseSchedule,
} from "@/shared/api/types";

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

export interface RelatedNovel extends NovelListItem {
  kind: RelationKind;
  kind_label: string;
  note?: string;
}

export const catalogApi = {
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
};
