import { qs, request } from "@/shared/api/client";
import type { Paged } from "@/shared/api/types";

export interface CommentAuthor {
  id: string;
  display_name: string;
  avatar_url?: string;
  role: string;
}

export interface Comment {
  id: string;
  chapter_id: string;
  parent_id?: string;
  body: string;
  is_spoiler_hidden: boolean;
  likes_count: number;
  liked: boolean;
  is_translator: boolean;
  created_at: string;
  author: CommentAuthor;
  replies: Comment[];
}

export interface Review {
  id: string;
  novel_id: string;
  rating: number;
  body?: string;
  created_at: string;
  author: CommentAuthor;
}

/**
 * How much of a series the reader follows.
 *
 * Three-valued rather than a boolean because ติดตามทั้งชุด fans out over
 * per-novel follows: a reader can follow some books and not others, and a book
 * joining later does not follow itself.
 */
export interface SeriesFollow {
  state: "none" | "partial" | "all";
  total: number;
  following: number;
}

export const socialApi = {
  listComments: (chapterId: string, sort = "popular") =>
    request<Paged<Comment>>(`/chapters/${chapterId}/comments${qs({ sort })}`),
  createComment: (chapterId: string, body: { body: string; parent_id?: string; is_spoiler_hidden?: boolean }) =>
    request<Comment>(`/chapters/${chapterId}/comments`, { method: "POST", body }),
  likeComment: (id: string) =>
    request<{ likes_count: number; liked: boolean }>(`/comments/${id}/like`, { method: "POST" }),
  unlikeComment: (id: string) =>
    request<{ likes_count: number; liked: boolean }>(`/comments/${id}/like`, { method: "DELETE" }),
  deleteComment: (id: string) => request<void>(`/comments/${id}`, { method: "DELETE" }),

  listReviews: (novelId: string) =>
    request<Paged<Review> & { my_review?: Review }>(`/novels/${novelId}/reviews`),
  upsertReview: (novelId: string, body: { rating: number; body?: string }) =>
    request<Review>(`/novels/${novelId}/reviews`, { method: "POST", body }),

  isFollowing: (novelId: string) => request<{ following: boolean }>(`/me/follows/${novelId}`),
  follow: (novelId: string) =>
    request<{ following: boolean }>(`/me/follows/${novelId}`, { method: "POST" }),
  unfollow: (novelId: string) =>
    request<{ following: boolean }>(`/me/follows/${novelId}`, { method: "DELETE" }),

  // ติดตามทั้งชุด — a fan-out over the per-novel follows above, which is why
  // the state is none / partial / all rather than a boolean.
  seriesFollowState: (seriesId: string) =>
    request<SeriesFollow>(`/series/${encodeURIComponent(seriesId)}/follow`),
  followSeries: (seriesId: string) =>
    request<SeriesFollow>(`/series/${encodeURIComponent(seriesId)}/follow`, { method: "POST" }),
  unfollowSeries: (seriesId: string) =>
    request<SeriesFollow>(`/series/${encodeURIComponent(seriesId)}/follow`, { method: "DELETE" }),
};
