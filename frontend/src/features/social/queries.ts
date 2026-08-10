import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { socialApi } from "./api";

export const socialKeys = {
  all: ["social"] as const,
  comments: (chapterId: string) => [...socialKeys.all, "comments", chapterId] as const,
  commentList: (chapterId: string, sort: string) =>
    [...socialKeys.comments(chapterId), sort] as const,
  reviews: (novelId: string) => [...socialKeys.all, "reviews", novelId] as const,
  follows: () => [...socialKeys.all, "follows"] as const,
  following: (novelId: string) => [...socialKeys.follows(), novelId] as const,
  seriesFollow: (seriesId: string) => [...socialKeys.all, "series-follow", seriesId] as const,
};

export function useComments(chapterId: string, sort: string) {
  return useQuery({
    queryKey: socialKeys.commentList(chapterId, sort),
    queryFn: () => socialApi.listComments(chapterId, sort),
    enabled: Boolean(chapterId),
  });
}

export function useCreateComment(chapterId: string, onDone?: () => void) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: { body: string; parent_id?: string; is_spoiler_hidden?: boolean }) =>
      socialApi.createComment(chapterId, body),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: socialKeys.comments(chapterId) });
      onDone?.();
    },
  });
}

export function useToggleCommentLike(chapterId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ commentId, liked }: { commentId: string; liked: boolean }) =>
      liked ? socialApi.unlikeComment(commentId) : socialApi.likeComment(commentId),
    onSuccess: () => qc.invalidateQueries({ queryKey: socialKeys.comments(chapterId) }),
  });
}

export function useDeleteComment(chapterId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (commentId: string) => socialApi.deleteComment(commentId),
    onSuccess: () => qc.invalidateQueries({ queryKey: socialKeys.comments(chapterId) }),
  });
}

export function useReviews(novelId: string | undefined) {
  return useQuery({
    queryKey: socialKeys.reviews(novelId ?? ""),
    queryFn: () => socialApi.listReviews(novelId!),
    enabled: Boolean(novelId),
  });
}

export function useUpsertReview(novelId: string, onDone?: () => void) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: { rating: number; body?: string }) => socialApi.upsertReview(novelId, body),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: socialKeys.reviews(novelId) });
      onDone?.();
    },
  });
}

export function useIsFollowing(novelId: string | undefined, enabled = true) {
  return useQuery({
    queryKey: socialKeys.following(novelId ?? ""),
    queryFn: () => socialApi.isFollowing(novelId!),
    enabled: Boolean(novelId) && enabled,
  });
}

export function useToggleFollow(novelId: string, onDone?: () => void) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (following: boolean) =>
      following ? socialApi.unfollow(novelId) : socialApi.follow(novelId),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: socialKeys.following(novelId) });
      onDone?.();
    },
  });
}

export function useSeriesFollow(seriesId: string | undefined, enabled = true) {
  return useQuery({
    queryKey: socialKeys.seriesFollow(seriesId ?? ""),
    queryFn: () => socialApi.seriesFollowState(seriesId!),
    enabled: Boolean(seriesId) && enabled,
  });
}

/**
 * ติดตามทั้งชุด fans out over the per-novel follows, so this invalidates every
 * `following` key as well as the series' own state.
 */
export function useToggleSeriesFollow(seriesId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (followAll: boolean) =>
      followAll ? socialApi.followSeries(seriesId) : socialApi.unfollowSeries(seriesId),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: socialKeys.seriesFollow(seriesId) });
      qc.invalidateQueries({ queryKey: socialKeys.follows() });
    },
  });
}
