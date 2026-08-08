import { useEffect, useState } from "react";
import { Link } from "react-router-dom";

import { api, Genre, NovelListItem } from "../lib/api";

export default function Browse() {
  const [q, setQ] = useState("");
  const [activeGenre, setActiveGenre] = useState<string>("");
  const [genres, setGenres] = useState<Genre[]>([]);
  const [results, setResults] = useState<NovelListItem[]>([]);

  useEffect(() => {
    api
      .listGenres()
      .then(setGenres)
      .catch(() => setGenres([]));
  }, []);

  useEffect(() => {
    const t = setTimeout(() => {
      api
        .listNovels({ q, genre: activeGenre })
        .then(setResults)
        .catch(() => setResults([]));
    }, 180);
    return () => clearTimeout(t);
  }, [q, activeGenre]);

  return (
    <section style={{ maxWidth: 1080 }}>
      <h1
        className="serif"
        style={{ fontSize: 28, fontWeight: 600, margin: 0 }}
      >
        หมวดหมู่และค้นหา
      </h1>
      <input
        value={q}
        onChange={(e) => setQ(e.target.value)}
        placeholder="ค้นหาชื่อเรื่อง ชื่อจีน ผู้แปล หรือชื่อตัวละคร"
        style={{
          width: "100%",
          marginTop: 22,
          padding: "15px 18px",
          border: "1px solid rgba(35,32,27,0.16)",
          borderRadius: 3,
          background: "var(--panel)",
          fontSize: 14,
          outline: "none",
        }}
      />

      <div style={{ display: "flex", flexWrap: "wrap", gap: 8, marginTop: 20 }}>
        <Chip
          label="ทั้งหมด"
          active={activeGenre === ""}
          onClick={() => setActiveGenre("")}
        />
        {genres.map((g) => (
          <Chip
            key={g.slug}
            label={g.name_th}
            active={activeGenre === g.slug}
            onClick={() => setActiveGenre(g.slug)}
          />
        ))}
      </div>

      <div style={{ marginTop: 34, fontSize: 12.5, color: "var(--soft)" }}>
        {results.length} เรื่อง
      </div>

      <div style={{ marginTop: 16 }}>
        {results.map((n) => (
          <Link
            key={n.id}
            to={`/novels/${encodeURIComponent(n.slug)}`}
            style={{
              display: "flex",
              gap: 20,
              padding: "18px 8px",
              borderTop: "1px solid rgba(35,32,27,0.09)",
              color: "inherit",
            }}
          >
            <div
              style={{
                flex: "0 0 62px",
                height: 84,
                borderRadius: 2,
                background:
                  "repeating-linear-gradient(135deg, #E7E0D2 0 6px, #F0EADE 6px 12px)",
                display: "flex",
                alignItems: "center",
                justifyContent: "center",
              }}
            >
              <div
                className="serif"
                style={{
                  writingMode: "vertical-rl",
                  fontSize: 13,
                  color: "var(--soft)",
                }}
              >
                {n.title_cn ?? n.title_th}
              </div>
            </div>
            <div style={{ flex: 1, minWidth: 0 }}>
              <div className="serif" style={{ fontSize: 17, fontWeight: 600 }}>
                {n.title_th}
              </div>
              <div
                style={{ fontSize: 11.5, color: "var(--gold)", marginTop: 4 }}
              >
                {n.genres.map((g) => g.name_th).join(" · ")}
              </div>
            </div>
            <div
              style={{
                flex: "0 0 84px",
                textAlign: "right",
                fontSize: 11.5,
                color: "var(--soft)",
              }}
            >
              <div className="mono" style={{ color: "var(--ink)" }}>
                {n.rating_avg.toFixed(1)}
              </div>
              <div style={{ marginTop: 4 }}>{n.chapters_count} บท</div>
            </div>
          </Link>
        ))}
      </div>
    </section>
  );
}

function Chip({
  label,
  active,
  onClick,
}: {
  label: string;
  active: boolean;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      style={{
        padding: "9px 16px",
        border: "1px solid rgba(35,32,27,0.14)",
        borderRadius: 999,
        background: active ? "var(--ink)" : "transparent",
        color: active ? "var(--bg)" : "var(--ink)",
        cursor: "pointer",
        fontSize: 12.5,
        fontFamily: "inherit",
      }}
    >
      {label}
    </button>
  );
}
