import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { readingApi, type Progress } from "./api";

export const readingKeys = {
  all: ["reading"] as const,
  chapter: (id: string) => [...readingKeys.all, "chapter", id] as const,
  progress: (novelId: string) => [...readingKeys.all, "progress", novelId] as const,
};

export function useChapter(id: string) {
  return useQuery({
    queryKey: readingKeys.chapter(id),
    queryFn: () => readingApi.getChapter(id),
    enabled: Boolean(id),
  });
}

export function useProgress(novelId: string | undefined) {
  return useQuery({
    queryKey: readingKeys.progress(novelId ?? ""),
    queryFn: () => readingApi.getProgress(novelId!),
    enabled: Boolean(novelId),
    // A reader with no saved position gets a 404; that is an answer, not a fault.
    retry: false,
  });
}

export function useSaveProgress(novelId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: Partial<Progress>) => readingApi.saveProgress(novelId, body),
    onSuccess: () => qc.invalidateQueries({ queryKey: readingKeys.progress(novelId) }),
  });
}
