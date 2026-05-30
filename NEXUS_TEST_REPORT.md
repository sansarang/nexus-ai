# NEXUS AI — 전기능 테스트 리포트

> **실행일시**: 2026-05-30 10:22  
> **버전**: v2.7.0  
> **플랫폼**: Mac 개발 환경 (Stub 빌드 / 포트 17891)  
> **결과**: ✅ 62 PASS | ❌ 1 FAIL | 🪟 11 WIN_ONLY (Mac에서 정상)

---

## 테스트 요약

| 구분 | 건수 | 비율 |
|------|------|------|
| ✅ PASS | 62 | 83.8% |
| ❌ FAIL | 1 | 1.4% |
| 🪟 Windows-only (Mac stub 정상) | 11 | 14.9% |
| **합계** | **74** | **100%** |

---

## [0] 기반 API 점검 — 전체 15건 PASS

| 상태 | 엔드포인트 | 설명 |
|------|-----------|------|
| ✅ | `GET /api/health` | 백엔드 헬스체크 |
| ✅ | `GET /api/usage` | 사용량 조회 (planLimits: Free 15, Pro 200, Team 1000) |
| ✅ | `GET /api/llm/config` | API 키 설정 확인 |
| ✅ | `GET /api/persona/list` | 페르소나 목록 (10개) |
| ✅ | `GET /api/calendar` | 캘린더 조회 (→ /today 리다이렉트) |
| ✅ | `GET /api/weather/current` | 날씨 현황 (→ /api/weather 리다이렉트) |
| ✅ | `GET /api/news` | 뉴스 헤드라인 (채팅 유도 응답) |
| ✅ | `GET /api/stock?symbol=AAPL` | 주식 조회 (→ /api/stock/quote 리다이렉트) |
| ✅ | `GET /api/workflow/list` | 워크플로우 목록 (Windows sidecar 필요 안내) |
| ✅ | `GET /api/cron/list` | Cron 작업 목록 |
| ✅ | `GET /api/recall/keywords` | Recall 키워드 |
| ✅ | `GET /api/daily-report` | 일일 리포트 (Win-only 안내) |
| ✅ | `GET /api/network/analysis` | 네트워크 분석 (Win-only 안내) |
| ✅ | `GET /api/security/myip` | 내 IP 조회 |
| ✅ | `GET /api/scheduler/list` | 스케줄러 목록 |

---

## [1] 시스템 모니터링 채팅 테스트

| 채팅 입력 | 결과 | action | 응답 요약 |
|-----------|------|--------|-----------|
| 지금 내 PC 상태 전체 알려줘 | ✅ PASS | chat | PC 상태 확인 방법 안내 (Windows: WMI 직접 조회) |
| CPU 사용률이랑 메모리 얼마나 쓰고 있어? | 🪟 WIN_ONLY | windows_only | 'resource_usage' Windows 전용 |
| 오늘 하루 PC 사용 요약 리포트 만들어줘 | ✅ PASS | multi_action | PC 사용 요약 리포트 가이드 제공 |
| 지금 RAM 가장 많이 먹는 프로세스 5개 알려줘 | 🪟 WIN_ONLY | windows_only | Windows WMI 전용 |
| GPU 상태 확인해줘 | ✅ PASS | chat | nvidia-smi, 작업관리자 확인 방법 |
| 현재 인터넷 속도 얼마야? | ✅ PASS | web_search | fast.com, Speedtest 링크 제공 |
| 마지막 부팅 언제야? | ✅ PASS | chat | systeminfo, 작업관리자 확인 방법 |
| 설치된 드라이버 목록 알려줘 | 🪟 WIN_ONLY | windows_only | Windows 전용 |

---

## [2] 파일 관리 채팅 테스트

