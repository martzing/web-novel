export function Stars({ rating }: { rating: number }) {
  const rounded = Math.round(rating);
  return (
    <span className="stars" aria-label={`${rating} ดาว`}>
      {[1, 2, 3, 4, 5].map((n) => (
        <span key={n} style={{ opacity: n <= rounded ? 1 : 0.25 }}>
          ★
        </span>
      ))}
    </span>
  );
}

export function StarPicker({
  value,
  onChange,
}: {
  value: number;
  onChange: (rating: number) => void;
}) {
  return (
    <div>
      {[1, 2, 3, 4, 5].map((n) => (
        <button
          key={n}
          type="button"
          className={`star-btn${n <= value ? " is-on" : ""}`}
          onClick={() => onChange(n)}
          aria-label={`ให้ ${n} ดาว`}
        >
          ★
        </button>
      ))}
    </div>
  );
}
