import { NavLink, Outlet, useNavigate } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";

import { api } from "../lib/api";
import { useAuth } from "../lib/auth";

interface NavEntry {
  to: string;
  label: string;
  icon: string;
  badge?: string;
}

/**
 * The application shell.
 *
 * One nav definition drives three presentations: the full sidebar on desktop,
 * an icon rail on tablets, and a bottom tab bar on phones. Which one shows is
 * decided entirely in CSS, so there is no viewport JavaScript to get wrong.
 */
export default function Shell() {
  const { user, isTranslator, logout } = useAuth();
  const navigate = useNavigate();

  const wallet = useQuery({
    queryKey: ["wallet"],
    queryFn: api.getWallet,
    enabled: Boolean(user),
    staleTime: 30_000,
  });

  const shelf = useQuery({
    queryKey: ["shelf", "counts"],
    queryFn: () => api.getShelf(),
    enabled: Boolean(user),
    staleTime: 60_000,
  });

  const readerNav: NavEntry[] = [
    { to: "/", label: "หน้าแรก", icon: "☰" },
    { to: "/browse", label: "หมวดหมู่และค้นหา", icon: "⌕" },
    {
      to: "/library",
      label: "ชั้นหนังสือ",
      icon: "▤",
      badge: shelf.data ? String(shelf.data.counts.total) : undefined,
    },
    {
      to: "/coins",
      label: "เหรียญ",
      icon: "◎",
      badge: wallet.data ? String(wallet.data.total) : undefined,
    },
  ];

  const writerNav: NavEntry[] = [
    { to: "/write", label: "เขียนบท", icon: "✎" },
    { to: "/stats", label: "สถิติผลงาน", icon: "◔" },
  ];

  const tabs: NavEntry[] = [
    ...readerNav,
    isTranslator ? writerNav[0] : { to: user ? "/account" : "/login", label: user ? "บัญชี" : "เข้าสู่ระบบ", icon: "◍" },
  ];

  return (
    <div className="shell">
      <aside className="sidebar">
        <div className="sidebar__brand">
          <div className="sidebar__title">หมอกจันทร์</div>
          <div className="sidebar__sub">หอนิยายจีนแปล</div>
        </div>

        <nav className="sidebar__nav">
          {readerNav.map((entry) => (
            <NavItem key={entry.to} entry={entry} />
          ))}
        </nav>

        {isTranslator && (
          <>
            <div className="sidebar__section eyebrow" style={{ margin: "24px 10px 10px" }}>
              สำหรับนักเขียน
            </div>
            <nav className="sidebar__nav">
              {writerNav.map((entry) => (
                <NavItem key={entry.to} entry={entry} />
              ))}
            </nav>
          </>
        )}

        <div className="sidebar__foot">
          {user ? (
            <>
              <NavLink to="/onboarding" className="navitem">
                <span className="navitem__label">ตั้งค่าแนวที่ชอบ</span>
              </NavLink>
              <button
                className="navitem"
                onClick={async () => {
                  await logout();
                  navigate("/");
                }}
              >
                <span className="navitem__label">ออกจากระบบ · {user.display_name}</span>
              </button>
            </>
          ) : (
            <NavLink to="/login" className="btn btn--block">
              เข้าสู่ระบบ
            </NavLink>
          )}
        </div>
      </aside>

      <div style={{ flex: 1, minWidth: 0, display: "flex", flexDirection: "column" }}>
        <header className="topbar">
          <div className="topbar__title">หมอกจันทร์</div>
          {user ? (
            <span className="mono muted" style={{ fontSize: 12 }}>
              ◎ {wallet.data?.total ?? 0}
            </span>
          ) : (
            <NavLink to="/login" className="btn btn--sm">
              เข้าสู่ระบบ
            </NavLink>
          )}
        </header>

        <main className="shell__main">
          <div className="shell__inner">
            <Outlet />
          </div>
        </main>
      </div>

      <nav className="tabbar" aria-label="เมนูหลัก">
        {tabs.map((entry) => (
          <NavLink
            key={entry.to}
            to={entry.to}
            end={entry.to === "/"}
            className={({ isActive }) => `tabbar__item${isActive ? " is-active" : ""}`}
          >
            <span className="tabbar__icon" aria-hidden="true">
              {entry.icon}
            </span>
            <span>{entry.label}</span>
          </NavLink>
        ))}
      </nav>
    </div>
  );
}

function NavItem({ entry }: { entry: NavEntry }) {
  return (
    <NavLink
      to={entry.to}
      end={entry.to === "/"}
      className={({ isActive }) => `navitem${isActive ? " is-active" : ""}`}
    >
      <span className="navitem__icon" aria-hidden="true">
        {entry.icon}
      </span>
      <span className="navitem__label">{entry.label}</span>
      {entry.badge && <span className="navitem__badge">{entry.badge}</span>}
    </NavLink>
  );
}
