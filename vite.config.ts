import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import path from "path";
import fs from "fs";

// 앱 버전 동기화: CI는 APP_VERSION env(2.9.<run_number>)를 주입하고, 로컬/dev는
// src-tauri/tauri.conf.json의 version으로 폴백. 이 값이 __APP_VERSION__ 로 번들에 주입돼
// Onboarding.tsx가 하드코딩 없이 실제 설치 바이너리 버전과 항상 일치한다.
function resolveAppVersion(): string {
  if (process.env.APP_VERSION) return process.env.APP_VERSION;
  try {
    const conf = JSON.parse(
      fs.readFileSync(path.resolve(__dirname, "src-tauri/tauri.conf.json"), "utf-8")
    );
    return typeof conf.version === "string" ? conf.version : "0.0.0";
  } catch {
    return "0.0.0";
  }
}

export default defineConfig(async () => ({
  plugins: [react()],
  define: {
    __APP_VERSION__: JSON.stringify(resolveAppVersion()),
  },
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
