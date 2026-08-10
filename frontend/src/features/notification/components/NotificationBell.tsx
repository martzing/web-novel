import { useEffect, useRef, useState } from "react";
import { useNavigate } from "react-router-dom";

import { relativeTime } from "@/shared/lib/format";

import type { Notification } from "../api";
import { useMarkNotificationsRead, useNotifications, useUnreadCount } from "../queries";

/**
 * The reader inbox, as a bell in the shell.
 *
 * The API and the publish-time fan-out have existed since phase 4; this is the
 * surface that finally makes R-17 true for a reader rather than only for the
 * database. The design mocks never drew it, so it borrows the sidebar's own
 * panel treatment instead of inventing a look.
 */
export default function NotificationBell() {
  const [open, setOpen] = useState(false);
  const navigate = useNavigate();
  const wrapRef = useRef<HTMLDivElement>(null);

  // The poll is long enough not to hit the API on every render, short enough
  // that a new chapter shows up while the reader is still on the page.
  const unread = useUnreadCount(true, 120_000);

  // The list is only worth fetching once the panel is actually open.
  const list = useNotifications(open);

  const markRead = useMarkNotificationsRead();

  // Close on an outside click or Escape. Without this the panel survives
  // navigation and covers the page it just moved to.
  useEffect(() => {
    if (!open) return;

    function onPointerDown(e: MouseEvent) {
      if (!wrapRef.current?.contains(e.target as Node)) setOpen(false);
    }
    function onKeyDown(e: KeyboardEvent) {
      if (e.key === "Escape") setOpen(false);
    }

    document.addEventListener("mousedown", onPointerDown);
    document.addEventListener("keydown", onKeyDown);
    return () => {
      document.removeEventListener("mousedown", onPointerDown);
      document.removeEventListener("keydown", onKeyDown);
    };
  }, [open]);

  const count = unread.data?.unread ?? 0;
  const items = list.data?.data ?? [];

  function openNotification(item: Notification) {
    setOpen(false);
    if (!item.read) markRead.mutate([Number(item.id)]);
    const to = destinationOf(item);
    if (to) navigate(to);
  }

  return (
    <div className="bell" ref={wrapRef}>
      <button
        className="bell__button"
        aria-label={count > 0 ? `การแจ้งเตือน ${count} รายการที่ยังไม่อ่าน` : "การแจ้งเตือน"}
        aria-expanded={open}
        onClick={() => setOpen((v) => !v)}
      >
        <span aria-hidden="true">✻</span>
        {count > 0 && <span className="bell__badge">{count > 99 ? "99+" : count}</span>}
      </button>

      {open && (
        <div className="bell__panel">
          <div className="bell__head">
            <span className="eyebrow">การแจ้งเตือน</span>
            {count > 0 && (
              <button
                className="bell__markall"
                onClick={() => markRead.mutate([])}
                disabled={markRead.isPending}
              >
                อ่านแล้วทั้งหมด
              </button>
            )}
          </div>

          {list.isLoading ? (
            <div className="bell__empty">กำลังโหลด…</div>
          ) : items.length === 0 ? (
            <div className="bell__empty">
              ยังไม่มีการแจ้งเตือน · ติดตามเรื่องที่ชอบไว้ แล้วเราจะบอกเมื่อมีบทใหม่
            </div>
          ) : (
            <ul className="bell__list">
              {items.map((item) => (
                <li key={item.id}>
                  <button
                    className={`bell__item${item.read ? "" : " is-unread"}`}
                    onClick={() => openNotification(item)}
                  >
                    <span className="bell__text">{describe(item)}</span>
                    <span className="bell__when">{relativeTime(item.created_at)}</span>
                  </button>
                </li>
              ))}
            </ul>
          )}
        </div>
      )}
    </div>
  );
}

/** Where a notification takes the reader when they tap it. */
function destinationOf(item: Notification): string | null {
  const chapterID = numeric(item.payload.chapter_id);

  switch (item.kind) {
    case "new_chapter":
      return chapterID ? `/read/${chapterID}` : null;
    case "reply":
      return chapterID ? `/chapters/${chapterID}/comments` : null;
    case "auto_unlock_failed":
    case "bonus_expiring":
      // Both are about the wallet, so both land where the reader can act on it.
      return "/coins";
    default:
      return null;
  }
}

function describe(item: Notification): string {
  switch (item.kind) {
    case "new_chapter":
      return "มีบทใหม่ในเรื่องที่คุณติดตาม";
    case "reply":
      return "มีคนตอบกลับความเห็นของคุณ";
    case "auto_unlock_failed":
      return "เหรียญไม่พอสำหรับปลดล็อกอัตโนมัติ · เติมเหรียญเพื่ออ่านต่อ";
    case "bonus_expiring":
      return "เหรียญโบนัสของคุณใกล้หมดอายุ";
    default:
      return "มีการแจ้งเตือนใหม่";
  }
}

/** Payload ids arrive as JSON numbers; anything else is not a usable id. */
function numeric(value: unknown): string | null {
  if (typeof value === "number" && Number.isFinite(value)) return String(value);
  if (typeof value === "string" && value !== "") return value;
  return null;
}
