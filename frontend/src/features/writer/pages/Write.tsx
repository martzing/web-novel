import { useEffect, useMemo, useRef, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import { useMutation, useQueryClient } from "@tanstack/react-query";

import { useAuth } from "@/features/identity";
import { chapterStatusLabel, numberTH, relativeTime } from "@/shared/lib/format";
import { Cover, Empty, ErrorNote, Loading } from "@/shared/ui";

import { writerApi, type WriterChapter, type WriterNovel, type WriterSeries } from "../api";
import {
  useWriterChapter,
  useWriterChapters,
  useWriterNovels,
  useWriterSeries,
  writerKeys,
} from "../queries";

const AUTOSAVE_MS = 20_000;

/**
 * เขียนบท, as a three-step wizard: pick the work, pick or create the chapter,
 * then edit.
 *
 * The steps are derived from the two ids rather than held in their own state,
 * so a deep link from จัดการผลงาน (`/write?work=…&chapter=…`) lands straight in
 * the editor instead of replaying the picker.
 */
export default function Write() {
  const { user, isTranslator } = useAuth();
  const qc = useQueryClient();
  const [params, setParams] = useSearchParams();

  const novelId = params.get("work") ?? "";
  const chapterId = params.get("chapter") ?? "";

  const setSelection = (work: string, chapter: string) => {
    const next = new URLSearchParams();
    if (work) next.set("work", work);
    if (chapter) next.set("chapter", chapter);
    setParams(next, { replace: true });
  };

  const [draft, setDraft] = useState({ chapter_no: 1, title: "", body_source: "", price_coins: 0 });
  const [savedAt, setSavedAt] = useState<string | null>(null);
  const [scheduledAt, setScheduledAt] = useState("");

  const bodyRef = useRef<HTMLTextAreaElement>(null);
  const autosave = useRef<number>();

  const novels = useWriterNovels(isTranslator);
  const seriesList = useWriterSeries(isTranslator);
  const chapters = useWriterChapters(novelId);
  const chapter = useWriterChapter(chapterId);

  // Load the selected chapter into the editor.
  useEffect(() => {
    if (!chapter.data) return;
    setDraft({
      chapter_no: chapter.data.chapter_no,
      title: chapter.data.title,
      body_source: chapter.data.body_source,
      price_coins: chapter.data.price_coins,
    });
    setSavedAt(chapter.data.updated_at ?? null);
  }, [chapter.data]);

  const save = useMutation({
    mutationFn: () => writerApi.saveWriterChapter(chapterId, draft),
    onSuccess: (saved) => {
      setSavedAt(saved.updated_at ?? new Date().toISOString());
      qc.invalidateQueries({ queryKey: writerKeys.chapters(novelId) });
    },
  });

  const create = useMutation({
    mutationFn: () => {
      const highest = (chapters.data?.data ?? []).reduce(
        (max, c) => Math.max(max, c.chapter_no),
        0,
      );
      const nextNo = highest + 1;
      return writerApi.createWriterChapter(novelId, {
        chapter_no: nextNo,
        title: `บทที่ ${nextNo}`,
        body_source: "",
        // 0 lets the novel's price_per_chapter apply; the server also forces
        // free below free_until_chapter.
        price_coins: 0,
      });
    },
    onSuccess: (created) => {
      qc.invalidateQueries({ queryKey: writerKeys.chapters(novelId) });
      setSelection(novelId, created.id);
    },
  });

  const publish = useMutation({
    mutationFn: () =>
      writerApi.publishChapter(chapterId, scheduledAt ? new Date(scheduledAt).toISOString() : undefined),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: writerKeys.chapters(novelId) });
      qc.invalidateQueries({ queryKey: writerKeys.chapter(chapterId) });
      // Publishing moves novels.chapters_count, which จัดการผลงาน and the work
      // picker both display — without this they keep showing the old total.
      qc.invalidateQueries({ queryKey: writerKeys.novels() });
      setScheduledAt("");
    },
  });

  // Autosave every 20s while there is unsaved work, per W-03.
  useEffect(() => {
    if (!chapterId) return;
    window.clearTimeout(autosave.current);
    autosave.current = window.setTimeout(() => save.mutate(), AUTOSAVE_MS);
    return () => window.clearTimeout(autosave.current);
    // Re-armed on every keystroke so the timer measures idle time.
  }, [draft, chapterId]); // eslint-disable-line react-hooks/exhaustive-deps

  if (!user) {
    return (
      <section>
        <h1 className="page-title">เขียนบท</h1>
        <Empty>
          <Link to="/login">เข้าสู่ระบบ</Link> ด้วยบัญชีนักแปลเพื่อใช้งานพื้นที่เขียน
        </Empty>
      </section>
    );
  }

  if (!isTranslator) {
    return (
      <section>
        <h1 className="page-title">เขียนบท</h1>
        <Empty>บัญชีนี้ยังไม่มีสิทธิ์นักแปล ติดต่อผู้ดูแลเพื่อขอเปิดใช้งาน</Empty>
      </section>
    );
  }

  /** Wraps the current selection in an editor marker. */
  function wrapSelection(prefix: string, suffix: string) {
    const el = bodyRef.current;
    if (!el) return;
    const { selectionStart: start, selectionEnd: end, value } = el;
    const selected = value.slice(start, end);
    const next = value.slice(0, start) + prefix + selected + suffix + value.slice(end);
    setDraft((d) => ({ ...d, body_source: next }));
    // Restore the caret after React re-renders the textarea.
    requestAnimationFrame(() => {
      el.focus();
      el.setSelectionRange(start + prefix.length, start + prefix.length + selected.length);
    });
  }

  const works = novels.data?.data ?? [];
  const selectedWork = works.find((w) => w.id === novelId) ?? null;
  const step = !novelId || !selectedWork ? 1 : !chapterId ? 2 : 3;

  return (
    <section>
      <div className="page-head">
        <div>
          <h1 className="page-title">เขียนบท</h1>
          <div className="muted" style={{ fontSize: 12, marginTop: 6 }}>
            ขั้นที่ {step} จาก 3 · {["เลือกเรื่อง", "เลือกบทหรือสร้างบทใหม่", "เขียน"][step - 1]}
          </div>
        </div>
        {step === 3 && (
          <div className="muted" style={{ fontSize: 12 }}>
            {save.isPending
              ? "กำลังบันทึก…"
              : savedAt
                ? `บันทึกอัตโนมัติ ${relativeTime(savedAt)}`
                : "ยังไม่ได้บันทึก"}
          </div>
        )}
      </div>

      <Breadcrumb
        step={step}
        work={selectedWork}
        chapterTitle={chapter.data?.title}
        onPickWork={() => setSelection("", "")}
        onPickChapter={() => setSelection(novelId, "")}
      />

      {novels.isLoading ? (
        <Loading rows={3} />
      ) : step === 1 ? (
        <WorkPicker
          works={works}
          series={seriesList.data?.data ?? []}
          onPick={(id) => setSelection(id, "")}
        />
      ) : step === 2 ? (
        <ChapterPicker
          chapters={chapters.data?.data ?? []}
          loading={chapters.isLoading}
          creating={create.isPending}
          onCreate={() => create.mutate()}
          onPick={(id) => setSelection(novelId, id)}
        />
      ) : (
        <div className="writer" style={{ marginTop: 18 }}>
          <aside className="writer__rail">
            <div className="eyebrow" style={{ marginBottom: 10 }}>
              ร่างและบทที่เผยแพร่
            </div>
            <button
              className="btn btn--block btn--sm"
              onClick={() => create.mutate()}
              disabled={create.isPending}
            >
              + สร้างบทใหม่
            </button>
            <div className="grid" style={{ gap: 2, marginTop: 12 }}>
              {(chapters.data?.data ?? []).map((c) => (
                <button
                  key={c.id}
                  className={`panel__item${c.id === chapterId ? " is-current" : ""}`}
                  onClick={() => setSelection(novelId, c.id)}
                >
                  <span className="panel__num">{c.chapter_no}</span>
                  <span style={{ flex: 1, minWidth: 0 }}>
                    <span style={{ display: "block" }}>{c.title}</span>
                    <span className="muted" style={{ fontSize: 11 }}>
                      {chapterStatusLabel(c.status)}
                    </span>
                  </span>
                </button>
              ))}
            </div>
          </aside>

          <div className="writer__editor">
            <input
              className="input"
              style={{ fontSize: 19, fontFamily: "var(--font-serif)" }}
              value={draft.title}
              onChange={(e) => setDraft((d) => ({ ...d, title: e.target.value }))}
              placeholder="ชื่อบท"
            />

            <div style={{ display: "flex", gap: 8, marginTop: 12, flexWrap: "wrap" }}>
              <button className="btn btn--sm" onClick={() => wrapSelection("[^", "]")}>
                แทรกเชิงอรรถ
              </button>
              <button className="btn btn--sm" onClick={() => wrapSelection("{{", "}}")}>
                ผูกศัพท์อภิธาน
              </button>
              <button className="btn btn--sm" onClick={() => wrapSelection("\n\n<hr/>\n\n", "")}>
                คั่นฉาก
              </button>
            </div>

            <textarea
              ref={bodyRef}
              className="textarea"
              style={{ marginTop: 12, minHeight: 380, fontFamily: "var(--font-reading)" }}
              value={draft.body_source}
              onChange={(e) => setDraft((d) => ({ ...d, body_source: e.target.value }))}
              placeholder="เขียนเนื้อหาบทที่นี่ ใช้ {{term_key}} เพื่อผูกศัพท์อภิธาน"
            />

            <div className="muted" style={{ fontSize: 11.5, marginTop: 8 }}>
              {numberTH(draft.body_source.trim().split(/\s+/).filter(Boolean).length)} คำ
            </div>

            <div className="card" style={{ marginTop: 18 }}>
              <div style={{ display: "flex", gap: 16, flexWrap: "wrap", alignItems: "end" }}>
                <label className="field" style={{ width: 150 }}>
                  <span className="field__label">ราคาปลดล็อก (เหรียญ)</span>
                  <input
                    className="input"
                    type="number"
                    min={0}
                    value={draft.price_coins}
                    onChange={(e) =>
                      setDraft((d) => ({ ...d, price_coins: Number(e.target.value) || 0 }))
                    }
                  />
                </label>
                <label className="field" style={{ width: 220 }}>
                  <span className="field__label">ตั้งเวลาเผยแพร่ (ไม่บังคับ)</span>
                  <input
                    className="input"
                    type="datetime-local"
                    value={scheduledAt}
                    onChange={(e) => setScheduledAt(e.target.value)}
                  />
                </label>
              </div>

              {(save.isError || publish.isError) && (
                <ErrorNote message={((save.error ?? publish.error) as Error).message} />
              )}

              <div style={{ display: "flex", gap: 10, marginTop: 16, flexWrap: "wrap" }}>
                <button className="btn" onClick={() => save.mutate()} disabled={save.isPending}>
                  บันทึกร่าง
                </button>
                <button
                  className="btn btn--primary"
                  onClick={async () => {
                    await save.mutateAsync();
                    publish.mutate();
                  }}
                  disabled={publish.isPending}
                >
                  {scheduledAt ? "ตั้งเวลาเผยแพร่" : "เผยแพร่ทันที"}
                </button>
                {chapter.data?.status === "published" && <PublishedLink chapter={chapter.data} />}
              </div>
            </div>
          </div>
        </div>
      )}
    </section>
  );
}

