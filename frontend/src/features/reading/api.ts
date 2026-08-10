import { request } from "@/shared/api/client";

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

export interface Progress {
  novel_id: string;
  last_chapter_id?: string;
  last_chapter_no?: number;
  para_anchor: number;
  pct: number;
}

export const readingApi = {
  getChapter: (id: string) => request<ChapterView>(`/chapters/${id}`),
  // `completed` marks a read that reached the end of the chapter; it is what
  // the writer's อ่านจบต่อบท KPI counts.
  readEvent: (id: string, completed = false) =>
    request<void>(`/chapters/${id}/read-event`, { method: "POST", body: { completed } }),
  getProgress: (novelId: string) => request<Progress>(`/me/progress/${novelId}`),
  saveProgress: (novelId: string, body: Partial<Progress>) =>
    request<Progress>(`/me/progress/${novelId}`, { method: "PUT", body }),
};
