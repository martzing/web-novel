import { request } from "@/shared/api/client";
import type { Paged } from "@/shared/api/types";

export interface User {
  id: string;
  username: string;
  email: string;
  display_name: string;
  avatar_url?: string;
  roles: string[];
  status: string;
}

export interface AuthResponse {
  user: User;
  token: string;
  refresh_token: string;
  expires_at: string;
}

export interface Prefs {
  theme: "light" | "sepia" | "dark";
  font: "loop" | "serif" | "sans";
  font_size: number;
  line_height: number;
  column_width: "narrow" | "normal" | "wide";
}

export interface GenrePref {
  genre_id: string;
  weight: number;
}

export const identityApi = {
  register: (body: { username: string; email: string; password: string; display_name?: string }) =>
    request<AuthResponse>("/auth/register", { method: "POST", body, auth: false }),
  login: (body: { email: string; password: string }) =>
    request<AuthResponse>("/auth/login", { method: "POST", body, auth: false }),
  logout: (refresh_token: string | null) =>
    request<void>("/auth/logout", { method: "POST", body: { refresh_token } }),
  me: () => request<User>("/auth/me"),
  updateMe: (body: { display_name?: string; avatar_url?: string }) =>
    request<User>("/users/me", { method: "PATCH", body }),

  getPrefs: () => request<Prefs>("/users/me/prefs"),
  setPrefs: (body: Prefs) => request<Prefs>("/users/me/prefs", { method: "PUT", body }),
  getGenrePrefs: () => request<Paged<GenrePref>>("/users/me/genre-prefs"),
  setGenrePrefs: (genres: GenrePref[]) =>
    request<Paged<GenrePref>>("/users/me/genre-prefs", { method: "PUT", body: { genres } }),
};
