# Conventions

## Go backend
- Two-file pattern per feature: `handlers_X.go` (//go:build windows) + `handlers_X_stub.go` (//go:build !windows)
- Shared types/funcs in untagged files (`types.go`, `intent_classify_shared.go`)
- Action names: snake_case, lowercase. Must match across:
  - backend `switch intent.Action` (handlers_command{,_stub}.go)
  - frontend `renderCommandResult` switch (index.tsx)
  - intent registry (`intentRegistry.ts`)
- Response shape: `CommandResponse{success, action, message, result, duration, ...}`
- Errors: prefer `json200(w, ...success:false)` over HTTP error codes (frontend expects 200)
- Auth: localhost bypass in `requireAuth`. External needs JWT in `Authorization: Bearer`.
- LLM keys: package-level vars (`llmGroqKey`, `llmTavilyKey`, etc.). Bundled defaults via `-ldflags -X main.bundledXKey=...`.

## TypeScript / React
- File naming: PascalCase for components (`ChatBubble.tsx`), camelCase for utils
- State: React hooks + `useAppStore` (Zustand) for global
- Backend calls: only via `src/lib/nexus/backendAPI.ts` — never `fetch` directly elsewhere
- Intents: type `Intent` from `intentDetector.ts`. New intent → update registry + add to `chatIntentImpl.ts` switch.
- Korean/English: every user-facing string must have both. Use `t(ko, en, lang)` helper or `userLang === 'en' ? ... : ...`.

## LLM prompts (자비스 톤 — recent)
- Max 1-3 sentences. Lead with answer.
- KO: "~이에요/예요" 친근체. NO "~입니다/합니다" 격식체.
- EN: contractions OK (I'll, you're). NO "In summary", "To answer your question".
- NO markdown headers (#, ##), bullets (•), bold (**) unless critical.
- Backend post-processes via `cleanJarvisTone` (handlers_llm.go).

## Voice (TTS)
- Default OFF. Sidebar toggle persists in localStorage `nexus-sound`.
- Force-once via `forceVoiceNextRef` (set by "말로/읽어줘" patterns in chatSenderImpl).
- Proactive alerts bypass toggle (call `speak()` directly).

## Markdown rendering
- ChatBubble renders AI messages via `renderMarkdown()` (line 4 of ChatBubble.tsx).
- Sidebar previews are PLAIN TEXT — must strip `**` before display.

## Build tag pitfalls
- `regexp` import needed in handlers_command{,_stub}.go for systemPatterns
- Adding new shared function → ensure both build tags reach it (use untagged file)
