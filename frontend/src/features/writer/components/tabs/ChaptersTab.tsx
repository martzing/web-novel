import { useState } from "react";
import { useNavigate } from "react-router-dom";

import { numberTH } from "@/shared/lib/format";
import { Empty, Loading } from "@/shared/ui";

import type { WriterArc, WriterNovel } from "../../api";
import { useWriterArcs, useWriterChapters } from "../../queries";
import { ArcSheet } from "../sheets/ArcSheet";

// ── Tab 3 · ภาคและบท ────────────────────────────────────────────────────────

export function ChaptersTab({ novel }: { novel: WriterNovel }) {
  const navigate = useNavigate();
  const [editing, setEditing] = useState<WriterArc | "new" | null>(null);

  const arcs = useWriterArcs(novel.id);
  const chapters = useWriterChapters(novel.id);

  // The API returns chapters newest-first, so the head of the list is the
  // latest — `.slice(-12)` took the twelve *oldest* it had loaded and showed
  // them under a "บทล่าสุด" heading.
  const recent = (chapters.data?.data ?? []).slice(0, 12);

  return (
    <div>
      <div className="section-head" style={{ marginTop: 0 }}>
        <div className="eyebrow">ภาคในเรื่องนี้</div>
        <button className="btn btn--sm" onClick={() => setEditing("new")}>
          + เพิ่มภาคใหม่
        </button>
      </div>

      {arcs.isLoading ? (
        <Loading rows={2} />
      ) : (arcs.data?.data ?? []).length === 0 ? (
        <Empty>ยังไม่มีภาค · แบ่งภาคช่วยให้ผู้อ่านหาบทได้ง่ายขึ้น</Empty>
      ) : (
        <div className="rows" style={{ marginTop: 12 }}>
          {arcs.data!.data.map((arc) => (
            <div key={arc.id} className="row">
              <span className="mono muted" style={{ minWidth: 62, fontSize: 12.5 }}>
                ภาคที่ {arc.arc_no}
              </span>
              <span style={{ flex: 1, minWidth: 0, fontSize: 13.5 }}>{arc.name}</span>
              <span className="mono muted" style={{ fontSize: 11.5 }}>
                {arc.from_chapter_no}–{arc.to_chapter_no}
              </span>
              <button className="btn btn--ghost btn--sm" onClick={() => setEditing(arc)}>
                แก้ไข
              </button>
            </div>
          ))}
        </div>
      )}

      <div className="section-head">
        <div className="eyebrow">บทล่าสุด</div>
        <span className="muted" style={{ fontSize: 12 }}>
          แปลแล้ว {numberTH(novel.chapters_count)} จาก{" "}
          {novel.source_chapters_count > 0 ? numberTH(novel.source_chapters_count) : "—"} บท
        </span>
      </div>

      {chapters.isLoading ? (
        <Loading rows={3} />
      ) : recent.length === 0 ? (
        <Empty>ยังไม่มีบท · เริ่มที่หน้าเขียนบท</Empty>
      ) : (
        <div className="rows" style={{ marginTop: 12 }}>
          {recent.map((c) => (
            <div key={c.id} className="row">
              <span className="mono muted" style={{ minWidth: 44, fontSize: 12.5 }}>
                {c.chapter_no}
              </span>
              <span style={{ flex: 1, minWidth: 0, fontSize: 13.5 }}>{c.title}</span>
              <span className="pill">{chapterStatusLabel(c.status)}</span>
              <button
                className="btn btn--ghost btn--sm"
                // Deep-links with both ids so the editor opens on the chapter
                // itself rather than dropping the translator back at step 1.
                onClick={() => navigate(`/write?work=${novel.id}&chapter=${c.id}`)}
              >
                เปิดในตัวแก้ไข
              </button>
            </div>
          ))}
        </div>
      )}

      {editing && (
        <ArcSheet
          novelId={novel.id}
          arc={editing === "new" ? null : editing}
          onClose={() => setEditing(null)}
        />
      )}
    </div>
  );
}

function chapterStatusLabel(status: string): string {
  switch (status) {
    case "published":
      return "เผยแพร่แล้ว";
    case "scheduled":
      return "ตั้งเวลาไว้";
    default:
      return "ฉบับร่าง";
  }
}

