import { newIdempotencyKey, qs, request } from "@/shared/api/client";
import type { Paged } from "@/shared/api/types";

export interface Wallet {
  balance: number;
  bonus_balance: number;
  bonus_expires_at?: string;
  total: number;
}

export interface LedgerEntry {
  id: string;
  delta: number;
  bonus_delta: number;
  kind: string;
  ref_type?: string;
  reason?: string;
  balance_after: number;
  created_at: string;
}

export interface CoinPack {
  id: string;
  coins: number;
  bonus_coins: number;
  price_satang: number;
  currency: string;
  is_best_value: boolean;
}

export interface PurchaseResult {
  purchase_id: string;
  status: string;
  amount_satang: number;
  mock_checkout_url: string;
}

export interface Receipt {
  ledger_id: string;
  coins_spent: number;
  balance_after: number;
  bonus_balance_after: number;
  replayed: boolean;
}

/** A quote for buying a whole arc, before the reader commits. */
export interface ArcBundle {
  arc_id: string;
  novel_id: string;
  arc_no: number;
  name: string;
  chapter_count: number;
  gross: number;
  discount_percent: number;
  discount: number;
  total: number;
  chapters: { chapter_id: string; chapter_no: number; list_price: number; coins: number }[];
}

/** One auto-unlock opt-in. */
export interface AutoUnlock {
  novel_id: string;
  novel_title_th?: string;
  novel_slug?: string;
  active: boolean;
  max_coins_per_chapter: number;
}

export const walletApi = {
  getWallet: () => request<Wallet>("/me/wallet"),
  getLedger: (cursor?: string) => request<Paged<LedgerEntry>>(`/me/wallet/ledger${qs({ cursor })}`),
  listPacks: () => request<Paged<CoinPack>>("/coin-packs", { auth: false }),
  createPurchase: (packId: string) =>
    request<PurchaseResult>("/purchases", {
      method: "POST",
      body: { pack_id: packId },
      idempotencyKey: newIdempotencyKey(),
    }),
  completePurchase: (purchaseId: string, key: string) =>
    request<Receipt>(`/purchases/${purchaseId}/mock-complete`, {
      method: "POST",
      idempotencyKey: key,
    }),
  failPurchase: (purchaseId: string) =>
    request<{ status: string }>(`/purchases/${purchaseId}/mock-fail`, { method: "POST" }),
  unlockChapter: (chapterId: string, key: string) =>
    request<Receipt>(`/chapters/${chapterId}/unlock`, { method: "POST", idempotencyKey: key }),
  quoteArcBundle: (arcId: string) => request<ArcBundle>(`/arcs/${arcId}/bundle`),
  unlockArc: (arcId: string, key: string) =>
    request<Receipt>(`/arcs/${arcId}/unlock`, { method: "POST", idempotencyKey: key }),
  tipChapter: (chapterId: string, coins: number, key: string) =>
    request<Receipt>(`/chapters/${chapterId}/tip`, {
      method: "POST",
      body: { coins },
      idempotencyKey: key,
    }),

  listAutoUnlock: () => request<Paged<AutoUnlock>>("/me/auto-unlock"),
  setAutoUnlock: (novelId: string, body: { active: boolean; max_coins_per_chapter: number }) =>
    request<AutoUnlock>(`/me/auto-unlock/${novelId}`, { method: "PUT", body }),
  removeAutoUnlock: (novelId: string) =>
    request<void>(`/me/auto-unlock/${novelId}`, { method: "DELETE" }),
};
