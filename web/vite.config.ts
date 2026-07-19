import path from "node:path";

import tailwindcss from "@tailwindcss/vite";
import { tanstackRouter } from "@tanstack/router-plugin/vite";
import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";

const projectDirectory = import.meta.dirname;
const workspaceRoot = path.resolve(projectDirectory, "..");

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
      "@": path.resolve(projectDirectory, "./src"),
      "@schema": path.resolve(projectDirectory, "../schema"),
    },
  },
  server: {
    fs: {
      // Allow Vite to serve the vendored osquery schema from the repo root.
      allow: [workspaceRoot, projectDirectory],
    },
    port: 5173,
    strictPort: true,
    proxy: {
      "/api": {
        changeOrigin: true,
        target: "https://woodstar:8443",
      },
    },
  },
});