/** The wizard's own back navigation — each completed step stays clickable. */
function Breadcrumb({
  step,
  work,
  chapterTitle,
  onPickWork,
  onPickChapter,
}: {
  step: number;
  work: WriterNovel | null;
  chapterTitle?: string;
  onPickWork: () => void;
  onPickChapter: () => void;
}) {
  if (step === 1) return null;

  return (
    <nav className="wizard-crumbs" aria-label="ขั้นตอน">
      <button className="btn btn--ghost btn--sm" onClick={onPickWork}>
        ← เลือกเรื่อง
      </button>
      {work && <span className="muted">{work.title_th}</span>}
      {step === 3 && (
        <>
          <span className="muted" aria-hidden="true">
            ·
          </span>
          <button className="btn btn--ghost btn--sm" onClick={onPickChapter}>
            เปลี่ยนบท
          </button>
          {chapterTitle && <span className="muted">{chapterTitle}</span>}
        </>
      )}
    </nav>
  );
}

/** Step 1 — search over title and series, then pick a work. */
function WorkPicker({
  works,
  series,
  onPick,
}: {
  works: WriterNovel[];
  series: WriterSeries[];
  onPick: (id: string) => void;
}) {
  const [q, setQ] = useState("");

  const seriesTitle = useMemo(() => {
    const byID = new Map(series.map((s) => [s.id, s.title]));
    return (id?: string) => (id ? (byID.get(id) ?? "") : "");
  }, [series]);

  const matches = useMemo(() => {
    const needle = q.trim().toLowerCase();
    if (!needle) return works;
    return works.filter((w) =>
      [w.title_th, w.title_cn ?? "", seriesTitle(w.series_id)]
        .join(" ")
        .toLowerCase()
        .includes(needle),
    );
  }, [q, works, seriesTitle]);

  if (works.length === 0) {
    return (
      <Empty>
        ยังไม่มีผลงานในบัญชีนี้ · <Link to="/works">เพิ่มเรื่องใหม่</Link>
      </Empty>
    );
  }

  return (
    <div style={{ marginTop: 18 }}>
      <input
        className="input"
        style={{ maxWidth: 360 }}
        value={q}
        onChange={(e) => setQ(e.target.value)}
        placeholder="ค้นหาจากชื่อเรื่องหรือชุดหนังสือ"
      />

      {matches.length === 0 ? (
        <Empty>ไม่พบผลงานที่ตรงกับคำค้น</Empty>
      ) : (
        <div className="grid grid--cards" style={{ marginTop: 16 }}>
          {matches.map((w) => (
            <button key={w.id} className="card work-pick" onClick={() => onPick(w.id)}>
              <Cover
                url={w.cover_url}
                titleCN={w.title_cn}
                style={w.cover_style}
                color={w.cover_color}
                text={w.cover_text}
                width={46}
                height={64}
              />
              <span style={{ minWidth: 0, textAlign: "left" }}>
                <span className="serif work-pick__title">{w.title_th}</span>
                <span className="muted work-pick__sub">
                  {seriesTitle(w.series_id) || "ไม่สังกัดชุด"} · แปลแล้ว{" "}
                  {numberTH(w.chapters_count)} บท
                </span>
              </span>
            </button>
          ))}
        </div>
      )}
    </div>
  );
}

