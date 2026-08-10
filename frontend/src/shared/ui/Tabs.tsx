export function Tabs<T extends string>({
  tabs,
  active,
  onChange,
}: {
  tabs: { key: T; label: string; badge?: number }[];
  active: T;
  onChange: (key: T) => void;
}) {
  return (
    <div className="tabs" role="tablist">
      {tabs.map((tab) => (
        <button
          key={tab.key}
          role="tab"
          aria-selected={tab.key === active}
          className={`tab${tab.key === active ? " is-active" : ""}`}
          onClick={() => onChange(tab.key)}
        >
          {tab.label}
          {tab.badge !== undefined && <span className="muted"> {tab.badge}</span>}
        </button>
      ))}
    </div>
  );
}
