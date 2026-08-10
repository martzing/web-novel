import { Link } from "react-router-dom";

import { numberTH } from "@/shared/lib/format";
import { Cover } from "@/shared/ui";

import type { NovelListItem } from "../api";

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
