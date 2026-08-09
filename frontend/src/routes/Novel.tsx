import { useMemo, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import {
  ApiError,
  api,
  newIdempotencyKey,
  type Arc,
  type ChapterListItem,
  type NovelDetail,
  type RelatedNovel,
} from "../lib/api";
import { useAuth } from "../lib/auth";
import { numberTH, relativeTime, thaiDate } from "../lib/format";
import {
  Empty,
  ErrorNote,
  Loading,
  Modal,
  NovelCover,
  StarPicker,
  Stars,
} from "../components";

export default function Novel() {
  const { slug = "" } = useParams();
  const { user } = useAuth();
  const qc = useQueryClient();

  const novel = useQuery({ queryKey: ["novel", slug], queryFn: () => api.getNovel(slug) });
  const novelId = novel.data?.id;

  const chapters = useQuery({
    queryKey: ["chapters", novelId],
    queryFn: () => api.listChapters(novelId!),
    enabled: Boolean(novelId),
  });

  const related = useQuery({
    queryKey: ["related", novelId],
    queryFn: () => api.listRelated(novelId!),
    enabled: Boolean(novelId),
  });

  const progress = useQuery({
    queryKey: ["progress", novelId],
    queryFn: () => api.getProgress(novelId!),
    enabled: Boolean(novelId && user),
    // A reader with no saved position gets a 404; that is an answer, not a fault.
    retry: false,
  });

  const following = useQuery({
    queryKey: ["following", novelId],
    queryFn: () => api.isFollowing(novelId!),
    enabled: Boolean(novelId && user),
  });

  const shelf = useQuery({
    queryKey: ["shelf", "counts"],
    queryFn: () => api.getShelf(),
    enabled: Boolean(user),
  });

  const onShelf = shelf.data?.data.some((item) => item.novel_id === novelId) ?? false;

  const toggleShelf = useMutation({
    mutationFn: async () => {
      if (onShelf) {
        await api.removeFromShelf(novelId!);
        return;
      }
      await api.setShelfStatus(novelId!, "reading");
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: ["shelf"] }),
  });

  const toggleFollow = useMutation({
    mutationFn: () => (following.data?.following ? api.unfollow(novelId!) : api.follow(novelId!)),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["following", novelId] });
      qc.invalidateQueries({ queryKey: ["novel", slug] });
    },
  });

  if (novel.isLoading) return <Loading rows={5} />;
  if (novel.isError) return <ErrorNote message={(novel.error as Error).message} />;
  if (!novel.data) return <Empty>ไม่พบนิยายเรื่องนี้</Empty>;

  const n = novel.data;
  const rows = chapters.data?.data ?? [];
  const resumeChapterId = progress.data?.last_chapter_id;
  const resumeChapterNo = progress.data?.last_chapter_no;
  const firstUnread = rows.find((c) => c.unlocked) ?? rows[0];

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

          {n.series_id && (
            <div style={{ marginTop: 10 }}>
              <Link to={`/series/${n.series_id}`} className="pill pill--gold">
                อยู่ในชุดหนังสือ →
              </Link>
            </div>
          )}

          <div style={{ display: "flex", gap: 10, marginTop: 20, flexWrap: "wrap" }}>
            {firstUnread && !resumeChapterId && (
              <Link to={`/read/${firstUnread.id}`} className="btn btn--primary">
                เริ่มอ่านบทที่ {firstUnread.chapter_no}
              </Link>
            )}
            {user && (
              <>
                <button
                  className="btn"
                  onClick={() => toggleShelf.mutate()}
                  disabled={toggleShelf.isPending}
                >
                  {onShelf ? "อยู่ในชั้นหนังสือแล้ว" : "เพิ่มลงชั้นหนังสือ"}
                </button>
                <button
                  className="btn"
                  onClick={() => toggleFollow.mutate()}
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
            <Stat value={numberTH(n.followers_count)} label="ผู้ติดตาม" />
          </div>

          <div className="muted" style={{ fontSize: 12.5, marginTop: 18, display: "flex", gap: 16, flexWrap: "wrap" }}>
            <span>อภิธานศัพท์ {numberTH(n.glossary_count)} คำ</span>
            <span>ความเห็น {numberTH(n.comments_count)}</span>
            <span>{n.arcs.length} ภาค</span>
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
  const qc = useQueryClient();
  const [cap, setCap] = useState(20);

  const subs = useQuery({ queryKey: ["auto-unlock"], queryFn: api.listAutoUnlock });
  const mine = subs.data?.data.find((s) => s.novel_id === novelId);
  const active = mine?.active ?? false;

  const toggle = useMutation({
    mutationFn: async () => {
      if (active) {
        await api.removeAutoUnlock(novelId);
        return;
      }
      await api.setAutoUnlock(novelId, { active: true, max_coins_per_chapter: cap });
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: ["auto-unlock"] }),
  });

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
        onClick={() => toggle.mutate()}
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

/**
 * The table of contents: filter pills over an arc-grouped, expandable list.
 *
 * Chapters beyond what has been translated are listed too, dimmed, up to
 * source_chapters_count. That is derivable on the client from the two counts,
 * so it costs no endpoint — and it is what tells a reader the work is ongoing
 * rather than finished at chapter 87.
 */
function TableOfContents({ novel, chapters }: { novel: NovelDetail; chapters: ChapterListItem[] }) {
  const [filter, setFilter] = useState<ChapterFilter>("all");
  const [expanded, setExpanded] = useState<Set<string>>(() => new Set());

  const visible = useMemo(() => {
    switch (filter) {
      case "free":
        return chapters.filter((c) => c.price_coins === 0);
      case "unlocked":
        return chapters.filter((c) => c.price_coins > 0 && c.unlocked);
      case "locked":
        return chapters.filter((c) => c.price_coins > 0 && !c.unlocked);
      default:
        return chapters;
    }
  }, [chapters, filter]);

  // Untranslated chapters only make sense on the unfiltered list: they have no
  // price and no unlock state to filter by.
  const untranslated = useMemo<TocRow[]>(() => {
    if (filter !== "all") return [];
    const highest = chapters.reduce((max, c) => Math.max(max, c.chapter_no), 0);
    const rows: TocRow[] = [];
    for (let no = highest + 1; no <= novel.source_chapters_count; no++) {
      rows.push({ key: `pending-${no}`, chapterNo: no });
    }
    // A very long source would otherwise render thousands of empty rows.
    return rows.slice(0, 50);
  }, [chapters, filter, novel.source_chapters_count]);

  const groups = useMemo(() => groupByArc(novel.arcs, visible, untranslated), [novel.arcs, visible, untranslated]);

  const toggle = (key: string) =>
    setExpanded((prev) => {
      const next = new Set(prev);
      if (next.has(key)) next.delete(key);
      else next.add(key);
      return next;
    });

  return (
    <div style={{ marginTop: 14 }}>
      <div className="chips" style={{ marginBottom: 16 }}>
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

      {groups.length === 0 ? (
        <Empty>ไม่มีบทที่ตรงกับตัวกรองนี้</Empty>
      ) : (
        groups.map((group) => {
          // The first arc opens by default and the rest stay closed, so a
          // 400-chapter novel does not arrive as a wall of rows. `expanded`
          // holds the arcs whose state has been *flipped* from that default,
          // which keeps "collapse the first one" working too.
          const openByDefault = group.index === 0;
          const isOpen = expanded.has(group.key) ? !openByDefault : openByDefault;

          return (
            <div key={group.key} style={{ marginBottom: 22 }}>
              <button className="arc-head" onClick={() => toggle(group.key)} aria-expanded={isOpen}>
                <span className="arc-head__caret" aria-hidden="true">
                  {isOpen ? "▾" : "▸"}
                </span>
                <span className="arc-head__name">{group.label}</span>
                <span className="muted mono" style={{ fontSize: 11.5 }}>
                  {numberTH(group.rows.length)} บท
                </span>
                {group.arc && novel.sell_by_arc && <ArcBundleButton arc={group.arc} />}
              </button>

              {isOpen && <ChapterRows rows={group.rows} />}
            </div>
          );
        })
      )}
    </div>
  );
}

interface TocGroup {
  key: string;
  index: number;
  label: string;
  arc?: Arc;
  rows: TocRow[];
}

function groupByArc(arcs: Arc[], chapters: ChapterListItem[], untranslated: TocRow[]): TocGroup[] {
  const rows: TocRow[] = [
    ...chapters.map((c) => ({ key: c.id, chapterNo: c.chapter_no, chapter: c })),
    ...untranslated,
  ];

  const groups: TocGroup[] = [];
  const claimed = new Set<string>();

  arcs.forEach((arc) => {
    // Membership by chapter number, matching how the backend prices a bundle —
    // arc_id is NULL on chapters written before their arc existed.
    const items = rows.filter(
      (r) => r.chapterNo >= arc.from_chapter_no && r.chapterNo <= arc.to_chapter_no,
    );
    items.forEach((r) => claimed.add(r.key));
    if (items.length > 0) {
      groups.push({
        key: `arc-${arc.id}`,
        index: groups.length,
        label: `ภาคที่ ${arc.arc_no} · ${arc.name}`,
        arc,
        rows: items,
      });
    }
  });

  const rest = rows.filter((r) => !claimed.has(r.key));
  if (rest.length > 0) {
    groups.push({
      key: "arc-none",
      index: groups.length,
      label: groups.length > 0 ? "บทอื่น ๆ" : "สารบัญ",
      rows: rest,
    });
  }
  return groups;
}

/** Buys a whole arc at the platform discount, after showing the quote. */
function ArcBundleButton({ arc }: { arc: Arc }) {
  const qc = useQueryClient();
  const [open, setOpen] = useState(false);
  const [key] = useState(newIdempotencyKey);

  const quote = useQuery({
    queryKey: ["arc-bundle", arc.id],
    queryFn: () => api.quoteArcBundle(arc.id),
    enabled: open,
    retry: false,
  });

  const buy = useMutation({
    mutationFn: () => api.unlockArc(arc.id, key),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["chapters"] });
      qc.invalidateQueries({ queryKey: ["wallet"] });
      setOpen(false);
    },
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

  const reviews = useQuery({
    queryKey: ["reviews", novelId],
    queryFn: () => api.listReviews(novelId),
  });

  const submit = useMutation({
    mutationFn: () => api.upsertReview(novelId, { rating, body }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["reviews", novelId] });
      qc.invalidateQueries({ queryKey: ["novel"] });
      setBody("");
    },
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
            onClick={() => submit.mutate()}
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
