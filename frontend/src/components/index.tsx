import type { CSSProperties, ReactNode } from "react";
import { Link } from "react-router-dom";

import type { CoverStyle, NovelListItem } from "../lib/api";
import { numberTH, percent } from "../lib/format";

/**
 * How light the cover's text has to be.
 *
 * The four template styles paint over a translator-chosen colour, so the text
 * has to pick its own contrast rather than inherit a theme token — a dark
 * swatch under dark theme would otherwise render ink-on-ink. This is the
 * standard relative-luminance test at the WCAG midpoint.
 */
function isDarkColor(hex?: string): boolean {
  if (!hex || hex.length !== 7 || hex[0] !== "#") return false;
  const r = parseInt(hex.slice(1, 3), 16) / 255;
  const g = parseInt(hex.slice(3, 5), 16) / 255;
  const b = parseInt(hex.slice(5, 7), 16) / 255;
  const lin = (c: number) => (c <= 0.03928 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4);
  return 0.2126 * lin(r) + 0.7152 * lin(g) + 0.0722 * lin(b) < 0.45;
}

/** The generated background for each template style. */
function templateBackground(style: CoverStyle, color: string): CSSProperties {
  switch (style) {
    case "ink":
      // Diagonal stripes, the mockup's woven-silk motif in the chosen colour.
      return {
        background: `repeating-linear-gradient(135deg, ${color} 0 7px, ${shade(color, 0.14)} 7px 14px)`,
      };
    case "seal":
      // A bordered block, echoing a carved name seal.
      return {
        background: color,
        boxShadow: `inset 0 0 0 2px ${shade(color, 0.3)}, inset 0 0 0 7px ${color}`,
      };
    case "brush":
      return { background: `linear-gradient(160deg, ${shade(color, -0.18)}, ${shade(color, 0.22)})` };
    default:
      return { background: color };
  }
}

/** Lightens (positive) or darkens (negative) a hex colour by a fraction. */
function shade(hex: string, amount: number): string {
  if (hex.length !== 7 || hex[0] !== "#") return hex;
  const channel = (start: number) => {
    const value = parseInt(hex.slice(start, start + 2), 16);
    const shifted = amount >= 0 ? value + (255 - value) * amount : value * (1 + amount);
    return Math.round(Math.min(255, Math.max(0, shifted)))
      .toString(16)
      .padStart(2, "0");
  };
  return `#${channel(1)}${channel(3)}${channel(5)}`;
}

/** The default swatches the cover editor offers. */
export const COVER_COLORS = [
  "#8B1E2D",
  "#A9803F",
  "#2F5D50",
  "#2B3A67",
  "#23201B",
  "#6B3A5B",
  "#C9B79C",
  "#4A5259",
];

export const COVER_STYLES: { key: CoverStyle; label: string }[] = [
  { key: "image", label: "ใช้ภาพที่อัปโหลด" },
  { key: "ink", label: "ลายทแยง" },
  { key: "seal", label: "ตราประทับ" },
  { key: "brush", label: "ไล่สี" },
  { key: "plain", label: "สีพื้น" },
];

export interface CoverProps {
  url?: string;
  titleCN?: string;
  width: number | string;
  height: number | string;
  /** Template fields. Without them this falls back to the woven placeholder. */
  style?: CoverStyle;
  color?: string;
  text?: string;
}

/**
 * Cover art: an uploaded image, a generated template, or the woven placeholder.
 *
 * A template is pure CSS so it costs no request and stays crisp at every size
 * the product uses — 40px in the work tree up to 132px on the detail page.
 */
export function Cover({ url, titleCN, width, height, style, color, text }: CoverProps) {
  const box: CSSProperties = {
    width,
    height,
    flex: `0 0 ${typeof width === "number" ? `${width}px` : width}`,
  };

  // An uploaded image always wins when there is one: "image" is the style that
  // means "show cover_url", and the others are only reachable without artwork.
  if (url && (style === undefined || style === "image")) {
    return (
      <div className="cover" style={box}>
        <img src={url} alt="" loading="lazy" />
      </div>
    );
  }

  if (style && style !== "image" && color) {
    const label = text || titleCN || "";
    return (
      <div
        className={`cover cover--template${isDarkColor(color) ? " cover--on-dark" : ""}`}
        style={{ ...box, ...templateBackground(style, color) }}
      >
        {label && <span className="cover__label">{label}</span>}
      </div>
    );
  }

  return (
    <div className="cover" style={box}>
      <span className="cover__cn">{titleCN ?? ""}</span>
    </div>
  );
}

/** Cover for a novel-shaped record, so callers stop restating the six fields. */
export function NovelCover({
  novel,
  width,
  height,
}: {
  novel: Pick<NovelListItem, "cover_url" | "title_cn" | "cover_style" | "cover_color" | "cover_text">;
  width: number | string;
  height: number | string;
}) {
  return (
    <Cover
      url={novel.cover_url}
      titleCN={novel.title_cn}
      style={novel.cover_style}
      color={novel.cover_color}
      text={novel.cover_text}
      width={width}
      height={height}
    />
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

/**
 * One shelf row: cover, title, progress, and two ways out.
 *
 * The row is deliberately *not* one big link any more. The design gives a
 * reader two destinations — the novel's detail page and the chapter they left
 * off at — and a link wrapping the whole row cannot contain either without
 * nesting interactive elements, which is invalid markup and unusable with a
 * keyboard or a screen reader.
 */
export function ShelfRow({
  slug,
  titleCN,
  titleTH,
  coverURL,
  coverStyle,
  coverColor,
  coverText,
  sub,
  pct,
  cta,
  continueTo,
}: {
  slug: string;
  titleCN?: string;
  titleTH: string;
  coverURL?: string;
  coverStyle?: CoverStyle;
  coverColor?: string;
  coverText?: string;
  sub: ReactNode;
  pct: number;
  cta: string;
  /** Where the primary CTA goes — the saved chapter, or the novel page. */
  continueTo: string;
}) {
  const detail = `/novels/${slug}`;

  return (
    <div className="card shelf-row">
      <Link to={detail} className="shelf-row__cover" aria-label={`${titleTH} · รายละเอียด`}>
        <Cover
          url={coverURL}
          titleCN={titleCN}
          style={coverStyle}
          color={coverColor}
          text={coverText}
          width={54}
          height={76}
        />
      </Link>

      <div className="shelf-row__body">
        <Link to={detail} className="shelf-row__title serif">
          {titleTH}
        </Link>
        <div className="muted shelf-row__sub">{sub}</div>
        <div style={{ marginTop: 12 }}>
          <ProgressBar pct={pct} />
        </div>
      </div>

      <div className="shelf-row__actions">
        <Link to={continueTo} className="btn btn--accent">
          {cta}
        </Link>
        <Link to={`${detail}#chapters`} className="btn btn--ghost">
          รายละเอียดและสารบัญ
        </Link>
        <span className="muted mono shelf-row__pct">{percent(pct)}</span>
      </div>
    </div>
  );
}
