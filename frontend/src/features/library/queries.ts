import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { libraryApi } from "./api";

export const libraryKeys = {
  all: ["library"] as const,
  shelf: () => [...libraryKeys.all, "shelf"] as const,
  /** No tab means the counts-only read the shell and the novel page make. */
  shelfTab: (tab = "counts") => [...libraryKeys.shelf(), tab] as const,
  bookmarks: (novelId: string) => [...libraryKeys.all, "bookmarks", novelId] as const,
};

export function useShelf(tab?: string, enabled = true) {
  return useQuery({
    queryKey: libraryKeys.shelfTab(tab),
    queryFn: () => libraryApi.getShelf(tab),
    enabled,
  });
}

/** The shell's badge only needs the counts, and re-reads them less often. */
export function useShelfCounts(enabled = true) {
  return useQuery({
    queryKey: libraryKeys.shelfTab(),
    queryFn: () => libraryApi.getShelf(),
    enabled,
    staleTime: 60_000,
  });
}

export function useSetShelfStatus(novelId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (status: string) => libraryApi.setShelfStatus(novelId, status),
    onSuccess: () => qc.invalidateQueries({ queryKey: libraryKeys.shelf() }),
  });
}

export function useRemoveFromShelf() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (novelId: string) => libraryApi.removeFromShelf(novelId),
    onSuccess: () => qc.invalidateQueries({ queryKey: libraryKeys.shelf() }),
  });
}

export function useBookmarks(novelId: string | undefined, enabled = true) {
  return useQuery({
    queryKey: libraryKeys.bookmarks(novelId ?? ""),
    queryFn: () => libraryApi.listBookmarks(novelId!),
    enabled: Boolean(novelId) && enabled,
  });
}

export function useCreateBookmark(novelId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: {
      novel_id: string;
      chapter_id: string;
      para_anchor: number;
      excerpt: string;
      note?: string;
    }) => libraryApi.createBookmark(body),
    onSuccess: () => qc.invalidateQueries({ queryKey: libraryKeys.bookmarks(novelId) }),
  });
}

export function useDeleteBookmark(novelId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => libraryApi.deleteBookmark(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: libraryKeys.bookmarks(novelId) }),
  });
}
