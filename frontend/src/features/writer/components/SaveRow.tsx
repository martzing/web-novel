import type { ReactNode } from "react";

/** The shared save affordance every tab ends with. */
export function SaveRow({
  saving,
  error,
  saved,
  children,
}: {
  saving: boolean;
  error: unknown;
  saved: boolean;
  children?: ReactNode;
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
