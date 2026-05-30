#!/usr/bin/env python3
"""
NEXUS AI 전기능 테스트 스크립트
채팅 지시 → 실제 응답 기록
실행: python3 nexus-fulltest.py
"""

import json, requests, time, sys, os
from datetime import datetime

BASE   = "http://127.0.0.1:17891"   # Mac stub 백엔드
WIN    = "http://127.0.0.1:17892"   # Windows VM (SSH 터널)
OUT    = os.path.expanduser("~/Desktop/뚝딱PC/NEXUS_TEST_REPORT.md")

results = []
PASS = FAIL = WINDOWS_ONLY = 0

def chat(label, message, base=BASE, timeout=30, expect_action=None):
    global PASS, FAIL, WINDOWS_ONLY
    ts = datetime.now().strftime("%H:%M:%S")
    try:
        r = requests.post(f"{base}/api/command",
                          json={"message": message, "user_email": "test@nexus.ai"},
                          timeout=timeout)
        d = r.json()
    except Exception as e:
        d = {"success": False, "action": "timeout", "message": str(e), "duration": "?"}

    action  = d.get("action", "?")
    success = d.get("success", False)
    msg     = str(d.get("message", ""))[:200]
    dur     = d.get("duration", "?")

    if action == "windows_only":
        tag = "🪟 WIN_ONLY"
        WINDOWS_ONLY += 1
    elif action == "timeout" or not r if 'r' in dir() else False:
        tag = "⏱ TIMEOUT"
        FAIL += 1
    elif success:
        tag = "✅ PASS"
        PASS += 1
    else:
        tag = "⚠️  FAIL"
        FAIL += 1

    if expect_action and action != expect_action and action != "windows_only":
        tag = f"❌ WRONG({action})"
        FAIL += 1

    row = {
        "ts": ts, "label": label, "input": message,
        "tag": tag, "action": action, "success": success,
        "message": msg, "duration": dur
    }
    results.append(row)

    col = "\033[92m" if "PASS" in tag else ("\033[93m" if "WIN" in tag else "\033[91m")
    rst = "\033[0m"
    print(f"{col}[{tag}]{rst} {label}")
    print(f"       입력  : {message[:70]}")
    print(f"       action: {action}  dur={dur}")
    print(f"       응답  : {msg[:120]}")
    print()
    return d

def api(label, method, path, body=None, base=BASE):
    global PASS, FAIL
    try:
        if method == "GET":
            r = requests.get(f"{base}{path}", timeout=10)
        else:
            r = requests.post(f"{base}{path}", json=body or {}, timeout=10)
        d = r.json()
        tag = "✅ PASS" if r.status_code < 400 else f"❌ {r.status_code}"
        if r.status_code < 400:
            PASS += 1
        else:
            FAIL += 1
    except Exception as e:
        d = {"error": str(e)}
        tag = "❌ ERR"
        FAIL += 1

    print(f"{tag} [{label}] {method} {path}")
    results.append({"ts": datetime.now().strftime("%H:%M:%S"),
                    "label": label, "input": f"{method} {path}",
                    "tag": tag, "action": "api", "message": str(d)[:150]})
    return d

# ══════════════════════════════════════════════════════════
print("\n" + "═"*65)
print(" NEXUS AI 전기능 테스트  —  " + datetime.now().strftime("%Y-%m-%d %H:%M"))
print("═"*65)

# ──────────────────────────────────────────────────────────
print("\n▶ [0] 기반 API 점검")
api("헬스체크",         "GET", "/api/health")
api("사용량 조회",      "GET", "/api/usage")
api("LLM 설정 조회",    "GET", "/api/llm/config")
api("페르소나 목록",    "GET", "/api/persona/list")
api("캘린더 조회",      "GET", "/api/calendar")
api("날씨 현황",        "GET", "/api/weather/current")
api("뉴스 헤드라인",    "GET", "/api/news")
api("주식 조회",        "GET", "/api/stock?symbol=AAPL")
api("워크플로우 목록",  "GET", "/api/workflow/list")
api("크론 목록",        "GET", "/api/cron/list")
api("Recall 키워드",    "GET", "/api/recall/keywords")
api("데일리 리포트",    "GET", "/api/daily-report")
api("네트워크 분석",    "GET", "/api/network/analysis")
api("보안 내 IP",       "GET", "/api/security/myip")
api("스케줄러 목록",    "GET", "/api/scheduler/list")

