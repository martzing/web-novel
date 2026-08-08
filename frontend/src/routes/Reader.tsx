import { useEffect, useState } from "react";
import { Link, useParams } from "react-router-dom";

import { api, ChapterFull } from "../lib/api";

export default function Reader() {
  const { id = "" } = useParams();
  const [ch, setCh] = useState<ChapterFull | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    setError(null);
    api
      .getChapter(id)
      .then((c) => !cancelled && setCh(c))
      .catch((e: Error) => !cancelled && setError(e.message));
    return () => {
      cancelled = true;
    };
  }, [id]);

  if (error)
    return <div style={{ color: "var(--red)", padding: 40 }}>{error}</div>;
  if (!ch)
    return <div style={{ color: "var(--soft)", padding: 40 }}>กำลังโหลด…</div>;

  return (
    <article
      className="reading"
      style={{
        maxWidth: 640,
        margin: "0 auto",
        padding: "80px 26px 200px",
        fontSize: 20,
        lineHeight: 2,
      }}
    >
      <Link
        to={-1 as unknown as string}
        style={{ fontSize: 12.5, color: "var(--soft)" }}
      >
        ← กลับ
      </Link>

      <div style={{ textAlign: "center", marginTop: 40, marginBottom: 62 }}>
        <div className="eyebrow">บทที่ {ch.chapter_no}</div>
        <h1
          className="serif"
          style={{ fontSize: 32, margin: "14px 0 0", fontWeight: 600 }}
        >
          {ch.title}
        </h1>
      </div>

      {ch.locked ? (
        <LockedNotice priceCoins={ch.price_coins} />
      ) : (
        <div dangerouslySetInnerHTML={{ __html: ch.body_html ?? "" }} />
      )}
    </article>
  );
}

function LockedNotice({ priceCoins }: { priceCoins: number }) {
  return (
    <div
      style={{
        padding: "32px 28px",
        border: "1px solid rgba(35,32,27,0.14)",
        borderRadius: 4,
        textAlign: "center",
        background: "var(--panel)",
      }}
    >
      <div className="eyebrow">บทนี้ต้องปลดล็อก</div>
      <div
        className="serif"
        style={{ fontSize: 22, fontWeight: 600, marginTop: 10 }}
      >
        ใช้ {priceCoins} เหรียญเพื่ออ่านบทนี้
      </div>
      <div
        style={{
          fontSize: 13,
          color: "var(--soft)",
          marginTop: 8,
          lineHeight: 1.9,
        }}
      >
        การปลดล็อกจะเปิดใช้งานใน Phase 2 พร้อมระบบเหรียญและ mock payment
      </div>
    </div>
  );
}
