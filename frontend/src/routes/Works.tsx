import { useEffect, useMemo, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import {
  api,
  type CoverStyle,
  type Genre,
  type NovelStatus,
  type RelationKind,
  type ReleaseSchedule,
  type WriterArc,
  type WriterNovel,
  type WriterNovelPatch,
  type WriterRelation,
  type WriterSeries,
  type WriterSeriesBook,
} from "../lib/api";
import { useAuth } from "../lib/auth";
import { moveItem, useReorder } from "../lib/reorder";
import { numberTH } from "../lib/format";
import {
  COVER_COLORS,
  COVER_STYLES,
  Cover,
  Empty,
  ErrorNote,
  Loading,
  Modal,
  Tabs,
} from "../components";

type WorkTab = "info" | "cover" | "chapters" | "series" | "pricing";

const TABS: { key: WorkTab; label: string }[] = [
  { key: "info", label: "ข้อมูลเรื่อง" },
  { key: "cover", label: "หน้าปก" },
  { key: "chapters", label: "ภาคและบท" },
  { key: "series", label: "ชุดและเรื่องเกี่ยวเนื่อง" },
  { key: "pricing", label: "ราคาและการเผยแพร่" },
];

const STATUSES: { key: NovelStatus; label: string }[] = [
  { key: "ongoing", label: "เผยแพร่ กำลังแปล" },
  { key: "complete", label: "เผยแพร่ จบแล้ว" },
  { key: "hiatus", label: "พักการแปลชั่วคราว" },
  { key: "hidden", label: "ซ่อนจากหน้าร้าน" },
];

const SCHEDULES: { key: ReleaseSchedule; label: string }[] = [
  { key: "irregular", label: "ไม่กำหนด" },
  { key: "daily", label: "ทุกวัน" },
  { key: "weekly", label: "สัปดาห์ละ 1 บท" },
  { key: "biweekly", label: "สัปดาห์ละ 2 บท" },
  { key: "monthly", label: "เดือนละครั้ง" },
];

const RELATION_KINDS: { key: RelationKind; label: string }[] = [
  { key: "sequel", label: "ภาคต่อโดยตรง" },
  { key: "prequel", label: "ปฐมบท" },
  { key: "spinoff", label: "ภาคแยก" },
  { key: "side_story", label: "ภาคพิเศษ" },
  { key: "same_world", label: "เกิดในโลกเดียวกัน" },
];

/** จัดการผลงาน — the translator's work-management workspace. */
export default function Works() {
  const { user, isTranslator } = useAuth();
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [tab, setTab] = useState<WorkTab>("info");
  const [sheet, setSheet] = useState<"work" | "series" | null>(null);

  const novels = useQuery({
    queryKey: ["writer-novels"],
    queryFn: api.listWriterNovels,
    enabled: Boolean(user && isTranslator),
  });

  const seriesList = useQuery({
    queryKey: ["writer-series"],
    queryFn: api.listWriterSeries,
    enabled: Boolean(user && isTranslator),
  });

  const works = useMemo(() => novels.data?.data ?? [], [novels.data]);

  // Selecting the first work on load is what makes the master/detail layout
  // usable — the mockup's right pane is never empty.
  useEffect(() => {
    if (!selectedId && works.length > 0) setSelectedId(works[0].id);
  }, [selectedId, works]);

  if (!user) {
    return (
      <section>
        <h1 className="page-title">จัดการผลงาน</h1>
        <Empty>
          <Link to="/login">เข้าสู่ระบบ</Link> เพื่อจัดการผลงานของคุณ
        </Empty>
      </section>
    );
  }
  if (!isTranslator) {
    return (
      <section>
        <h1 className="page-title">จัดการผลงาน</h1>
        <Empty>บัญชีนี้ยังไม่ได้รับสิทธิ์นักแปล</Empty>
      </section>
    );
  }

  const selected = works.find((w) => w.id === selectedId) ?? null;
  const groups = groupWorks(works, seriesList.data?.data ?? []);

  return (
    <section>
      <div className="page-head">
        <div>
          <h1 className="page-title">จัดการผลงาน</h1>
          <div className="muted" style={{ fontSize: 12.5, marginTop: 6 }}>
            {numberTH(works.length)} เรื่อง · {numberTH(seriesList.data?.data.length ?? 0)} ชุดหนังสือ
          </div>
        </div>
        <button className="btn btn--primary" onClick={() => setSheet("work")}>
          + เพิ่มเรื่องใหม่
        </button>
      </div>

      {novels.isError ? (
        <ErrorNote message={(novels.error as Error).message} />
      ) : novels.isLoading ? (
        <Loading rows={4} />
      ) : (
        <div className="works">
          <aside className="works__tree card">
            <div className="eyebrow">ชุดและเรื่อง</div>

            {groups.map((group) => (
              <div key={group.key} style={{ marginTop: 16 }}>
                <div className="works__group">
                  <span>{group.label}</span>
                  <span className="mono muted">{group.works.length}</span>
                </div>
                {group.works.map((work) => (
                  <button
                    key={work.id}
                    className={`works__item${work.id === selectedId ? " is-active" : ""}`}
                    onClick={() => setSelectedId(work.id)}
                  >
                    <Cover
                      url={work.cover_url}
                      titleCN={work.title_cn}
                      style={work.cover_style}
                      color={work.cover_color}
                      text={work.cover_text}
                      width={30}
                      height={42}
                    />
                    <span style={{ minWidth: 0 }}>
                      <span className="works__item-title">{work.title_th}</span>
                      <span className="works__item-sub muted">
                        {statusLabel(work.status)} · แปลแล้ว {numberTH(work.chapters_count)}
                      </span>
                    </span>
                  </button>
                ))}
              </div>
            ))}

            <button
              className="btn btn--ghost btn--block"
              style={{ marginTop: 18 }}
              onClick={() => setSheet("series")}
            >
              + สร้างชุดหนังสือใหม่
            </button>
          </aside>

          <div className="works__detail card">
            {!selected ? (
              <Empty>ยังไม่มีผลงาน · กด “เพิ่มเรื่องใหม่” เพื่อเริ่ม</Empty>
            ) : (
              <>
                <Tabs<WorkTab> tabs={TABS} active={tab} onChange={setTab} />
                <div style={{ marginTop: 22 }}>
                  {tab === "info" && <InfoTab key={selected.id} novel={selected} />}
                  {tab === "cover" && <CoverTab key={selected.id} novel={selected} />}
                  {tab === "chapters" && <ChaptersTab key={selected.id} novel={selected} />}
                  {tab === "series" && (
                    <SeriesTab
                      key={selected.id}
                      novel={selected}
                      seriesList={seriesList.data?.data ?? []}
                      works={works}
                    />
                  )}
                  {tab === "pricing" && <PricingTab key={selected.id} novel={selected} />}
                </div>
              </>
            )}
          </div>
        </div>
      )}

      {sheet === "work" && (
        <NewWorkSheet
          onClose={() => setSheet(null)}
          onCreated={(id) => {
            setSelectedId(id);
            setTab("info");
            setSheet(null);
          }}
        />
      )}
      {sheet === "series" && <NewSeriesSheet onClose={() => setSheet(null)} />}
    </section>
  );
}

interface WorkGroup {
  key: string;
  label: string;
  works: WriterNovel[];
}

/** Groups the work tree by series, with the unfiled works last. */
function groupWorks(works: WriterNovel[], series: WriterSeries[]): WorkGroup[] {
  const groups: WorkGroup[] = series.map((s) => ({
    key: s.id,
    label: s.title,
    works: works.filter((w) => w.series_id === s.id),
  }));

  const unfiled = works.filter((w) => !w.series_id || !series.some((s) => s.id === w.series_id));
  if (unfiled.length > 0) {
    groups.push({ key: "none", label: "ไม่สังกัดชุด", works: unfiled });
  }
  return groups.filter((g) => g.works.length > 0);
}

function statusLabel(status: NovelStatus): string {
  return STATUSES.find((s) => s.key === status)?.label ?? status;
}

/** Saves a patch and keeps the work tree and detail pane in step. */
function useSaveNovel(novelId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (patch: WriterNovelPatch) => api.updateWriterNovel(novelId, patch),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["writer-novels"] });
      qc.invalidateQueries({ queryKey: ["writer-series"] });
    },
  });
}

