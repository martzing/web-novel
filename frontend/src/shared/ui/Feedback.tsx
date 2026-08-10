import type { ReactNode } from "react";

/** The three states every list screen shares: empty, loading, failed. */

export function Empty({ children }: { children: ReactNode }) {
  return <div className="empty">{children}</div>;
}

export function Loading({ rows = 3 }: { rows?: number }) {
  return (
    <div className="grid" style={{ marginTop: 16 }}>
      {Array.from({ length: rows }, (_, i) => (
        <div key={i} className="skeleton" style={{ height: 78 }} />
      ))}
    </div>
  );
}

export function ErrorNote({ message }: { message: string }) {
  return (
    <div className="empty" style={{ color: "var(--red)" }}>
      {message}
    </div>
  );
}
