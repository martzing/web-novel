import { useState } from "react";

import type { NovelStatus, ReleaseSchedule } from "@/shared/api/types";

import type { WriterNovel } from "../../api";
import { SCHEDULES, STATUSES } from "../../constants";
import { useSaveNovel } from "../../queries";
import { SaveRow } from "../SaveRow";

// ── Tab 5 · ราคาและการเผยแพร่ ───────────────────────────────────────────────

export function PricingTab({ novel }: { novel: WriterNovel }) {
  const save = useSaveNovel(novel.id);

  const [price, setPrice] = useState(novel.price_per_chapter);
  const [freeUntil, setFreeUntil] = useState(novel.free_until_chapter);
  const [sellByArc, setSellByArc] = useState(novel.sell_by_arc);
  const [earlyAccess, setEarlyAccess] = useState(novel.early_access_hours > 0);
  const [tips, setTips] = useState(novel.tips_enabled);
  const [status, setStatus] = useState<NovelStatus>(novel.status);
  const [schedule, setSchedule] = useState<ReleaseSchedule>(novel.release_schedule ?? "irregular");

  return (
    <div>
      <div className="form-grid">
        <label className="field">
          <span className="field__label">ราคาต่อบท (เหรียญ)</span>
          <input
            className="input"
            type="number"
            min={0}
            max={999}
            value={price}
            onChange={(e) => setPrice(Math.max(0, Number(e.target.value) || 0))}
          />
        </label>
        <label className="field">
          <span className="field__label">อ่านฟรีถึงบทที่</span>
          <input
            className="input"
            type="number"
            min={0}
            value={freeUntil}
            onChange={(e) => setFreeUntil(Math.max(0, Number(e.target.value) || 0))}
          />
        </label>
      </div>

      <div className="card toggles" style={{ marginTop: 18 }}>
        <Toggle
          checked={sellByArc}
          onChange={setSellByArc}
          label="เปิดขายเป็นภาค ผู้อ่านซื้อทั้งภาคได้ในราคาลด 15%"
        />
        <Toggle
          checked={earlyAccess}
          onChange={setEarlyAccess}
          label="ปล่อยบทล่วงหน้าให้ผู้ที่เปิดปลดล็อกอัตโนมัติก่อน 24 ชั่วโมง"
        />
        <Toggle checked={tips} onChange={setTips} label="เปิดรับทิปจากผู้อ่านท้ายบท" />
      </div>

      <div className="form-grid" style={{ marginTop: 18 }}>
        <label className="field">
          <span className="field__label">สถานะการเผยแพร่</span>
          <select
            className="select"
            value={status}
            onChange={(e) => setStatus(e.target.value as NovelStatus)}
          >
            {STATUSES.map((s) => (
              <option key={s.key} value={s.key}>
                {s.label}
              </option>
            ))}
          </select>
        </label>
        <label className="field">
          <span className="field__label">รอบปล่อยบทใหม่</span>
          <select
            className="select"
            value={schedule}
            onChange={(e) => setSchedule(e.target.value as ReleaseSchedule)}
          >
            {SCHEDULES.map((s) => (
              <option key={s.key} value={s.key}>
                {s.label}
              </option>
            ))}
          </select>
        </label>
      </div>

      {status === "hidden" && (
        <div className="muted" style={{ fontSize: 12.5, marginTop: 12, lineHeight: 1.9 }}>
          เรื่องที่ซ่อนจะไม่ปรากฏในหน้าร้าน ค้นหา หรืออันดับ — แต่คุณยังเปิดและแก้ไขได้ตามปกติ
        </div>
      )}

      <SaveRow saving={save.isPending} error={save.error} saved={save.isSuccess}>
        <button
          className="btn btn--primary"
          disabled={save.isPending}
          onClick={() =>
            save.mutate({
              price_per_chapter: price,
              free_until_chapter: freeUntil,
              sell_by_arc: sellByArc,
              tips_enabled: tips,
              // The design offers a switch, not an hours field, so the toggle
              // maps to the platform's single 24-hour window.
              early_access_hours: earlyAccess ? 24 : 0,
              status,
              release_schedule: schedule,
            })
          }
        >
          บันทึกการตั้งค่า
        </button>
      </SaveRow>
    </div>
  );
}

function Toggle({
  checked,
  onChange,
  label,
}: {
  checked: boolean;
  onChange: (next: boolean) => void;
  label: string;
}) {
  return (
    <label className="toggle">
      <input type="checkbox" checked={checked} onChange={(e) => onChange(e.target.checked)} />
      <span>{label}</span>
    </label>
  );
}