function SaveRow({
  saving,
  error,
  saved,
  children,
}: {
  saving: boolean;
  error: unknown;
  saved: boolean;
  children?: React.ReactNode;
}) {
  return (
    <div style={{ marginTop: 22, display: "flex", gap: 12, alignItems: "center", flexWrap: "wrap" }}>
      {children}
      {saving && <span className="muted" style={{ fontSize: 12 }}>กำลังบันทึก…</span>}
      {saved && !saving && <span className="muted" style={{ fontSize: 12 }}>บันทึกแล้ว</span>}
      {Boolean(error) && <span className="form-error">{(error as Error).message}</span>}
    </div>
  );
}

// ── Tab 1 · ข้อมูลเรื่อง ────────────────────────────────────────────────────

const MAX_GENRES = 3;

function InfoTab({ novel }: { novel: WriterNovel }) {
  const save = useSaveNovel(novel.id);
  const genres = useQuery({ queryKey: ["genres"], queryFn: api.listGenres });

  const [titleTH, setTitleTH] = useState(novel.title_th);
  const [titleCN, setTitleCN] = useState(novel.title_cn ?? "");
  const [author, setAuthor] = useState(novel.author_name ?? "");
  const [description, setDescription] = useState(novel.description ?? "");
  const [genreIds, setGenreIds] = useState<string[]>(novel.genre_ids);
  const [sourceCount, setSourceCount] = useState(novel.source_chapters_count);

  const toggleGenre = (id: string) =>
    setGenreIds((prev) => {
      if (prev.includes(id)) return prev.filter((g) => g !== id);
      // The cap is enforced here rather than by disabling the unselected
      // chips, so a translator can always see the full list.
      if (prev.length >= MAX_GENRES) return prev;
      return [...prev, id];
    });

  return (
    <div>
      <div className="form-grid">
        <label className="field">
          <span className="field__label">ชื่อเรื่องภาษาไทย</span>
          <input className="input" value={titleTH} onChange={(e) => setTitleTH(e.target.value)} />
        </label>
        <label className="field">
          <span className="field__label">ชื่อต้นฉบับ</span>
          <input className="input" value={titleCN} onChange={(e) => setTitleCN(e.target.value)} />
        </label>
        <label className="field">
          <span className="field__label">ผู้แต่ง</span>
          <input className="input" value={author} onChange={(e) => setAuthor(e.target.value)} />
        </label>
        <label className="field">
          <span className="field__label">บทในต้นฉบับ</span>
          <input
            className="input"
            type="number"
            min={0}
            value={sourceCount}
            onChange={(e) => setSourceCount(Math.max(0, Number(e.target.value) || 0))}
          />
        </label>
      </div>

      <label className="field" style={{ marginTop: 18 }}>
        <span className="field__label">เรื่องย่อ</span>
        <textarea
          className="textarea"
          value={description}
          onChange={(e) => setDescription(e.target.value)}
        />
      </label>

      <div style={{ marginTop: 18 }}>
        <div className="field__label" style={{ marginBottom: 8 }}>
          หมวดหมู่ เลือกได้สูงสุด {MAX_GENRES} หมวด
        </div>
        <div className="chips">
          {(genres.data?.data ?? []).map((g: Genre) => (
            <button
              key={g.id}
              className={`chip${genreIds.includes(g.id) ? " is-active" : ""}`}
              onClick={() => toggleGenre(g.id)}
            >
              {g.name_th}
            </button>
          ))}
        </div>
      </div>

      <SaveRow saving={save.isPending} error={save.error} saved={save.isSuccess}>
        <button
          className="btn btn--primary"
          disabled={save.isPending || titleTH.trim() === ""}
          onClick={() =>
            save.mutate({
              title_th: titleTH,
              title_cn: titleCN,
              author_name: author,
              description,
              genre_ids: genreIds,
              source_chapters_count: sourceCount,
            })
          }
        >
          บันทึกการแก้ไข
        </button>
        <Link to={`/novels/${novel.slug}`} className="btn">
          ดูตัวอย่างหน้าเรื่อง
        </Link>
      </SaveRow>
    </div>
  );
}

