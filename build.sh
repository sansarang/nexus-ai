#!/bin/bash
# Nexus AI 빌드 스크립트 (Mac 개발 환경)
set -e

echo "═══════════════════════════════════════"
echo "  Nexus AI Build Script"
echo "═══════════════════════════════════════"

BACKEND_DIR="./backend"
FRONTEND_DIR="."

# 색상
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

log_ok()   { echo -e "${GREEN}✓ $1${NC}"; }
log_warn() { echo -e "${YELLOW}⚠ $1${NC}"; }
log_err()  { echo -e "${RED}✗ $1${NC}"; exit 1; }

# ── 1. 프론트엔드 빌드 ───────────────────────────────────────
echo ""
echo "▶ 프론트엔드 빌드 중..."
cd "$FRONTEND_DIR"
npm run build > /dev/null 2>&1 && log_ok "프론트엔드 빌드 완료" || log_err "프론트엔드 빌드 실패"

# ── 2. 백엔드 빌드 (Mac 개발용) ────────────────────────────
echo ""
echo "▶ 백엔드 빌드 중 (Mac 개발용)..."
cd "$BACKEND_DIR"
go build -tags "!windows" -o nexus_backend_mac . 2>&1 && log_ok "Mac 백엔드 빌드 완료" || log_err "Mac 백엔드 빌드 실패"
cd ..

# ── 3. Windows 크로스컴파일 (옵션: CROSS=1 환경변수) ─────
if [ "$CROSS" = "1" ]; then
  echo ""
  echo "▶ Windows 크로스컴파일 중..."
  cd "$BACKEND_DIR"
  GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -tags windows -o nexus_backend.exe . 2>&1 \
    && log_ok "Windows 백엔드 빌드 완료 (nexus_backend.exe)" \
    || log_warn "Windows 크로스컴파일 실패 (CGO 제한 — Windows 머신에서 직접 빌드 필요)"
  cd ..
fi

# ── 4. Tauri 앱 빌드 (옵션: TAURI=1 환경변수) ────────────
if [ "$TAURI" = "1" ]; then
  echo ""
  echo "▶ Tauri 앱 빌드 중..."
  npm run tauri build 2>&1 && log_ok "Tauri 앱 빌드 완료" || log_err "Tauri 빌드 실패"
fi

echo ""
echo "═══════════════════════════════════════"
log_ok "빌드 완료!"
echo ""
echo "실행 방법:"
echo "  Mac 개발:   ./backend/nexus_backend_mac &"
echo "  Windows:    .\\backend\\nexus_backend.exe"
echo "  Tauri dev:  npm run tauri dev"
echo "═══════════════════════════════════════"
