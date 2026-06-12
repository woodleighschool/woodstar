import tailwindcss from "@tailwindcss/vite";
import react from "@vitejs/plugin-react";
import path from "node:path";
import { defineConfig } from "vite";

const __dirname = import.meta.dirname;

export default defineConfig({
  build: {
    chunkSizeWarningLimit: 750,
    rolldownOptions: {
      output: {
        codeSplitting: {
          groups: [
            {
              name: "react-vendor",
              priority: 30,
              test: /node_modules[\\/](react|react-dom|react-hook-form)([\\/]|$)/,
            },
            {
              name: "tanstack",
              priority: 20,
              test: /node_modules[\\/]@tanstack[\\/]/,
            },
            {
              name: "editor",
              priority: 15,
              test: /node_modules[\\/](@codemirror|@uiw|@lezer)[\\/]/,
            },
            {
              name: "ui-vendor",
              priority: 10,
              test: /node_modules[\\/](@base-ui|radix-ui|lucide-react|cmdk|motion)([\\/]|$)/,
            },
            {
              name: "content-vendor",
              priority: 5,
              test: /node_modules[\\/](react-markdown|zod|date-fns)([\\/]|$)/,
            },
          ],
        },
      },
    },
    sourcemap: false,
  },
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
  server: {
    fs: {
      // Allow Vite to serve the vendored osquery schema from the repo root.
      allow: [path.resolve(__dirname, ".."), path.resolve(__dirname)],
    },
    port: 5173,
    proxy: {
      "/api": {
        changeOrigin: true,
        target: "http://localhost:8080",
      },
    },
  },
});
