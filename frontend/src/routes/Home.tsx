import { useEffect, useState } from "react";
import { Link } from "react-router-dom";

import { api, NovelListItem } from "../lib/api";

export default function Home() {
  const [popular, setPopular] = useState<NovelListItem[] | null>(null);
  const [latest, setLatest] = useState<NovelListItem[] | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    Promise.all([
      api.listNovels({ sort: "popular" }),
      api.listNovels({ sort: "latest" }),
    ])
      .then(([p, l]) => {
        if (cancelled) return;
        setPopular(p);
        setLatest(l);
      })
      .catch((e: Error) => !cancelled && setError(e.message));
    return () => {
      cancelled = true;
    };
  }, []);

  return (
    <section style={{ maxWidth: 1080 }}>
      <div
        style={{
          display: "flex",
          alignItems: "baseline",
          justifyContent: "space-between",
          gap: 20,
        }}
      >
        <h1
          className="serif"
          style={{ fontSize: 30, fontWeight: 600, margin: 0 }}
        >
          สวัสดียามบ่าย
        </h1>
        <div style={{ fontSize: 12.5, color: "var(--soft)" }}>
          อ่านสะสมสัปดาห์นี้ 4 ชั่วโมง 12 นาที
        </div>
      </div>

      {error && (
        <div
          style={{
            marginTop: 24,
            padding: 14,
            background: "#FFF3F0",
            border: "1px solid rgba(168,56,43,0.2)",
            borderRadius: 3,
            color: "var(--red)",
            fontSize: 13,
          }}
        >
          {error}
        </div>
      )}

      <div
        style={{
          marginTop: 46,
          display: "grid",
          gridTemplateColumns: "1.35fr 1fr",
          gap: 40,
          alignItems: "start",
        }}
      >
        <div>
          <div className="eyebrow">อันดับสัปดาห์นี้</div>
          <div style={{ marginTop: 16, display: "grid" }}>
            {(popular ?? Array.from({ length: 5 }, () => null)).map((n, i) => (
              <RankRow key={n?.id ?? `p${i}`} rank={i + 1} n={n} />
            ))}
          </div>
        </div>

        <div>
          <div className="eyebrow">อัปเดตล่าสุด</div>
          <div
            style={{
              marginTop: 18,
              display: "grid",
              gridTemplateColumns: "repeat(auto-fill, minmax(158px, 1fr))",
              gap: "26px 22px",
            }}
          >
            {(latest ?? Array.from({ length: 4 }, () => null)).map((n, i) => (
              <CoverCard key={n?.id ?? `l${i}`} n={n} />
            ))}
          </div>
        </div>
      </div>
    </section>
  );
}

function RankRow({ rank, n }: { rank: number; n: NovelListItem | null }) {
  if (!n) return <SkeletonRow />;
  return (
    <Link
      to={`/novels/${encodeURIComponent(n.slug)}`}
      style={{
        display: "flex",
        gap: 16,
        alignItems: "center",
        padding: "14px 6px",
        borderTop: "1px solid rgba(35,32,27,0.09)",
        color: "inherit",
      }}
    >
      <div
        className="serif"
        style={{ fontSize: 21, color: "var(--red)", minWidth: 24 }}
      >
        {rank}
      </div>
      <div style={{ flex: 1, minWidth: 0 }}>
        <div
          style={{
            fontSize: 14,
            fontWeight: 600,
            whiteSpace: "nowrap",
            overflow: "hidden",
            textOverflow: "ellipsis",
          }}
        >
          {n.title_th}
        </div>
        <div style={{ fontSize: 11.5, color: "var(--soft)", marginTop: 3 }}>
          {n.genres.map((g) => g.name_th).join(" · ")} · {n.chapters_count} บท
        </div>
      </div>
      <div className="mono" style={{ fontSize: 11.5, color: "var(--soft)" }}>
        {n.rating_avg.toFixed(1)}
      </div>
    </Link>
  );
}

function CoverCard({ n }: { n: NovelListItem | null }) {
  if (!n) return <SkeletonCard />;
  return (
    <Link
      to={`/novels/${encodeURIComponent(n.slug)}`}
      style={{ color: "inherit" }}
    >
      <div
        style={{
          aspectRatio: "3 / 4",
          borderRadius: 3,
          background:
            "repeating-linear-gradient(135deg, #E7E0D2 0 6px, #F0EADE 6px 12px)",
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          border: "1px solid rgba(35,32,27,0.08)",
        }}
      >
        <div
          className="serif"
          style={{
            writingMode: "vertical-rl",
            fontSize: 17,
            color: "var(--soft)",
            letterSpacing: "0.18em",
          }}
        >
          {n.title_cn ?? n.title_th}
        </div>
      </div>
      <div
        style={{
          fontSize: 13.5,
          fontWeight: 600,
          marginTop: 10,
          lineHeight: 1.5,
        }}
      >
        {n.title_th}
      </div>
      <div style={{ fontSize: 11.5, color: "var(--soft)", marginTop: 4 }}>
        {n.chapters_count} บท · {n.followers_count.toLocaleString()} ผู้ติดตาม
      </div>
    </Link>
  );
}

function SkeletonRow() {
  return (
    <div
      style={{
        padding: "14px 6px",
        borderTop: "1px solid rgba(35,32,27,0.09)",
        color: "var(--soft)",
        fontSize: 12,
      }}
    >
      กำลังโหลด…
    </div>
  );
}

function SkeletonCard() {
  return (
    <div>
      <div
        style={{
          aspectRatio: "3 / 4",
          borderRadius: 3,
          background: "rgba(35,32,27,0.05)",
          border: "1px solid rgba(35,32,27,0.06)",
        }}
      />
      <div
        style={{
          height: 12,
          marginTop: 10,
          background: "rgba(35,32,27,0.08)",
          borderRadius: 2,
        }}
      />
    </div>
  );
}
