import tailwindcss from "@tailwindcss/vite";
import react from "@vitejs/plugin-react";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { defineConfig } from "vite";

const __dirname = path.dirname(fileURLToPath(import.meta.url));

export default defineConfig({
  plugins: [
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
  build: {
    rolldownOptions: {
      output: {
        codeSplitting: {
          groups: [
            {
              name: "react-vendor",
              test: /node_modules[\\/](react|react-dom|react-hook-form)([\\/]|$)/,
              priority: 30,
            },
            {
              name: "tanstack",
              test: /node_modules[\\/]@tanstack[\\/]/,
              priority: 20,
            },
            {
              name: "editor",
              test: /node_modules[\\/](@codemirror|@uiw|@lezer)[\\/]/,
              priority: 15,
            },
            {
              name: "ui-vendor",
              test: /node_modules[\\/](@base-ui|radix-ui|lucide-react|cmdk|motion)([\\/]|$)/,
              priority: 10,
            },
            {
              name: "content-vendor",
              test: /node_modules[\\/](react-markdown|zod|date-fns)([\\/]|$)/,
              priority: 5,
            },
          ],
        },
      },
    },
    chunkSizeWarningLimit: 750,
    sourcemap: false,
  },
  server: {
    port: 5173,
    proxy: {
      "/api": {
        target: "http://localhost:8080",
        changeOrigin: true,
      },
    },
    fs: {
      // Allow Vite to serve the vendored osquery schema from the repo root.
      allow: [path.resolve(__dirname, ".."), path.resolve(__dirname)],
    },
  },
});
