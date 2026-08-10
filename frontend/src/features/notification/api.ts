import { request } from "@/shared/api/client";
import type { Paged } from "@/shared/api/types";

export interface Notification {
  id: string;
  kind: string;
  payload: Record<string, unknown>;
  read: boolean;
  created_at: string;
}

export const notificationApi = {
  listNotifications: () => request<Paged<Notification>>("/me/notifications"),
  unreadCount: () => request<{ unread: number }>("/me/notifications/unread-count"),
  markNotificationsRead: (ids: number[] = []) =>
    request<{ unread: number }>("/me/notifications/read", { method: "POST", body: { ids } }),
};
