import type { ReactNode } from "react";
import { BrowserRouter } from "react-router-dom";
import { QueryClientProvider } from "@tanstack/react-query";

import { AuthProvider } from "@/features/identity";

import { queryClient } from "./queryClient";

/**
 * Every cross-cutting provider the app needs, in one place.
 *
 * Tests mount their own tree, so keeping this separate from `main.tsx` lets a
 * test reuse the real provider order without pulling in `createRoot`.
 */
export default function Providers({ children }: { children: ReactNode }) {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <AuthProvider>{children}</AuthProvider>
      </BrowserRouter>
    </QueryClientProvider>
  );
}