# ──────────────────────────────────────────────────────────
print("\n▶ [1] 시스템 모니터링 채팅 테스트")
chat("PC 상태 요약",       "지금 내 PC 상태 전체 알려줘")
chat("CPU·RAM 확인",       "CPU 사용률이랑 메모리 얼마나 쓰고 있어?")
chat("일일 리포트",        "오늘 하루 PC 사용 요약 리포트 만들어줘")
chat("TOP 프로세스",       "지금 RAM 가장 많이 먹는 프로세스 5개 알려줘")
chat("GPU 상태",           "GPU 상태 확인해줘")
chat("네트워크 속도",      "현재 인터넷 속도 얼마야?")
chat("부팅 시간",          "마지막 부팅 언제야?")
chat("드라이버 정보",      "설치된 드라이버 목록 알려줘")

# ──────────────────────────────────────────────────────────
print("\n▶ [2] 파일 관리 채팅 테스트")
chat("파일 검색",          "바탕화면에 있는 PDF 파일 찾아줘")
chat("중복 파일",          "중복 파일 찾아줘")
chat("대용량 파일",        "1GB 이상 파일 어디 있어?")
chat("문서 AI 요약",       "Downloads 폴더 최근 문서 요약해줘")
chat("엑셀 읽기",          "바탕화면 sales.xlsx 열어서 내용 보여줘")
chat("OCR 실행",           "스크린샷 찍어서 텍스트 추출해줘")
chat("폴더 분석",          "Downloads 폴더 용량 분석해줘")

# ──────────────────────────────────────────────────────────
print("\n▶ [3] LLM·AI 코어 채팅 테스트")
chat("기본 채팅 한국어",   "안녕! 유튜브 썸네일 클릭률 높이는 팁 3가지 알려줘")
chat("기본 채팅 영어",     "Give me 3 viral video ideas about AI for 2026")
chat("웹 검색",            "2026년 최신 AI 뉴스 검색해줘",         timeout=45)
chat("Perplexity 검색",    "GPT-5 최신 정보 찾아줘",               timeout=45)
chat("다단계 질문1",       "테슬라 주가 어때?")
chat("다단계 질문2",       "삼성전자 주식 분석해줘")
chat("감정분석",           "요즘 유튜버들이 번아웃 오는 이유가 뭐야?")

# ──────────────────────────────────────────────────────────
print("\n▶ [4] 브라우저 자동화 채팅 테스트")
chat("유튜브 영상 검색",   "유튜브에서 AI 관련 최신 영상 5개 찾아줘",    timeout=45)
chat("TikTok 트렌드",      "틱톡에서 요즘 뜨는 영상 트렌드 알려줘",      timeout=45)
chat("쿠팡 가격 비교",     "쿠팡에서 맥북 프로 14인치 가격 얼마야?",     timeout=45)
chat("뉴스 크롤링",        "오늘 테크 뉴스 3개 크롤링해줘",             timeout=45)
chat("PDF 추출",           "https://arxiv.org/pdf/2312.00752 PDF 내용 요약해줘", timeout=60)

# ──────────────────────────────────────────────────────────
print("\n▶ [5] 이메일·캘린더 채팅 테스트")
chat("Outlook 받은 편지함", "최근 받은 이메일 5개 보여줘")
chat("이메일 답장",         "김철수한테 '내일 회의 확인' 이메일 보내줘")
chat("미팅 일정 조회",      "이번 주 미팅 일정 알려줘")
chat("미팅 자동 제안",      "내일 오후 2시 팀 회의 잡아줘")
chat("캘린더 이벤트 추가",  "다음 주 월요일 오전 10시 기획 미팅 추가해줘")

