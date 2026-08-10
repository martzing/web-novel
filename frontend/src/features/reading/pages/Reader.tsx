import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Link, useNavigate, useParams, useSearchParams } from "react-router-dom";
import { useQueryClient } from "@tanstack/react-query";

import { catalogKeys, useChapters, useGlossary } from "@/features/catalog";
import { useAuth, usePrefs } from "@/features/identity";
import { useBookmarks, useCreateBookmark, useDeleteBookmark } from "@/features/library";
import { useTipChapter, useUnlockChapter, useWallet } from "@/features/wallet";
import type { GlossaryEntry } from "@/shared/api/types";
import { numberTH, relativeTime } from "@/shared/lib/format";
import { ErrorNote } from "@/shared/ui";

import { readingApi } from "../api";
import { readingKeys, useChapter } from "../queries";

type Panel = "toc" | "glossary" | "marks" | "settings" | null;

interface NotePosition {
  entry: GlossaryEntry;
  x: number;
  y: number;
}

/** Only the three content panels are worth deep-linking; settings is not. */
function initialPanel(value: string | null): Panel {
  return value === "toc" || value === "glossary" || value === "marks" ? value : null;
}

export default function Reader() {
  const { id = "" } = useParams();
  const [search] = useSearchParams();
  const { user } = useAuth();
  const navigate = useNavigate();
  const qc = useQueryClient();

  const [chrome, setChrome] = useState(true);
  // The novel page links here as /read/:id?panel=glossary, so a reader tapping
  // "อภิธานศัพท์ 27 คำ" lands with the panel already open.
  const [panel, setPanel] = useState<Panel>(() => initialPanel(search.get("panel")));
  const [note, setNote] = useState<NotePosition | null>(null);
  const [glossaryFocus, setGlossaryFocus] = useState<string | undefined>();
  const [pct, setPct] = useState(0);

  const articleRef = useRef<HTMLElement>(null);
  const hideTimer = useRef<number>();
  const saveTimer = useRef<number>();
  // One completion per chapter view, so a reader scrolling back and forth over
  // the end does not inflate the writer's อ่านจบต่อบท figure.
  const completedRef = useRef(false);

  const { prefs, setPref } = usePrefs();

  const chapter = useChapter(id);
  const novelId = chapter.data?.novel_id;

  const chapters = useChapters(novelId);
  const glossary = useGlossary(novelId);
  const bookmarks = useBookmarks(novelId, Boolean(user));
  const wallet = useWallet(Boolean(user));

  // Terms are indexed by their data-k key, which is what the rendered body uses.
  const termsByKey = useMemo(() => {
    const map = new Map<string, GlossaryEntry>();
    for (const group of glossary.data?.data ?? []) {
      for (const entry of group.entries) map.set(entry.term_key, entry);
    }
    return map;
  }, [glossary.data]);

  // The wallet hook invalidates its own balance; a freshly unlocked chapter
  // also stales this screen's body and the novel's table of contents.
  const unlock = useUnlockChapter(id, () => {
    qc.invalidateQueries({ queryKey: readingKeys.chapter(id) });
    qc.invalidateQueries({ queryKey: catalogKeys.chapterList(novelId ?? "") });
  });

  const showChrome = useCallback(() => {
    setChrome(true);
    window.clearTimeout(hideTimer.current);
    hideTimer.current = window.setTimeout(() => setChrome(false), 3400);
  }, []);

  // Record the read once per chapter view; failures are deliberately ignored
  // because analytics must never interrupt reading.
  useEffect(() => {
    if (!id) return;
    readingApi.readEvent(id).catch(() => {});
    completedRef.current = false;
    window.scrollTo(0, 0);
    showChrome();
  }, [id, showChrome]);

  // Track scroll progress and persist it, debounced.
  useEffect(() => {
    function onScroll() {
      const height = document.body.scrollHeight - window.innerHeight;
      const next = height > 0 ? Math.min(100, Math.round((window.scrollY / height) * 100)) : 0;
      setPct(next);
      setNote(null);

      // อ่านจบต่อบท. Fired once per chapter view, at the point the reader has
      // reached the end rather than on unload: an unload beacon would miss the
      // common case of tapping straight through to the next chapter. 92% rather
      // than 100% because the footer and the tip box sit below the last
      // paragraph, and nobody scrolls past the text they came to read.
      if (!completedRef.current && next >= 92) {
        completedRef.current = true;
        readingApi.readEvent(id, true).catch(() => {});
      }

      if (!user || !novelId || !chapter.data) return;
      window.clearTimeout(saveTimer.current);
      saveTimer.current = window.setTimeout(() => {
        readingApi
          .saveProgress(novelId, {
            last_chapter_id: chapter.data!.id,
            last_chapter_no: chapter.data!.chapter_no,
            para_anchor: currentParagraph(articleRef.current),
            pct: next,
          })
          .catch(() => {});
      }, 600);
    }

    window.addEventListener("scroll", onScroll, { passive: true });
    return () => {
      window.removeEventListener("scroll", onScroll);
      window.clearTimeout(saveTimer.current);
    };
  }, [id, user, novelId, chapter.data]);

  /**
   * Immersive tap: the top and bottom eighths reveal the chrome, the middle
   * toggles it. Taps on interactive elements and text selections are ignored.
   */
  function onSurfaceTap(e: React.MouseEvent) {
    const target = e.target as HTMLElement;
    if (target.closest("a, button, input, [data-k]")) return;
    if (String(window.getSelection() ?? "")) return;

    const y = e.clientY;
    const h = window.innerHeight;
    if (y < h * 0.14 || y > h * 0.86) {
      showChrome();
      return;
    }
    if (note) {
      setNote(null);
      return;
    }
    setChrome((on) => !on);
  }

  /** Opens the glossary popover for an inline term. */
  function onBodyClick(e: React.MouseEvent) {
    const span = (e.target as HTMLElement).closest("[data-k]");
    if (!span) return;
    e.stopPropagation();

    const key = span.getAttribute("data-k");
    const entry = key ? termsByKey.get(key) : undefined;
    if (!entry) return;

    const rect = span.getBoundingClientRect();
    const width = Math.min(320, window.innerWidth - 32);
    const x = Math.max(16, Math.min(window.innerWidth - width - 16, rect.left + rect.width / 2 - width / 2));
    const below = rect.bottom + 12;
    const y = below + 220 > window.innerHeight ? Math.max(16, rect.top - 232) : below;

    setNote({ entry, x, y });
  }

  const createBookmark = useCreateBookmark(novelId ?? "");
  const removeBookmark = useDeleteBookmark(novelId ?? "");

  const addBookmark = () => {
    const paragraph = currentParagraph(articleRef.current);
    const text = paragraphText(articleRef.current, paragraph);
    createBookmark.mutate({
      novel_id: novelId!,
      chapter_id: id,
      para_anchor: paragraph,
      excerpt: text || chapter.data!.title,
    });
  };

  if (chapter.isLoading) {
    return <div className="empty">กำลังโหลด…</div>;
  }
  if (chapter.isError) {
    return <ErrorNote message={(chapter.error as Error).message} />;
  }
  if (!chapter.data) return null;

  const c = chapter.data;

  return (
    <div
      className={`reader${chrome ? " is-chrome" : ""}`}
      data-theme={prefs.theme}
      style={
        {
          "--reader-font": readerFont(prefs.font),
          "--reader-size": `${prefs.font_size}px`,
          "--reader-leading": String(prefs.line_height),
          "--reader-width": readerWidth(prefs.column_width),
        } as React.CSSProperties
      }
      onClick={onSurfaceTap}
    >
      <div className="reader__progress">
        <div className="reader__progress-bar" style={{ width: `${pct}%` }} />
      </div>

      <header className="reader__chrome reader__header">
        <Link to={`/novels/${c.novel_slug}`} className="reader__tool">
          ‹ หน้าแรก
        </Link>
        <div className="reader__titles">
          <div className="reader__novel">{c.novel_title_th}</div>
          <div className="reader__crumb">
            {c.arc_name ? `ภาคที่ ${c.arc_no} · ${c.arc_name} · ` : ""}บทที่ {c.chapter_no}
          </div>
        </div>
        <nav className="reader__tools">
          <button className="reader__tool" onClick={() => setPanel("marks")}>
            ที่คั่น
          </button>
          <button className="reader__tool" onClick={() => setPanel("toc")}>
            สารบัญ
          </button>
          <button className="reader__tool" onClick={() => setPanel("glossary")}>
            อภิธานศัพท์
          </button>
          <button className="reader__tool" onClick={() => setPanel("settings")}>
            ตั้งค่า
          </button>
        </nav>
      </header>

      <article className="reader__article" ref={articleRef}>
        <div className="reader__chapter-head">
          <div className="eyebrow">บทที่ {c.chapter_no}</div>
          <h1 className="reader__chapter-title">{c.title}</h1>
          <div className="reader__byline">
            {c.word_count > 0 ? `${numberTH(c.word_count)} คำ` : ""}
          </div>
        </div>

        {c.locked && c.locked_reason === "early_access" ? (
          // Early access is not a paywall: no amount of coins opens this yet,
          // so offering a purchase button would just produce a 403.
          <div className="paywall">
            <div className="eyebrow">เปิดให้อ่านก่อนใคร</div>
            <h2 style={{ fontSize: 21, marginTop: 10 }}>
              บทที่ {c.chapter_no} — {c.title}
            </h2>
            <div className="muted" style={{ fontSize: 13, marginTop: 14, lineHeight: 1.9 }}>
              บทนี้เปิดให้ผู้ที่เปิดปลดล็อกอัตโนมัติอ่านก่อน 24 ชั่วโมง
              จากนั้นจะเปิดให้ทุกคนโดยอัตโนมัติ
            </div>
            <div style={{ marginTop: 18, display: "grid", gap: 10 }}>
              <Link to={`/novels/${c.novel_slug}`} className="btn btn--primary btn--block">
                เปิดปลดล็อกอัตโนมัติ
              </Link>
              {c.prev_id && (
                <Link to={`/read/${c.prev_id}`} className="btn btn--block">
                  ← กลับไปบทก่อนหน้า
                </Link>
              )}
            </div>
          </div>
        ) : c.locked ? (
          <div className="paywall">
            <div className="eyebrow">บทนี้ยังไม่ปลดล็อก</div>
            <h2 style={{ fontSize: 21, marginTop: 10 }}>
              บทที่ {c.chapter_no} — {c.title}
            </h2>
            <div className="muted" style={{ fontSize: 13, marginTop: 14, lineHeight: 1.9 }}>
              {user ? (
                <>
                  ยอดคงเหลือ <span className="mono">{wallet.data?.total ?? 0}</span> เหรียญ
                </>
              ) : (
                "เข้าสู่ระบบเพื่อปลดล็อกบทนี้"
              )}
            </div>

            {unlock.isError && (
              <div className="form-error" style={{ marginTop: 12 }}>
                {unlockMessage((unlock.error as Error & { code?: string }).code)}
              </div>
            )}

            <div style={{ marginTop: 18, display: "grid", gap: 10 }}>
              {user ? (
                <>
                  <button
                    className="btn btn--primary btn--block"
                    onClick={() => unlock.mutate()}
                    disabled={unlock.isPending}
                  >
                    ปลดล็อกบทนี้ {c.price_coins} เหรียญ
                  </button>
                  <Link to="/coins" className="btn btn--block">
                    เติมเหรียญ
                  </Link>
                </>
              ) : (
                <Link to="/login" className="btn btn--primary btn--block">
                  เข้าสู่ระบบ
                </Link>
              )}
            </div>
          </div>
        ) : (
          <>
            <div
              className="chapter-body"
              onClick={onBodyClick}
              dangerouslySetInnerHTML={{ __html: c.body_html ?? "" }}
            />
            <div className="reader__end">
              <div className="eyebrow">จบบทที่ {c.chapter_no}</div>
              {c.next_id && (
                <Link
                  to={`/read/${c.next_id}`}
                  className="btn btn--primary"
                  style={{ marginTop: 16 }}
                >
                  อ่านบทถัดไป →
                </Link>
              )}
              <div style={{ marginTop: 16 }}>
                <Link to={`/chapters/${c.id}/comments`} className="btn">
                  ความเห็นของบทนี้
                </Link>
              </div>
              {c.tips_enabled && user && <TipControl chapterId={c.id} />}
            </div>
          </>
        )}
      </article>

      <footer className="reader__chrome reader__footer">
        <button
          className="reader__tool"
          disabled={!c.prev_id}
          onClick={() => c.prev_id && navigate(`/read/${c.prev_id}`)}
        >
          ← บทก่อนหน้า
        </button>
        <span className="mono muted" style={{ fontSize: 12 }}>
          {/* Against the translated count, not a bare number: the reader is
              seeing how far through the Thai text they are, and the source may
              run hundreds of chapters ahead. */}
          {c.chapter_no} / {numberTH(chapters.data?.data.length ?? c.chapter_no)} บทที่แปลแล้ว ·{" "}
          {pct}%
        </span>
        <button
          className="reader__tool"
          disabled={!c.next_id}
          onClick={() => c.next_id && navigate(`/read/${c.next_id}`)}
        >
          บทถัดไป →
        </button>
      </footer>

      {note && (
        <div className="note" style={{ left: note.x, top: note.y }} onClick={(e) => e.stopPropagation()}>
          {note.entry.kind && <div className="note__kind">{note.entry.kind}</div>}
          <div className="note__title">{note.entry.title_th}</div>
          {note.entry.title_cn && <div className="note__cn">{note.entry.title_cn}</div>}
          <div className="note__body">{note.entry.body}</div>
          <div style={{ display: "flex", gap: 8, marginTop: 10, flexWrap: "wrap" }}>
            {/* A term popover answers "what is this word"; the panel answers
                "what else is like it". The reader needs a way from one to the
                other without hunting for the toolbar. */}
            <button
              className="btn btn--ghost btn--sm"
              onClick={() => {
                setNote(null);
                setPanel("glossary");
                setGlossaryFocus(note.entry.term_key);
              }}
            >
              ดูในอภิธานศัพท์ →
            </button>
            <button className="btn btn--ghost btn--sm" onClick={() => setNote(null)}>
              ปิด
            </button>
          </div>
        </div>
      )}

      {panel && (
        <div className="panel" onClick={(e) => e.stopPropagation()}>
          <div className="panel__head">
            <div className="panel__title">
              {panel === "toc" && "สารบัญ"}
              {panel === "glossary" && "อภิธานศัพท์"}
              {panel === "marks" && "ที่คั่นหนังสือ"}
              {panel === "settings" && "การอ่าน"}
            </div>
            <button className="btn btn--ghost btn--sm" onClick={() => setPanel(null)}>
              ปิด
            </button>
          </div>

          <div className="panel__body">
            {panel === "toc" && (
              <TocPanel
                chapters={chapters.data?.data ?? []}
                currentId={c.id}
                onPick={(chapterId) => {
                  setPanel(null);
                  navigate(`/read/${chapterId}`);
                }}
              />
            )}

            {panel === "glossary" && (
              <GlossaryPanel groups={glossary.data?.data ?? []} focusTermKey={glossaryFocus} />
            )}

            {panel === "marks" && (
              <MarksPanel
                bookmarks={bookmarks.data?.data ?? []}
                canAdd={Boolean(user) && !c.locked}
                onAdd={addBookmark}
                onDelete={(bookmarkId) => removeBookmark.mutate(bookmarkId)}
                onGo={(chapterId) => {
                  setPanel(null);
                  navigate(`/read/${chapterId}`);
                }}
              />
            )}

            {panel === "settings" && <SettingsPanel prefs={prefs} setPref={setPref} />}
          </div>
        </div>
      )}
    </div>
  );
}

