# Nexus Windows 빌드 및 실행

## 1. Mac에서 Windows .exe 빌드

```bash
cd /Users/youngjinjung/Desktop/뚝딱PC/backend
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -tags windows -ldflags="-s -w" -o nexus-backend.exe .
```

빌드 성공 시: `nexus-backend.exe` (~15MB) 생성됨

## 2. Windows에서 실행

`nexus-backend.exe`를 더블클릭 (또는 CMD):
```cmd
nexus-backend.exe
```

백그라운드에서 `127.0.0.1:17891` 포트로 서버 시작됨.

## 3. 기능 테스트 — PowerShell로 직접 확인

### ① 헬스체크
```powershell
Invoke-RestMethod http://localhost:17891/api/health
# → {"status":"ok"}
```

### ② 에어팟 프로 검색 → PDF 생성 (핵심 기능)
```powershell
$body = @{
    query     = "에어팟 프로"
    max_items = 5
    open_after = $true
} | ConvertTo-Json -Compress

$result = Invoke-RestMethod -Method POST `
    -Uri "http://localhost:17891/api/browser/search-and-pdf" `
    -ContentType "application/json" `
    -Body $body

Write-Host "PDF 경로: $($result.pdf_path)"
Write-Host "제품 수: $($result.item_count)"
Write-Host "요약: $($result.summary)"
```

**결과**: 바탕화면에 `에어팟_프로_20260505_XXXXXX.pdf` 파일 생성됨  
Chrome이 열려서 PDF 렌더링 후 자동으로 저장됨.

### ③ 어떤 제품이든 동적으로 처리
```powershell
# 삼성 노트북
$body = @{ query = "삼성 갤럭시북 노트북"; max_items = 5 } | ConvertTo-Json
Invoke-RestMethod -Method POST -Uri "http://localhost:17891/api/browser/search-and-pdf" -ContentType "application/json" -Body $body

# 다이슨 청소기
$body = @{ query = "다이슨 청소기 2026"; max_items = 5 } | ConvertTo-Json
Invoke-RestMethod -Method POST -Uri "http://localhost:17891/api/browser/search-and-pdf" -ContentType "application/json" -Body $body

# 삼성전자 주가
$body = @{ query = "삼성전자 주가 목표주가"; max_items = 5 } | ConvertTo-Json
Invoke-RestMethod -Method POST -Uri "http://localhost:17891/api/browser/search-and-pdf" -ContentType "application/json" -Body $body
```

## 4. 동작 흐름 (실제 Windows 실행 시)

```
사용자 → Nexus UI → "에어팟 프로 제품설명서 만들어줘"
            ↓
Groq LLM → search_and_pdf 도구 선택
            ↓
Go 백엔드 → POST /api/browser/search-and-pdf
            ↓
Chrome (스텔스) → 쿠팡 검색
  → 봇 차단 시: 자동 fallback (공식 데이터)
            ↓
Groq AI → 제품 분석 요약 생성
            ↓
HTML 제품설명서 → Chrome DevTools PrintToPDF
            ↓
바탕화면에 PDF 저장 + 자동 열기
            ↓
Nexus UI: "✅ PDF 저장됨: C:\Users\...\Desktop\에어팟_프로_20260505.pdf"
```

## 5. 필요 조건 (Windows)

| 항목 | 조건 |
|------|------|
| Chrome | 설치되어 있어야 함 (C:\Program Files\Google\Chrome 또는 Program Files (x86)) |
| Windows 버전 | Windows 10/11 (64비트) |
| 인터넷 연결 | 쿠팡 검색 + Groq API 호출 |
| Groq API 키 | Nexus UI 설정에서 입력 |

## 6. Groq API 키 설정 (첫 실행 시)

Nexus UI → 설정 → API 키 → Groq API Key 입력  
또는 환경 변수: `GROQ_API_KEY=your-key-here nexus-backend.exe`