// ── Tab 2 · หน้าปก ──────────────────────────────────────────────────────────

function CoverTab({ novel }: { novel: WriterNovel }) {
  const qc = useQueryClient();
  const save = useSaveNovel(novel.id);

  const [style, setStyle] = useState<CoverStyle>(novel.cover_style);
  const [color, setColor] = useState(novel.cover_color ?? COVER_COLORS[0]);
  const [text, setText] = useState(novel.cover_text ?? "");

  const upload = useMutation({
    mutationFn: (file: File) => api.uploadCover(novel.id, file),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["writer-novels"] }),
  });

  return (
    <div className="cover-editor">
      <div className="cover-editor__preview">
        {/* The preview renders from the working state, not the saved novel, so
            picking a swatch shows the result before committing to it. */}
        <Cover
          url={novel.cover_url}
          titleCN={novel.title_cn}
          style={style}
          color={color}
          text={text || novel.title_cn}
          width={198}
          height={264}
        />
        <div className="muted" style={{ fontSize: 11.5, marginTop: 10, textAlign: "center" }}>
          ตัวอย่างหน้าปก 600 × 800
        </div>
      </div>

      <div className="cover-editor__controls">
        <div className="field">
          <span className="field__label">อัปโหลดภาพปก</span>
          <div className="dropzone">
            <div style={{ fontSize: 13 }}>ลากไฟล์มาวาง หรือเลือกจากเครื่อง</div>
            <div className="muted" style={{ fontSize: 11.5, marginTop: 6 }}>
              JPG หรือ PNG อัตราส่วน 3:4 อย่างน้อย 600 × 800 px
            </div>
            <label className="btn btn--sm" style={{ marginTop: 12, cursor: "pointer" }}>
              เลือกไฟล์
              <input
                type="file"
                accept="image/jpeg,image/png,image/webp"
                hidden
                onChange={(e) => {
                  const file = e.target.files?.[0];
                  if (file) upload.mutate(file);
                }}
              />
            </label>
          </div>
          {upload.isPending && <span className="muted" style={{ fontSize: 12 }}>กำลังอัปโหลด…</span>}
          {upload.isError && <span className="form-error">{(upload.error as Error).message}</span>}
        </div>

        <div className="field" style={{ marginTop: 18 }}>
          <span className="field__label">หรือสร้างปกจากแม่แบบ</span>
          <div className="chips">
            {COVER_STYLES.map((s) => (
              <button
                key={s.key}
                className={`chip${style === s.key ? " is-active" : ""}`}
                onClick={() => setStyle(s.key)}
              >
                {s.label}
              </button>
            ))}
          </div>
        </div>

        <div className="field" style={{ marginTop: 18 }}>
          <span className="field__label">สีพื้นปก</span>
          <div className="swatches">
            {COVER_COLORS.map((c) => (
              <button
                key={c}
                className={`swatch${color === c ? " is-active" : ""}`}
                style={{ background: c }}
                aria-label={`สี ${c}`}
                onClick={() => setColor(c)}
              />
            ))}
          </div>
        </div>

        <label className="field" style={{ marginTop: 18 }}>
          <span className="field__label">ตัวอักษรบนปก</span>
          <input
            className="input"
            value={text}
            maxLength={40}
            onChange={(e) => setText(e.target.value)}
            placeholder={novel.title_cn ?? novel.title_th}
          />
        </label>

        <SaveRow saving={save.isPending} error={save.error} saved={save.isSuccess}>
          <button
            className="btn btn--primary"
            disabled={save.isPending}
            onClick={() =>
              save.mutate({ cover_style: style, cover_color: color, cover_text: text })
            }
          >
            ใช้ปกนี้
          </button>
          <button
            className="btn"
            onClick={() => {
              setStyle(novel.cover_style);
              setColor(novel.cover_color ?? COVER_COLORS[0]);
              setText(novel.cover_text ?? "");
            }}
          >
            คืนค่าปกเดิม
          </button>
        </SaveRow>
      </div>
    </div>
  );
}

