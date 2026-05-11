#!/bin/bash
# ============================================================
# build-windows.sh
# Mac에서 Windows용 Nexus 빌드 스크립트
# 실행: chmod +x build-windows.sh && ./build-windows.sh
# ============================================================
set -e

PROJECT_ROOT="$(cd "$(dirname "$0")" && pwd)"
BACKEND_DIR="$PROJECT_ROOT/backend"
SIDECAR_DIR="$PROJECT_ROOT/src-tauri/binaries"
OUT_DIR="$PROJECT_ROOT/windows-dist"

GREEN='\033[0;32m'; YELLOW='\033[1;33m'; RED='\033[0;31m'; NC='\033[0m'
info()  { echo -e "${GREEN}[✓] $1${NC}"; }
warn()  { echo -e "${YELLOW}[!] $1${NC}"; }
error() { echo -e "${RED}[✗] $1${NC}"; exit 1; }

echo ""
echo "============================================"
echo "  Nexus Windows 빌드 시작"
echo "  $(date '+%Y-%m-%d %H:%M:%S')"
echo "============================================"
echo ""

# ── 사전 확인 ────────────────────────────────────────────────
command -v go      >/dev/null 2>&1 || error "Go가 설치되지 않았습니다. https://go.dev/dl/"
command -v node    >/dev/null 2>&1 || error "Node.js가 설치되지 않았습니다. https://nodejs.org/"
command -v rustup  >/dev/null 2>&1 || error "Rust가 설치되지 않았습니다. https://rustup.rs/"

# ── STEP 1: Go 백엔드 Windows 크로스 컴파일 ─────────────────
info "STEP 1: Go 백엔드 Windows 빌드 중..."
mkdir -p "$SIDECAR_DIR"
cd "$BACKEND_DIR"

go mod tidy

GOOS=windows GOARCH=amd64 CGO_ENABLED=0 \
  go build -tags windows \
  -ldflags="-s -w" \
  -o "$SIDECAR_DIR/backend-x86_64-pc-windows-gnu.exe" .

BACKEND_SIZE=$(du -sh "$SIDECAR_DIR/backend-x86_64-pc-windows-gnu.exe" | cut -f1)
info "백엔드 빌드 완료: $BACKEND_SIZE"

# ── STEP 2: Rust Windows 타겟 확인 ───────────────────────────
info "STEP 2: Rust Windows 타겟 확인 중..."
if ! rustup target list --installed | grep -q "x86_64-pc-windows-gnu"; then
    warn "x86_64-pc-windows-gnu 타겟 추가 중..."
    rustup target add x86_64-pc-windows-gnu
fi

# MinGW 링커 확인
if ! command -v x86_64-w64-mingw32-gcc >/dev/null 2>&1; then
    error "MinGW가 설치되지 않았습니다.\n  실행: brew install mingw-w64"
fi
info "Rust Windows 타겟 준비됨"

# ── STEP 3: 프론트엔드 빌드 ──────────────────────────────────
info "STEP 3: 프론트엔드 (React) 빌드 중..."
cd "$PROJECT_ROOT"

if [ ! -d "node_modules" ]; then
    warn "node_modules 없음, npm install 실행 중..."
    npm install
fi

npm run build
info "프론트엔드 빌드 완료"

# ── STEP 4: Tauri Windows 크로스 컴파일 (raw exe) ────────────
info "STEP 4: Tauri Windows 앱 빌드 중... (5~15분 소요)"
cd "$PROJECT_ROOT"

# GNU 타겟으로 크로스 컴파일, 번들러(NSIS) 제외하고 raw exe만 생성
npm run tauri build -- \
    --target x86_64-pc-windows-gnu \
    --no-bundle 2>&1 || {
    warn "--no-bundle 옵션 실패, 기본 빌드 시도 중..."
    npm run tauri build -- --target x86_64-pc-windows-gnu 2>&1 || {
        error "Tauri 빌드 실패. Windows VM에서 빌드를 권장합니다."
    }
}

# ── STEP 5: 배포 폴더 구성 ───────────────────────────────────
info "STEP 5: 배포 패키지 구성 중..."
mkdir -p "$OUT_DIR"

# Tauri exe 복사
TAURI_EXE=$(find "$PROJECT_ROOT/src-tauri/target/x86_64-pc-windows-gnu/release" \
    -maxdepth 1 -name "*.exe" ! -name "*setup*" 2>/dev/null | head -1)

if [ -n "$TAURI_EXE" ]; then
    cp "$TAURI_EXE" "$OUT_DIR/Nexus.exe"
    info "Tauri exe 복사 완료"
else
    warn "Tauri exe를 찾지 못했습니다."
fi

# 백엔드 exe 복사 (Nexus.exe와 같은 폴더에 있어야 함)
cp "$SIDECAR_DIR/backend-x86_64-pc-windows-gnu.exe" \
   "$OUT_DIR/backend-x86_64-pc-windows-gnu.exe"

# ── 완료 ─────────────────────────────────────────────────────
echo ""
echo "============================================"
echo "  ✅ 빌드 완료!"
echo "============================================"
echo ""
echo "📁 결과물 위치: $OUT_DIR/"
ls -lh "$OUT_DIR/" 2>/dev/null
echo ""
echo "📋 Windows PC 전송 방법:"
echo "   USB: 위 폴더 전체를 Windows에 복사"
echo "   Wi-Fi: python3 -m http.server 9999 후 Windows에서 다운로드"
echo ""
echo "▶ Windows에서 실행: Nexus.exe 더블클릭"
echo "  (backend-*.exe는 같은 폴더에 있어야 함)"
echo ""
