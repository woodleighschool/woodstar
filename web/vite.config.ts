import path from "node:path";

import tailwindcss from "@tailwindcss/vite";
import { tanstackRouter } from "@tanstack/router-plugin/vite";
import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";

const __dirname = import.meta.dirname;
const workspaceRoot = path.resolve(__dirname, "..");

export default defineConfig({
  plugins: [
    tanstackRouter({
      target: "react",
      autoCodeSplitting: true,
    }),
    react({
      jsxRuntime: "automatic",
    }),
    tailwindcss(),
  ],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
      "@schema": path.resolve(__dirname, "../schema"),
    },
  },
  server: {
    fs: {
      // Allow Vite to serve the vendored osquery schema from the repo root.
      allow: [workspaceRoot, path.resolve(__dirname)],
    },
    port: 5173,
    strictPort: true,
    proxy: {
      "/api": {
        changeOrigin: true,
        target: "https://localhost:8443",
      },
    },
  },
});