// ── Tab 3 · ภาคและบท ────────────────────────────────────────────────────────

function ChaptersTab({ novel }: { novel: WriterNovel }) {
  const navigate = useNavigate();
  const [editing, setEditing] = useState<WriterArc | "new" | null>(null);

  const arcs = useQuery({
    queryKey: ["writer-arcs", novel.id],
    queryFn: () => api.listWriterArcs(novel.id),
  });

  const chapters = useQuery({
    queryKey: ["writer-chapters", novel.id],
    queryFn: () => api.listWriterChapters(novel.id),
  });

  const recent = (chapters.data?.data ?? []).slice(-12).reverse();

  return (
    <div>
      <div className="section-head" style={{ marginTop: 0 }}>
        <div className="eyebrow">ภาคในเรื่องนี้</div>
        <button className="btn btn--sm" onClick={() => setEditing("new")}>
          + เพิ่มภาคใหม่
        </button>
      </div>

      {arcs.isLoading ? (
        <Loading rows={2} />
      ) : (arcs.data?.data ?? []).length === 0 ? (
        <Empty>ยังไม่มีภาค · แบ่งภาคช่วยให้ผู้อ่านหาบทได้ง่ายขึ้น</Empty>
      ) : (
        <div className="rows" style={{ marginTop: 12 }}>
          {arcs.data!.data.map((arc) => (
            <div key={arc.id} className="row">
              <span className="mono muted" style={{ minWidth: 62, fontSize: 12.5 }}>
                ภาคที่ {arc.arc_no}
              </span>
              <span style={{ flex: 1, minWidth: 0, fontSize: 13.5 }}>{arc.name}</span>
              <span className="mono muted" style={{ fontSize: 11.5 }}>
                {arc.from_chapter_no}–{arc.to_chapter_no}
              </span>
              <button className="btn btn--ghost btn--sm" onClick={() => setEditing(arc)}>
                แก้ไข
              </button>
            </div>
          ))}
        </div>
      )}

      <div className="section-head">
        <div className="eyebrow">บทล่าสุด</div>
        <span className="muted" style={{ fontSize: 12 }}>
          แปลแล้ว {numberTH(novel.chapters_count)} จาก{" "}
          {novel.source_chapters_count > 0 ? numberTH(novel.source_chapters_count) : "—"} บท
        </span>
      </div>

      {chapters.isLoading ? (
        <Loading rows={3} />
      ) : recent.length === 0 ? (
        <Empty>ยังไม่มีบท · เริ่มที่หน้าเขียนบท</Empty>
      ) : (
        <div className="rows" style={{ marginTop: 12 }}>
          {recent.map((c) => (
            <div key={c.id} className="row">
              <span className="mono muted" style={{ minWidth: 44, fontSize: 12.5 }}>
                {c.chapter_no}
              </span>
              <span style={{ flex: 1, minWidth: 0, fontSize: 13.5 }}>{c.title}</span>
              <span className="pill">{chapterStatusLabel(c.status)}</span>
              <button
                className="btn btn--ghost btn--sm"
                // Deep-links with both ids so the editor opens on the chapter
                // itself rather than dropping the translator back at step 1.
                onClick={() => navigate(`/write?work=${novel.id}&chapter=${c.id}`)}
              >
                เปิดในตัวแก้ไข
              </button>
            </div>
          ))}
        </div>
      )}

      {editing && (
        <ArcSheet
          novelId={novel.id}
          arc={editing === "new" ? null : editing}
          onClose={() => setEditing(null)}
        />
      )}
    </div>
  );
}

