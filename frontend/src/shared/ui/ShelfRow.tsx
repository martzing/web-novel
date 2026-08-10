import type { ReactNode } from "react";
import { Link } from "react-router-dom";

import type { CoverStyle } from "@/shared/api/types";
import { percent } from "@/shared/lib/format";

import { Cover } from "./Cover";
import { ProgressBar } from "./ProgressBar";

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
