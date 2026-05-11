#!/bin/bash
# Mac → Windows 빌드용 소스 압축 스크립트
# 실행: bash mac_to_windows_export.sh

set -e
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
OUTPUT="$HOME/Desktop/nexus-source-$(date +%Y%m%d).zip"

echo "╔══════════════════════════════════════════╗"
echo "║   NEXUS AI — Windows 빌드용 소스 압축    ║"
echo "╚══════════════════════════════════════════╝"
echo ""
echo "프로젝트 경로: $SCRIPT_DIR"
echo "출력 파일:     $OUTPUT"
echo ""

cd "$(dirname "$SCRIPT_DIR")"
PROJECT=$(basename "$SCRIPT_DIR")

echo "압축 중... (node_modules, dist, target 제외)"

zip -r "$OUTPUT" "$PROJECT" \
  --exclude "$PROJECT/node_modules/*" \
  --exclude "$PROJECT/dist/*" \
  --exclude "$PROJECT/src-tauri/target/*" \
  --exclude "$PROJECT/.git/*" \
  --exclude "$PROJECT/nexus-backend.exe" \
  --exclude "$PROJECT/go_build_error.log" \
  --exclude "$PROJECT/build_log.txt" \
  --exclude "$PROJECT/*.zip" \
  --exclude "*.DS_Store" \
  -q

SIZE=$(du -sh "$OUTPUT" | cut -f1)
echo ""
echo "완료: $OUTPUT ($SIZE)"
echo ""
echo "다음 단계:"
echo "  1. 위 ZIP 파일을 Windows PC로 복사 (USB / Google Drive / 이메일)"
echo "  2. Windows에서 압축 해제"
echo "  3. build_windows.bat 더블클릭"
echo ""