function chapterStatusLabel(status: string): string {
  switch (status) {
    case "published":
      return "เผยแพร่แล้ว";
    case "scheduled":
      return "ตั้งเวลาไว้";
    default:
      return "ฉบับร่าง";
  }
}

// ── Tab 4 · ชุดและเรื่องเกี่ยวเนื่อง ────────────────────────────────────────

function SeriesTab({
  novel,
  seriesList,
  works,
}: {
  novel: WriterNovel;
  seriesList: WriterSeries[];
  works: WriterNovel[];
}) {
  const qc = useQueryClient();
  const save = useSaveNovel(novel.id);
  const [linking, setLinking] = useState(false);

  const books = useQuery({
    queryKey: ["writer-series-books", novel.series_id],
    queryFn: () => api.listWriterSeriesBooks(novel.series_id!),
    enabled: Boolean(novel.series_id),
  });

  const relations = useQuery({
    queryKey: ["writer-relations", novel.id],
    queryFn: () => api.listWriterRelations(novel.id),
  });

  const [order, setOrder] = useState<WriterSeriesBook[]>([]);
  useEffect(() => setOrder(books.data?.data ?? []), [books.data]);

  const reorder = useMutation({
    mutationFn: (ids: string[]) => api.reorderWriterSeries(novel.series_id!, ids),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["writer-series-books", novel.series_id] }),
  });

  const unlink = useMutation({
    mutationFn: (relatedId: string) => api.unlinkNovels(novel.id, relatedId),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["writer-relations", novel.id] }),
  });

  const applyMove = (from: number, to: number) => {
    const next = moveItem(order, from, to);
    setOrder(next);
    reorder.mutate(next.map((b) => b.novel_id));
  };

  const { handlersFor, classFor } = useReorder(applyMove);

  return (
    <div>
      <label className="field">
        <span className="field__label">ชุดหนังสือที่เรื่องนี้สังกัด</span>
        <select
          className="select"
          value={novel.series_id ?? ""}
          onChange={(e) =>
            // The empty option means "leave the series", which the API expresses
            // as an explicit null rather than an omitted key.
            save.mutate({ series_id: e.target.value === "" ? null : e.target.value })
          }
        >
          <option value="">ไม่สังกัดชุดใด</option>
          {seriesList.map((s) => (
            <option key={s.id} value={s.id}>
              {s.title}
            </option>
          ))}
        </select>
      </label>
      {save.isError && <div className="form-error">{(save.error as Error).message}</div>}

      {novel.series_id && (
        <>
          <div className="section-head">
            <div className="eyebrow">ลำดับการอ่านในชุด</div>
            <span className="muted" style={{ fontSize: 12 }}>
              ลากเพื่อจัดลำดับ ผู้อ่านจะเห็นลำดับนี้ในหน้าชุดหนังสือ
            </span>
          </div>

          {books.isLoading ? (
            <Loading rows={2} />
          ) : (
            <div className="rows" style={{ marginTop: 12 }}>
              {order.map((book, index) => (
                <div key={book.novel_id} className={`row drag-row${classFor(index)}`} {...handlersFor(index)}>
                  <span className="drag-handle" aria-hidden="true">
                    ⣿
                  </span>
                  <span className="mono muted" style={{ minWidth: 24, fontSize: 12.5 }}>
                    {index + 1}
                  </span>
                  <span style={{ flex: 1, minWidth: 0, fontSize: 13.5 }}>{book.title_th}</span>
                  <SeriesNote novelId={book.novel_id} note={book.note ?? ""} />
                  {/* Drag is pointer-only, so the same move is always reachable
                      from the keyboard through these. */}
                  <button
                    className="btn btn--ghost btn--sm"
                    disabled={index === 0}
                    aria-label="เลื่อนขึ้น"
                    onClick={() => applyMove(index, index - 1)}
                  >
                    ↑
                  </button>
                  <button
                    className="btn btn--ghost btn--sm"
                    disabled={index === order.length - 1}
                    aria-label="เลื่อนลง"
                    onClick={() => applyMove(index, index + 1)}
                  >
                    ↓
                  </button>
                </div>
              ))}
            </div>
          )}
        </>
      )}

      <div className="section-head">
        <div className="eyebrow">เรื่องเกี่ยวเนื่อง</div>
        <button className="btn btn--sm" onClick={() => setLinking(true)}>
          + ผูกเรื่องเกี่ยวเนื่อง
        </button>
      </div>

      {relations.isLoading ? (
        <Loading rows={2} />
      ) : (relations.data?.data ?? []).length === 0 ? (
        <Empty>ยังไม่มีเรื่องเกี่ยวเนื่อง</Empty>
      ) : (
        <div className="rows" style={{ marginTop: 12 }}>
          {relations.data!.data.map((rel: WriterRelation) => (
            <div key={rel.related_novel_id} className="row">
              <span className="pill">{rel.kind_label}</span>
              <span style={{ flex: 1, minWidth: 0, fontSize: 13.5 }}>{rel.title_th}</span>
              {rel.mirrored ? (
                // Declared on the other novel: shown for context, but its note
                // and order belong over there.
                <span className="muted" style={{ fontSize: 11.5 }}>
                  ผูกจากอีกเรื่อง
                </span>
              ) : (
                <button
                  className="btn btn--ghost btn--sm"
                  disabled={unlink.isPending}
                  onClick={() => unlink.mutate(rel.related_novel_id)}
                >
                  ปลด
                </button>
              )}
            </div>
          ))}
        </div>
      )}

      {linking && (
        <LinkSheet
          novel={novel}
          candidates={works.filter((w) => w.id !== novel.id)}
          onClose={() => setLinking(false)}
        />
      )}
    </div>
  );
}

