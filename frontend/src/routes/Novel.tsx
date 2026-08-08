import { useEffect, useState } from "react";
import { Link, useParams } from "react-router-dom";

import { api, ChapterListItem, NovelDetail } from "../lib/api";

export default function Novel() {
  const { slug = "" } = useParams();
  const [novel, setNovel] = useState<NovelDetail | null>(null);
  const [chapters, setChapters] = useState<ChapterListItem[]>([]);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    setError(null);
    api
      .getNovel(slug)
      .then((n) => {
        if (cancelled) return;
        setNovel(n);
        return api.listChapters(n.id);
      })
      .then((cs) => !cancelled && cs && setChapters(cs))
      .catch((e: Error) => !cancelled && setError(e.message));
    return () => {
      cancelled = true;
    };
  }, [slug]);

  if (error) {
    return <div style={{ color: "var(--red)", fontSize: 13 }}>{error}</div>;
  }
  if (!novel)
    return <div style={{ color: "var(--soft)", fontSize: 13 }}>กำลังโหลด…</div>;

  return (
    <section style={{ maxWidth: 940 }}>
      <Link
        to="/"
        style={{
          fontSize: 12.5,
          color: "var(--soft)",
          display: "inline-block",
          padding: "0 0 22px",
        }}
      >
        ← กลับ
      </Link>

      <div style={{ display: "flex", gap: 38, alignItems: "flex-start" }}>
        <div style={{ flex: "0 0 200px" }}>
          <div
            style={{
              aspectRatio: "3 / 4",
              borderRadius: 3,
              border: "1px solid rgba(35,32,27,0.08)",
              background:
                "repeating-linear-gradient(135deg, #E7E0D2 0 7px, #F0EADE 7px 14px)",
              display: "flex",
              alignItems: "center",
              justifyContent: "center",
            }}
          >
            <div
              className="serif"
              style={{
                writingMode: "vertical-rl",
                fontSize: 26,
                color: "var(--soft)",
                letterSpacing: "0.2em",
              }}
            >
              {novel.title_cn ?? novel.title_th}
            </div>
          </div>
        </div>

        <div style={{ flex: 1, minWidth: 0 }}>
          <div style={{ fontSize: 12, color: "var(--gold)" }}>
            {novel.genres.map((g) => g.name_th).join(" · ")}
            {novel.title_cn && ` · แปลจาก ${novel.title_cn}`}
          </div>
          <h1
            className="serif"
            style={{ fontSize: 34, fontWeight: 600, margin: "8px 0 0" }}
          >
            {novel.title_th}
          </h1>
          {novel.author_name && (
            <div style={{ fontSize: 13, color: "var(--soft)", marginTop: 8 }}>
              ผู้แต่ง {novel.author_name}
            </div>
          )}

          <div
            style={{
              display: "flex",
              gap: 26,
              flexWrap: "wrap",
              marginTop: 24,
              padding: "18px 0",
              borderTop: "1px solid rgba(35,32,27,0.09)",
              borderBottom: "1px solid rgba(35,32,27,0.09)",
            }}
          >
            <Stat
              value={novel.rating_avg.toFixed(1)}
              label={`${novel.rating_count.toLocaleString()} รีวิว`}
            />
            <Stat
              value={novel.chapters_count.toLocaleString()}
              label="บททั้งหมด"
            />
            <Stat value={String(novel.arcs.length)} label="ภาค" />
            <Stat
              value={novel.followers_count.toLocaleString()}
              label="ผู้ติดตาม"
            />
          </div>

          {novel.description && (
            <p
              className="reading"
              style={{
                fontSize: 14.5,
                lineHeight: 2,
                color: "#3A342B",
                margin: "22px 0 0",
              }}
            >
              {novel.description}
            </p>
          )}
        </div>
      </div>

      <div
        style={{
          marginTop: 48,
          display: "flex",
          alignItems: "baseline",
          justifyContent: "space-between",
        }}
      >
        <div className="serif" style={{ fontSize: 20, fontWeight: 600 }}>
          สารบัญ
        </div>
        <div style={{ fontSize: 12, color: "var(--soft)" }}>
          {chapters.length} บทที่เผยแพร่แล้ว
        </div>
      </div>

      <div
        style={{ marginTop: 18, borderTop: "1px solid rgba(35,32,27,0.09)" }}
      >
        {chapters.map((c) => (
          <ChapterRow key={c.id} c={c} />
        ))}
        {chapters.length === 0 && (
          <div
            style={{ padding: "20px 8px", color: "var(--soft)", fontSize: 13 }}
          >
            ยังไม่มีบทเผยแพร่
          </div>
        )}
      </div>
    </section>
  );
}

function Stat({ value, label }: { value: string; label: string }) {
  return (
    <div>
      <div className="mono" style={{ fontSize: 19 }}>
        {value}
      </div>
      <div style={{ fontSize: 11, color: "var(--soft)", marginTop: 3 }}>
        {label}
      </div>
    </div>
  );
}

function ChapterRow({ c }: { c: ChapterListItem }) {
  const locked = c.price_coins > 0;
  return (
    <Link
      to={`/read/${c.id}`}
      style={{
        display: "flex",
        alignItems: "center",
        gap: 12,
        padding: "13px 8px",
        borderBottom: "1px solid rgba(35,32,27,0.07)",
        color: "inherit",
      }}
    >
      <div
        className="mono"
        style={{ fontSize: 11.5, color: "var(--soft)", minWidth: 34 }}
      >
        {c.chapter_no}
      </div>
      <div style={{ flex: 1, minWidth: 0, fontSize: 14 }}>{c.title}</div>
      <div style={{ fontSize: 11.5, color: "var(--soft)" }}>
        {locked ? `${c.price_coins} เหรียญ` : "อ่านฟรี"}
      </div>
    </Link>
  );
}