function TocPanel({
  chapters,
  currentId,
  onPick,
}: {
  chapters: { id: string; chapter_no: number; title: string; price_coins: number; unlocked: boolean }[];
  currentId: string;
  onPick: (id: string) => void;
}) {
  const [query, setQuery] = useState("");
  const filtered = chapters.filter(
    (c) => !query || c.title.includes(query) || String(c.chapter_no).includes(query),
  );

  return (
    <>
      <input
        className="input"
        value={query}
        onChange={(e) => setQuery(e.target.value)}
        placeholder="ค้นหาบท"
        style={{ marginBottom: 14 }}
      />
      {filtered.map((c) => (
        <button
          key={c.id}
          className={`panel__item${c.id === currentId ? " is-current" : ""}`}
          onClick={() => onPick(c.id)}
        >
          <span className="panel__num">{c.chapter_no}</span>
          <span style={{ flex: 1 }}>{c.title}</span>
          {/* A tag rather than a bare ◎: "อ่านฟรี" and "ปลดล็อกแล้ว" are as
              worth knowing as the price when scanning a long table. */}
          {c.price_coins === 0 ? (
            <span className="pill">อ่านฟรี</span>
          ) : c.unlocked ? (
            <span className="pill pill--gold">ปลดล็อกแล้ว</span>
          ) : (
            <span className="pill pill--red">◎ {c.price_coins}</span>
          )}
        </button>
      ))}
    </>
  );
}