/** The per-book note in a series' reading order, saved on blur. */
function SeriesNote({ novelId, note }: { novelId: string; note: string }) {
  const qc = useQueryClient();
  const [value, setValue] = useState(note);
  useEffect(() => setValue(note), [note]);

  const save = useMutation({
    mutationFn: (next: string) => api.setSeriesNote(novelId, next),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["writer-series-books"] }),
  });

  return (
    <input
      className="input series-note"
      value={value}
      placeholder="โน้ตสำหรับผู้อ่าน"
      onChange={(e) => setValue(e.target.value)}
      onBlur={() => value !== note && save.mutate(value)}
    />
  );
}

// ── Tab 5 · ราคาและการเผยแพร่ ───────────────────────────────────────────────

function PricingTab({ novel }: { novel: WriterNovel }) {
  const save = useSaveNovel(novel.id);

  const [price, setPrice] = useState(novel.price_per_chapter);
  const [freeUntil, setFreeUntil] = useState(novel.free_until_chapter);
  const [sellByArc, setSellByArc] = useState(novel.sell_by_arc);
  const [earlyAccess, setEarlyAccess] = useState(novel.early_access_hours > 0);
  const [tips, setTips] = useState(novel.tips_enabled);
  const [status, setStatus] = useState<NovelStatus>(novel.status);
  const [schedule, setSchedule] = useState<ReleaseSchedule>(novel.release_schedule ?? "irregular");

  return (
    <div>
      <div className="form-grid">
        <label className="field">
          <span className="field__label">ราคาต่อบท (เหรียญ)</span>
          <input
            className="input"
            type="number"
            min={0}
            max={999}
            value={price}
            onChange={(e) => setPrice(Math.max(0, Number(e.target.value) || 0))}
          />
        </label>
        <label className="field">
          <span className="field__label">อ่านฟรีถึงบทที่</span>
          <input
            className="input"
            type="number"
            min={0}
            value={freeUntil}
            onChange={(e) => setFreeUntil(Math.max(0, Number(e.target.value) || 0))}
          />
        </label>
      </div>

      <div className="card toggles" style={{ marginTop: 18 }}>
        <Toggle
          checked={sellByArc}
          onChange={setSellByArc}
          label="เปิดขายเป็นภาค ผู้อ่านซื้อทั้งภาคได้ในราคาลด 15%"
        />
        <Toggle
          checked={earlyAccess}
          onChange={setEarlyAccess}
          label="ปล่อยบทล่วงหน้าให้ผู้ที่เปิดปลดล็อกอัตโนมัติก่อน 24 ชั่วโมง"
        />
        <Toggle checked={tips} onChange={setTips} label="เปิดรับทิปจากผู้อ่านท้ายบท" />
      </div>

      <div className="form-grid" style={{ marginTop: 18 }}>
        <label className="field">
          <span className="field__label">สถานะการเผยแพร่</span>
          <select
            className="select"
            value={status}
            onChange={(e) => setStatus(e.target.value as NovelStatus)}
          >
            {STATUSES.map((s) => (
              <option key={s.key} value={s.key}>
                {s.label}
              </option>
            ))}
          </select>
        </label>
        <label className="field">
          <span className="field__label">รอบปล่อยบทใหม่</span>
          <select
            className="select"
            value={schedule}
            onChange={(e) => setSchedule(e.target.value as ReleaseSchedule)}
          >
            {SCHEDULES.map((s) => (
              <option key={s.key} value={s.key}>
                {s.label}
              </option>
            ))}
          </select>
        </label>
      </div>

      {status === "hidden" && (
        <div className="muted" style={{ fontSize: 12.5, marginTop: 12, lineHeight: 1.9 }}>
          เรื่องที่ซ่อนจะไม่ปรากฏในหน้าร้าน ค้นหา หรืออันดับ — แต่คุณยังเปิดและแก้ไขได้ตามปกติ
        </div>
      )}

      <SaveRow saving={save.isPending} error={save.error} saved={save.isSuccess}>
        <button
          className="btn btn--primary"
          disabled={save.isPending}
          onClick={() =>
            save.mutate({
              price_per_chapter: price,
              free_until_chapter: freeUntil,
              sell_by_arc: sellByArc,
              tips_enabled: tips,
              // The design offers a switch, not an hours field, so the toggle
              // maps to the platform's single 24-hour window.
              early_access_hours: earlyAccess ? 24 : 0,
              status,
              release_schedule: schedule,
            })
          }
        >
          บันทึกการตั้งค่า
        </button>
      </SaveRow>
    </div>
  );
}

