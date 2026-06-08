# Nexus AI 전체 기능 테스트 가이드

설치된 Nexus 앱의 기능을 **기능별로 한 번에** 두드려 보고 PASS/FAIL/SKIP을 확인하는 도구입니다.

## 1. 사전 조건
- **Nexus 앱이 실행 중**이어야 합니다 (트레이에 아이콘 → 백엔드 `127.0.0.1:17891` + 사이드카 `127.0.0.1:17893`).
- Python 3.x 설치. (선택) `pip install openpyxl` → xlsx 픽스처까지 생성.

## 2. 실행 (둘 중 하나)
**A. 더블클릭 (Windows 권장)**
```
scripts\run_feature_test.bat
```
**B. 직접 실행**
```
python scripts\feature_test.py --setup --run     # 픽스처 생성 + 전체 테스트
python scripts\feature_test.py --setup           # 픽스처(파일/샘플영상)만 생성
python scripts\feature_test.py                   # 테스트만 (픽스처 이미 있을 때)
python scripts\feature_test.py --only 자동화,비전   # 특정 카테고리만
```

## 3. 픽스처 (자동 생성 → `~/nexus_feature_test/`)
| 파일 | 용도 |
|---|---|
| `messy/` (jpg·png·pdf·txt·csv·md·mp3·zip 8개) | 파일 자동정리 테스트 |
| `test.txt` / `test.pdf` / `test.png` | 문서요약 / PDF / 비전 |
| `test.xlsx` (openpyxl 있을 때) | 엑셀 |
| `sample.mp4` (자동 다운로드, ~2.8MB) | 영상 기능 |

## 4. 결과 해석
- **✅ PASS** — 기능이 정상 응답.
- **⏭️ SKIP** — 정상적인 건너뜀. 사유:
  - **API 키 미설정** — 설치본은 번들 키로 동작하므로 보통 PASS. 키 없는 개발환경에선 SKIP.
  - **Windows 전용** — 클립보드/데스크탑 자동화 등은 Mac에서 SKIP.
  - **자동화 엔진 미준비** — `run 실행`이 **501**이면 SKIP. UIA(pywinauto)가 아직 `Available=false`라는 뜻 (정상). Windows에서 pywinauto 검증 완료 후 활성화됩니다.
  - **Claude 키 필요** — 스크린샷 분석(Claude Vision)은 Claude 키 없으면 SKIP.
- **❌ FAIL** — 실제 문제. 하단 "실패 항목"에서 사유 확인.

## 5. 기능별 기대치
| 카테고리 | 항목 | Windows 설치본 기대 |
|---|---|---|
| 헬스 | /api/health, /health, automation status, uia status | ✅ (uia status는 available=false여도 응답 PASS) |
| LLM | 한국어/영어 응답 | ✅ (번들 키) |
| 검색 | 웹·딥서치·뉴스·영상검색 | ✅ (Tavily 번들 키) |
| 정보 | 날씨·주가·환율·길찾기 | ✅ |
| 시스템 | PC상태·보안스캔·번역 | ✅ (PC상태는 실제 시스템 값) |
| 파일 | 파일정리·엑셀작성·문서요약 | ✅ |
| 생산성 | 스케줄·클립보드·워크플로·페르소나 | ✅ (클립보드는 Windows 실동작) |
| 비전 | 스크린샷 분석 | Claude 키 있으면 ✅, 없으면 SKIP |
| 영상 | 샘플 픽스처·영상 다운로드 | 픽스처 ✅, 다운로드는 yt-dlp/ffmpeg 필요 |
| **자동화** | 워크플로 저장·목록·조회·dry-run | ✅ (오케스트레이션) |
| **자동화** | run 실행 | UIA 준비 전 **501 SKIP**(정상), 준비 후 ✅ |

## 6. 자동화 엔진(신규) 주의
- 워크플로 **저장/목록/조회/dry-run**은 키·플랫폼 무관하게 PASS여야 합니다(오케스트레이션 검증됨).
- **run 실행**은 안전상 `Available()=true`일 때만 동작합니다. 그 전엔 **501로 거부**(SKIP)되며 이는 의도된 안전 게이트입니다.
- 실제 UIA 동작/95% 신뢰성은 `scripts/uia_poc_webform.py` + `docs/automation/windows-qa-runbook.md`로 별도 검증.

## 7. 종료 코드
- `0` = FAIL 0건, `1` = FAIL 존재, `2` = 백엔드 미기동(앱 먼저 실행).
