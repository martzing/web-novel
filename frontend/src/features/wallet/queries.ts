import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { newIdempotencyKey } from "@/shared/api/client";

import { walletApi, type Receipt } from "./api";

/**
 * Every cache key this context owns, built from one root.
 *
 * Keys are produced here and nowhere else. Typing them by hand is what let
 * `["writer-series-books", id]` and `["writer-series-books"]` drift apart —
 * an invalidation that silently matched nothing. `walletKeys.all` is a prefix
 * of every other key, so invalidating it clears the whole context at once.
 */
export const walletKeys = {
  all: ["wallet"] as const,
  balance: () => [...walletKeys.all, "balance"] as const,
  ledger: () => [...walletKeys.all, "ledger"] as const,
  packs: () => [...walletKeys.all, "packs"] as const,
  autoUnlock: () => [...walletKeys.all, "auto-unlock"] as const,
  arcBundle: (arcId: string) => [...walletKeys.all, "arc-bundle", arcId] as const,
};

export function useWallet(enabled = true) {
  return useQuery({
    queryKey: walletKeys.balance(),
    queryFn: walletApi.getWallet,
    enabled,
    staleTime: 30_000,
  });
}

export function useLedger(enabled = true) {
  return useQuery({
    queryKey: walletKeys.ledger(),
    queryFn: () => walletApi.getLedger(),
    enabled,
  });
}

export function useCoinPacks() {
  return useQuery({ queryKey: walletKeys.packs(), queryFn: walletApi.listPacks });
}

export function useAutoUnlockSubs(enabled = true) {
  return useQuery({
    queryKey: walletKeys.autoUnlock(),
    queryFn: walletApi.listAutoUnlock,
    enabled,
  });
}

export function useArcBundle(arcId: string, enabled = true) {
  return useQuery({
    queryKey: walletKeys.arcBundle(arcId),
    queryFn: () => walletApi.quoteArcBundle(arcId),
    enabled,
    // A refused quote (nothing left to buy, arc not for sale) is an answer, not
    // a blip — retrying only delays the message the reader needs.
    retry: false,
  });
}

/**
 * Buys a pack and completes the mock payment in one step.
 *
 * The completion key is generated once per attempt, so a retry of the same
 * attempt replays rather than double-crediting.
 */
export function useBuyCoinPack(packId: string, onDone?: (receipt: Receipt) => void) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async () => {
      const purchase = await walletApi.createPurchase(packId);
      return walletApi.completePurchase(purchase.purchase_id, newIdempotencyKey());
    },
    onSuccess: (receipt) => {
      qc.invalidateQueries({ queryKey: walletKeys.all });
      onDone?.(receipt);
    },
  });
}

/**
 * The coin-spending mutations.
 *
 * Each takes an `onSpent` callback rather than reaching for another context's
 * keys: unlocking a chapter also stales the reader's chapter and the novel's
 * table of contents, and only the calling screen knows which of those it has
 * on display. Wallet invalidates wallet.
 */
export function useUnlockChapter(chapterId: string, onSpent?: (receipt: Receipt) => void) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () => walletApi.unlockChapter(chapterId, newIdempotencyKey()),
    onSuccess: (receipt) => {
      qc.invalidateQueries({ queryKey: walletKeys.all });
      onSpent?.(receipt);
    },
  });
}

export function useUnlockArc(arcId: string, key: string, onSpent?: (receipt: Receipt) => void) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () => walletApi.unlockArc(arcId, key),
    onSuccess: (receipt) => {
      qc.invalidateQueries({ queryKey: walletKeys.all });
      onSpent?.(receipt);
    },
  });
}

export function useTipChapter(chapterId: string, coins: number, onSpent?: (receipt: Receipt) => void) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () => walletApi.tipChapter(chapterId, coins, newIdempotencyKey()),
    onSuccess: (receipt) => {
      qc.invalidateQueries({ queryKey: walletKeys.all });
      onSpent?.(receipt);
    },
  });
}

/** Turns auto-unlock on at a per-chapter cap, or off by removing the opt-in. */
export function useSetAutoUnlock(novelId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async ({ active, cap }: { active: boolean; cap: number }) => {
      if (!active) {
        await walletApi.removeAutoUnlock(novelId);
        return;
      }
      await walletApi.setAutoUnlock(novelId, { active: true, max_coins_per_chapter: cap });
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: walletKeys.autoUnlock() }),
  });
}
