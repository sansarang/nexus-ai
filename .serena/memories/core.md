# Nexus AI (뚝딱PC) — Project Map

Tauri 2 desktop app: Korean PC assistant with Jarvis-style AI chat.

## Architecture (3 processes)

- **Tauri Rust** (`src-tauri/src/main.rs`) — auth server on port 17891, TCP proxy → 17892 Go.
- **Go backend** (`backend/`, port 17892) — main service. Build tag split:
  - Windows: `//go:build windows` (production, full features incl. PowerShell)
  - !windows: `//go:build !windows` (Mac dev stubs)
- **Python sidecar** (`backend/nexus_python/`, port 17893) — ML heavy (OCR, embeddings, yt-dlp). PyInstaller bundled.

## Frontend
- React 18 + TS, Vite, Framer Motion
- Main UI: `src/components/FloatingCharacter/` (3-Pane: Sidebar + Chat + Canvas modal)
- Intent registry: `src/lib/nexus/intentRegistry.ts` (127 intents)
- Backend client: `src/lib/nexus/backendAPI.ts` (calls `http://127.0.0.1:17891`)

## Intent flow (CRITICAL invariant)
1. Frontend `sendTextImpl` (chatSenderImpl.ts) entry
2. Greeting/continuation/voice patterns short-circuit FIRST
3. FAST_INTENTS detectIntent → backend handler direct
4. 1st: LLM router via `/api/command` (backend handleCommand)
5. 2nd: local detectIntent keyword
6. 3rd: dynamic LLM chat

## Backend command routing (handleCommand)
Order:
1. Shopping/video pre-route by keyword (handlers_command{,_stub}.go)
2. **systemPatterns regex** (stats/scan/clean/email_inbox/price_compare/video_search/weather)
   → matched? skip LLM, `goto haikuRouted` (Win) or set `systemPreRouted=true` (Mac)
3. Claude Haiku intent classify via Supabase proxy (`callHaikuIntentClassifyViaProxy` in `intent_classify_shared.go`)
4. Groq Function Calling (gsk_ keys)
5. JSON prompt fallback (`callGroqWithFallback`)

Action names must match BOTH backend switch case AND frontend `renderCommandResult` cases (index.tsx).

## LLM provider chain
`callGroqWithFallback` (handlers_llm.go):
1. Supabase proxy (`perplexity_chat` action)
2. Direct Claude API (sk-ant- key)
3. Direct Groq/Perplexity
4. Direct Claude (non sk-ant)
Output post-processed via `cleanJarvisTone` (strips markdown, biz preambles).

## Bundled keys (production)
GitHub Secrets injected via `-ldflags`:
- `bundledGroqKey` (gsk_)
- `bundledTavilyKey` (tvly-)
- `bundledOpenAIKey` (sk-proj-)
Frontend uses backend `/api/llm/chat` proxy — users need no API keys.

## Auth
`requireAuth` (handlers_proxy.go) allows localhost 127.0.0.1:* freely. External requires JWT.

## Voice policy (recent)
Default OFF. ON only via: sidebar toggle, "말로 알려줘", proactive alerts.

## Detailed maps
- Tech stack: `mem:tech_stack`
- Commands: `mem:suggested_commands`
- Conventions: `mem:conventions`
- Task completion: `mem:task_completion`
- Backend internals: `mem:backend/core`
- Frontend internals: `mem:frontend/core`
