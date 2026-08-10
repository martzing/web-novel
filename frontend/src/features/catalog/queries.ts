import { keepPreviousData, useQuery } from "@tanstack/react-query";

import { catalogApi } from "./api";

export interface NovelListParams {
  q?: string;
  genre?: string;
  sort?: string;
  limit?: number;
}

/** Every cache key the catalogue owns. See `walletKeys` for the rationale. */
export const catalogKeys = {
  all: ["catalog"] as const,
  genres: () => [...catalogKeys.all, "genres"] as const,
  novels: () => [...catalogKeys.all, "novels"] as const,
  novelList: (params: NovelListParams) => [...catalogKeys.novels(), params] as const,
  novel: () => [...catalogKeys.all, "novel"] as const,
  novelDetail: (idOrSlug: string) => [...catalogKeys.novel(), idOrSlug] as const,
  chapters: () => [...catalogKeys.all, "chapters"] as const,
  chapterList: (novelId: string) => [...catalogKeys.chapters(), novelId] as const,
  arcs: (novelId: string) => [...catalogKeys.all, "arcs", novelId] as const,
  glossary: (novelId: string) => [...catalogKeys.all, "glossary", novelId] as const,
  ranking: (limit: number) => [...catalogKeys.all, "ranking", "weekly", limit] as const,
  series: (idOrSlug: string) => [...catalogKeys.all, "series", idOrSlug] as const,
  related: (novelId: string) => [...catalogKeys.all, "related", novelId] as const,
};

export function useGenres() {
  return useQuery({
    queryKey: catalogKeys.genres(),
    queryFn: catalogApi.listGenres,
    // The genre list changes at most when a migration runs.
    staleTime: 10 * 60_000,
  });
}

export function useNovels(params: NovelListParams) {
  return useQuery({
    queryKey: catalogKeys.novelList(params),
    queryFn: () => catalogApi.listNovels(params),
    // Keeping the previous page visible avoids a flash of empty state while a
    // new filter loads.
    placeholderData: keepPreviousData,
  });
}

export function useNovel(idOrSlug: string) {
  return useQuery({
    queryKey: catalogKeys.novelDetail(idOrSlug),
    queryFn: () => catalogApi.getNovel(idOrSlug),
    enabled: Boolean(idOrSlug),
  });
}

export function useChapters(novelId: string | undefined) {
  return useQuery({
    queryKey: catalogKeys.chapterList(novelId ?? ""),
    queryFn: () => catalogApi.listChapters(novelId!),
    enabled: Boolean(novelId),
  });
}

export function useGlossary(novelId: string | undefined) {
  return useQuery({
    queryKey: catalogKeys.glossary(novelId ?? ""),
    queryFn: () => catalogApi.getGlossary(novelId!),
    enabled: Boolean(novelId),
  });
}

export function useWeeklyRanking(limit = 5) {
  return useQuery({
    queryKey: catalogKeys.ranking(limit),
    queryFn: () => catalogApi.weeklyRanking(limit),
  });
}

export function useSeries(idOrSlug: string | undefined) {
  return useQuery({
    queryKey: catalogKeys.series(idOrSlug ?? ""),
    queryFn: () => catalogApi.getSeries(idOrSlug!),
    enabled: Boolean(idOrSlug),
  });
}

export function useRelatedNovels(novelId: string | undefined) {
  return useQuery({
    queryKey: catalogKeys.related(novelId ?? ""),
    queryFn: () => catalogApi.listRelated(novelId!),
    enabled: Boolean(novelId),
  });
}
