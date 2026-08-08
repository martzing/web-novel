import { useState } from "react";
import { Link } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";

import { api } from "../lib/api";
import { useAuth } from "../lib/auth";
import { Empty, ErrorNote, Loading, ShelfRow, Tabs } from "../components";

type Tab = "reading" | "saved" | "done";

const TAB_CTA: Record<Tab, string> = {
  reading: "อ่านต่อ",
  saved: "เริ่มอ่าน",
  done: "อ่านซ้ำ",
};

export default function Library() {
  const { user } = useAuth();
  const [tab, setTab] = useState<Tab>("reading");

  const shelf = useQuery({
    queryKey: ["shelf", tab],
    queryFn: () => api.getShelf(tab),
    enabled: Boolean(user),
  });

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
              sub={
                item.last_chapter_no
                  ? `บทที่ ${item.last_chapter_no} จาก ${item.chapters_count}`
                  : `${item.chapters_count} บท · ยังไม่เริ่ม`
              }
              pct={item.pct}
              cta={TAB_CTA[tab]}
            />
          ))}
        </div>
      )}
    </section>
  );
}
