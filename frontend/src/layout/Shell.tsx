import { NavLink, Outlet } from "react-router-dom";

const navItemStyle: React.CSSProperties = {
  display: "flex",
  alignItems: "center",
  justifyContent: "space-between",
  textAlign: "left",
  padding: "10px 12px",
  borderRadius: 3,
  fontSize: 13.5,
  textDecoration: "none",
  color: "var(--soft)",
};

const active: React.CSSProperties = {
  background: "rgba(35,32,27,0.06)",
  color: "var(--ink)",
};

function link(to: string, label: string, badge?: string) {
  return (
    <NavLink
      to={to}
      end={to === "/"}
      style={({ isActive }) =>
        isActive ? { ...navItemStyle, ...active } : navItemStyle
      }
    >
      <span>{label}</span>
      {badge && (
        <span style={{ fontSize: 11, color: "var(--soft)" }}>{badge}</span>
      )}
    </NavLink>
  );
}

export default function Shell() {
  return (
    <div style={{ display: "flex", minHeight: "100vh", alignItems: "stretch" }}>
      <aside
        style={{
          display: "flex",
          flexDirection: "column",
          width: 244,
          flex: "0 0 244px",
          position: "sticky",
          top: 0,
          height: "100vh",
          background: "#F3EFE6",
          borderRight: "1px solid rgba(35,32,27,0.09)",
          padding: "28px 18px",
        }}
      >
        <div style={{ padding: "0 10px 26px" }}>
          <div
            className="serif"
            style={{ fontSize: 21, fontWeight: 600, letterSpacing: "0.01em" }}
          >
            หมอกจันทร์
          </div>
          <div style={{ fontSize: 11.5, color: "var(--soft)", marginTop: 3 }}>
            หอนิยายจีนแปล
          </div>
        </div>

        <nav style={{ display: "grid", gap: 2 }}>
          {link("/", "หน้าแรก")}
          {link("/browse", "หมวดหมู่และค้นหา")}
          {link("/library", "ชั้นหนังสือ", "12")}
          {link("/coins", "เหรียญ", "240")}
        </nav>

        <div
          style={{
            margin: "24px 10px 10px",
            fontSize: 10.5,
            letterSpacing: "0.18em",
            color: "var(--gold)",
            textTransform: "uppercase",
          }}
        >
          สำหรับนักเขียน
        </div>
        <nav style={{ display: "grid", gap: 2 }}>
          {link("/write", "เขียนบท")}
          {link("/stats", "สถิติผลงาน")}
        </nav>

        <div
          style={{
            marginTop: "auto",
            display: "grid",
            gap: 8,
            padding: "0 2px",
          }}
        >
          <a
            href="/read"
            style={{
              display: "block",
              padding: 12,
              textAlign: "center",
              border: "1px solid rgba(35,32,27,0.16)",
              borderRadius: 3,
              fontSize: 12.5,
              color: "var(--ink)",
            }}
          >
            เปิดหน้าอ่าน →
          </a>
        </div>
      </aside>

      <main style={{ flex: 1, minWidth: 0, padding: "44px 56px 96px" }}>
        <Outlet />
      </main>
    </div>
  );
}
