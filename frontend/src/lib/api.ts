const BASE = import.meta.env.VITE_API_BASE ?? "/api/v1";

async function get<T>(path: string): Promise<T> {
  const res = await fetch(`${BASE}${path}`, { credentials: "include" });
  if (!res.ok) {
    let msg = `HTTP ${res.status}`;
    try {
      const body = (await res.json()) as {
        error?: { code?: string; message?: string };
      };
      if (body?.error?.message) msg = body.error.message;
    } catch {
      /* ignore */
    }
    throw new Error(msg);
  }
  return (await res.json()) as T;
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
  cover_url?: string;
  status: "ongoing" | "complete" | "hiatus";
  rating_avg: number;
  rating_count: number;
  followers_count: number;
  chapters_count: number;
  genres: Genre[];
}

export interface Arc {
  id: string;
  arc_no: number;
  name: string;
  from_chapter_no: number;
  to_chapter_no: number;
}

export interface NovelDetail extends NovelListItem {
  author_name?: string;
  description?: string;
  arcs: Arc[];
}

export interface ChapterListItem {
  id: string;
  chapter_no: number;
  title: string;
  price_coins: number;
  published_at?: string;
  arc_id?: string;
}

export interface ChapterFull {
  id: string;
  novel_id: string;
  chapter_no: number;
  title: string;
  price_coins: number;
  locked: boolean;
  body_html?: string;
}

export const api = {
  listGenres: () => get<{ data: Genre[] }>("/genres").then((r) => r.data),
  listNovels: (
    params: { q?: string; genre?: string; sort?: "popular" | "latest" } = {},
  ) => {
    const p = new URLSearchParams();
    if (params.q) p.set("q", params.q);
    if (params.genre) p.set("genre", params.genre);
    if (params.sort) p.set("sort", params.sort);
    const qs = p.toString();
    return get<{ data: NovelListItem[] }>(`/novels${qs ? `?${qs}` : ""}`).then(
      (r) => r.data,
    );
  },
  getNovel: (slug: string) =>
    get<NovelDetail>(`/novels/${encodeURIComponent(slug)}`),
  listChapters: (novelId: string) =>
    get<{ data: ChapterListItem[] }>(`/novels/${novelId}/chapters`).then(
      (r) => r.data,
    ),
  getChapter: (id: string) => get<ChapterFull>(`/chapters/${id}`),
};