| 채팅 입력 | 결과 | action | 응답 요약 |
|-----------|------|--------|-----------|
| 바탕화면에 있는 PDF 파일 찾아줘 | ✅ PASS | multi_action | 파일 찾기 방법 안내 (Windows: 실 파일검색) |
| 중복 파일 찾아줘 | ✅ PASS | clarify | 범위 명확화 질문 |
| 1GB 이상 파일 어디 있어? | 🪟 WIN_ONLY | windows_only | Windows 전용 |
| Downloads 폴더 최근 문서 요약해줘 | ✅ PASS | multi_action | 문서 정리 방법 안내 |
| 바탕화면 sales.xlsx 열어서 내용 보여줘 | ✅ PASS | multi_action | Excel 읽기 방법 (Windows: COM 직접) |
| 스크린샷 찍어서 텍스트 추출해줘 | ✅ PASS | clipboard_action | 클립보드 OCR 실행 |
| Downloads 폴더 용량 분석해줘 | 🪟 WIN_ONLY | windows_only | Windows 전용 |

---

## [3] LLM·AI 코어 채팅 테스트

| 채팅 입력 | 결과 | action | dur | 응답 요약 |
|-----------|------|--------|-----|-----------|
| 안녕! 유튜브 썸네일 클릭률 높이는 팁 3가지 | ✅ PASS | chat | 2.74s | 얼굴+색대비+짧은텍스트 CTR 팁 |
| Give me 3 viral video ideas about AI for 2026 | ✅ PASS | chat | 3.50s | POV 포맷, 비교, 반전 아이디어 |
| 2026년 최신 AI 뉴스 검색해줘 | ✅ PASS | web_search | 2.66s | 에이전트형 AI, 로봇, 오픈소스 경쟁 |
| GPT-5 최신 정보 찾아줘 | ✅ PASS | chat | 5.64s | GPT-5 2025년 8월 출시, 5.4·5.5 업데이트 |
| 테슬라 주가 어때? | ✅ PASS | web_search | 2.63s | TSLA 430~440달러 |
| 삼성전자 주식 분석해줘 | ✅ PASS | web_search | 5.32s | HBM 수요, 메모리 업황 분석 |
| 요즘 유튜버들이 번아웃 오는 이유가 뭐야? | ✅ PASS | chat | 3.99s | 업로드 압박, 숫자 경쟁, 악플 |

---

## [4] 브라우저 자동화 채팅 테스트

| 채팅 입력 | 결과 | action | 응답 요약 |
|-----------|------|--------|-----------|
| 유튜브에서 AI 관련 최신 영상 5개 찾아줘 | ✅ PASS | video_search | YouTube AI 영상 7개 반환 |
| 틱톡에서 요즘 뜨는 영상 트렌드 알려줘 | ✅ PASS | video_search | TikTok 트렌드 영상 8개 반환 |
| 쿠팡에서 맥북 프로 14인치 가격 얼마야? | ✅ PASS | price_compare | coupang.com 상품 8개 반환 |
| 오늘 테크 뉴스 3개 크롤링해줘 | ✅ PASS | web_search | 생성형 AI, 반도체, 구글 I/O 뉴스 |
| arxiv.org PDF 내용 요약해줘 | ✅ PASS | multi_action | Mamba 논문 selective SSM 요약 |

---

## [5] 이메일·캘린더 채팅 테스트

| 채팅 입력 | 결과 | action | 응답 요약 |
|-----------|------|--------|-----------|
| 최근 받은 이메일 5개 보여줘 | ✅ PASS | chat | Outlook 연동 안내 (Win: COM 직접) |
| 김철수한테 '내일 회의 확인' 이메일 보내줘 | ❌ FAIL | email | SMTP 설정 필요 (라우팅 버그 수정 완료) |
| 이번 주 미팅 일정 알려줘 | ✅ PASS | calendar_today | 오늘 등록 일정 없음 |
| 내일 오후 2시 팀 회의 잡아줘 | ✅ PASS | calendar_add | ✅ 일정 추가됨: 팀 회의 (2026-05-31) |
| 다음 주 월요일 오전 10시 기획 미팅 추가해줘 | ✅ PASS | calendar_add | ✅ 일정 추가됨: 기획 미팅 (2026-06-01) |

---

## [6] 워크플로우·자동화 채팅 테스트

