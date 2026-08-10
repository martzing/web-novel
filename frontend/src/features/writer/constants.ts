import type { NovelStatus, RelationKind, ReleaseSchedule } from "@/shared/api/types";

export type WorkTab = "info" | "cover" | "chapters" | "glossary" | "series" | "pricing";

// The design draws five tabs. อภิธานศัพท์ is the sixth because W-05 promises
// full glossary management and the editor's {{term_key}} binding is useless
// without somewhere to define the terms. It sits after ภาคและบท, next to the
// other per-chapter work.
export const TABS: { key: WorkTab; label: string }[] = [
  { key: "info", label: "ข้อมูลเรื่อง" },
  { key: "cover", label: "หน้าปก" },
  { key: "chapters", label: "ภาคและบท" },
  { key: "glossary", label: "อภิธานศัพท์" },
  { key: "series", label: "ชุดและเรื่องเกี่ยวเนื่อง" },
  { key: "pricing", label: "ราคาและการเผยแพร่" },
];

export const STATUSES: { key: NovelStatus; label: string }[] = [
  { key: "ongoing", label: "เผยแพร่ กำลังแปล" },
  { key: "complete", label: "เผยแพร่ จบแล้ว" },
  { key: "hiatus", label: "พักการแปลชั่วคราว" },
  { key: "hidden", label: "ซ่อนจากหน้าร้าน" },
];

export const SCHEDULES: { key: ReleaseSchedule; label: string }[] = [
  { key: "irregular", label: "ไม่กำหนด" },
  { key: "daily", label: "ทุกวัน" },
  { key: "weekly", label: "สัปดาห์ละ 1 บท" },
  { key: "biweekly", label: "สัปดาห์ละ 2 บท" },
  { key: "monthly", label: "เดือนละครั้ง" },
];

export const RELATION_KINDS: { key: RelationKind; label: string }[] = [
  { key: "sequel", label: "ภาคต่อโดยตรง" },
  { key: "prequel", label: "ปฐมบท" },
  { key: "spinoff", label: "ภาคแยก" },
  { key: "side_story", label: "ภาคพิเศษ" },
  { key: "same_world", label: "เกิดในโลกเดียวกัน" },
];

export function statusLabel(status: NovelStatus): string {
  return STATUSES.find((s) => s.key === status)?.label ?? status;
}
