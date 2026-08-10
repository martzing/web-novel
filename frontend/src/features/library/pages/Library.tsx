import { useState } from "react";
import { Link } from "react-router-dom";

import { useAuth } from "@/features/identity";
import { numberTH } from "@/shared/lib/format";
import { Empty, ErrorNote, Loading, ShelfRow, Tabs } from "@/shared/ui";

import type { ShelfItem } from "../api";
import { useShelf } from "../queries";

type Tab = "reading" | "saved" | "done";

const TAB_CTA: Record<Tab, string> = {
  reading: "อ่านต่อ",
  saved: "เริ่มอ่าน",
  done: "อ่านซ้ำ",
};

/**
 * The shelf's progress line, in translated chapters rather than a percentage.
 *
 * "บทที่ 87 จาก 88 บทที่แปลแล้ว" tells a reader something a bar cannot: how
 * much is left, and that new chapters have landed since they were last here.
 */
function shelfProgress(item: ShelfItem) {
  const translated = numberTH(item.chapters_count);

  if (!item.last_chapter_no) {
    return `${translated} บทที่แปลแล้ว · ยังไม่เริ่ม`;
  }

  const unread = item.chapters_count - item.last_chapter_no;
  return (
    <>
      บทที่ {numberTH(item.last_chapter_no)} จาก {translated} บทที่แปลแล้ว
      {unread > 0 && <span style={{ color: "var(--gold)" }}> · มีบทใหม่ {numberTH(unread)} บท</span>}
    </>
  );
}

export default function Library() {
  const { user } = useAuth();
  const [tab, setTab] = useState<Tab>("reading");

  const shelf = useShelf(tab, Boolean(user));

  if (!user) {
    return (
      <section>
        <h1 className="page-title">ชั้นหนังสือ</h1>
        <Empty>
          <Link to="/login">เข้าสู่ระบบ</Link> เพื่อเก็บนิยายไว้ในชั้นหนังสือของคุณ
        </Empty>
      </section>
    );
  }

  const counts = shelf.data?.counts;
  const items = shelf.data?.data ?? [];

  return (
    <section>
      <div className="page-head">
        <h1 className="page-title">ชั้นหนังสือ</h1>
      </div>

      <div style={{ marginTop: 20 }}>
        <Tabs<Tab>
          active={tab}
          onChange={setTab}
          tabs={[
            { key: "reading", label: "กำลังอ่าน", badge: counts?.reading },
            { key: "saved", label: "บันทึกไว้", badge: counts?.saved },
            { key: "done", label: "อ่านจบแล้ว", badge: counts?.done },
          ]}
        />
      </div>

      {shelf.isError ? (
        <ErrorNote message={(shelf.error as Error).message} />
      ) : shelf.isLoading ? (
        <Loading />
      ) : items.length === 0 ? (
        <Empty>
          ยังไม่มีนิยายในชั้นนี้ · <Link to="/browse">เลือกอ่านสักเรื่อง</Link>
        </Empty>
      ) : (
        <div className="grid" style={{ marginTop: 18 }}>
          {items.map((item) => (
            <ShelfRow
              key={item.novel_id}
              slug={item.slug}
              titleCN={item.title_cn}
              titleTH={item.title_th}
              coverURL={item.cover_url}
              coverStyle={item.cover_style}
              coverColor={item.cover_color}
              coverText={item.cover_text}
              sub={shelfProgress(item)}
              pct={item.pct}
              cta={TAB_CTA[tab]}
              // Resume the saved chapter when there is one; otherwise the novel
              // page, which is where "เริ่มอ่าน" belongs anyway.
              continueTo={
                item.last_chapter_id ? `/read/${item.last_chapter_id}` : `/novels/${item.slug}`
              }
            />
          ))}
        </div>
      )}
    </section>
  );
}
