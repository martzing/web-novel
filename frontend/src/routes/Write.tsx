import { useEffect, useRef, useState } from "react";
import { Link } from "react-router-dom";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { api, type WriterChapter } from "../lib/api";
import { useAuth } from "../lib/auth";
import { chapterStatusLabel, numberTH, relativeTime } from "../lib/format";
import { Empty, ErrorNote, Loading } from "../components";

const AUTOSAVE_MS = 20_000;

export default function Write() {
  const { user, isTranslator } = useAuth();
  const qc = useQueryClient();

  const [novelId, setNovelId] = useState<string>("");
  const [chapterId, setChapterId] = useState<string>("");
  const [draft, setDraft] = useState({ chapter_no: 1, title: "", body_source: "", price_coins: 0 });
  const [savedAt, setSavedAt] = useState<string | null>(null);
  const [scheduledAt, setScheduledAt] = useState("");

  const bodyRef = useRef<HTMLTextAreaElement>(null);
  const autosave = useRef<number>();

  const novels = useQuery({
    queryKey: ["writer", "novels"],
    queryFn: api.listWriterNovels,
    enabled: isTranslator,
  });

  // Default to the first novel once the list arrives.
  useEffect(() => {
    if (!novelId && novels.data?.data.length) setNovelId(novels.data.data[0].id);
  }, [novels.data, novelId]);

  const chapters = useQuery({
    queryKey: ["writer", "chapters", novelId],
    queryFn: () => api.listWriterChapters(novelId),
    enabled: Boolean(novelId),
  });

  const chapter = useQuery({
    queryKey: ["writer", "chapter", chapterId],
    queryFn: () => api.getWriterChapter(chapterId),
    enabled: Boolean(chapterId),
  });

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
    mutationFn: () => api.saveWriterChapter(chapterId, draft),
    onSuccess: (saved) => {
      setSavedAt(saved.updated_at ?? new Date().toISOString());
      qc.invalidateQueries({ queryKey: ["writer", "chapters", novelId] });
    },
  });

  const create = useMutation({
    mutationFn: () => {
      const nextNo = (chapters.data?.data[0]?.chapter_no ?? 0) + 1;
      return api.createWriterChapter(novelId, {
        chapter_no: nextNo,
        title: `บทที่ ${nextNo}`,
        body_source: "",
        price_coins: 0,
      });
    },
    onSuccess: (created) => {
      qc.invalidateQueries({ queryKey: ["writer", "chapters", novelId] });
      setChapterId(created.id);
    },
  });

  const publish = useMutation({
    mutationFn: () =>
      api.publishChapter(chapterId, scheduledAt ? new Date(scheduledAt).toISOString() : undefined),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["writer", "chapters", novelId] });
      qc.invalidateQueries({ queryKey: ["writer", "chapter", chapterId] });
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

  const rail = chapters.data?.data ?? [];

  return (
    <section>
      <div className="page-head">
        <h1 className="page-title">เขียนบท</h1>
        <div className="muted" style={{ fontSize: 12 }}>
          {save.isPending
            ? "กำลังบันทึก…"
            : savedAt
              ? `บันทึกอัตโนมัติ ${relativeTime(savedAt)}`
              : "ยังไม่ได้บันทึก"}
        </div>
      </div>

      {novels.isLoading ? (
        <Loading rows={2} />
      ) : rail.length === 0 && (novels.data?.data.length ?? 0) === 0 ? (
        <Empty>ยังไม่มีผลงานในบัญชีนี้</Empty>
      ) : (
        <>
          {(novels.data?.data.length ?? 0) > 1 && (
            <select
              className="select"
              style={{ marginTop: 18, maxWidth: 320 }}
              value={novelId}
              onChange={(e) => {
                setNovelId(e.target.value);
                setChapterId("");
              }}
            >
              {novels.data!.data.map((n) => (
                <option key={n.id} value={n.id}>
                  {n.title_th}
                </option>
              ))}
            </select>
          )}

          <div className="writer">
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
                {rail.map((c) => (
                  <button
                    key={c.id}
                    className={`panel__item${c.id === chapterId ? " is-current" : ""}`}
                    onClick={() => setChapterId(c.id)}
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
              {!chapterId ? (
                <Empty>เลือกบทจากรายการ หรือสร้างบทใหม่</Empty>
              ) : (
                <>
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
                    <button
                      className="btn btn--sm"
                      onClick={() => wrapSelection("\n\n<hr/>\n\n", "")}
                    >
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
                      <ErrorNote
                        message={((save.error ?? publish.error) as Error).message}
                      />
                    )}

                    <div style={{ display: "flex", gap: 10, marginTop: 16, flexWrap: "wrap" }}>
                      <button
                        className="btn"
                        onClick={() => save.mutate()}
                        disabled={save.isPending}
                      >
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
                      {chapter.data?.status === "published" && (
                        <PublishedLink chapter={chapter.data} />
                      )}
                    </div>
                  </div>
                </>
              )}
            </div>
          </div>
        </>
      )}
    </section>
  );
}

function PublishedLink({ chapter }: { chapter: WriterChapter }) {
  return (
    <Link to={`/read/${chapter.id}`} className="btn btn--ghost">
      ดูหน้าอ่าน →
    </Link>
  );
}