# ──────────────────────────────────────────────────────────
print("\n▶ [6] 워크플로우·자동화 채팅 테스트")
chat("워크플로우 생성",    "매일 오전 9시에 날씨 알려주는 자동화 만들어줘")
chat("매크로 실행",        "폴더 정리 매크로 실행해줘")
chat("스케줄러 등록",      "매주 월요일 아침 8시에 PC 상태 체크해줘")
chat("Cron 작업",          "30분마다 백업 작업 등록해줘")
chat("트리거 설정",        "CPU 90% 넘으면 알림 줘")
chat("워크플로우 텍스트",  "파일 다운로드되면 자동으로 정리해줘")

# ──────────────────────────────────────────────────────────
print("\n▶ [7] 보안·개인정보 채팅 테스트")
chat("보안 상태 점검",     "내 PC 보안 상태 점검해줘")
chat("원격접속 탐지",      "지금 내 PC에 원격으로 접속한 사람 있어?")
chat("Windows Defender",   "Windows Defender 상태 확인해줘")
chat("의심 프로세스",      "악성코드 의심 프로세스 있어?")
chat("내 IP 확인",         "내 공인 IP 주소 알려줘")
chat("VirusTotal",         "chrome.exe 바이러스 검사해줘")

# ──────────────────────────────────────────────────────────
print("\n▶ [8] 기억·RAG 채팅 테스트")
chat("대화 기억",          "내가 지난번에 뭐 물어봤었어?")
chat("키워드 회상",        "AI 관련해서 내가 얘기한 거 기억해?")
chat("일일 브리핑",        "오늘 브리핑 해줘")
chat("지식 저장",          "다음을 기억해줘: 우리 채널 이름은 NexusTV야")
chat("저장 확인",          "우리 채널 이름이 뭐야?")

# ──────────────────────────────────────────────────────────
print("\n▶ [9] 기타 기능 채팅 테스트")
chat("페르소나 전환",      "지금부터 투자 전문가로 행동해줘")
chat("날씨",               "서울 내일 날씨 알려줘")
chat("주식",               "애플 주가 실시간으로 알려줘")
chat("미디어 검색",        "BTS 최신 유튜브 영상 찾아줘",           timeout=45)
chat("번역",               "Hello, how are you? 한국어로 번역해줘")
chat("미팅 요약",          "방금 회의를 10분 했어. 요약해줘")
chat("음성 받아쓰기",      "음성 입력 모드로 전환해줘")
chat("PowerShell 실행",    "PowerShell로 현재 사용자 이름 출력해줘")
chat("레지스트리 조회",    "레지스트리에서 Windows 버전 확인해줘")
chat("Excel 분석",         "sales.xlsx 파일 열어서 총 매출 계산해줘")

# ══════════════════════════════════════════════════════════
# 결과 집계
# ══════════════════════════════════════════════════════════
total = PASS + FAIL + WINDOWS_ONLY
print("\n" + "═"*65)
print(f" 테스트 완료: {total}건  ✅{PASS} PASS  ❌{FAIL} FAIL  🪟{WINDOWS_ONLY} WIN_ONLY")
print("═"*65)

# ── 마크다운 리포트 저장 ──────────────────────────────────
now = datetime.now().strftime("%Y-%m-%d %H:%M:%S")
md = [
    f"# NEXUS AI 전기능 테스트 리포트",
    f"> 실행일시: {now}  |  백엔드: {BASE}",
    f"> ✅ {PASS} PASS  |  ❌ {FAIL} FAIL  |  🪟 {WINDOWS_ONLY} Windows-only (Mac stub)",
    "",
    "---",
    "",
]

current_cat = ""
for row in results:
    cat = row['label'].split()[0] if row['label'] else "?"
    # 카테고리 구분 헤더
    inp = row['input']
    if inp.startswith(("GET", "POST")):
        md.append(f"| `{row['ts']}` | **{row['tag']}** | `{row['action']}` | {row['label']} | `{inp}` | {row['message'][:80]} |")
    else:
        md.append("")
        md.append(f"### {row['label']}")
        md.append(f"- **채팅 입력**: `{inp}`")
        md.append(f"- **결과**: {row['tag']}  `action={row['action']}`  `dur={row.get('duration','?')}`")
        md.append(f"- **응답 내용**: {row['message'][:200]}")

with open(OUT, "w", encoding="utf-8") as f:
    f.write("\n".join(md))

print(f"\n📄 리포트 저장: {OUT}")
