# Tech Stack

## Frontend
- React 18, TypeScript, Vite
- Framer Motion (animations), Tauri 2 IPC
- @paddle/paddle-js (subscription), @supabase/supabase-js (auth)
- @react-three/fiber + drei (3D avatar — currently disabled)
- npm scripts: `dev`, `build`, `preview`, `tauri`

## Desktop
- Tauri 2 (Rust 1.x stable, NSIS installer Windows)
- src-tauri/Cargo.toml — Rust deps for auth proxy + WebView2

## Backend
- Go 1.26
- module: `ttuktak-backend` (`backend/go.mod`)
- Build tag dual: `windows` vs `!windows` (Mac dev stubs)
- chromedp (stealth browser scraping), excelize (xlsx)
- No CGO (`CGO_ENABLED=0`), cross-compiled Windows from Mac for tests

## Python sidecar (optional)
- FastAPI + uvicorn
- yt-dlp (video), easyocr (OCR), sentence-transformers (embeddings), faiss-cpu, PyMuPDF
- PyInstaller --onefile bundle in GitHub Actions

## External APIs
- Claude Haiku (intent classify) via Supabase Edge Function `claude_intent`
- Perplexity sonar-pro (web search) via Supabase `perplexity_chat`
- Groq llama-3.3-70b (chat) — direct + Supabase
- Tavily (search aggregation) — direct + Supabase
- OpenAI (TTS, vision fallback)

## CI
- `.github/workflows/build-windows.yml`
- Builds: Go backend → PyInstaller sidecar → Tauri NSIS installer
- Auto-tag releases v2.7.1-build<N>
- ~30-40 min per run