/** Step 2 — search the chapter rail, or start a new chapter. */
function ChapterPicker({
  chapters,
  loading,
  creating,
  onCreate,
  onPick,
}: {
  chapters: WriterChapter[];
  loading: boolean;
  creating: boolean;
  onCreate: () => void;
  onPick: (id: string) => void;
}) {
  const [q, setQ] = useState("");

  const matches = useMemo(() => {
    const needle = q.trim().toLowerCase();
    if (!needle) return chapters;
    return chapters.filter(
      (c) => c.title.toLowerCase().includes(needle) || String(c.chapter_no).includes(needle),
    );
  }, [q, chapters]);

  if (loading) return <Loading rows={4} />;

  return (
    <div style={{ marginTop: 18 }}>
      <div style={{ display: "flex", gap: 10, flexWrap: "wrap", alignItems: "center" }}>
        <input
          className="input"
          style={{ maxWidth: 320 }}
          value={q}
          onChange={(e) => setQ(e.target.value)}
          placeholder="ค้นหาบทจากชื่อหรือเลขบท"
        />
        <button className="btn btn--primary" onClick={onCreate} disabled={creating}>
          + สร้างบทใหม่
        </button>
      </div>

      {chapters.length === 0 ? (
        <Empty>ยังไม่มีบทในเรื่องนี้ · กด “สร้างบทใหม่” เพื่อเริ่มบทแรก</Empty>
      ) : matches.length === 0 ? (
        <Empty>ไม่พบบทที่ตรงกับคำค้น</Empty>
      ) : (
        <div className="rows" style={{ marginTop: 16 }}>
          {matches.map((c) => (
            <button key={c.id} className="row" onClick={() => onPick(c.id)}>
              <span className="mono muted" style={{ minWidth: 44, fontSize: 12.5 }}>
                {c.chapter_no}
              </span>
              <span style={{ flex: 1, minWidth: 0, fontSize: 13.5, textAlign: "left" }}>
                {c.title}
              </span>
              <span className="pill">{chapterStatusLabel(c.status)}</span>
              {c.updated_at && (
                <span className="muted" style={{ fontSize: 11.5, whiteSpace: "nowrap" }}>
                  {relativeTime(c.updated_at)}
                </span>
              )}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}

function PublishedLink({ chapter }: { chapter: WriterChapter }) {
  return (
    <Link to={`/read/${chapter.id}`} className="btn btn--ghost">
      ดูหน้าอ่าน →
    </Link>
  );
}
