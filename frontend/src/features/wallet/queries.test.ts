import { describe, expect, it } from "vitest";

import { walletKeys } from "./queries";

/**
 * The invariant the factory exists to protect.
 *
 * Every key must start with `all`, because mutations invalidate `all` and rely
 * on prefix matching to clear the rest. A key that forgets the root would be
 * silently missed — which is exactly how the writer workspace's series-books
 * key drifted apart from its own invalidation.
 */
describe("walletKeys", () => {
  const keys = [
    walletKeys.balance(),
    walletKeys.ledger(),
    walletKeys.packs(),
    walletKeys.autoUnlock(),
    walletKeys.arcBundle("arc-1"),
  ];

  it("roots every key at walletKeys.all", () => {
    for (const key of keys) {
      expect(key.slice(0, walletKeys.all.length)).toEqual([...walletKeys.all]);
    }
  });

  it("gives each query its own key", () => {
    const serialised = keys.map((key) => JSON.stringify(key));
    expect(new Set(serialised).size).toBe(keys.length);
  });

  it("separates one arc's bundle quote from another's", () => {
    expect(walletKeys.arcBundle("arc-1")).not.toEqual(walletKeys.arcBundle("arc-2"));
  });
});
