# Commands

## Frontend
- `npm run dev` — Vite dev server (port 1420)
- `npm run build` — production bundle
- `npx tsc --noEmit` — type check only (use for verification)

## Backend (Mac dev simulation)
- `cd backend && go build -o /tmp/nexus-mac . && /tmp/nexus-mac > /tmp/nx.log 2>&1 &`
- `curl http://127.0.0.1:17891/api/health`
- Kill: `lsof -i :17891 -t | xargs kill`

## Backend (Windows cross-compile from Mac)
- `cd backend && GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -tags windows -o /tmp/nx-win.exe .`

## Tauri (full app)
- `npx tauri dev` — dev mode
- `npx tauri build --target x86_64-pc-windows-msvc` — Windows release (CI only)

## Test
- E2E script: `test/e2e.ps1` (PowerShell, 60 queries). User runs in VM:
  `iex (irm https://raw.githubusercontent.com/sansarang/nexus-ai/main/test/e2e.ps1)`

## Release
- Push to main → GitHub Actions `build-windows.yml` triggers
- `gh run list --workflow=build-windows.yml --limit=3` — status
- `gh release list --limit 3` — latest exe URLs
- Releases: https://github.com/sansarang/nexus-ai/releases/tag/v2.7.1-build<N>

## Darwin-specific
- `pbcopy` / `pbpaste` for clipboard tests
- `screencapture -x` for screenshots (Parallels VM included)
- `cliclick` for GUI automation (unreliable for Korean IME — prefer API tests)
