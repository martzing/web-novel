import { describe, expect, it } from "vitest";

import type { LedgerEntry } from "./api";
import { coinsSpentThisMonth, estimateUnlockableChapters } from "./summary";

function entry(partial: Partial<LedgerEntry> & { created_at: string }): LedgerEntry {
  return {
    id: partial.id ?? crypto.randomUUID(),
    delta: partial.delta ?? 0,
    bonus_delta: partial.bonus_delta ?? 0,
    kind: partial.kind ?? "spend_unlock",
    balance_after: partial.balance_after ?? 0,
    created_at: partial.created_at,
  };
}

describe("estimateUnlockableChapters", () => {
  it("floors the estimate rather than promising a partial chapter", () => {
    expect(estimateUnlockableChapters(12)).toBe(2);
  });

  it("says nothing when the balance buys nothing", () => {
    expect(estimateUnlockableChapters(0)).toBeNull();
    expect(estimateUnlockableChapters(4)).toBeNull();
    expect(estimateUnlockableChapters(undefined)).toBeNull();
  });
});

describe("coinsSpentThisMonth", () => {
  // The month boundary is local, not UTC — "เดือนนี้" means the reader's month.
  // These fixtures are therefore built from local components so the test says
  // the same thing in Bangkok as it does in CI.
  const local = (y: number, m: number, d: number, h = 12) =>
    new Date(y, m - 1, d, h).toISOString();
  const now = new Date(2026, 7, 10, 12);

  it("counts debits and ignores top-ups", () => {
    const spent = coinsSpentThisMonth(
      [
        entry({ created_at: local(2026, 8, 2), delta: -5 }),
        entry({ created_at: local(2026, 8, 5), delta: -3, bonus_delta: -2 }),
        entry({ created_at: local(2026, 8, 6), delta: 100, kind: "topup" }),
      ],
      now,
    );
    expect(spent).toBe(10);
  });

  it("excludes spending from before the first of the month", () => {
    const spent = coinsSpentThisMonth(
      [
        entry({ created_at: local(2026, 7, 31, 23), delta: -50 }),
        entry({ created_at: local(2026, 8, 1, 1), delta: -7 }),
      ],
      now,
    );
    expect(spent).toBe(7);
  });

  it("returns zero before the ledger has loaded", () => {
    expect(coinsSpentThisMonth(undefined, now)).toBe(0);
  });
});
