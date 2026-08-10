/**
 * The wallet context's public surface.
 *
 * Other features import from `@/features/wallet` and never from a path inside
 * it. That keeps the coupling visible in one file: if this list grows, the
 * context is leaking, and the fix is a narrower hook rather than a deeper
 * import — the same discipline the Go side gets from its bounded contexts.
 */
export { walletApi } from "./api";
export type { ArcBundle, AutoUnlock, CoinPack, LedgerEntry, Receipt, Wallet } from "./api";
export {
  useArcBundle,
  useAutoUnlockSubs,
  useBuyCoinPack,
  useCoinPacks,
  useLedger,
  useSetAutoUnlock,
  useTipChapter,
  useUnlockArc,
  useUnlockChapter,
  useWallet,
  walletKeys,
} from "./queries";
export { default as Checkout } from "./pages/Checkout";
export { default as Coins } from "./pages/Coins";
