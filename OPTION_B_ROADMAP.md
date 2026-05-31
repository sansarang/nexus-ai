# 🛠️ Option B 로드맵 — Python 의존성 → Go 점진적 이전

## 목표
**현재**: Python sidecar 300-400MB (ML 라이브러리 무거움)
**6개월 후**: Python 의존도 50% 이하, 단일 Go .exe 100MB

## 원칙
- **작동하면서 이전** — 한 번에 한 기능씩, 사용 중인 서비스 중단 X
- **검증 우선** — 새 Go 구현이 Python 결과와 동일한지 자동 테스트
- **클라우드 활용** — 무거운 ML은 외부 API (OpenAI/Anthropic/Groq)

---

## 📋 우선순위 — 무거운 것부터

### 🥇 Wave 1: 임베딩 (Second Brain) — 가장 큰 절감
- **현재**: `sentence-transformers` (~500MB 모델 + faiss)
- **이전**: OpenAI `text-embedding-3-small` API (서버사이드)
- **저장**: faiss → SQLite + 코사인 유사도 검색
- **이득**: -400MB
- **작업**: 2일

### 🥈 Wave 2: OCR — 두 번째 큰 절감
- **현재**: `easyocr` (~200MB GPU 모델)
- **이전**:
  - Mac: macOS Vision API 호출
  - Win: Windows OCR API (PowerShell)
  - 폴백: Google Cloud Vision API
- **이득**: -200MB
- **작업**: 3일

### 🥉 Wave 3: PDF 추출
- **현재**: `PyMuPDF` (~50MB)
- **이전**: Go `unidoc/unipdf` 또는 `ledongthuc/pdf`
- **이득**: -50MB
- **작업**: 1일

### Wave 4: Excel 분석
- **현재**: pandas + openpyxl
- **이전**: Go `xuri/excelize` (이미 사용 중!) + 자체 통계 함수
- **이득**: -100MB
- **작업**: 2일

### Wave 5: 주식 데이터
- **현재**: `yfinance` Python
- **이전**: Go HTTP 클라이언트 + Yahoo Finance API 직접
- **이득**: -30MB
- **작업**: 1일

### Wave 6: 영상 다운로드 (보류)
- **현재**: `yt-dlp` (~80MB)
- **이전**: 어려움 — yt-dlp가 가장 강력
- **결정**: **Python 유지** (대체 불가)
- 분리: `yt-dlp.exe` 단독 다운로드 (이미 NSIS hook에 있음)

### Wave 7: 마우스 자동화
- **현재**: `pyautogui` Python
- **이전**: 이미 Go robotgo 또는 Tauri WebView 자동화로 구현됨
- **결정**: Python 버전 제거 (중복 기능)

---

## 📊 예상 절감 효과

| Wave | 의존성 | 절감 (MB) | 누적 (MB) |
|------|--------|----------|-----------|
| 시작 | All Python | 400 | 400 |
| 1 | -sentence-transformers, -faiss | -400 | 0 → 200 (코어만) |
| 2 | -easyocr | -200 | 100 |
| 3 | -PyMuPDF | -50 | 80 |
| 4 | -pandas (대용 분석만) | -100 | 60 |
| 5 | -yfinance | -30 | 50 |
| 7 | -pyautogui | -10 | 40 |
| **최종** | yt-dlp + 코어만 | | **~80MB** |

→ Go 백엔드 + 가벼운 Python = 180MB (현재 400MB의 절반)

---

## 🗓️ 일정 (제안)

| 시기 | Wave | 비고 |
|------|------|------|
| 1개월차 | Wave 1 (임베딩 → OpenAI) | 가장 큰 절감 |
| 2개월차 | Wave 2 (OCR → OS Native) | 두 번째 큰 |
| 3개월차 | Wave 3, 4 (PDF, Excel) | 작은 절감, 빠른 진행 |
| 4개월차 | Wave 5, 7 (주식, 마우스 정리) | |
| 5-6개월차 | 안정화, 테스트, Python 최소화 | yt-dlp 만 남음 |

---

## ⚠️ 주의사항

1. **각 Wave 후 자동 테스트** — 결과 일치 확인
2. **API 비용 모니터링** — 임베딩/OCR을 클라우드로 이전 시 비용 추적
3. **사용자 통지** — 큰 변경 시 changelog
4. **롤백 가능** — Git tag로 각 Wave 백업

---

## 💡 추가 고려

- **Python 사이드카 자체를 옵션으로** — 사용자가 OCR/Brain 기능 안 쓰면 다운로드 안 하게
- **Lazy loading** — 첫 OCR 호출 시 Python 모듈 다운로드
- **WASM** — 일부 Python 기능을 WASM으로 컴파일 (실험적)

---

*작성: 2026-05-31 — Option A (PyInstaller 빌드) 완료 직후*
