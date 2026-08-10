import type { ReactNode } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { walletApi } from "./api";
import { useUnlockChapter, useWallet, walletKeys } from "./queries";

function wrapper(client: QueryClient) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
  };
}

function newClient() {
  return new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
}

describe("useWallet", () => {
  beforeEach(() => {
    vi.spyOn(walletApi, "getWallet").mockResolvedValue({
      balance: 40,
      bonus_balance: 10,
      total: 50,
    });
  });

  it("reads the balance under the key the factory produces", async () => {
    const client = newClient();
    const { result } = renderHook(() => useWallet(), { wrapper: wrapper(client) });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data?.total).toBe(50);
    expect(client.getQueryData(walletKeys.balance())).toEqual(result.current.data);
  });

  it("does not call the API while signed out", () => {
    const client = newClient();
    renderHook(() => useWallet(false), { wrapper: wrapper(client) });
    expect(walletApi.getWallet).not.toHaveBeenCalled();
  });
});

/**
 * The contract the reader depends on: unlocking a chapter stales the balance
 * without the screen having to know the wallet's keys, and hands the receipt
 * back so the screen can invalidate what it owns.
 */
describe("useUnlockChapter", () => {
  it("invalidates the wallet and reports the receipt to its caller", async () => {
    const client = newClient();
    client.setQueryData(walletKeys.balance(), { balance: 40, bonus_balance: 10, total: 50 });

    const receipt = {
      ledger_id: "l-1",
      coins_spent: 5,
      balance_after: 35,
      bonus_balance_after: 10,
      replayed: false,
    };
    vi.spyOn(walletApi, "unlockChapter").mockResolvedValue(receipt);
    const onSpent = vi.fn();

    const { result } = renderHook(() => useUnlockChapter("chapter-1", onSpent), {
      wrapper: wrapper(client),
    });

    result.current.mutate();

    await waitFor(() => expect(onSpent).toHaveBeenCalledWith(receipt));
    expect(client.getQueryState(walletKeys.balance())?.isInvalidated).toBe(true);
  });
});
