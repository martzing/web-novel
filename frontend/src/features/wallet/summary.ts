import type { LedgerEntry } from "./api";

/**
 * The going rate for a paid chapter, used only to turn a balance into
 * "ปลดล็อกได้อีกราว N บท".
 *
 * Deliberately a constant and deliberately labelled ราว: chapter prices differ
 * per novel, so any exact figure would be wrong for most of what the reader is
 * about to open. The estimate answers "roughly how far does this go", which is
 * the question a balance actually raises.
 */
export const TYPICAL_CHAPTER_PRICE = 5;

export function estimateUnlockableChapters(total?: number): number | null {
  if (!total || total < TYPICAL_CHAPTER_PRICE) return null;
  return Math.floor(total / TYPICAL_CHAPTER_PRICE);
}

/**
 * Coins spent since the first of this month, from the ledger page already on
 * screen. Only debits count, so a top-up does not read as spending.
 */
export function coinsSpentThisMonth(entries?: LedgerEntry[], now = new Date()): number {
  if (!entries) return 0;

  const monthStart = new Date(now.getFullYear(), now.getMonth(), 1);

  return entries.reduce((sum, entry) => {
    const amount = entry.delta + entry.bonus_delta;
    if (amount >= 0) return sum;
    return new Date(entry.created_at) >= monthStart ? sum + Math.abs(amount) : sum;
  }, 0);
}
