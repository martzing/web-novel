import { useEffect, useMemo, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { useQueryClient } from "@tanstack/react-query";

import { useAuth } from "@/features/identity";
import {
  useRemoveFromShelf,
  useSetShelfStatus,
  useShelfCounts,
} from "@/features/library";
import { useProgress } from "@/features/reading";
import { useIsFollowing, useReviews, useToggleFollow, useUpsertReview } from "@/features/social";
import {
  useArcBundle,
  useAutoUnlockSubs,
  useSetAutoUnlock,
  useUnlockArc,
} from "@/features/wallet";
import { ApiError, newIdempotencyKey } from "@/shared/api/client";
import { numberTH, releaseScheduleLabel, relativeTime, thaiDate } from "@/shared/lib/format";
import {
  Empty,
  ErrorNote,
  Loading,
  Modal,
  NovelCover,
  StarPicker,
  Stars,
} from "@/shared/ui";

import type { Arc, ChapterListItem, NovelDetail, RelatedNovel } from "../api";
import {
  catalogKeys,
  useChapters,
  useNovel,
  useRelatedNovels,
  useSeries,
} from "../queries";

export default function Novel() {
  const { slug = "" } = useParams();
  const { user } = useAuth();
  const qc = useQueryClient();

  const novel = useNovel(slug);
  const novelId = novel.data?.id;

  const chapters = useChapters(novelId);
  const related = useRelatedNovels(novelId);
  const progress = useProgress(user ? novelId : undefined);
  const following = useIsFollowing(novelId, Boolean(user));
  const shelf = useShelfCounts(Boolean(user));

  const onShelf = shelf.data?.data.some((item) => item.novel_id === novelId) ?? false;

  const addToShelf = useSetShelfStatus(novelId ?? "");
  const dropFromShelf = useRemoveFromShelf();
  const shelfPending = addToShelf.isPending || dropFromShelf.isPending;
  const toggleShelf = () =>
    onShelf ? dropFromShelf.mutate(novelId!) : addToShelf.mutate("reading");

  // A follow changes the novel's followers_count, which this page displays.
  const toggleFollow = useToggleFollow(novelId ?? "", () =>
    qc.invalidateQueries({ queryKey: catalogKeys.novelDetail(slug) }),
  );

  if (novel.isLoading) return <Loading rows={5} />;
  if (novel.isError) return <ErrorNote message={(novel.error as Error).message} />;
  if (!novel.data) return <Empty>ไม่พบนิยายเรื่องนี้</Empty>;

  const n = novel.data;
  const rows = chapters.data?.data ?? [];
  const resumeChapterId = progress.data?.last_chapter_id;
  const resumeChapterNo = progress.data?.last_chapter_no;
  const firstUnread = rows.find((c) => c.unlocked) ?? rows[0];
  // Comments are chapter-scoped, so "the novel's comments" has to point
  // somewhere concrete — the newest published chapter is where the
  // conversation actually is.
  const latestChapter = rows.length > 0 ? rows[rows.length - 1] : undefined;

  return (
    <section>
      <Link to="/browse" className="muted" style={{ fontSize: 12.5 }}>
        ← กลับ
      </Link>

      <div style={{ display: "flex", gap: 26, marginTop: 20, flexWrap: "wrap" }}>
        <NovelCover novel={n} width={168} height={236} />

        <div style={{ flex: "1 1 320px", minWidth: 0 }}>
          <div className="muted" style={{ fontSize: 12.5 }}>
            {n.genres.map((g) => g.name_th).join(" · ")}
            {n.title_cn && ` · แปลจาก ${n.title_cn}`}
          </div>
          <h1 className="page-title" style={{ marginTop: 8 }}>
            {n.title_th}
          </h1>
          <div className="muted" style={{ fontSize: 13, marginTop: 8 }}>
            {n.author_name ? `ผู้แต่ง ${n.author_name}` : "ไม่ระบุผู้แต่ง"}
          </div>

          {n.series_id && <SeriesLink seriesId={n.series_id} />}

          <div style={{ display: "flex", gap: 10, marginTop: 20, flexWrap: "wrap" }}>
            {firstUnread && !resumeChapterId && (
              <Link to={`/read/${firstUnread.id}`} className="btn btn--primary">
                เริ่มอ่านบทที่ {firstUnread.chapter_no}
              </Link>
            )}
            {user && (
              <>
                <button className="btn" onClick={toggleShelf} disabled={shelfPending}>
                  {onShelf ? "อยู่ในชั้นหนังสือแล้ว" : "เพิ่มลงชั้นหนังสือ"}
                </button>
                <button
                  className="btn"
                  onClick={() => toggleFollow.mutate(following.data?.following ?? false)}
                  disabled={toggleFollow.isPending}
                >
                  {following.data?.following ? "ติดตามอยู่" : "ติดตาม"}
                </button>
              </>
            )}
          </div>

          {/* Two counts, never one: the split is the whole point of the change. */}
          <div className="grid grid--kpi" style={{ marginTop: 26 }}>
            <Stat value={n.rating_avg.toFixed(1)} label={`${numberTH(n.rating_count)} รีวิว`}>
              <Stars rating={n.rating_avg} />
            </Stat>
            <Stat value={numberTH(n.chapters_count)} label="บทที่แปลแล้ว" />
            <Stat
              value={n.source_chapters_count > 0 ? numberTH(n.source_chapters_count) : "—"}
              label="บทในต้นฉบับ"
            />
            <Stat value={numberTH(n.arcs.length)} label="ภาค" />
            <Stat value={numberTH(n.followers_count)} label="ผู้ติดตาม" />
          </div>

          {/* These three are destinations, not decoration: each one is the
              answer to a question a reader asks on this page. */}
          <div className="meta-links">
            {n.glossary_count > 0 && firstUnread ? (
              <Link to={`/read/${firstUnread.id}?panel=glossary`}>
                อภิธานศัพท์ {numberTH(n.glossary_count)} คำ
              </Link>
            ) : (
              <span className="muted">อภิธานศัพท์ {numberTH(n.glossary_count)} คำ</span>
            )}
            {latestChapter ? (
              <Link to={`/chapters/${latestChapter.id}/comments`}>
                ความเห็น {numberTH(n.comments_count)}
              </Link>
            ) : (
              <span className="muted">ความเห็น {numberTH(n.comments_count)}</span>
            )}
            {n.release_schedule && (
              <span className="muted">
                รอบปล่อยบทใหม่ · {releaseScheduleLabel(n.release_schedule)}
              </span>
            )}
          </div>
        </div>
      </div>

      {resumeChapterId && (
        <ResumeCard
          chapterId={resumeChapterId}
          chapterNo={resumeChapterNo}
          translated={n.chapters_count}
          pct={progress.data?.pct ?? 0}
        />
      )}

      {n.description && (
        <p style={{ marginTop: 26, fontSize: 14, lineHeight: 2, maxWidth: 720 }}>{n.description}</p>
      )}

      {user && <AutoUnlockControl novelId={n.id} />}

      <div className="section-head" id="chapters">
        <div className="eyebrow">สารบัญ</div>
        <span className="muted" style={{ fontSize: 12 }}>
          {numberTH(rows.length)} บทที่เผยแพร่แล้ว
        </span>
      </div>

      {pricingSummary(n) && (
        <div className="muted price-summary">{pricingSummary(n)}</div>
      )}

      {chapters.isLoading ? (
        <Loading />
      ) : rows.length === 0 ? (
        <Empty>ยังไม่มีบทที่เผยแพร่</Empty>
      ) : (
        <TableOfContents novel={n} chapters={rows} />
      )}

      {related.data && related.data.data.length > 0 && (
        <RelatedWorks items={related.data.data} />
      )}

      <Reviews novelId={n.id} />
    </section>
  );
}

/**
 * The deal, in one line: "บทที่ 1–48 อ่านฟรี · บทหลังจากนั้น 5 เหรียญต่อบท".
 *
 * Without it a reader has to scroll the table of contents and infer the rule
 * from the per-row tags. Returns "" for a wholly free novel, where there is no
 * deal to explain.
 */
function pricingSummary(n: NovelDetail): string {
  if (n.price_per_chapter <= 0) return "";

  const price = `${numberTH(n.price_per_chapter)} เหรียญต่อบท`;
  if (n.free_until_chapter > 0) {
    return `บทที่ 1–${numberTH(n.free_until_chapter)} อ่านฟรี · บทหลังจากนั้น ${price}`;
  }
  return `บทที่ยังไม่ปลดล็อก ${price}`;
}

/**
 * The series link, carrying the numbers the design puts on it.
 *
 * It loads the series to say "5 เรื่อง 8 ภาค" rather than a bare arrow: the
 * size of a series is exactly what decides whether a reader clicks through.
 */
function SeriesLink({ seriesId }: { seriesId: string }) {
  const series = useSeries(seriesId);

  const label = series.data
    ? `ดูทั้งชุด · ${numberTH(series.data.books.length)} เรื่อง ${numberTH(series.data.arcs_count)} ภาค`
    : "ดูทั้งชุด →";

  return (
    <div style={{ marginTop: 10 }}>
      <Link to={`/series/${seriesId}`} className="pill pill--gold">
        {label}
      </Link>
    </div>
  );
}

/** Picks up where the reader left off, with how far they have to go. */
function ResumeCard({
  chapterId,
  chapterNo,
  translated,
  pct,
}: {
  chapterId: string;
  chapterNo?: number;
  translated: number;
  pct: number;
}) {
  const remaining = chapterNo ? Math.max(translated - chapterNo, 0) : 0;

  return (
    <div className="card resume-card">
      <div style={{ minWidth: 0, flex: 1 }}>
        <div className="eyebrow">อ่านค้างไว้</div>
        <div className="serif" style={{ fontSize: 17, fontWeight: 600, marginTop: 6 }}>
          {chapterNo ? `บทที่ ${numberTH(chapterNo)}` : "กลับไปอ่านต่อ"}
        </div>
        <div className="muted" style={{ fontSize: 12.5, marginTop: 6 }}>
          อ่านไปแล้ว {Math.round(pct)}%
          {remaining > 0 && ` · เหลืออีก ${numberTH(remaining)} บทที่แปลแล้ว`}
        </div>
      </div>
      <Link to={`/read/${chapterId}`} className="btn btn--primary">
        อ่านต่อ
      </Link>
    </div>
  );
}

function Stat({ value, label, children }: { value: string; label: string; children?: React.ReactNode }) {
  return (
    <div>
      <div className="serif" style={{ fontSize: 22, fontWeight: 600 }}>
        {value}
      </div>
      {children}
      <div className="muted" style={{ fontSize: 11.5, marginTop: 3 }}>
        {label}
      </div>
    </div>
  );
}

/** ปลดล็อกอัตโนมัติ — the opt-in that also grants the early-access window. */
function AutoUnlockControl({ novelId }: { novelId: string }) {
  const [cap, setCap] = useState(20);

  const subs = useAutoUnlockSubs();
  const mine = subs.data?.data.find((s) => s.novel_id === novelId);
  const active = mine?.active ?? false;

  const toggle = useSetAutoUnlock(novelId);

  return (
    <div className="card auto-unlock">
      <div style={{ minWidth: 0, flex: 1 }}>
        <div className="serif" style={{ fontSize: 15, fontWeight: 600 }}>
          ปลดล็อกอัตโนมัติ
        </div>
        <div className="muted" style={{ fontSize: 12.5, marginTop: 6, lineHeight: 1.9 }}>
          {active
            ? `เปิดอยู่ · ใช้ได้ไม่เกิน ${numberTH(mine?.max_coins_per_chapter ?? cap)} เหรียญต่อบท`
            : "เปิดไว้เพื่อปลดล็อกบทใหม่ให้อัตโนมัติ และอ่านก่อนใครในช่วง 24 ชั่วโมงแรก"}
        </div>
      </div>

      {!active && (
        <label className="field auto-unlock__cap">
          <span className="field__label">ไม่เกิน (เหรียญ/บท)</span>
          <input
            className="input"
            type="number"
            min={1}
            max={999}
            value={cap}
            onChange={(e) => setCap(Math.max(1, Number(e.target.value) || 1))}
          />
        </label>
      )}

      <button
        className={`btn${active ? "" : " btn--accent"}`}
        onClick={() => toggle.mutate({ active: !active, cap })}
        disabled={toggle.isPending}
      >
        {active ? "ปิด" : "เปิด"}
      </button>
    </div>
  );
}

type ChapterFilter = "all" | "free" | "unlocked" | "locked";

const FILTERS: { key: ChapterFilter; label: string }[] = [
  { key: "all", label: "ทั้งหมด" },
  { key: "free", label: "อ่านฟรี" },
  { key: "unlocked", label: "ปลดล็อกแล้ว" },
  { key: "locked", label: "ยังไม่ปลดล็อก" },
];

/** One ToC row, translated or not. */
interface TocRow {
  key: string;
  chapterNo: number;
  chapter?: ChapterListItem;
}

const TOC_RANGE_SIZE = 50;

interface TocRange {
  key: string;
  label: string;
  from: number;
  to: number;
}

/**
 * The table of contents: search by chapter number or title, filter by status
 * or arc, sort old/new, and jump around a long list via fixed-size chapter
 * ranges — mirroring the read/reader "jump to chapter" affordance readers
 * already expect from other Thai novel sites.
 *
 * Chapters beyond what has been translated are listed too, dimmed, up to
 * source_chapters_count. That is derivable on the client from the two counts,
 * so it costs no endpoint — and it is what tells a reader the work is ongoing
 * rather than finished at chapter 87. Ranges are built over translated +
 * this (capped) pending count, never over the raw source count, so a
 * long-running source doesn't produce dozens of near-empty ranges.
 */
function TableOfContents({ novel, chapters }: { novel: NovelDetail; chapters: ChapterListItem[] }) {
  const [filter, setFilter] = useState<ChapterFilter>("all");
  const [query, setQuery] = useState("");
  const [arcId, setArcId] = useState("all");
  const [sortDesc, setSortDesc] = useState(false);
  const [rangeKey, setRangeKey] = useState("all");
  const [shown, setShown] = useState(TOC_RANGE_SIZE);

  // Any change to what's being looked at should land back on the first page
  // of results, rather than leaving `shown` pointing past a now-shorter list.
  useEffect(() => {
    setShown(TOC_RANGE_SIZE);
  }, [query, arcId, sortDesc, rangeKey, filter]);

  const highestTranslated = useMemo(
    () => chapters.reduce((max, c) => Math.max(max, c.chapter_no), 0),
    [chapters],
  );

  const untranslated = useMemo<TocRow[]>(() => {
    const rows: TocRow[] = [];
    for (let no = highestTranslated + 1; no <= novel.source_chapters_count; no++) {
      rows.push({ key: `pending-${no}`, chapterNo: no });
    }
    // A very long source would otherwise render thousands of empty rows.
    return rows.slice(0, 50);
  }, [highestTranslated, novel.source_chapters_count]);

  const maxChapterNo = highestTranslated + untranslated.length;

  const chapterByNo = useMemo(() => {
    const map = new Map<number, ChapterListItem>();
    chapters.forEach((c) => map.set(c.chapter_no, c));
    return map;
  }, [chapters]);

  const pendingByNo = useMemo(() => {
    const map = new Map<number, TocRow>();
    untranslated.forEach((r) => map.set(r.chapterNo, r));
    return map;
  }, [untranslated]);

  const ranges = useMemo<TocRange[]>(() => {
    const list: TocRange[] = [{ key: "all", label: "ทั้งหมด", from: 1, to: maxChapterNo }];
    for (let from = 1; from <= maxChapterNo; from += TOC_RANGE_SIZE) {
      const to = Math.min(maxChapterNo, from + TOC_RANGE_SIZE - 1);
      list.push({ key: `r${from}`, label: `${from}–${to}`, from, to });
    }
    return list;
  }, [maxChapterNo]);

  const rangeIndex = Math.max(0, ranges.findIndex((r) => r.key === rangeKey));
  const activeRange = ranges[rangeIndex];
  const selectedArc = novel.arcs.find((a) => a.id === arcId);

  const trimmedQuery = query.trim();
  const isChapterJump = /^\d+$/.test(trimmedQuery);

  // Choosing an arc and choosing a range are two lenses over the same list —
  // picking one clears the other rather than compounding them.
  const pickArc = (id: string) => {
    setArcId(id);
    setRangeKey("all");
  };
  const pickRange = (key: string) => {
    setRangeKey(key);
    setArcId("all");
  };

  const allRows = useMemo<TocRow[]>(() => {
    let nums: number[] = [];
    if (trimmedQuery && isChapterJump) {
      // A pure chapter number is a "jump near here" request, not a filter —
      // it ignores whatever arc or range is currently selected.
      const target = Number(trimmedQuery);
      for (let n = Math.max(1, target - 3); n <= Math.min(maxChapterNo, target + 6); n++) nums.push(n);
    } else {
      const lo = selectedArc ? selectedArc.from_chapter_no : activeRange.from;
      const hi = selectedArc ? selectedArc.to_chapter_no : activeRange.to;
      for (let n = lo; n <= hi; n++) nums.push(n);
    }
    if (sortDesc) nums.reverse();

    let rows: TocRow[] = nums.map((n) => {
      const chapter = chapterByNo.get(n);
      if (chapter) return { key: chapter.id, chapterNo: n, chapter };
      return pendingByNo.get(n) ?? { key: `pending-${n}`, chapterNo: n };
    });

    rows = rows.filter((r) => {
      if (filter === "all") return true;
      // Untranslated rows have no price or unlock state, so they only ever
      // make sense on the unfiltered list.
      if (!r.chapter) return false;
      if (filter === "free") return r.chapter.price_coins === 0;
      if (filter === "unlocked") return r.chapter.price_coins > 0 && r.chapter.unlocked;
      return r.chapter.price_coins > 0 && !r.chapter.unlocked; // "locked"
    });

    if (trimmedQuery && !isChapterJump) {
      const needle = trimmedQuery.toLowerCase();
      rows = rows.filter((r) => r.chapter?.title.toLowerCase().includes(needle));
    }

    return rows;
  }, [trimmedQuery, isChapterJump, maxChapterNo, selectedArc, activeRange, sortDesc, chapterByNo, pendingByNo, filter]);

  const visibleCount = Math.min(allRows.length, shown);
  const visibleRows = allRows.slice(0, visibleCount);
  const remaining = allRows.length - visibleCount;

  const heading = selectedArc
    ? `ภาคที่ ${selectedArc.arc_no} · ${selectedArc.name}`
    : trimmedQuery && isChapterJump
      ? `ใกล้บทที่ ${trimmedQuery}`
      : `ช่วงบท ${activeRange.label}`;

  const rangePosLabel =
    arcId === "all" ? `ช่วงที่ ${numberTH(rangeIndex + 1)} จาก ${numberTH(ranges.length)}` : "กรองตามภาค";

  return (
    <div style={{ marginTop: 14 }}>
      <div className="toc-bar">
        <input
          className="input"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder="พิมพ์เลขบท เช่น 1450 หรือค้นชื่อบท"
        />
        <select className="select" value={arcId} onChange={(e) => pickArc(e.target.value)}>
          <option value="all">ทุกภาค</option>
          {novel.arcs.map((a) => (
            <option key={a.id} value={a.id}>
              ภาคที่ {a.arc_no} · {a.name}
            </option>
          ))}
        </select>
        <button className="btn" onClick={() => setSortDesc((d) => !d)}>
          {sortDesc ? "บทใหม่ → เก่า" : "บทเก่า → ใหม่"}
        </button>
      </div>

      <div className="chips" style={{ marginTop: 12 }}>
        {FILTERS.map((f) => (
          <button
            key={f.key}
            className={`chip${filter === f.key ? " is-active" : ""}`}
            onClick={() => setFilter(f.key)}
          >
            {f.label}
          </button>
        ))}
      </div>

      <div className="toc-heading">
        <span className="toc-heading__label">
          {heading}
          {selectedArc && novel.sell_by_arc && <ArcBundleButton arc={selectedArc} />}
        </span>
        <span className="muted" style={{ fontSize: 11.5 }}>
          {numberTH(allRows.length)} บท · แสดง {numberTH(visibleCount)} บท
        </span>
      </div>

      {allRows.length === 0 ? (
        <Empty>ไม่พบบทที่ตรงกับเงื่อนไข ลองล้างตัวกรองหรือเปลี่ยนช่วงบท</Empty>
      ) : (
        <ChapterRows rows={visibleRows} />
      )}

      {remaining > 0 && (
        <button
          className="btn btn--ghost btn--block"
          onClick={() => setShown((n) => n + TOC_RANGE_SIZE)}
        >
          แสดงอีก {numberTH(Math.min(TOC_RANGE_SIZE, remaining))} บท จากที่เหลือ {numberTH(remaining)} บท
        </button>
      )}

      {ranges.length > 2 && (
        <>
          <div className="toc-range">
            <button
              className="btn btn--ghost btn--sm"
              onClick={() => pickRange(ranges[Math.max(0, rangeIndex - 1)].key)}
              disabled={rangeIndex <= 0}
            >
              ‹ ช่วงก่อนหน้า
            </button>
            <select className="select mono" value={activeRange.key} onChange={(e) => pickRange(e.target.value)}>
              {ranges.map((r) => (
                <option key={r.key} value={r.key}>
                  {r.label}
                </option>
              ))}
            </select>
            <button
              className="btn btn--ghost btn--sm"
              onClick={() => pickRange(ranges[Math.min(ranges.length - 1, rangeIndex + 1)].key)}
              disabled={rangeIndex >= ranges.length - 1}
            >
              ช่วงถัดไป ›
            </button>
          </div>
          <div className="toc-range__pos muted">{rangePosLabel}</div>
        </>
      )}
    </div>
  );
}

/** Buys a whole arc at the platform discount, after showing the quote. */
function ArcBundleButton({ arc }: { arc: Arc }) {
  const qc = useQueryClient();
  const [open, setOpen] = useState(false);
  const [key] = useState(newIdempotencyKey);

  const quote = useArcBundle(arc.id, open);

  // Wallet keeps its own balance fresh; the newly bought chapters are this
  // screen's table of contents.
  const buy = useUnlockArc(arc.id, key, () => {
    qc.invalidateQueries({ queryKey: catalogKeys.chapters() });
    setOpen(false);
  });

  return (
    <>
      <span
        className="pill pill--gold arc-head__bundle"
        role="button"
        tabIndex={0}
        onClick={(e) => {
          // The header is itself a button; without this the arc collapses.
          e.stopPropagation();
          setOpen(true);
        }}
        onKeyDown={(e) => {
          if (e.key === "Enter" || e.key === " ") {
            e.preventDefault();
            e.stopPropagation();
            setOpen(true);
          }
        }}
      >
        ซื้อทั้งภาค
      </span>

      {open && (
        <Modal title={`ซื้อทั้งภาค · ${arc.name}`} onClose={() => setOpen(false)}>
          {quote.isLoading ? (
            <Loading rows={2} />
          ) : quote.isError ? (
            <ErrorNote message={bundleMessage(quote.error)} />
          ) : quote.data ? (
            <div>
              <div style={{ fontSize: 13.5, lineHeight: 2 }}>
                <div>
                  {numberTH(quote.data.chapter_count)} บทที่ยังไม่ปลดล็อก
                </div>
                <div className="muted">
                  ราคาปกติ ◎ {numberTH(quote.data.gross)} · ส่วนลด {quote.data.discount_percent}% (◎{" "}
                  {numberTH(quote.data.discount)})
                </div>
                <div className="serif" style={{ fontSize: 20, fontWeight: 600, marginTop: 10 }}>
                  รวม ◎ {numberTH(quote.data.total)}
                </div>
              </div>

              {buy.isError && <div className="form-error" style={{ marginTop: 12 }}>{bundleMessage(buy.error)}</div>}

              <button
                className="btn btn--primary btn--block"
                style={{ marginTop: 16 }}
                disabled={buy.isPending || quote.data.chapter_count === 0}
                onClick={() => buy.mutate()}
              >
                ยืนยันซื้อ ◎ {numberTH(quote.data.total)}
              </button>
            </div>
          ) : null}
        </Modal>
      )}
    </>
  );
}

function bundleMessage(error: unknown): string {
  if (error instanceof ApiError) {
    switch (error.code) {
      case "ARC_ALREADY_OWNED":
        return "คุณปลดล็อกทุกบทในภาคนี้แล้ว";
      case "ARC_NOT_FOR_SALE":
        return "เรื่องนี้ยังไม่เปิดขายแบบทั้งภาค";
      case "INSUFFICIENT_COINS":
        return "เหรียญไม่พอ · เติมเหรียญก่อนแล้วลองใหม่";
      case "ARC_BUNDLE_STALE":
        return "รายการในภาคเปลี่ยนไป กรุณาเปิดใหม่อีกครั้ง";
    }
    return error.message;
  }
  return (error as Error)?.message ?? "เกิดข้อผิดพลาด";
}

function ChapterRows({ rows }: { rows: TocRow[] }) {
  return (
    <div className="rows">
      {rows.map((row) =>
        row.chapter ? (
          <Link key={row.key} to={`/read/${row.chapter.id}`} className="row">
            <span className="mono muted" style={{ minWidth: 44, fontSize: 12.5 }}>
              {row.chapterNo}
            </span>
            <span style={{ flex: 1, minWidth: 0, fontSize: 13.5 }}>{row.chapter.title}</span>
            <span className="muted" style={{ fontSize: 11.5, whiteSpace: "nowrap" }}>
              {thaiDate(row.chapter.published_at)}
            </span>
            <span style={{ whiteSpace: "nowrap" }}>
              <ChapterTag chapter={row.chapter} />
            </span>
          </Link>
        ) : (
          // Not translated yet: shown so the reader can see how far the source
          // runs ahead, but deliberately not a link.
          <div key={row.key} className="row row--pending" aria-disabled="true">
            <span className="mono muted" style={{ minWidth: 44, fontSize: 12.5 }}>
              {row.chapterNo}
            </span>
            <span className="muted" style={{ flex: 1, minWidth: 0, fontSize: 13.5 }}>
              บทที่ {row.chapterNo}
            </span>
            <span className="pill">ยังไม่แปล</span>
          </div>
        ),
      )}
    </div>
  );
}

function ChapterTag({ chapter }: { chapter: ChapterListItem }) {
  if (chapter.price_coins === 0) return <span className="pill">อ่านฟรี</span>;
  if (chapter.unlocked) return <span className="pill pill--gold">ปลดล็อกแล้ว</span>;
  return <span className="pill pill--red">◎ {chapter.price_coins}</span>;
}

/** เรื่องเกี่ยวเนื่อง, grouped by the kind the translator declared. */
function RelatedWorks({ items }: { items: RelatedNovel[] }) {
  const groups = useMemo(() => {
    const byKind = new Map<string, { label: string; items: RelatedNovel[] }>();
    for (const item of items) {
      const group = byKind.get(item.kind) ?? { label: item.kind_label, items: [] };
      group.items.push(item);
      byKind.set(item.kind, group);
    }
    return [...byKind.values()];
  }, [items]);

  return (
    <>
      <div className="section-head">
        <div className="eyebrow">เรื่องเกี่ยวเนื่อง</div>
      </div>

      {groups.map((group) => (
        <div key={group.label} style={{ marginTop: 16 }}>
          <div className="muted" style={{ fontSize: 12.5, marginBottom: 8 }}>
            {group.label}
          </div>
          <div className="grid grid--cards">
            {group.items.map((item) => (
              <Link key={item.id} to={`/novels/${item.slug}`} className="card related-card">
                <NovelCover novel={item} width={48} height={68} />
                <div style={{ minWidth: 0 }}>
                  <div className="serif" style={{ fontSize: 14.5, fontWeight: 600, lineHeight: 1.5 }}>
                    {item.title_th}
                  </div>
                  <div className="muted" style={{ fontSize: 11.5, marginTop: 4 }}>
                    {numberTH(item.chapters_count)} บทที่แปลแล้ว
                  </div>
                  {item.note && (
                    <div className="muted" style={{ fontSize: 11.5, marginTop: 4, lineHeight: 1.8 }}>
                      {item.note}
                    </div>
                  )}
                </div>
              </Link>
            ))}
          </div>
        </div>
      ))}
    </>
  );
}

function Reviews({ novelId }: { novelId: string }) {
  const { user } = useAuth();
  const qc = useQueryClient();
  const [rating, setRating] = useState(0);
  const [body, setBody] = useState("");

  const reviews = useReviews(novelId);

  // A new review moves the novel's rating average, shown at the top of the page.
  const submit = useUpsertReview(novelId, () => {
    qc.invalidateQueries({ queryKey: catalogKeys.novel() });
    setBody("");
  });

  const mine = reviews.data?.my_review;

  return (
    <>
      <div className="section-head">
        <div className="eyebrow">รีวิว</div>
      </div>

      {user ? (
        <div className="card" style={{ marginTop: 14 }}>
          <div className="muted" style={{ fontSize: 12.5 }}>
            {mine ? "แก้ไขรีวิวของคุณ" : "ให้คะแนนเรื่องนี้"}
          </div>
          <div style={{ marginTop: 8 }}>
            <StarPicker value={rating || mine?.rating || 0} onChange={setRating} />
          </div>
          <textarea
            className="textarea"
            style={{ marginTop: 12, minHeight: 90 }}
            value={body}
            onChange={(e) => setBody(e.target.value)}
            placeholder={mine?.body ?? "เขียนความรู้สึกสั้น ๆ (ไม่บังคับ)"}
          />
          {submit.isError && <div className="form-error">{(submit.error as Error).message}</div>}
          <button
            className="btn btn--primary"
            style={{ marginTop: 12 }}
            disabled={(rating || mine?.rating || 0) === 0 || submit.isPending}
            onClick={() => submit.mutate({ rating, body })}
          >
            {mine ? "บันทึกรีวิว" : "ส่งรีวิว"}
          </button>
        </div>
      ) : (
        <Empty>
          <Link to="/login">เข้าสู่ระบบ</Link> เพื่อให้คะแนนและเขียนรีวิว
        </Empty>
      )}

      {reviews.data && reviews.data.data.length > 0 && (
        <div className="grid" style={{ marginTop: 16 }}>
          {reviews.data.data.map((r) => (
            <div key={r.id} className="card">
              <div style={{ display: "flex", justifyContent: "space-between", gap: 12 }}>
                <div style={{ fontSize: 13.5 }}>{r.author.display_name}</div>
                <Stars rating={r.rating} />
              </div>
              {r.body && (
                <p style={{ fontSize: 13.5, lineHeight: 1.95, marginTop: 8, marginBottom: 0 }}>
                  {r.body}
                </p>
              )}
              <div className="muted" style={{ fontSize: 11.5, marginTop: 8 }}>
                {relativeTime(r.created_at)}
              </div>
            </div>
          ))}
        </div>
      )}
    </>
  );
}
