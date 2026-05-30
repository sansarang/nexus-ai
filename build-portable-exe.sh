#!/bin/bash
# ╔══════════════════════════════════════════════════════════════╗
# ║  build-portable-exe.sh                                       ║
# ║  Mac → Windows 단일 portable exe 빌드                         ║
# ║  실행: chmod +x build-portable-exe.sh && ./build-portable-exe.sh ║
# ╚══════════════════════════════════════════════════════════════╝
set -e

ROOT="$(cd "$(dirname "$0")" && pwd)"
BACKEND_DIR="$ROOT/backend"
BIN_DIR="$ROOT/src-tauri/backend-bin"

GREEN='\033[0;32m'; YELLOW='\033[1;33m'; RED='\033[0;31m'; BOLD='\033[1m'; NC='\033[0m'
ok()   { echo -e "${GREEN}✓${NC} $1"; }
warn() { echo -e "${YELLOW}!${NC} $1"; }
fail() { echo -e "${RED}✗ $1${NC}"; exit 1; }
step() { echo -e "\n${BOLD}━━ $1 ━━${NC}"; }

echo -e "${BOLD}"
echo "  ███╗   ██╗███████╗██╗  ██╗██╗   ██╗███████╗"
echo "  ████╗  ██║██╔════╝╚██╗██╔╝██║   ██║██╔════╝"
echo "  ██╔██╗ ██║█████╗   ╚███╔╝ ██║   ██║███████╗"
echo "  ██║╚██╗██║██╔══╝   ██╔██╗ ██║   ██║╚════██║"
echo "  ██║ ╚████║███████╗██╔╝ ██╗╚██████╔╝███████║"
echo "  ╚═╝  ╚═══╝╚══════╝╚═╝  ╚═╝ ╚═════╝ ╚══════╝"
echo "  Windows Portable EXE Builder"
echo -e "${NC}"
echo "  시작: $(date '+%Y-%m-%d %H:%M:%S')"
echo ""

# ── 사전 조건 체크 ────────────────────────────────────────────
step "사전 조건 확인"

command -v go >/dev/null 2>&1      || fail "Go 미설치: https://go.dev/dl/"
command -v node >/dev/null 2>&1    || fail "Node.js 미설치: https://nodejs.org/"
command -v rustup >/dev/null 2>&1  || fail "Rust 미설치: https://rustup.rs/"

ok "Go: $(go version | awk '{print $3}')"
ok "Node: $(node --version)"
ok "Rust: $(rustc --version | awk '{print $2}')"

# ── STEP 1: Go 백엔드 빌드 ────────────────────────────────────
step "STEP 1: Go 백엔드 빌드 (Windows x64)"

mkdir -p "$BIN_DIR"
cd "$BACKEND_DIR"

go mod tidy
ok "go mod tidy 완료"

GOOS=windows GOARCH=amd64 CGO_ENABLED=0 \
  go build -tags windows \
  -ldflags="-s -w" \
  -o "$BIN_DIR/nexus-backend.exe" .

SIZE=$(du -sh "$BIN_DIR/nexus-backend.exe" | cut -f1)
ok "Go 백엔드 빌드 완료: $SIZE"

# ── STEP 2: Rust Windows 타겟 확인 ───────────────────────────
step "STEP 2: Rust 크로스 컴파일 환경 확인"

if ! rustup target list --installed | grep -q "x86_64-pc-windows-gnu"; then
    warn "x86_64-pc-windows-gnu 타겟 추가 중..."
    rustup target add x86_64-pc-windows-gnu
fi
ok "Rust 타겟 준비됨"

