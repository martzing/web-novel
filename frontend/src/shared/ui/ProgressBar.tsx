export function ProgressBar({ pct }: { pct: number }) {
  return (
    <div className="progress">
      <div className="progress__bar" style={{ width: `${Math.min(Math.max(pct, 0), 100)}%` }} />
    </div>
  );
}
