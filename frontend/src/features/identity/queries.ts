import { useMutation, useQuery } from "@tanstack/react-query";

import { identityApi, type GenrePref } from "./api";

export const identityKeys = {
  all: ["identity"] as const,
  genrePrefs: () => [...identityKeys.all, "genre-prefs"] as const,
};

export function useGenrePrefs(enabled = true) {
  return useQuery({
    queryKey: identityKeys.genrePrefs(),
    queryFn: identityApi.getGenrePrefs,
    enabled,
  });
}

export function useSaveGenrePrefs(onDone?: () => void) {
  return useMutation({
    mutationFn: (genres: GenrePref[]) => identityApi.setGenrePrefs(genres),
    onSuccess: () => onDone?.(),
  });
}