/** ทิปท้ายบท — a direct payment to the translator, paid coins only. */
const TIP_AMOUNTS = [5, 10, 20, 50];

function TipControl({ chapterId }: { chapterId: string }) {
  const [coins, setCoins] = useState(10);
  const [done, setDone] = useState(false);

  const tip = useTipChapter(chapterId, coins, () => setDone(true));

  if (done) {
    return (
      <div className="tip-box">
        <div className="muted" style={{ fontSize: 13 }}>
          ขอบคุณที่สนับสนุนผู้แปล · ส่งไปแล้ว {coins} เหรียญ
        </div>
      </div>
    );
  }

  return (
    <div className="tip-box">
      <div style={{ fontSize: 13.5 }}>ชอบบทนี้ไหม ส่งทิปให้ผู้แปลได้</div>
      <div className="chips" style={{ marginTop: 12, justifyContent: "center" }}>
        {TIP_AMOUNTS.map((amount) => (
          <button
            key={amount}
            className={`chip${coins === amount ? " is-active" : ""}`}
            onClick={() => setCoins(amount)}
          >
            ◎ {amount}
          </button>
        ))}
      </div>
      {tip.isError && (
        <div className="form-error" style={{ marginTop: 12 }}>
          {tipMessage((tip.error as Error & { code?: string }).code, (tip.error as Error).message)}
        </div>
      )}
      <button
        className="btn btn--accent"
        style={{ marginTop: 14 }}
        disabled={tip.isPending}
        onClick={() => tip.mutate()}
      >
        ส่งทิป ◎ {coins}
      </button>
    </div>
  );
}

