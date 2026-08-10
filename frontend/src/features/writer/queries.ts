import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { writerApi, type WriterNovelPatch } from "./api";

/**
 * Every cache key the writer workspace owns.
 *
 * This factory is the fix for a real defect: the series tab invalidated
 * `["writer-series-books", seriesId]` in one place and `["writer-series-books"]`
 * in another, so a note saved from the second path never refreshed the list.
 * Keys built here cannot drift, and `writerKeys.all` is a prefix of all of
 * them.
 */
export const writerKeys = {
  all: ["writer"] as const,
  novels: () => [...writerKeys.all, "novels"] as const,
  series: () => [...writerKeys.all, "series"] as const,
  seriesBooks: () => [...writerKeys.all, "series-books"] as const,
  seriesBooksOf: (seriesId: string) => [...writerKeys.seriesBooks(), seriesId] as const,
  relations: (novelId: string) => [...writerKeys.all, "relations", novelId] as const,
  arcs: (novelId: string) => [...writerKeys.all, "arcs", novelId] as const,
  chapters: (novelId: string) => [...writerKeys.all, "chapters", novelId] as const,
  chapter: (chapterId: string) => [...writerKeys.all, "chapter", chapterId] as const,
  glossary: (novelId: string) => [...writerKeys.all, "glossary", novelId] as const,
  stats: (novelId: string, period: string) =>
    [...writerKeys.all, "stats", novelId, period] as const,
  earnings: () => [...writerKeys.all, "earnings"] as const,
};

export function useWriterNovels(enabled = true) {
  return useQuery({
    queryKey: writerKeys.novels(),
    queryFn: () => writerApi.listWriterNovels(),
    enabled,
  });
}

export function useWriterSeries(enabled = true) {
  return useQuery({
    queryKey: writerKeys.series(),
    queryFn: writerApi.listWriterSeries,
    enabled,
  });
}

export function useWriterSeriesBooks(seriesId: string | undefined) {
  return useQuery({
    queryKey: writerKeys.seriesBooksOf(seriesId ?? ""),
    queryFn: () => writerApi.listWriterSeriesBooks(seriesId!),
    enabled: Boolean(seriesId),
  });
}

export function useWriterRelations(novelId: string) {
  return useQuery({
    queryKey: writerKeys.relations(novelId),
    queryFn: () => writerApi.listWriterRelations(novelId),
    enabled: Boolean(novelId),
  });
}

export function useWriterArcs(novelId: string) {
  return useQuery({
    queryKey: writerKeys.arcs(novelId),
    queryFn: () => writerApi.listWriterArcs(novelId),
    enabled: Boolean(novelId),
  });
}

export function useWriterChapters(novelId: string | undefined) {
  return useQuery({
    queryKey: writerKeys.chapters(novelId ?? ""),
    queryFn: () => writerApi.listWriterChapters(novelId!),
    enabled: Boolean(novelId),
  });
}

export function useWriterChapter(chapterId: string | undefined) {
  return useQuery({
    queryKey: writerKeys.chapter(chapterId ?? ""),
    queryFn: () => writerApi.getWriterChapter(chapterId!),
    enabled: Boolean(chapterId),
  });
}

export function useWriterGlossary(novelId: string) {
  return useQuery({
    queryKey: writerKeys.glossary(novelId),
    queryFn: () => writerApi.listWriterGlossary(novelId),
    enabled: Boolean(novelId),
  });
}

export function useWriterStats(novelId: string | undefined, period: string) {
  return useQuery({
    queryKey: writerKeys.stats(novelId ?? "", period),
    queryFn: () => writerApi.getStats(novelId!, period),
    enabled: Boolean(novelId),
  });
}

export function useEarnings() {
  return useQuery({ queryKey: writerKeys.earnings(), queryFn: () => writerApi.listEarnings() });
}

/** Saves a patch and keeps the work tree and detail pane in step. */
export function useSaveNovel(novelId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (patch: WriterNovelPatch) => writerApi.updateWriterNovel(novelId, patch),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: writerKeys.novels() });
      qc.invalidateQueries({ queryKey: writerKeys.series() });
    },
  });
}
