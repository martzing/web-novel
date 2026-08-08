import type { ReactNode } from "react";
import { Link } from "react-router-dom";

import type { NovelListItem } from "../lib/api";
import { numberTH, percent } from "../lib/format";

/** Cover art, or the mockup's woven placeholder with the Chinese title. */
export function Cover({
  url,
  titleCN,
  width,
  height,
}: {
  url?: string;
  titleCN?: string;
  width: number | string;
  height: number | string;
}) {
  return (
    <div className="cover" style={{ width, height, flex: `0 0 ${typeof width === "number" ? `${width}px` : width}` }}>
      {url ? <img src={url} alt="" loading="lazy" /> : <span className="cover__cn">{titleCN ?? ""}</span>}
    </div>
  );
}

export function Stars({ rating }: { rating: number }) {
  const rounded = Math.round(rating);
  return (
    <span className="stars" aria-label={`${rating} ดาว`}>
      {[1, 2, 3, 4, 5].map((n) => (
        <span key={n} style={{ opacity: n <= rounded ? 1 : 0.25 }}>
          ★
        </span>
      ))}
    </span>
  );
}

export function StarPicker({
  value,
  onChange,
}: {
  value: number;
  onChange: (rating: number) => void;
}) {
  return (
    <div>
      {[1, 2, 3, 4, 5].map((n) => (
        <button
          key={n}
          type="button"
          className={`star-btn${n <= value ? " is-on" : ""}`}
          onClick={() => onChange(n)}
          aria-label={`ให้ ${n} ดาว`}
        >
          ★
        </button>
      ))}
    </div>
  );
}

export function ProgressBar({ pct }: { pct: number }) {
  return (
    <div className="progress">
      <div className="progress__bar" style={{ width: `${Math.min(Math.max(pct, 0), 100)}%` }} />
    </div>
  );
}

export function Tabs<T extends string>({
  tabs,
  active,
  onChange,
}: {
  tabs: { key: T; label: string; badge?: number }[];
  active: T;
  onChange: (key: T) => void;
}) {
  return (
    <div className="tabs" role="tablist">
      {tabs.map((tab) => (
        <button
          key={tab.key}
          role="tab"
          aria-selected={tab.key === active}
          className={`tab${tab.key === active ? " is-active" : ""}`}
          onClick={() => onChange(tab.key)}
        >
          {tab.label}
          {tab.badge !== undefined && <span className="muted"> {tab.badge}</span>}
        </button>
      ))}
    </div>
  );
}

export function Empty({ children }: { children: ReactNode }) {
  return <div className="empty">{children}</div>;
}

export function Loading({ rows = 3 }: { rows?: number }) {
  return (
    <div className="grid" style={{ marginTop: 16 }}>
      {Array.from({ length: rows }, (_, i) => (
        <div key={i} className="skeleton" style={{ height: 78 }} />
      ))}
    </div>
  );
}

export function ErrorNote({ message }: { message: string }) {
  return (
    <div className="empty" style={{ color: "var(--red)" }}>
      {message}
    </div>
  );
}

export function Modal({
  title,
  onClose,
  children,
}: {
  title: string;
  onClose: () => void;
  children: ReactNode;
}) {
  return (
    <div
      className="scrim"
      role="dialog"
      aria-modal="true"
      aria-label={title}
      onClick={onClose}
    >
      <div className="modal" onClick={(e) => e.stopPropagation()}>
        <div style={{ display: "flex", justifyContent: "space-between", alignItems: "start", gap: 12 }}>
          <h2 style={{ fontSize: 19 }}>{title}</h2>
          <button className="btn btn--ghost" onClick={onClose} aria-label="ปิด">
            ✕
          </button>
        </div>
        <div style={{ marginTop: 16 }}>{children}</div>
      </div>
    </div>
  );
}

/** The catalogue card used on the home and browse screens. */
export function NovelCard({ novel }: { novel: NovelListItem }) {
  return (
    <Link to={`/novels/${novel.slug}`} className="card" style={{ display: "flex", gap: 16, color: "inherit" }}>
      <Cover url={novel.cover_url} titleCN={novel.title_cn} width={62} height={88} />
      <div style={{ minWidth: 0, flex: 1 }}>
        <div className="muted" style={{ fontSize: 11.5 }}>
          {novel.title_cn}
        </div>
        <div className="serif" style={{ fontSize: 16, fontWeight: 600, marginTop: 2, lineHeight: 1.5 }}>
          {novel.title_th}
        </div>
        <div className="muted" style={{ fontSize: 12, marginTop: 6 }}>
          {novel.genres.map((g) => g.name_th).join(" · ") || "—"}
        </div>
        <div style={{ display: "flex", gap: 12, marginTop: 8, fontSize: 12 }} className="muted">
          <span className="mono">{novel.rating_avg.toFixed(1)}</span>
          <span>{numberTH(novel.chapters_count)} บท</span>
        </div>
      </div>
    </Link>
  );
}

/** One shelf row: cover, title, progress. */
export function ShelfRow({
  slug,
  titleCN,
  titleTH,
  coverURL,
  sub,
  pct,
  cta,
}: {
  slug: string;
  titleCN?: string;
  titleTH: string;
  coverURL?: string;
  sub: string;
  pct: number;
  cta: string;
}) {
  return (
    <Link to={`/novels/${slug}`} className="card" style={{ display: "flex", gap: 16, color: "inherit" }}>
      <Cover url={coverURL} titleCN={titleCN} width={54} height={76} />
      <div style={{ minWidth: 0, flex: 1 }}>
        <div className="serif" style={{ fontSize: 15.5, fontWeight: 600 }}>
          {titleTH}
        </div>
        <div className="muted" style={{ fontSize: 12, marginTop: 4 }}>
          {sub}
        </div>
        <div style={{ marginTop: 12 }}>
          <ProgressBar pct={pct} />
        </div>
      </div>
      <div
        style={{
          alignSelf: "center",
          fontSize: 12,
          color: "var(--red)",
          whiteSpace: "nowrap",
        }}
      >
        {cta} · {percent(pct)}
      </div>
    </Link>
  );
}