function tipMessage(code: string | undefined, fallback: string): string {
  switch (code) {
    case "INSUFFICIENT_PAID_COINS":
      // Distinct from a plain shortfall: bonus coins cannot fund a tip, and a
      // reader looking at a bonus balance deserves to be told why.
      return "ทิปใช้ได้เฉพาะเหรียญที่ซื้อเอง (ไม่รวมเหรียญโบนัส) · เติมเหรียญก่อนแล้วลองใหม่";
    case "CANNOT_TIP_SELF":
      return "ให้ทิปผลงานของตัวเองไม่ได้";
    case "TIPS_DISABLED":
      return "เรื่องนี้ปิดรับทิป";
    default:
      return fallback;
  }
}

function GlossaryPanel({
  groups,
  focusTermKey,
}: {
  groups: { id: string; name: string; entries: GlossaryEntry[] }[];
  focusTermKey?: string;
}) {
  const [query, setQuery] = useState("");
  const focusRef = useRef<HTMLDivElement>(null);

  // Arriving from a term popover, scroll the entry into view and mark it, so
  // "ดูในอภิธานศัพท์" lands on the word rather than the top of a long list.
  useEffect(() => {
    if (focusTermKey) focusRef.current?.scrollIntoView({ block: "center" });
  }, [focusTermKey, groups]);

  const filtered = groups
    .map((g) => ({
      ...g,
      entries: g.entries.filter(
        (e) =>
          !query ||
          e.title_th.includes(query) ||
          e.body.includes(query) ||
          (e.title_cn ?? "").includes(query),
      ),
    }))
    .filter((g) => g.entries.length > 0);

  return (
    <>
      <input
        className="input"
        value={query}
        onChange={(e) => setQuery(e.target.value)}
        placeholder="ค้นหาศัพท์"
        style={{ marginBottom: 14 }}
      />
      {filtered.length === 0 && <div className="empty">ไม่พบศัพท์ที่ค้นหา</div>}
      {filtered.map((group) => (
        <div key={group.id} className="panel__group">
          <div className="panel__group-name">{group.name}</div>
          {group.entries.map((entry) => (
            <div
              key={entry.id}
              ref={entry.term_key === focusTermKey ? focusRef : undefined}
              className={entry.term_key === focusTermKey ? "glossary-entry is-focused" : "glossary-entry"}
            >
              <div style={{ fontSize: 14 }}>
                {entry.title_th}
                {entry.title_cn && <span className="muted"> {entry.title_cn}</span>}
              </div>
              <div className="muted" style={{ fontSize: 12.5, lineHeight: 1.9, marginTop: 3 }}>
                {entry.body}
              </div>
            </div>
          ))}
        </div>
      ))}
    </>
  );
}