function Toggle({
  checked,
  onChange,
  label,
}: {
  checked: boolean;
  onChange: (next: boolean) => void;
  label: string;
}) {
  return (
    <label className="toggle">
      <input type="checkbox" checked={checked} onChange={(e) => onChange(e.target.checked)} />
      <span>{label}</span>
    </label>
  );
}

// ── Sheets ──────────────────────────────────────────────────────────────────

function NewWorkSheet({
  onClose,
  onCreated,
}: {
  onClose: () => void;
  onCreated: (id: string) => void;
}) {
  const qc = useQueryClient();
  const [titleTH, setTitleTH] = useState("");
  const [titleCN, setTitleCN] = useState("");
  const [author, setAuthor] = useState("");

  const create = useMutation({
    mutationFn: () =>
      api.createWriterNovel({ title_th: titleTH, title_cn: titleCN, author_name: author }),
    onSuccess: (novel) => {
      qc.invalidateQueries({ queryKey: ["writer-novels"] });
      onCreated(novel.id);
    },
  });

  return (
    <Modal title="เพิ่มเรื่องใหม่" onClose={onClose}>
      <div className="grid" style={{ gap: 14 }}>
        <label className="field">
          <span className="field__label">ชื่อเรื่องภาษาไทย</span>
          <input className="input" value={titleTH} onChange={(e) => setTitleTH(e.target.value)} />
        </label>
        <label className="field">
          <span className="field__label">ชื่อต้นฉบับ</span>
          <input className="input" value={titleCN} onChange={(e) => setTitleCN(e.target.value)} />
        </label>
        <label className="field">
          <span className="field__label">ผู้แต่ง</span>
          <input className="input" value={author} onChange={(e) => setAuthor(e.target.value)} />
        </label>
      </div>
      {create.isError && <div className="form-error">{(create.error as Error).message}</div>}
      <button
        className="btn btn--primary btn--block"
        style={{ marginTop: 18 }}
        disabled={titleTH.trim() === "" || create.isPending}
        onClick={() => create.mutate()}
      >
        สร้างเรื่อง
      </button>
    </Modal>
  );
}

function NewSeriesSheet({ onClose }: { onClose: () => void }) {
  const qc = useQueryClient();
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");

  const create = useMutation({
    mutationFn: () => api.createWriterSeries({ title, description }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["writer-series"] });
      onClose();
    },
  });

  return (
    <Modal title="สร้างชุดหนังสือใหม่" onClose={onClose}>
      <div className="grid" style={{ gap: 14 }}>
        <label className="field">
          <span className="field__label">ชื่อชุด</span>
          <input className="input" value={title} onChange={(e) => setTitle(e.target.value)} />
        </label>
        <label className="field">
          <span className="field__label">คำอธิบาย</span>
          <textarea
            className="textarea"
            style={{ minHeight: 90 }}
            value={description}
            onChange={(e) => setDescription(e.target.value)}
          />
        </label>
      </div>
      {create.isError && <div className="form-error">{(create.error as Error).message}</div>}
      <button
        className="btn btn--primary btn--block"
        style={{ marginTop: 18 }}
        disabled={title.trim() === "" || create.isPending}
        onClick={() => create.mutate()}
      >
        สร้างชุดหนังสือ
      </button>
    </Modal>
  );
}

