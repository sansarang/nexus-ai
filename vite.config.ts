import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import path from "path";

export default defineConfig(async () => ({
  plugins: [react()],
  resolve: {
    alias: { "@": path.resolve(__dirname, "./src") },
  },
  clearScreen: false,
  build: {
    target: "chrome105",  // top-level await 지원 (Tauri WebView 기준)
    rollupOptions: {
      output: {
        // 코드 스플리팅 비활성화 → 단일 번들 (Tauri 배포에 적합)
        inlineDynamicImports: false,
      },
    },
  },
  server: {
    port: 1420,
    strictPort: true,
    watch: {
      ignored: ["**/src-tauri/**"],
    },
  },
}));