function MarksPanel({
  bookmarks,
  canAdd,
  onAdd,
  onDelete,
  onGo,
}: {
  bookmarks: { id: string; chapter_id: string; chapter_no: number; title: string; excerpt: string; created_at: string }[];
  canAdd: boolean;
  onAdd: () => void;
  onDelete: (id: string) => void;
  onGo: (chapterId: string) => void;
}) {
  return (
    <>
      {canAdd && (
        <button className="btn btn--block" onClick={onAdd} style={{ marginBottom: 16 }}>
          คั่นหน้าไว้ตรงนี้
        </button>
      )}
      {bookmarks.length === 0 && <div className="empty">ยังไม่มีที่คั่น</div>}
      {bookmarks.map((m) => (
        <div key={m.id} style={{ padding: "12px 8px", borderBottom: "1px solid var(--line)" }}>
          <button className="panel__item" onClick={() => onGo(m.chapter_id)}>
            <span className="panel__num">{m.chapter_no}</span>
            <span style={{ flex: 1 }}>{m.title}</span>
          </button>
          <div className="muted" style={{ fontSize: 12.5, lineHeight: 1.85, padding: "0 8px" }}>
            {m.excerpt}
          </div>
          <div
            style={{
              display: "flex",
              justifyContent: "space-between",
              alignItems: "center",
              padding: "6px 8px 0",
            }}
          >
            <span className="muted" style={{ fontSize: 11.5 }}>
              {relativeTime(m.created_at)}
            </span>
            <button className="btn btn--ghost btn--sm" onClick={() => onDelete(m.id)}>
              ลบ
            </button>
          </div>
        </div>
      ))}
    </>
  );
}