function ArcSheet({
  novelId,
  arc,
  onClose,
}: {
  novelId: string;
  arc: WriterArc | null;
  onClose: () => void;
}) {
  const qc = useQueryClient();
  const [arcNo, setArcNo] = useState(arc?.arc_no ?? 1);
  const [name, setName] = useState(arc?.name ?? "");
  const [from, setFrom] = useState(arc?.from_chapter_no ?? 1);
  const [to, setTo] = useState(arc?.to_chapter_no ?? 1);

  const submit = useMutation({
    mutationFn: () => {
      const body = { arc_no: arcNo, name, from_chapter_no: from, to_chapter_no: to };
      return arc ? api.updateWriterArc(arc.id, body) : api.createWriterArc(novelId, body);
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["writer-arcs", novelId] });
      onClose();
    },
  });

  return (
    <Modal title={arc ? "แก้ไขภาค" : "เพิ่มภาคใหม่"} onClose={onClose}>
      <div className="form-grid">
        <label className="field">
          <span className="field__label">ภาคที่</span>
          <input
            className="input"
            type="number"
            min={1}
            value={arcNo}
            onChange={(e) => setArcNo(Math.max(1, Number(e.target.value) || 1))}
          />
        </label>
        <label className="field">
          <span className="field__label">ชื่อภาค</span>
          <input className="input" value={name} onChange={(e) => setName(e.target.value)} />
        </label>
        <label className="field">
          <span className="field__label">ตั้งแต่บทที่</span>
          <input
            className="input"
            type="number"
            min={1}
            value={from}
            onChange={(e) => setFrom(Math.max(1, Number(e.target.value) || 1))}
          />
        </label>
        <label className="field">
          <span className="field__label">ถึงบทที่</span>
          <input
            className="input"
            type="number"
            min={1}
            value={to}
            onChange={(e) => setTo(Math.max(1, Number(e.target.value) || 1))}
          />
        </label>
      </div>
      {to < from && (
        <div className="form-error" style={{ marginTop: 10 }}>
          บทสุดท้ายต้องไม่น้อยกว่าบทแรก
        </div>
      )}
      {submit.isError && <div className="form-error">{(submit.error as Error).message}</div>}
      <button
        className="btn btn--primary btn--block"
        style={{ marginTop: 18 }}
        disabled={name.trim() === "" || to < from || submit.isPending}
        onClick={() => submit.mutate()}
      >
        {arc ? "บันทึกภาค" : "เพิ่มภาค"}
      </button>
    </Modal>
  );
}

function LinkSheet({
  novel,
  candidates,
  onClose,
}: {
  novel: WriterNovel;
  candidates: WriterNovel[];
  onClose: () => void;
}) {
  const qc = useQueryClient();
  const [relatedId, setRelatedId] = useState(candidates[0]?.id ?? "");
  const [kind, setKind] = useState<RelationKind>("sequel");
  const [note, setNote] = useState("");

  const link = useMutation({
    mutationFn: () =>
      api.linkNovels(novel.id, { related_novel_id: relatedId, kind, note }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["writer-relations", novel.id] });
      onClose();
    },
  });

  return (
    <Modal title="ผูกเรื่องเกี่ยวเนื่อง" onClose={onClose}>
      {candidates.length === 0 ? (
        <Empty>ต้องมีผลงานอย่างน้อยสองเรื่องจึงจะผูกกันได้</Empty>
      ) : (
        <>
          <div className="grid" style={{ gap: 14 }}>
            <label className="field">
              <span className="field__label">เรื่องที่จะผูก</span>
              <select
                className="select"
                value={relatedId}
                onChange={(e) => setRelatedId(e.target.value)}
              >
                {candidates.map((w) => (
                  <option key={w.id} value={w.id}>
                    {w.title_th}
                  </option>
                ))}
              </select>
            </label>
            <label className="field">
              <span className="field__label">ความสัมพันธ์ · {novel.title_th} เป็น…</span>
              <select
                className="select"
                value={kind}
                onChange={(e) => setKind(e.target.value as RelationKind)}
              >
                {RELATION_KINDS.map((k) => (
                  <option key={k.key} value={k.key}>
                    {k.label}
                  </option>
                ))}
              </select>
            </label>
            <label className="field">
              <span className="field__label">โน้ต (ไม่บังคับ)</span>
              <input className="input" value={note} onChange={(e) => setNote(e.target.value)} />
            </label>
          </div>
          {link.isError && <div className="form-error">{(link.error as Error).message}</div>}
          <button
            className="btn btn--primary btn--block"
            style={{ marginTop: 18 }}
            disabled={relatedId === "" || link.isPending}
            onClick={() => link.mutate()}
          >
            ผูกเรื่อง
          </button>
        </>
      )}
    </Modal>
  );
}