# MinGW 링커 확인
if ! command -v x86_64-w64-mingw32-gcc >/dev/null 2>&1; then
    echo ""
    warn "MinGW가 없습니다. Tauri 크로스 컴파일에 필요합니다."
    echo "  설치 명령: brew install mingw-w64"
    echo ""
    echo "  MinGW 없이 계속하려면 Windows PC에서 build-on-windows.ps1을 실행하세요."
    echo "  Go 백엔드만 빌드하고 종료합니다."
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "  Go 백엔드 빌드 완료!"
    echo "  파일 위치: $BIN_DIR/nexus-backend.exe"
    echo ""
    echo "  다음 단계:"
    echo "  1. 프로젝트 폴더를 Windows PC로 전송"
    echo "  2. Windows에서: .\\build-on-windows.ps1 실행"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    exit 0
fi
ok "MinGW 링커 확인됨"

# ── STEP 3: .cargo/config.toml 링커 설정 ─────────────────────
step "STEP 3: 크로스 컴파일 링커 설정"

CARGO_CFG="$ROOT/.cargo/config.toml"
mkdir -p "$(dirname "$CARGO_CFG")"
if ! grep -q "x86_64-pc-windows-gnu" "$CARGO_CFG" 2>/dev/null; then
    cat >> "$CARGO_CFG" << 'EOF'

[target.x86_64-pc-windows-gnu]
linker = "x86_64-w64-mingw32-gcc"
ar     = "x86_64-w64-mingw32-ar"
EOF
    ok ".cargo/config.toml 링커 설정 추가됨"
else
    ok ".cargo/config.toml 이미 설정됨"
fi

# ── STEP 4: npm 의존성 ───────────────────────────────────────
step "STEP 4: 프론트엔드 의존성 설치"
cd "$ROOT"

if [ ! -d "node_modules" ]; then
    warn "node_modules 없음, npm install 실행..."
    npm install
fi
ok "npm 의존성 준비됨"

# ── STEP 5: Tauri 빌드 ───────────────────────────────────────
step "STEP 5: Tauri Windows 앱 빌드 (첫 빌드는 15~20분 소요)"
cd "$ROOT"

echo "  Rust 크레이트 다운로드 중... (최초 1회)"
echo ""

# GNU 타겟으로 크로스 컴파일, NSIS 번들러 없이 raw exe만 생성
npm run tauri build -- \
    --target x86_64-pc-windows-gnu \
    --no-bundle

# raw exe 경로
EXE_PATH="$ROOT/src-tauri/target/x86_64-pc-windows-gnu/release/Nexus.exe"
if [ ! -f "$EXE_PATH" ]; then
    # 다른 가능한 경로 탐색
    EXE_PATH=$(find "$ROOT/src-tauri/target/x86_64-pc-windows-gnu/release" \
        -maxdepth 1 -name "*.exe" ! -name "*build*" 2>/dev/null | head -1)
fi

# ── STEP 6: 배포 패키지 구성 ─────────────────────────────────
step "STEP 6: 배포 패키지 구성"

DIST_DIR="$ROOT/windows-portable"
mkdir -p "$DIST_DIR"

if [ -n "$EXE_PATH" ] && [ -f "$EXE_PATH" ]; then
    cp "$EXE_PATH" "$DIST_DIR/Nexus.exe"
    ok "Nexus.exe 복사 완료"
else
    fail "Tauri exe를 찾을 수 없습니다. Windows에서 build-on-windows.ps1 사용을 권장합니다."
fi

# ── 완료 ─────────────────────────────────────────────────────
echo ""
echo -e "${GREEN}${BOLD}"
echo "  ╔══════════════════════════════════════╗"
echo "  ║  ✅ 빌드 완료!                        ║"
echo "  ╚══════════════════════════════════════╝"
echo -e "${NC}"
echo "  📁 결과물: $DIST_DIR/"
ls -lh "$DIST_DIR/"
echo ""
echo "  📋 Windows PC 전송:"
echo "     USB: $DIST_DIR/Nexus.exe 복사"
echo "     Wi-Fi: python3 -m http.server 9999 → Chrome에서 다운로드"
echo ""
echo "  ▶ Windows에서: Nexus.exe 더블클릭 → 바로 실행!"
echo ""