function SettingsPanel({
  prefs,
  setPref,
}: {
  prefs: ReturnType<typeof usePrefs>["prefs"];
  setPref: ReturnType<typeof usePrefs>["setPref"];
}) {
  return (
    <>
      <Setting
        label="ธีม"
        options={[
          ["light", "สว่าง"],
          ["sepia", "กระดาษ"],
          ["dark", "มืด"],
        ]}
        value={prefs.theme}
        onPick={(v) => setPref("theme", v as typeof prefs.theme)}
      />
      <Setting
        label="ฟอนต์"
        options={[
          ["loop", "มีหัว"],
          ["serif", "เซริฟ"],
          ["sans", "ไม่มีหัว"],
        ]}
        value={prefs.font}
        onPick={(v) => setPref("font", v as typeof prefs.font)}
      />

      <div className="setting">
        <div className="setting__label">ขนาด</div>
        <div className="setting__stepper">
          <button
            className="setting__step"
            onClick={() => setPref("font_size", Math.max(14, prefs.font_size - 1))}
          >
            A−
          </button>
          <span className="setting__value">{prefs.font_size}px</span>
          <button
            className="setting__step"
            onClick={() => setPref("font_size", Math.min(28, prefs.font_size + 1))}
          >
            A+
          </button>
        </div>
      </div>

      <div className="setting">
        <div className="setting__label">ระยะบรรทัด</div>
        <div className="setting__stepper">
          <button
            className="setting__step"
            onClick={() => setPref("line_height", round1(Math.max(1.4, prefs.line_height - 0.1)))}
          >
            −
          </button>
          <span className="setting__value">{prefs.line_height.toFixed(1)}</span>
          <button
            className="setting__step"
            onClick={() => setPref("line_height", round1(Math.min(2.4, prefs.line_height + 0.1)))}
          >
            +
          </button>
        </div>
      </div>

      <Setting
        label="ความกว้างคอลัมน์"
        options={[
          ["narrow", "แคบ"],
          ["normal", "ปกติ"],
          ["wide", "กว้าง"],
        ]}
        value={prefs.column_width}
        onPick={(v) => setPref("column_width", v as typeof prefs.column_width)}
      />
    </>
  );
}

