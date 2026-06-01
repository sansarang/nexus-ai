# Task Completion Checklist

After modifying Go backend:
1. `cd backend && go build -o /tmp/nexus-mac .` (Mac stub builds)
2. `cd backend && GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -tags windows -o /tmp/nx-win.exe .` (Windows prod builds)
3. Restart local backend, hit `/api/health`
4. Optional: run `python3 /tmp/nx_e2e.py` (Mac sim) or instruct user to run `test/e2e.ps1` in VM

After modifying TypeScript / React:
1. `npx tsc --noEmit` — must pass clean
2. If touching Tauri config (`src-tauri/tauri.conf.json`), verify JSON valid

Before commit:
1. `git status --short` — review changes
2. Ensure new symbol names match between backend switch + frontend renderCommandResult
3. For LLM/intent changes: verify systemPatterns cover both KO + EN keywords

After commit + push to main:
- GitHub Actions auto-triggers `build-windows.yml`
- ~30-40 min until new `.exe` available
- New build URL: https://github.com/sansarang/nexus-ai/releases/tag/v2.7.1-build<N>
- Latest build number: `gh release list --limit 1`

NEVER push without:
- TS clean (`tsc --noEmit`)
- Both Go builds clean (windows AND non-windows tags)
