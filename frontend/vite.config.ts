/// <reference types="vitest" />
import { fileURLToPath, URL } from "node:url";

import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      "@": fileURLToPath(new URL("./src", import.meta.url)),
    },
  },
  server: {
    port: 5173,
    proxy: {
      "/api": "http://localhost:8080",
      "/health": "http://localhost:8080",
    },
  },
  test: {
    environment: "jsdom",
    // Tests live next to what they cover, so a component and its test move
    // together when a feature is reorganised.
    include: ["src/**/*.test.{ts,tsx}"],
    restoreMocks: true,
  },
});