function Setting({
  label,
  options,
  value,
  onPick,
}: {
  label: string;
  options: [string, string][];
  value: string;
  onPick: (value: string) => void;
}) {
  return (
    <div className="setting">
      <div className="setting__label">{label}</div>
      <div className="setting__options">
        {options.map(([key, text]) => (
          <button
            key={key}
            className={`setting__option${value === key ? " is-active" : ""}`}
            onClick={() => onPick(key)}
          >
            {text}
          </button>
        ))}
      </div>
    </div>
  );
}

function readerFont(font: string): string {
  switch (font) {
    case "serif":
      return '"Noto Serif Thai"';
    case "sans":
      return '"IBM Plex Sans Thai"';
    default:
      return '"Sarabun"';
  }
}

function readerWidth(width: string): string {
  switch (width) {
    case "narrow":
      return "540px";
    case "wide":
      return "760px";
    default:
      return "640px";
  }
}

function round1(v: number): number {
  return Math.round(v * 10) / 10;
}

/** Index of the paragraph nearest the top of the viewport. */
function currentParagraph(article: HTMLElement | null): number {
  if (!article) return 0;
  const paragraphs = Array.from(article.querySelectorAll("p"));
  for (let i = 0; i < paragraphs.length; i++) {
    if (paragraphs[i].getBoundingClientRect().bottom > 120) return i;
  }
  return Math.max(paragraphs.length - 1, 0);
}

function paragraphText(article: HTMLElement | null, index: number): string {
  if (!article) return "";
  const paragraphs = article.querySelectorAll("p");
  return paragraphs[index]?.textContent?.slice(0, 160) ?? "";
}

function unlockMessage(code?: string): string {
  switch (code) {
    case "INSUFFICIENT_COINS":
      return "เหรียญไม่พอ กรุณาเติมเหรียญก่อน";
    case "CHAPTER_ALREADY_UNLOCKED":
      return "คุณปลดล็อกบทนี้ไว้แล้ว ลองรีเฟรชหน้านี้";
    case "CHAPTER_NOT_FOR_SALE":
      return "บทนี้อ่านฟรี ไม่ต้องปลดล็อก";
    default:
      return "ปลดล็อกไม่สำเร็จ กรุณาลองใหม่";
  }
}
