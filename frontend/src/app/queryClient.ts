import { QueryClient } from "@tanstack/react-query";

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30_000,
      refetchOnWindowFocus: false,
      // A 401 is resolved by the client's refresh-and-retry, and a 404 will not
      // fix itself; retrying either just delays the error the user needs to see.
      retry: 1,
    },
  },
});