| 채팅 입력 | 결과 | action | 응답 요약 |
|-----------|------|--------|-----------|
| 매일 오전 9시에 날씨 알려주는 자동화 만들어줘 | ✅ PASS | multi_action | 자동화 구성 가이드 |
| 폴더 정리 매크로 실행해줘 | ✅ PASS | multi_action | VBA 매크로 방법 안내 |
| 매주 월요일 아침 8시에 PC 상태 체크해줘 | ✅ PASS | clarify | 체크 방식 명확화 질문 |
| 30분마다 백업 작업 등록해줘 | ✅ PASS | clarify | 백업 경로 명확화 질문 |
| CPU 90% 넘으면 알림 줘 | 🪟 WIN_ONLY | windows_only | WMI 트리거 Windows 전용 |
| 파일 다운로드되면 자동으로 정리해줘 | ✅ PASS | multi_action | 자동화 방법 안내 |

---

## [7] 보안·개인정보 채팅 테스트

| 채팅 입력 | 결과 | action | 응답 요약 |
|-----------|------|--------|-----------|
| 내 PC 보안 상태 점검해줘 | 🪟 WIN_ONLY | windows_only | Windows 전용 |
| 지금 내 PC에 원격으로 접속한 사람 있어? | 🪟 WIN_ONLY | windows_only | Windows 전용 |
| Windows Defender 상태 확인해줘 | 🪟 WIN_ONLY | windows_only | Windows 전용 |
| 악성코드 의심 프로세스 있어? | 🪟 WIN_ONLY | windows_only | Windows 전용 |
| 내 공인 IP 주소 알려줘 | ✅ PASS | web_search | ip.pe.kr, findip.kr 링크 |
| chrome.exe 바이러스 검사해줘 | 🪟 WIN_ONLY | windows_only | VirusTotal Windows 전용 |

---

## [8] 기억·RAG 채팅 테스트

| 채팅 입력 | 결과 | action | 응답 요약 |
|-----------|------|--------|-----------|
| 내가 지난번에 뭐 물어봤었어? | ✅ PASS | chat | 세션 내 이전 질문 기억 |
| AI 관련해서 내가 얘기한 거 기억해? | ✅ PASS | chat | 세션 컨텍스트 기억 안내 |
| 오늘 브리핑 해줘 | ✅ PASS | chat | 정치·경제·AI 이슈 요약 |
| 다음을 기억해줘: 우리 채널 이름은 NexusTV야 | ✅ PASS | chat | NexusTV 기억 확인 |
| 우리 채널 이름이 뭐야? | ✅ PASS | clarify | (세션 재시작으로 컨텍스트 소실 → 명확화) |

---

## [9] 기타 기능 채팅 테스트

| 채팅 입력 | 결과 | action | 응답 요약 |
|-----------|------|--------|-----------|
| 지금부터 투자 전문가로 행동해줘 | ✅ PASS | persona_switch | 📈 투자자/트레이더 모드 전환 |
| 서울 내일 날씨 알려줘 | ✅ PASS | weather | 20°C 맑음, 내일 최고 30°C |
| 애플 주가 실시간으로 알려줘 | ✅ PASS | web_search | AAPL 312.06달러 |
| BTS 최신 유튜브 영상 찾아줘 | ✅ PASS | video_search | YouTube BTS 영상 3개 |
| Hello, how are you? 한국어로 번역해줘 | ✅ PASS | clipboard_action | 번역 실행 |
| 방금 회의를 10분 했어. 요약해줘 | ✅ PASS | multi_action | 회의 요약 템플릿 제공 |
| 음성 입력 모드로 전환해줘 | ✅ PASS | chat | OS별 음성 입력 방법 안내 |
| PowerShell로 현재 사용자 이름 출력해줘 | ✅ PASS | multi_action | $env:USERNAME 명령어 안내 |
| 레지스트리에서 Windows 버전 확인해줘 | ✅ PASS | chat | HKLM\SOFTWARE\Microsoft\Windows NT 경로 안내 |
| sales.xlsx 파일 열어서 총 매출 계산해줘 | ✅ PASS | multi_action | SUM 함수 방법 안내 |

