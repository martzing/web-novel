import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { notificationApi } from "./api";

export const notificationKeys = {
  all: ["notifications"] as const,
  unread: () => [...notificationKeys.all, "unread"] as const,
  list: () => [...notificationKeys.all, "list"] as const,
};

export function useUnreadCount(enabled = true, refetchInterval?: number) {
  return useQuery({
    queryKey: notificationKeys.unread(),
    queryFn: notificationApi.unreadCount,
    enabled,
    staleTime: 60_000,
    refetchInterval,
  });
}

export function useNotifications(enabled = true) {
  return useQuery({
    queryKey: notificationKeys.list(),
    queryFn: notificationApi.listNotifications,
    enabled,
  });
}

export function useMarkNotificationsRead() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (ids?: number[]) => notificationApi.markNotificationsRead(ids),
    onSuccess: () => qc.invalidateQueries({ queryKey: notificationKeys.all }),
  });
}
