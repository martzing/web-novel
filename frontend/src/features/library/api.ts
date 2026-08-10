import { qs, request } from "@/shared/api/client";
import type { CoverStyle, Paged } from "@/shared/api/types";

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

export const libraryApi = {
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
};
