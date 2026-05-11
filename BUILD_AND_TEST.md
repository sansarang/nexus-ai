# Nexus 빌드 및 실제 테스트 가이드

## 현재 구현된 기능 (실제 작동)

### ★ 핵심: 사용자가 말하면 → 실제 결과물(PDF) 생성

```
사용자: "에어팟 프로 제품설명서 만들어줘"
        ↓
Nexus: search_and_pdf 도구 실행
        ↓
Go 백엔드: POST /api/browser/search-and-pdf
        ↓
1. Chrome (스텔스 모드) → 쿠팡 검색
2. 봇 차단 시 → Apple 공식 데이터 자동 사용
3. Groq AI → 제품 분석 요약
4. HTML 제품설명서 생성
5. Chrome DevTools PrintToPDF → PDF 저장
6. 바탕화면에 PDF 파일 생성
        ↓
사용자 화면에: "✅ PDF 생성 완료! 📄 C:\Users\xxx\Desktop\에어팟_프로_20260505_090000.pdf"
```

## Mac에서 Windows용 빌드

```bash
cd /Users/youngjinjung/Desktop/뚝딱PC/backend

# Windows 64비트 빌드
GOOS=windows GOARCH=amd64 go build -tags windows -o nexus-backend.exe .

# Tauri 빌드 (전체 앱)
cd ..
npm run tauri build
```

## Windows에서 직접 테스트

### 1. 백엔드 단독 실행
```cmd
nexus-backend.exe
```

### 2. API 직접 테스트 (PowerShell)
```powershell
# 에어팟 검색 → PDF 생성 테스트
$body = @{
    query = "에어팟 프로"
    max_items = 5
    open_after = $true
} | ConvertTo-Json

Invoke-RestMethod -Method POST `
  -Uri "http://127.0.0.1:17891/api/browser/search-and-pdf" `
  -ContentType "application/json" `
  -Body $body

# 결과: 바탕화면에 PDF 파일 생성됨
```

### 3. 다른 요청 예시 (동적으로 무엇이든 처리)
```powershell
# 삼성 노트북 최저가 비교
$body = @{ query = "삼성 갤럭시북 노트북 최저가"; max_items = 10 } | ConvertTo-Json
Invoke-RestMethod -Method POST -Uri "http://localhost:17891/api/browser/search-and-pdf" -ContentType "application/json" -Body $body

# 다이슨 청소기 비교
$body = @{ query = "다이슨 청소기 2026 신모델"; max_items = 5 } | ConvertTo-Json
Invoke-RestMethod -Method POST -Uri "http://localhost:17891/api/browser/search-and-pdf" -ContentType "application/json" -Body $body

# 주식 정보 PDF
$body = @{ query = "삼성전자 주가 목표주가 2026"; max_items = 5 } | ConvertTo-Json
Invoke-RestMethod -Method POST -Uri "http://localhost:17891/api/browser/search-and-pdf" -ContentType "application/json" -Body $body
```

## 동작 원리 — "모든 상황이 동적으로 제어"

```
사용자 자연어 명령
        ↓
Groq LLM (llama-3.3-70b) → 어떤 도구를 쓸지 자동 결정
        ↓
┌─────────────────────────────────────────────────────────┐
│  검색+PDF 요청    → search_and_pdf 도구                  │
│  가격 비교 요청   → browser_collect_price 도구           │
│  뉴스/주가 요청   → browser_news_collect 도구            │
│  일정 자동화     → schedule_task 도구                    │
│  파일 관리       → Go 직접 실행 (LLM 없이)               │
│  PC 제어         → Go 직접 실행 (LLM 없이)               │
└─────────────────────────────────────────────────────────┘
        ↓
실제 결과물 생성:
  - PDF 파일 (바탕화면)
  - Excel 파일 (바탕화면)
  - 화면 표시 (카드)
  - 알림 / 메일 발송
```

## 현재 구현 완료 목록

| 기능 | 파일 | 상태 |
|------|------|------|
| 스텔스 브라우저 | handlers_browser_stealth.go | ✅ |
| 검색+PDF 생성 | handlers_browser_pdf.go | ✅ |
| 브라우저 에이전트 | handlers_browser_agent.go | ✅ |
| Excel 저장 | handlers_excel.go | ✅ |
| 자연어 스케줄러 | handlers_scheduler.go | ✅ |
| 장기 메모리 | handlers_memory.go | ✅ |
| API 라우트 | main.go | ✅ |
| 프론트 도구 스키마 | gemini_engine.ts | ✅ |
| 프론트 API 클라이언트 | backendAPI.ts | ✅ |

## 안티봇 처리 전략

쿠팡/네이버 봇 차단 시 자동 처리:

1. **1차**: Chrome 스텔스 모드 (webdriver 숨김, 인간 행동 시뮬레이션)
2. **2차**: 재시도 (지수 백오프, 최대 3회)
3. **3차**: 공식 데이터 fallback (결과는 항상 생성됨)
4. **결과**: PDF는 반드시 생성 (봇 차단 여부와 무관)
