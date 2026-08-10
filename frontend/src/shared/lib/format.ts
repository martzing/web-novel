/** Thai-first formatting helpers shared across screens. */

const THAI_MONTHS = [
  "ม.ค.", "ก.พ.", "มี.ค.", "เม.ย.", "พ.ค.", "มิ.ย.",
  "ก.ค.", "ส.ค.", "ก.ย.", "ต.ค.", "พ.ย.", "ธ.ค.",
];

/** Formats a date as "8 ส.ค. 2569" in the Buddhist era Thai readers expect. */
export function thaiDate(iso?: string): string {
  if (!iso) return "";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "";
  return `${d.getDate()} ${THAI_MONTHS[d.getMonth()]} ${d.getFullYear() + 543}`;
}

/** Coarse relative time, matching the mockups' phrasing. */
export function relativeTime(iso?: string): string {
  if (!iso) return "";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "";

  const minutes = Math.floor((Date.now() - d.getTime()) / 60000);
  if (minutes < 2) return "เมื่อสักครู่";
  if (minutes < 60) return `${minutes} นาทีที่แล้ว`;

  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours} ชั่วโมงที่แล้ว`;

  const days = Math.floor(hours / 24);
  if (days === 1) return "เมื่อวาน";
  if (days < 7) return `${days} วันที่แล้ว`;
  if (days < 30) return `${Math.floor(days / 7)} สัปดาห์ที่แล้ว`;
  return thaiDate(iso);
}

export function numberTH(value: number): string {
  return value.toLocaleString("th-TH");
}

/** Prices are stored in satang; display them as baht. */
export function baht(satang: number): string {
  return `฿${(satang / 100).toLocaleString("th-TH", { maximumFractionDigits: 0 })}`;
}

export function percent(value: number): string {
  return `${Math.round(value)}%`;
}

export function trend(pct: number): string {
  const sign = pct >= 0 ? "+" : "";
  return `${sign}${pct.toFixed(1)}%`;
}

/** Human-readable Thai label for each coin_ledger kind. */
export function ledgerLabel(kind: string): string {
  switch (kind) {
    case "topup":
      return "เติมเหรียญ";
    case "spend_unlock":
      return "ปลดล็อกบท";
    case "refund":
      return "คืนเหรียญ";
    case "bonus_grant":
      return "โบนัส";
    case "bonus_expire":
      return "โบนัสหมดอายุ";
    case "adjust":
      return "ปรับยอดโดยผู้ดูแล";
    default:
      return kind;
  }
}

export function chapterStatusLabel(status: string): string {
  switch (status) {
    case "draft":
      return "ร่าง";
    case "scheduled":
      return "ตั้งเวลาไว้";
    case "published":
      return "เผยแพร่แล้ว";
    default:
      return status;
  }
}

/**
 * รอบปล่อยบทใหม่, as a reader-facing label.
 *
 * Shared rather than duplicated: the translator picks the cadence in
 * จัดการผลงาน and the reader sees it on the novel's detail page, and the two
 * must never disagree about what `weekly` reads as.
 */
export function releaseScheduleLabel(schedule?: string): string {
  switch (schedule) {
    case "daily":
      return "ทุกวัน";
    case "weekly":
      return "สัปดาห์ละ 1 บท";
    case "biweekly":
      return "สัปดาห์ละ 2 บท";
    case "monthly":
      return "เดือนละครั้ง";
    case "irregular":
      return "ไม่กำหนด";
    default:
      return "";
  }
}

export function greeting(now = new Date()): string {
  const hour = now.getHours();
  if (hour < 12) return "สวัสดียามเช้า";
  if (hour < 17) return "สวัสดียามบ่าย";
  return "สวัสดียามค่ำ";
}
