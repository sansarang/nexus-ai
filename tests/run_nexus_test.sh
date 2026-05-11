#!/bin/bash
# Nexus 테스트 실행 스크립트
# 사용법: bash run_nexus_test.sh [검색어]
# 예시:   bash run_nexus_test.sh "에어팟 프로"

QUERY="${1:-에어팟 프로}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

echo "================================================"
echo "  Nexus 실제 동작 테스트"
echo "  검색어: $QUERY"
echo "================================================"

python3 "$SCRIPT_DIR/nexus_real_test.py" "$QUERY"
