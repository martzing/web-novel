/**
 * Wire types spoken by more than one bounded context.
 *
 * This mirrors the backend's `domain/page` and `domain/roles`: shared
 * vocabulary, deliberately small. A type only one feature uses belongs in that
 * feature's `api.ts`, not here — this file exists so that `catalog` and
 * `writer` can describe the same novel without importing each other.
 */

/** The cursor-paged envelope every list endpoint returns. */
export interface Paged<T> {
  data: T[];
  next_cursor?: string;
}

/**
 * Cover template styles, mirroring the CHECK on novels.cover_style.
 *
 * Shared because the reader renders covers and the writer workspace edits them.
 */
export type CoverStyle = "image" | "ink" | "seal" | "brush" | "plain";

/** ซ่อนจากหน้าร้าน is only ever seen by the novel's own translator. */
export type NovelStatus = "ongoing" | "complete" | "hiatus" | "hidden";

/** รอบปล่อยบทใหม่ — display-only metadata; it does not drive the scheduler. */
export type ReleaseSchedule = "irregular" | "daily" | "weekly" | "biweekly" | "monthly";

/** The five kinds of เรื่องเกี่ยวเนื่อง. */
export type RelationKind = "sequel" | "prequel" | "spinoff" | "side_story" | "same_world";

/**
 * Glossary shape, shared because the reader reads what the writer edits — one
 * server contract, two screens.
 */
export interface GlossaryEntry {
  id: string;
  term_key: string;
  title_th: string;
  title_cn?: string;
  body: string;
  kind?: string;
}

export interface GlossaryGroup {
  id: string;
  name: string;
  sort_no: number;
  entries: GlossaryEntry[];
}
