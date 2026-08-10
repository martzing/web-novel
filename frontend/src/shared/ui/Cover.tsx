import type { CSSProperties } from "react";

import type { CoverStyle } from "@/shared/api/types";

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
  { key: "ink", label: "หมึกจีนแนวตั้ง" },
  { key: "seal", label: "ตราประทับ" },
  { key: "brush", label: "ลายพู่กัน" },
  { key: "plain", label: "เรียบ ตัวอักษรอย่างเดียว" },
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

/**
 * The cover-bearing fields of a novel.
 *
 * Stated structurally rather than as `Pick<NovelListItem, …>` so that
 * `shared/` never imports a feature — every novel-shaped record satisfies it.
 */
export interface CoverFields {
  cover_url?: string;
  title_cn?: string;
  cover_style?: CoverStyle;
  cover_color?: string;
  cover_text?: string;
}

/** Cover for a novel-shaped record, so callers stop restating the five fields. */
export function NovelCover({
  novel,
  width,
  height,
}: {
  novel: CoverFields;
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