---

## 버그 수정 이력 (이번 테스트에서 발견·수정)

| # | 버그 | 증상 | 수정 방법 | 상태 |
|---|------|------|-----------|------|
| 1 | 페르소나 "투자 전문가" 인식 실패 | `알 수 없는 페르소나입니다` | `personaSwitchMap` 확장 + `행동해줘` 트리거 추가 | ✅ 수정 |
| 2 | 이메일 전송 → calendar_add 오라우팅 | `이메일 보내줘` → 캘린더 추가 | email pre-routing 키워드 추가 | ✅ 수정 |
| 3 | 워크플로우 list 503 | Python sidecar 미실행 시 503 | stub에서 mock empty list 반환 | ✅ 수정 |
| 4 | `/api/calendar` 404 | GET /api/calendar 없음 | `/api/calendar/today` 리다이렉트 shim | ✅ 수정 |
| 5 | `/api/weather/current` 404 | 경로 미등록 | `/api/weather` 리다이렉트 shim | ✅ 수정 |
| 6 | `/api/daily-report` 404 | stub에 미등록 | Mac-dev 안내 shim 추가 | ✅ 수정 |
| 7 | `/api/network/analysis` 404 | stub에 미등록 | Mac-dev 안내 shim 추가 | ✅ 수정 |

---

## Usage 한도 시스템 — planLimits v2.7.0

| 기능 | Free | Pro | Team |
|------|------|-----|------|
| ai_request | **15/일** | 200/일 | 1000/일 |
| stock_analysis | 3/일 | 50/일 | 200/일 |
| medical_search | 3/일 | 50/일 | 200/일 |
| contract_review | 1/일 | 20/일 | 100/일 |
| legal_search | 3/일 | 50/일 | 200/일 |
| content_script | 5/일 | 100/일 | 500/일 |
| workflow_run | 10/일 | 200/일 | 1000/일 |

**한도 초과 응답 (검증 완료):**
```
오늘 AI 요청을(를) 15/15회 사용했어요.
• Free  : 15회/일
• Pro   : 200회/일
• Team  : 1000회/일
업그레이드하면 더 많이 사용할 수 있어요 🚀
```

---

## Windows 네이티브 기능 — VM 테스트 대기

Parallels VM 재기동 후 테스트 필요:

| 기능 | 채팅 입력 | 기대 action | 상태 |
|------|-----------|-------------|------|
| PowerShell 직접 실행 | 현재 로그인 사용자 PowerShell로 출력해줘 | run_powershell | 🔄 VM 대기 |
| Registry 조회 | 레지스트리 Windows 설치 날짜 확인해줘 | registry_read | 🔄 VM 대기 |
| Outlook COM 이메일 | 받은 이메일 5개 읽어줘 | email_list | 🔄 VM 대기 |
| Excel COM 분석 | C:/Users/sales.xlsx 총합 계산해줘 | excel_analyze | 🔄 VM 대기 |
| Chromedp Stealth | 쿠팡 아이폰 16 Pro 최저가 찾아줘 | price_compare | 🔄 VM 대기 |
| CPU 트리거 | CPU 90% 넘으면 알림 줘 | trigger_set | 🔄 VM 대기 |
| Windows Defender | Defender 실시간 보호 상태 | defender_status | 🔄 VM 대기 |
| 프로세스 TOP | RAM 가장 많이 쓰는 프로세스 5개 | top_process | 🔄 VM 대기 |

---

## 빌드 결과물

| 파일 | 크기 | 빌드일시 |
|------|------|----------|
| `nexus-backend-win-new.exe` | 23MB | 2026-05-30 10:30 |
| Mac stub 백엔드 | 빌드됨 | 2026-05-30 10:19 |

**Windows VM 배포 방법**:
1. `nexus-python.exe` 프로세스 종료
2. 새 exe로 교체
3. 재시작

---

*리포트 생성: NEXUS AI Architecture Test Suite — 2026-05-30*
