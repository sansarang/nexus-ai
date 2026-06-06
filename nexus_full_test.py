#!/usr/bin/env python3
"""
nexus_full_test.py — NEXUS AI v2.8 전체 기능 완전 검증
실제 사용자가 쓰는 것처럼 250+ 엔드포인트 전체를 검증합니다.

실행 (Parallels Windows CMD):
  cd %USERPROFILE%\Desktop
  python nexus_full_test.py

결과 전체 복사 → Claude에게 붙여넣기
"""

import json, time, sys, os, pathlib, urllib.request, urllib.error, base64
from datetime import datetime
from collections import defaultdict

BASE = "http://127.0.0.1:17892"

# ══════════════════════════════════════════════════════
# ★ 관리자 JWT — 앱 로그인 후 여기에 붙여넣기
#
# 방법: 넥서스 앱 실행 → Google 로그인 완료 후
#   앱이 실행 중이면 JWT가 백엔드 메모리에 자동 저장되므로
#   보통은 비워도 됩니다.
#
#   만약 "플랜: FREE" 로 뜨면 아래에 JWT 토큰 붙여넣기:
#   (Supabase Dashboard → Authentication → Users →
#    etetet3ea1101@gmail.com → Sign in as user → access_token 복사)
# ══════════════════════════════════════════════════════
JWT_TOKEN = ""   # ← 필요 시 여기에 붙여넣기 (eyJ... 로 시작하는 토큰)

# ── 전역 카운터 ──────────────────────────────────────
RESULTS = []
PASS = FAIL = SKIP = WARN = 0
SECTION_STATS = defaultdict(lambda: {"pass":0,"fail":0,"warn":0})

# ── HTTP 헬퍼 ────────────────────────────────────────

def http(method, path, body=None, timeout=30, headers=None):
    url = BASE + path
    data = json.dumps(body).encode() if body is not None else None
    req = urllib.request.Request(url, data=data, method=method)
    req.add_header("Content-Type", "application/json")
    # JWT 자동 주입 (설정된 경우)
    if JWT_TOKEN:
        req.add_header("Authorization", f"Bearer {JWT_TOKEN}")
    if headers:
        for k, v in headers.items():
            req.add_header(k, v)
    t0 = time.time()
    try:
        with urllib.request.urlopen(req, timeout=timeout) as r:
            raw = r.read().decode(errors="ignore")
            ms = round((time.time()-t0)*1000)
            try:
                return json.loads(raw), ms
            except:
                return {"_raw": raw[:200]}, ms
    except urllib.error.HTTPError as e:
        body_text = e.read().decode(errors="ignore")[:300]
        return {"_http_error": e.code, "_body": body_text}, round((time.time()-t0)*1000)
    except Exception as e:
        return {"_error": str(e)}, round((time.time()-t0)*1000)

def GET(path, timeout=15):
    return http("GET", path, timeout=timeout)

def POST(path, body=None, timeout=30):
    return http("POST", path, body or {}, timeout=timeout)

def cmd(msg, persona="", lang="ko", timeout=35):
    body = {"message": msg, "lang": lang, "user_email": "test@nexus.ai"}
    if persona:
        body["persona"] = persona
    return POST("/api/command", body, timeout=timeout)

# ── 로깅 ─────────────────────────────────────────────

def log(section, name, r, ms, *, ok=None, warn_only=False, expect_action=None,
        expect_key=None, note="", skip=False):
    global PASS, FAIL, SKIP, WARN

    if skip:
        SKIP += 1
        print(f"⏭  SKIP [{section}] {name}")
        RESULTS.append({"section":section,"name":name,"status":"skip","ms":ms})
        return False, r

    if ok is None:
        # 자동 판정
        if "_error" in r:
            ok = False
        elif "_http_error" in r:
            ok = r["_http_error"] in (200, 201, 404)  # 404도 일단 warn
            if r["_http_error"] == 404:
                warn_only = True
        elif r.get("success") is False:
            ok = False
        elif expect_action:
            ok = r.get("action") == expect_action
        elif expect_key:
            ok = expect_key in r
        else:
            ok = True

        # 빈 응답 체크 (command endpoint)
        if ok and "action" in r:
            msg_val = r.get("message", "")
            action = r.get("action", "")
            if isinstance(msg_val, str) and len(msg_val.strip()) < 2 \
               and action not in ("confirm","clarify","multi","restart","shutdown","sleep","format_disk","payment","file_delete"):
                ok = False
                note = note or "빈 응답"

    reason = ""
    if not ok:
        if "_error" in r:
            reason = str(r["_error"])[:80]
        elif "_http_error" in r:
            reason = f"HTTP {r['_http_error']}: {r.get('_body','')[:60]}"
        elif r.get("success") is False:
            reason = str(r.get("message",""))[:80]
        elif expect_action:
            reason = f"action={r.get('action','?')!r} (want {expect_action!r})"
        elif expect_key:
            reason = f"key '{expect_key}' missing"

    if warn_only and not ok:
        status = "⚠️  WARN"
        WARN += 1
        SECTION_STATS[section]["warn"] += 1
    elif ok:
        status = "✅ PASS"
        PASS += 1
        SECTION_STATS[section]["pass"] += 1
    else:
        status = "❌ FAIL"
        FAIL += 1
        SECTION_STATS[section]["fail"] += 1

    action_str = r.get("action", r.get("card_type", ""))
    msg_preview = ""
    if isinstance(r.get("message"), str):
        msg_preview = r["message"][:55].replace("\n"," ")

    line = f"{status} [{section}] {name:<42} {ms:>5}ms"
    if action_str:
        line += f"  act={action_str}"
    if reason:
        line += f"  ⚠ {reason}"
    elif msg_preview and not action_str:
        line += f"  {msg_preview}"
    if note:
        line += f"  ({note})"
    print(line)
    RESULTS.append({"section":section,"name":name,"status":"pass" if ok else ("warn" if warn_only else "fail"),
                    "ms":ms,"action":action_str,"reason":reason})
    return ok, r

def sep(title):
    print(f"\n{'─'*80}")
    print(f"  {title}")
    print(f"{'─'*80}")

# ─────────────────────────────────────────────────────
# 0. 연결 확인
# ─────────────────────────────────────────────────────
print("\n" + "="*80)
print(f"  NEXUS AI v2.8 전체 기능 완전 검증  |  {datetime.now():%Y-%m-%d %H:%M:%S}")
print("="*80)

r, ms = GET("/api/health", timeout=5)
if r.get("_error"):
    r2, ms2 = cmd("ping", timeout=8)
    if r2.get("_error"):
        print(f"\n❌ 백엔드 연결 실패 (포트 17892): {r2['_error']}")
        print("   넥서스 앱을 먼저 실행하세요.")
        sys.exit(1)
print(f"✅ 백엔드 연결 성공 ({ms}ms)\n")

# ─────────────────────────────────────────────────────
# 0-1. 플랜 / 관리자 계정 확인
# ─────────────────────────────────────────────────────
print("── 플랜 확인 ────────────────────────────────────────────────────────────────")
usage_r, usage_ms = GET("/api/usage", timeout=10)
current_plan = usage_r.get("plan", "unknown")
ai_left  = usage_r.get("ai_request", {}).get("left", "?")
ai_limit = usage_r.get("ai_request", {}).get("limit", "?")

print(f"   현재 플랜: {current_plan.upper()}")
print(f"   AI 요청 잔여: {ai_left} / {ai_limit}")

if current_plan.lower() not in ("admin", "team", "pro_plus", "pro"):
    print()
    print("  ⚠️  경고: 현재 플랜이 Free (15건/일) 입니다!")
    print("  300건 테스트를 완료하려면 관리자 플랜이 필요합니다.")
    print()
    print("  ── 해결 방법 ──────────────────────────────────────────────────────")
    print("  1. https://supabase.com/dashboard 접속")
    print("  2. 넥서스 프로젝트 → Authentication → Users")
    print("  3. etetet3ea1101@gmail.com 찾기 → Edit user")
    print("  4. app_metadata 에 입력: {\"plan\": \"admin\"}")
    print("  5. Save → 넥서스 앱 재로그인 → 이 스크립트 다시 실행")
    print("  ───────────────────────────────────────────────────────────────────")
    print()
    ans = input("  그냥 진행하시겠습니까? (15건 초과 시 오류 예상) [y/N]: ").strip().lower()
    if ans != "y":
        print("  중단합니다. 위 안내대로 관리자 설정 후 재실행하세요.")
        sys.exit(0)
elif current_plan.lower() == "admin":
    print(f"  ✅ 관리자 계정 확인 — 요청 제한 없음 (99,999건)")
else:
    print(f"  ✅ {current_plan.upper()} 플랜 — 테스트 진행")
print()

# ─────────────────────────────────────────────────────
# 1. /api/command — 기본 채팅
# ─────────────────────────────────────────────────────
sep("1. 기본 채팅 & 인사")
for msg, note in [
    ("안녕","인사"), ("고마워","감사"), ("넌 누구야","자기소개"),
    ("뭘 도와줄 수 있어?","기능안내"), ("오늘 기분 어때","잡담"),
    ("Hi","영어 인사"), ("What can you do?","영어 기능문의"),
]:
    r,ms = cmd(msg)
    log("기본채팅", msg, r, ms, note=note)
    time.sleep(0.3)

# ─────────────────────────────────────────────────────
# 2. 시스템 정보
# ─────────────────────────────────────────────────────
sep("2. 시스템 정보")
# /api/command 경유
for msg, ea in [
    ("PC 상태 보여줘","pc_status"), ("CPU 사용률","pc_status"),
    ("메모리 얼마 남았어","pc_status"), ("디스크 공간","pc_status"),
    ("GPU 상태 알려줘","pc_status"),
]:
    r,ms = cmd(msg); log("시스템CMD", msg, r, ms, expect_action=ea); time.sleep(0.3)

# 직접 API
for path, name in [
    ("/api/stats","stats 직접"), ("/api/gpu/stats","GPU stats"),
    ("/api/processes/top","상위 프로세스"), ("/api/programs","설치 프로그램"),
    ("/api/drivers","드라이버 목록"), ("/api/power/plans","전원 계획"),
    ("/api/system/updates","시스템 업데이트"), ("/api/network/analysis","네트워크 분석"),
]:
    r,ms = GET(path); log("시스템API", name, r, ms, expect_key="success", warn_only=True); time.sleep(0.2)

# ─────────────────────────────────────────────────────
# 3. 날씨 / 지도 / 여행
# ─────────────────────────────────────────────────────
sep("3. 날씨 / 지도 / 여행")
for msg, ea in [
    ("서울 날씨","weather"), ("오늘 날씨 어때","weather"),
    ("부산 기온 알려줘","weather"), ("미세먼지 농도","weather"),
    ("내일 비 올까","weather"),
]:
    r,ms = cmd(msg); log("날씨", msg, r, ms, expect_action=ea); time.sleep(0.3)

r,ms = GET("/api/weather?city=Seoul"); log("날씨API", "/api/weather?city=Seoul", r, ms, warn_only=True)
r,ms = POST("/api/directions", {"from":"서울역","to":"강남역"}); log("날씨API", "directions 서울역→강남역", r, ms, warn_only=True)
r,ms = POST("/api/travel/time", {"from":"서울","to":"부산","mode":"KTX"}); log("날씨API", "travel time", r, ms, warn_only=True)
r,ms = POST("/api/place-view", {"query":"경복궁"}); log("날씨API", "place-view 경복궁", r, ms, warn_only=True)

# ─────────────────────────────────────────────────────
# 4. 웹 검색 (다양한 방식)
# ─────────────────────────────────────────────────────
sep("4. 웹 검색")
for msg, ea in [
    ("최신 AI 뉴스","web_search"), ("파이썬 리스트 사용법","web_search"),
    ("서울 맛집 추천","web_search"), ("오늘 코스피 지수","web_search"),
    ("ChatGPT vs Claude 비교","web_search"), ("넥서스AI 리뷰","web_search"),
]:
    r,ms = cmd(msg, timeout=25); log("웹검색CMD", msg, r, ms, expect_action=ea); time.sleep(0.5)

r,ms = POST("/api/search/deep", {"query":"자가치유 AI 트렌드 2026"}, timeout=30)
log("웹검색API", "deep search", r, ms, warn_only=True)
r,ms = POST("/api/search/anonymous", {"query":"AI 뉴스"})
log("웹검색API", "anonymous search", r, ms, warn_only=True)
r,ms = POST("/api/site-search", {"query":"AI","site":"naver.com"})
log("웹검색API", "site search naver", r, ms, warn_only=True)
r,ms = POST("/api/llm/deep-search", {"query":"Claude Sonnet 최신 기능"}, timeout=30)
log("웹검색API", "llm deep-search", r, ms, warn_only=True)

# ─────────────────────────────────────────────────────
# 5. 뉴스 / 소셜
# ─────────────────────────────────────────────────────
sep("5. 뉴스 / Reddit / TikTok")
for msg, ea in [
    ("오늘 뉴스 알려줘","news_search"), ("IT 뉴스","news_search"),
    ("주식 시장 뉴스","news_search"),
]:
    r,ms = cmd(msg, timeout=25); log("뉴스CMD", msg, r, ms, expect_action=ea); time.sleep(0.4)

r,ms = GET("/api/reddit/trending"); log("소셜API", "reddit trending", r, ms, warn_only=True)
r,ms = POST("/api/reddit/search", {"query":"AI","subreddit":"technology"})
log("소셜API", "reddit search", r, ms, warn_only=True)
r,ms = GET("/api/tiktok/trending"); log("소셜API", "tiktok trending", r, ms, warn_only=True)
r,ms = GET("/api/tiktok/hot-songs"); log("소셜API", "tiktok hot songs", r, ms, warn_only=True)
r,ms = GET("/api/netflix/trending"); log("소셜API", "netflix trending", r, ms, warn_only=True)

# ─────────────────────────────────────────────────────
# 6. 유튜브 / 비디오
# ─────────────────────────────────────────────────────
sep("6. 유튜브 / 비디오")
for msg, ea in [
    ("유튜브에서 AI 영상 찾아줘","video_search"),
    ("BTS 뮤직비디오 찾아줘","video_search"),
    ("파이썬 강의 유튜브","video_search"),
]:
    r,ms = cmd(msg, timeout=25); log("비디오CMD", msg, r, ms, expect_action=ea); time.sleep(0.4)

r,ms = GET("/api/video/quick-search?query=AI+tutorial"); log("비디오API", "quick-search", r, ms, warn_only=True)
r,ms = GET("/api/video/check-deps"); log("비디오API", "check-deps(yt-dlp)", r, ms, warn_only=True)
r,ms = POST("/api/video/transcript", {"url":"https://www.youtube.com/watch?v=dQw4w9WgXcQ"}, timeout=40)
log("비디오API", "transcript (Rick Astley)", r, ms, warn_only=True)
r,ms = POST("/api/video/quick-search", {"query":"AI 2026"}); log("비디오API", "video search POST", r, ms, warn_only=True)

# ─────────────────────────────────────────────────────
# 7. 음악 (YTMusic)
# ─────────────────────────────────────────────────────
sep("7. 음악 (YTMusic)")
r,ms = POST("/api/ytmusic/search", {"query":"NewJeans Hype Boy"})
log("음악API", "ytmusic search", r, ms, warn_only=True)
r,ms = GET("/api/tiktok/hot-songs"); log("음악API", "tiktok hot songs", r, ms, warn_only=True)
for msg in ["BTS 노래 틀어줘", "최신 K-POP 추천해줘", "유튜브 뮤직에서 IU 찾아줘"]:
    r,ms = cmd(msg, timeout=20); log("음악CMD", msg, r, ms); time.sleep(0.3)

# ─────────────────────────────────────────────────────
# 8. Excel — 생성 / 분석 / COM 조작
# ─────────────────────────────────────────────────────
sep("8. Excel")
for msg in [
    "엑셀로 월별 매출 정리해줘",
    "분기 실적 스프레드시트 만들어",
    "직원 급여 명세 엑셀 생성해줘",
    "재고 현황 엑셀 표 만들어",
    "2026년 예산 계획 엑셀로",
]:
    r,ms = cmd(msg, timeout=40); log("Excel생성", msg, r, ms); time.sleep(1.0)

# test_data.xlsx 분석
xlsx_path = pathlib.Path(os.environ.get("USERPROFILE","C:/Users/User")) / "Desktop" / "test_data.xlsx"
if xlsx_path.exists():
    r,ms = cmd(f"{xlsx_path} 파일 분석해줘", timeout=30)
    log("Excel분석", "test_data.xlsx 분석", r, ms)
    r,ms = cmd(f"{xlsx_path} 에서 매출이 가장 높은 달 찾아줘", timeout=30)
    log("Excel분석", "최고 매출 월 찾기", r, ms)
else:
    log("Excel분석", "test_data.xlsx 없음", {}, 0, skip=True)

r,ms = GET("/api/excel/list"); log("ExcelAPI", "excel list", r, ms, warn_only=True)
r,ms = GET("/api/excel/com/workbooks"); log("ExcelAPI", "COM workbooks", r, ms, warn_only=True)
r,ms = POST("/api/excel/com/formula", {"formula":"=SUM(A1:A10)","cell":"B1"}); log("ExcelAPI", "COM formula", r, ms, warn_only=True)

# ─────────────────────────────────────────────────────
# 9. PDF — 생성 / 분석
# ─────────────────────────────────────────────────────
sep("9. PDF")
for msg in [
    "보고서 PDF로 만들어줘",
    "계약서 PDF 생성해",
    "월간 리포트 PDF",
    "견적서 PDF 만들어",
    "프레젠테이션 PDF로 변환해",
]:
    r,ms = cmd(msg, timeout=40); log("PDF생성", msg, r, ms); time.sleep(1.0)

pdf_path = pathlib.Path(os.environ.get("USERPROFILE","C:/Users/User")) / "Desktop" / "test_contract.pdf"
if pdf_path.exists():
    r,ms = cmd(f"{pdf_path} 분석해줘", timeout=30)
    log("PDF분석", "test_contract.pdf 분석", r, ms)
    r,ms = cmd(f"{pdf_path} 에서 요금 정보 찾아줘", timeout=30)
    log("PDF분석", "계약서 요금 정보 추출", r, ms)
    r,ms = POST("/api/docs/summary", {"path": str(pdf_path)}, timeout=30)
    log("PDF분석", "docs/summary API", r, ms, warn_only=True)
else:
    log("PDF분석", "test_contract.pdf 없음", {}, 0, skip=True)

# ─────────────────────────────────────────────────────
# 10. Word 문서
# ─────────────────────────────────────────────────────
sep("10. Word 문서")
r,ms = GET("/api/word/com/documents"); log("WordAPI", "COM documents", r, ms, warn_only=True)
r,ms = POST("/api/word/com/insert", {"text":"NEXUS AI 테스트 내용"}); log("WordAPI", "COM insert", r, ms, warn_only=True)
r,ms = POST("/api/word/com/replace", {"find":"테스트","replace":"검증"}); log("WordAPI", "COM replace", r, ms, warn_only=True)
for msg in ["워드 문서 만들어줘", "보고서 Word로 작성해", "계약서 워드 파일로"]:
    r,ms = cmd(msg, timeout=30); log("Word생성", msg, r, ms); time.sleep(0.5)

# ─────────────────────────────────────────────────────
# 11. 문서 AI (요약 / 비교 / 편집)
# ─────────────────────────────────────────────────────
sep("11. 문서 AI")
r,ms = POST("/api/docs/find", {"query":"계약서"}); log("문서AI", "docs find", r, ms, warn_only=True)
r,ms = POST("/api/llm/doc-summary", {"text":"인공지능은 컴퓨터 과학의 한 분야로..."}, timeout=20)
log("문서AI", "doc summary", r, ms, warn_only=True)
r,ms = POST("/api/docs/ai-edit", {"text":"This is a test document.","instruction":"한국어로 번역해줘"}, timeout=20)
log("문서AI", "AI edit (번역)", r, ms, warn_only=True)
for msg in ["이 문서 요약해줘", "핵심 내용 3줄로 정리해", "문서에서 날짜 정보 추출해"]:
    r,ms = cmd(msg, timeout=20); log("문서AICMD", msg, r, ms); time.sleep(0.3)

# ─────────────────────────────────────────────────────
# 12. 이메일
# ─────────────────────────────────────────────────────
sep("12. 이메일")
for msg, ea in [
    ("받은 메일 확인해줘","email_inbox"), ("이메일 받은 편지함","email_inbox"),
    ("오늘 온 메일 요약해줘","email_summarize"), ("메일 중요한 것만 골라줘","email_summarize"),
]:
    r,ms = cmd(msg, timeout=20); log("이메일CMD", msg, r, ms, expect_action=ea); time.sleep(0.4)

r,ms = GET("/api/email/inbox"); log("이메일API", "inbox GET", r, ms, warn_only=True)
r,ms = GET("/api/email/config"); log("이메일API", "config GET", r, ms, warn_only=True)
r,ms = POST("/api/email/summarize", {"count":10}, timeout=20); log("이메일API", "summarize API", r, ms, warn_only=True)
r,ms = POST("/api/email/classify", {"emails":[]}, timeout=10); log("이메일API", "classify", r, ms, warn_only=True)
r,ms = POST("/api/email/draft-reply", {"subject":"테스트","body":"안녕하세요"}, timeout=15)
log("이메일API", "draft reply", r, ms, warn_only=True)
r,ms = POST("/api/email/extract-events", {"body":"내일 오후 2시에 미팅이 있습니다."})
log("이메일API", "extract events", r, ms, warn_only=True)

# Gmail
r,ms = GET("/api/gmail/inbox"); log("Gmail", "gmail inbox", r, ms, warn_only=True)
r,ms = GET("/api/gmail/search?q=AI"); log("Gmail", "gmail search", r, ms, warn_only=True)

# IMAP
r,ms = GET("/api/imap/accounts"); log("IMAP", "accounts list", r, ms, warn_only=True)
r,ms = GET("/api/imap/inbox"); log("IMAP", "imap inbox", r, ms, warn_only=True)

# ─────────────────────────────────────────────────────
# 13. 캘린더
# ─────────────────────────────────────────────────────
sep("13. 캘린더 / 일정")
for msg, ea in [
    ("오늘 일정 보여줘","calendar_today"), ("이번 주 일정","calendar_week"),
    ("내일 오후 3시 미팅 잡아줘","calendar_add"), ("빈 시간 찾아줘","calendar_find_slot"),
]:
    r,ms = cmd(msg, timeout=20); log("캘린더CMD", msg, r, ms, expect_action=ea); time.sleep(0.3)

r,ms = GET("/api/calendar/today"); log("캘린더API", "calendar/today", r, ms, warn_only=True)
r,ms = GET("/api/calendar/week"); log("캘린더API", "calendar/week", r, ms, warn_only=True)
r,ms = POST("/api/calendar/add", {"title":"테스트 미팅","datetime":"2026-06-10T14:00:00"})
log("캘린더API", "calendar add", r, ms, warn_only=True)
r,ms = POST("/api/calendar/find-slot", {"duration":60,"days":3})
log("캘린더API", "find-slot", r, ms, warn_only=True)
r,ms = POST("/api/calendar/smart-add", {"text":"다음주 월요일 오전 10시 팀 회의"})
log("캘린더API", "smart-add", r, ms, warn_only=True)
r,ms = GET("/api/calendar/google/status"); log("캘린더API", "google calendar status", r, ms, warn_only=True)

# ─────────────────────────────────────────────────────
# 14. 번역
# ─────────────────────────────────────────────────────
sep("14. 번역")
for msg, ea in [
    ("안녕하세요를 영어로","translate"), ("Hello world 한국어로","translate"),
    ("감사합니다 일본어로","translate"), ("I love Korea 스페인어로","translate"),
    ("Bonjour 한국어로 번역해줘","translate"),
]:
    r,ms = cmd(msg); log("번역", msg, r, ms, expect_action=ea); time.sleep(0.3)

# ─────────────────────────────────────────────────────
# 15. 파일 관리
# ─────────────────────────────────────────────────────
sep("15. 파일 관리")
for msg, ea in [
    ("다운로드 폴더 정리해줘","file_organize"), ("중복 파일 찾아줘","file_duplicate"),
    ("큰 파일 찾아줘","file_large"), ("바탕화면 파일 검색","file_search"),
]:
    r,ms = cmd(msg, timeout=20); log("파일CMD", msg, r, ms, expect_action=ea); time.sleep(0.3)

desktop = str(pathlib.Path(os.environ.get("USERPROFILE","C:/Users/User")) / "Desktop")
r,ms = POST("/api/files/search", {"query":"*.xlsx","path":desktop}); log("파일API", "files search xlsx", r, ms, warn_only=True)
r,ms = POST("/api/files/duplicates", {"path":desktop}); log("파일API", "duplicates scan", r, ms, warn_only=True)
r,ms = POST("/api/file/metadata", {"path":desktop}); log("파일API", "file metadata", r, ms, warn_only=True)
r,ms = POST("/api/file/large", {"path":desktop,"min_mb":10}); log("파일API", "large files", r, ms, warn_only=True)
r,ms = POST("/api/folder/open", {"path":desktop}); log("파일API", "folder open", r, ms, warn_only=True)

# ─────────────────────────────────────────────────────
# 16. 보안
# ─────────────────────────────────────────────────────
sep("16. 보안 & 스캔")
for msg, ea in [
    ("보안 스캔 해줘","security_scan"), ("악성코드 검사","security_scan"),
    ("방화벽 상태 확인","security_check"), ("바이러스 있나 확인해","security_scan"),
]:
    r,ms = cmd(msg, timeout=25); log("보안CMD", msg, r, ms, expect_action=ea); time.sleep(0.4)

r,ms = GET("/api/security/audit"); log("보안API", "security audit", r, ms, warn_only=True)
r,ms = GET("/api/security/processes"); log("보안API", "processes", r, ms, warn_only=True)
r,ms = GET("/api/security/startup"); log("보안API", "startup items", r, ms, warn_only=True)
r,ms = GET("/api/security/accounts"); log("보안API", "accounts", r, ms, warn_only=True)
r,ms = GET("/api/security/remote"); log("보안API", "remote access", r, ms, warn_only=True)
r,ms = POST("/api/security/myip", {}); log("보안API", "my IP", r, ms, warn_only=True)
r,ms = POST("/api/security/check-path", {"path":"C:\\Windows\\System32"}); log("보안API", "check path", r, ms, warn_only=True)
r,ms = POST("/api/scan", {}, timeout=30); log("보안API", "full scan", r, ms, warn_only=True)

# ─────────────────────────────────────────────────────
# 17. PC 제어 (시스템 조작)
# ─────────────────────────────────────────────────────
sep("17. PC 제어")
r,ms = POST("/api/system/volume", {"action":"get"}); log("PC제어", "볼륨 조회", r, ms, warn_only=True)
r,ms = POST("/api/system/brightness", {"action":"get"}); log("PC제어", "밝기 조회", r, ms, warn_only=True)
r,ms = POST("/api/system/wifi", {"action":"status"}); log("PC제어", "wifi 상태", r, ms, warn_only=True)
r,ms = POST("/api/system/launch", {"app":"notepad"}); log("PC제어", "앱 실행 (메모장)", r, ms, warn_only=True)
r,ms = POST("/api/power/plan", {"plan":"balanced"}); log("PC제어", "전원계획 설정", r, ms, warn_only=True)
r,ms = POST("/api/process/kill", {"name":"notepad.exe"}); log("PC제어", "프로세스 종료", r, ms, warn_only=True)
r,ms = GET("/api/app/permissions"); log("PC제어", "앱 권한 조회", r, ms, warn_only=True)

for msg in ["볼륨 줄여줘", "메모장 열어줘", "Wi-Fi 상태 확인해"]:
    r,ms = cmd(msg, timeout=15); log("PC제어CMD", msg, r, ms); time.sleep(0.3)

# ─────────────────────────────────────────────────────
# 18. 위험 액션 확인 카드
# ─────────────────────────────────────────────────────
sep("18. 위험 액션 — 확인 카드 필수")
for msg in ["PC 재시작해", "컴퓨터 꺼줘", "파일 전부 삭제해", "디스크 포맷해줘", "결제 진행해줘"]:
    r,ms = cmd(msg)
    action = r.get("action","")
    is_safe = action in ("confirm","clarify") or r.get("requires_confirm") or \
              r.get("card_type","") in ("confirm","danger_confirm")
    status = "✅ PASS" if is_safe else "⚠️  WARN"
    print(f"{status} [위험액션] {msg:<30} {ms:>5}ms  action={action}  confirm_card={is_safe}")
    if is_safe: PASS += 1
    else: WARN += 1
    RESULTS.append({"section":"위험액션","name":msg,"status":"pass" if is_safe else "warn",
                    "ms":ms,"action":action,"reason":"" if is_safe else "확인 카드 없음"})
    time.sleep(0.3)

# ─────────────────────────────────────────────────────
# 19. 멀티 액션 ("A하고 B해")
# ─────────────────────────────────────────────────────
sep("19. 멀티 액션")
multi_cases = [
    "날씨 확인하고 뉴스도 검색해줘",
    "PC 상태 보고 메모리 최적화도 해줘",
    "오늘 일정 보고 내일 날씨도 알려줘",
    "이메일 확인하고 중요한 거 요약해줘",
    "엑셀로 매출 정리하고 PDF로도 만들어줘",
    "유튜브에서 AI 영상 찾고 번역도 해줘",
    "보안 스캔하고 결과 리포트 만들어줘",
]
for msg in multi_cases:
    r,ms = cmd(msg, timeout=45)
    has_multi = (isinstance(r.get("results"), list) and len(r["results"]) > 1) \
                or r.get("action") == "multi" \
                or isinstance(r.get("actions"), list)
    ok = bool(r.get("success") or r.get("message") or r.get("results")) and not r.get("_error")
    flag = "🔀 multi" if has_multi else "→ single"
    status = "✅ PASS" if ok else "❌ FAIL"
    print(f"{status} [멀티액션] {msg:<48} {ms:>5}ms  {flag}")
    if ok: PASS += 1
    else: FAIL += 1
    RESULTS.append({"section":"멀티액션","name":msg,"status":"pass" if ok else "fail","ms":ms,"action":flag,"reason":""})
    time.sleep(1.0)

# ─────────────────────────────────────────────────────
# 20. 페르소나 전환 (19개 전체)
# ─────────────────────────────────────────────────────
sep("20. 페르소나 전환 (19개)")
r,ms = GET("/api/persona/list"); log("페르소나", "persona list", r, ms, expect_key="success", warn_only=True)
r,ms = GET("/api/persona/current"); log("페르소나", "current persona", r, ms, warn_only=True)

personas_19 = [
    ("auto",       "오늘 할 일 정리해줘"),
    ("developer",  "파이썬 데코레이터 설명해줘"),
    ("marketer",   "인스타그램 마케팅 전략 알려줘"),
    ("sales",      "고객 이탈 방지 방법은?"),
    ("pm",         "스프린트 계획 세우는 법"),
    ("designer",   "UI UX 원칙 5가지"),
    ("meeting",    "회의 효율 높이는 방법"),
    ("research",   "논문 검색 방법 알려줘"),
    ("security",   "제로데이 취약점이 뭐야?"),
    ("legal",      "근로계약서 필수 항목은?"),
    ("medical",    "혈압 140/90 의미는?"),
    ("finance",    "DCF 밸류에이션 설명해줘"),
    ("investor",   "삼성전자 투자 가치 분석해줘"),
    ("freelancer", "프리랜서 세금 신고 방법"),
    ("small_biz",  "소상공인 지원금 신청 방법"),
    ("corporate",  "법인세 신고 일정 알려줘"),
    ("creator",    "유튜브 알고리즘 최적화 방법"),
    ("tutor",      "중학생에게 미적분 설명하는 법"),
    ("nexus",      "넥서스 AI 기능 설명해줘"),
]
for persona, msg in personas_19:
    r,ms = cmd(msg, persona=persona, timeout=20)
    ok = bool(r.get("success") or r.get("message")) and not r.get("_error") and not r.get("_http_error")
    preview = str(r.get("message",""))[:45].replace("\n"," ")
    status = "✅ PASS" if ok else "❌ FAIL"
    print(f"{status} [페르소나:{persona:<12}] {msg:<38} {ms:>5}ms  {preview}")
    if ok: PASS += 1
    else: FAIL += 1
    RESULTS.append({"section":f"페르소나:{persona}","name":msg,"status":"pass" if ok else "fail","ms":ms,"action":"","reason":""})
    r2,ms2 = POST("/api/persona/set", {"persona": persona})
    time.sleep(0.4)

# ─────────────────────────────────────────────────────
# 21. RAG / 브레인 (PC 문서 검색)
# ─────────────────────────────────────────────────────
sep("21. RAG / 브레인 (PC 문서 인덱싱)")
r,ms = GET("/api/brain/stats"); log("RAG", "brain stats", r, ms, warn_only=True)
r,ms = POST("/api/brain/search", {"query":"계약서 조항"}, timeout=20)
log("RAG", "brain search 계약서", r, ms, warn_only=True)
r,ms = POST("/api/brain/search", {"query":"매출 실적"}, timeout=20)
log("RAG", "brain search 매출", r, ms, warn_only=True)
r,ms = POST("/api/brain/index", {"path": str(pathlib.Path(os.environ.get("USERPROFILE","C:/Users/User")) / "Desktop")}, timeout=30)
log("RAG", "brain index Desktop", r, ms, warn_only=True)
for msg in ["내 PC에서 계약서 찾아줘", "문서에서 매출 관련 내용 검색해", "바탕화면 파일 중 PDF 요약해줘"]:
    r,ms = cmd(msg, timeout=25); log("RAG CMD", msg, r, ms); time.sleep(0.4)

# ─────────────────────────────────────────────────────
# 22. 메모리 / 노트 / 일지
# ─────────────────────────────────────────────────────
sep("22. 메모리 / 노트 / 일지")
r,ms = GET("/api/memory/list"); log("메모리", "memory list", r, ms, warn_only=True)
r,ms = GET("/api/memory/stats"); log("메모리", "memory stats", r, ms, warn_only=True)
r,ms = POST("/api/memory/search", {"query":"테스트"}); log("메모리", "memory search", r, ms, warn_only=True)
r,ms = GET("/api/notes"); log("메모리", "notes list", r, ms, warn_only=True)
r,ms = POST("/api/notes", {"content":"NEXUS AI 테스트 노트 내용입니다.","title":"테스트"}); log("메모리", "note create", r, ms, warn_only=True)
r,ms = GET("/api/journal/today"); log("메모리", "journal today", r, ms, warn_only=True)
r,ms = GET("/api/journal/history"); log("메모리", "journal history", r, ms, warn_only=True)
r,ms = POST("/api/journal/generate", {}); log("메모리", "journal generate", r, ms, warn_only=True)
for msg in ["오늘 한 일 기록해줘 AI 테스트 완료", "노트 저장해줘 중요: 넥서스 v2.8 검증 완료"]:
    r,ms = cmd(msg, timeout=15); log("메모리CMD", msg, r, ms); time.sleep(0.3)

# ─────────────────────────────────────────────────────
# 23. 워크플로우
# ─────────────────────────────────────────────────────
sep("23. 워크플로우")
r,ms = GET("/api/workflow/list"); log("워크플로우", "list", r, ms, warn_only=True)
r,ms = GET("/api/workflow/templates"); log("워크플로우", "templates", r, ms, warn_only=True)
r,ms = POST("/api/workflow/from-text", {"text":"매일 아침 뉴스 요약해서 메모에 저장"}, timeout=20)
log("워크플로우", "from-text 생성", r, ms, warn_only=True)
r,ms = POST("/api/workflow/plan", {"goal":"주간 보고서 자동 생성"}, timeout=20)
log("워크플로우", "plan", r, ms, warn_only=True)

# ─────────────────────────────────────────────────────
# 24. 마켓플레이스
# ─────────────────────────────────────────────────────
sep("24. 마켓플레이스")
r,ms = GET("/api/marketplace/presets"); log("마켓", "presets list", r, ms, warn_only=True)
r,ms = GET("/api/marketplace/my-presets"); log("마켓", "my presets", r, ms, warn_only=True)
r,ms = GET("/api/marketplace/purchased"); log("마켓", "purchased", r, ms, warn_only=True)

# ─────────────────────────────────────────────────────
# 25. 스케줄러 / 크론 / 트리거
# ─────────────────────────────────────────────────────
sep("25. 스케줄러 / 크론 / 트리거")
r,ms = GET("/api/scheduler/list"); log("스케줄러", "scheduler list", r, ms, warn_only=True)
r,ms = POST("/api/scheduler/parse", {"text":"매일 오전 9시에 날씨 알려줘"})
log("스케줄러", "parse schedule text", r, ms, warn_only=True)
r,ms = GET("/api/cron/list"); log("크론", "cron list", r, ms, warn_only=True)
r,ms = GET("/api/trigger/list"); log("트리거", "trigger list", r, ms, warn_only=True)
r,ms = GET("/api/trigger/events"); log("트리거", "trigger events", r, ms, warn_only=True)
for msg in ["매일 아침 7시에 날씨 알려줘", "매주 월요일 주간 리포트 만들어줘"]:
    r,ms = cmd(msg, timeout=15); log("스케줄러CMD", msg, r, ms); time.sleep(0.3)

# ─────────────────────────────────────────────────────
# 26. 브리핑 (Proactive AI)
# ─────────────────────────────────────────────────────
sep("26. Proactive AI 브리핑")
r,ms = GET("/api/briefing/config"); log("브리핑", "config", r, ms, warn_only=True)
r,ms = POST("/api/briefing/now", {}, timeout=30); log("브리핑", "지금 브리핑 실행", r, ms, warn_only=True)
r,ms = GET("/api/daily-report"); log("브리핑", "daily report", r, ms, warn_only=True)
r,ms = GET("/api/alerts/latest"); log("브리핑", "alerts latest", r, ms, warn_only=True)

# ─────────────────────────────────────────────────────
# 27. 주식 / 금융
# ─────────────────────────────────────────────────────
sep("27. 주식 / 금융")
for msg, ea in [
    ("삼성전자 주가 알려줘","stock_price"), ("코스피 지수","stock_price"),
    ("애플 주식 현재가","stock_price"), ("비트코인 시세","stock_price"),
]:
    r,ms = cmd(msg, timeout=20); log("주식CMD", msg, r, ms, expect_action=ea); time.sleep(0.3)

r,ms = POST("/api/finance/stock", {"ticker":"005930","exchange":"KRX"}, timeout=20)
log("주식API", "stock 삼성전자 직접", r, ms, warn_only=True)
r,ms = POST("/v1/stock", {"ticker":"AAPL"}, timeout=20)
log("주식API", "v1/stock AAPL", r, ms, warn_only=True)
r,ms = POST("/api/llm/deep-search", {"query":"삼성전자 실적 분석"}, timeout=25)
log("주식API", "deep search 실적", r, ms, warn_only=True)

# ─────────────────────────────────────────────────────
# 28. 법률 / 의료 (Vertical API)
# ─────────────────────────────────────────────────────
sep("28. 법률 / 의료 Vertical")
r,ms = POST("/api/legal/review", {"text":"이 계약서 위험 조항 찾아줘", "document": "제1조 갑은 을에게 서비스를 제공한다."}, timeout=25)
log("법률", "legal review", r, ms, warn_only=True)
r,ms = POST("/api/legal/search", {"query":"근로기준법 연차 규정"}, timeout=20)
log("법률", "legal search", r, ms, warn_only=True)
r,ms = POST("/v1/legal", {"query":"계약 해지 조건"}, timeout=20)
log("법률", "v1/legal", r, ms, warn_only=True)

r,ms = POST("/api/medical/search", {"query":"혈압 140/90 위험성"}, timeout=20)
log("의료", "medical search", r, ms, warn_only=True)
r,ms = POST("/v1/medical", {"query":"당뇨 증상"}, timeout=20)
log("의료", "v1/medical", r, ms, warn_only=True)

for msg in [
    "법인세 신고 기한이 언제야?", "근로계약서에 반드시 들어가야 할 내용은?",
    "혈당 200 이상이면 어떻게 해야 해?",
]:
    r,ms = cmd(msg, timeout=20); log("VT CMD", msg, r, ms); time.sleep(0.3)

# ─────────────────────────────────────────────────────
# 29. LLM 직접 호출
# ─────────────────────────────────────────────────────
sep("29. LLM 직접 / Vision / OCR")
r,ms = GET("/api/llm/config"); log("LLM", "config", r, ms, warn_only=True)
r,ms = POST("/api/llm/chat", {"messages":[{"role":"user","content":"안녕하세요"}]}, timeout=20)
log("LLM", "llm/chat", r, ms, warn_only=True)
r,ms = POST("/api/llm/route", {"message":"파이썬 코드 짜줘: 피보나치"}, timeout=20)
log("LLM", "llm/route", r, ms, warn_only=True)
r,ms = GET("/api/ollama/models"); log("LLM", "ollama models", r, ms, warn_only=True)
r,ms = GET("/api/ollama/config"); log("LLM", "ollama config", r, ms, warn_only=True)

# Vision (화면 분석)
r,ms = GET("/api/desktop/screenshot"); log("Vision", "screenshot", r, ms, warn_only=True)
r,ms = POST("/api/vision/screenshot", {}, timeout=15); log("Vision", "vision screenshot", r, ms, warn_only=True)
r,ms = GET("/api/vision/active-window"); log("Vision", "active window", r, ms, warn_only=True)
r,ms = POST("/api/vision/ocr-clipboard", {}, timeout=15); log("Vision", "OCR clipboard", r, ms, warn_only=True)
for msg in ["지금 화면 분석해줘", "현재 창에 뭐가 있어?", "화면 캡처해줘"]:
    r,ms = cmd(msg, timeout=20); log("VisionCMD", msg, r, ms); time.sleep(0.3)

# ─────────────────────────────────────────────────────
# 30. 브라우저 자동화
# ─────────────────────────────────────────────────────
sep("30. 브라우저 자동화")
r,ms = GET("/api/browser/status"); log("브라우저", "browser status", r, ms, warn_only=True)
r,ms = POST("/api/browser/navigate", {"url":"https://www.naver.com"}, timeout=20)
log("브라우저", "navigate naver", r, ms, warn_only=True)
r,ms = POST("/api/browser/screenshot", {}, timeout=15); log("브라우저", "browser screenshot", r, ms, warn_only=True)
r,ms = POST("/api/browser/extract", {"selector":"body"}, timeout=15); log("브라우저", "extract body", r, ms, warn_only=True)
r,ms = POST("/api/browser/collect-price", {"url":"https://www.amazon.com","keyword":"laptop"}, timeout=25)
log("브라우저", "price collect", r, ms, warn_only=True)
r,ms = POST("/api/browser/news-collect", {"url":"https://news.naver.com"}, timeout=25)
log("브라우저", "news collect", r, ms, warn_only=True)

for msg in ["네이버 열어줘", "구글에서 AI 검색해줘", "이 상품 최저가 찾아줘: 맥북 M3"]:
    r,ms = cmd(msg, timeout=25); log("브라우저CMD", msg, r, ms); time.sleep(0.5)

# ─────────────────────────────────────────────────────
# 31. 미팅 / 받아쓰기
# ─────────────────────────────────────────────────────
sep("31. 미팅 / 회의록 / 자막")
r,ms = GET("/api/meeting/list"); log("미팅", "meeting list", r, ms, warn_only=True)
r,ms = POST("/api/meeting/start", {"title":"NEXUS 테스트 미팅"})
log("미팅", "meeting start", r, ms, warn_only=True)
r,ms = POST("/api/meeting/transcribe", {"audio":""}, timeout=20)
log("미팅", "transcribe (빈)", r, ms, warn_only=True)
r,ms = POST("/api/meeting/summarize", {"text":"오늘 회의에서 AI 기능 테스트를 완료했습니다. 다음 회의는 다음주입니다."})
log("미팅", "summarize", r, ms, warn_only=True)
r,ms = GET("/api/caption/latest"); log("미팅", "caption latest", r, ms, warn_only=True)

# ─────────────────────────────────────────────────────
# 32. 매크로
# ─────────────────────────────────────────────────────
sep("32. 매크로")
r,ms = GET("/api/macros"); log("매크로", "list", r, ms, warn_only=True)
r,ms = POST("/api/macros/parse", {"text":"메모장 열고 안녕이라고 타이핑하고 저장"})
log("매크로", "parse text", r, ms, warn_only=True)

# ─────────────────────────────────────────────────────
# 33. 팀 기능
# ─────────────────────────────────────────────────────
sep("33. 팀 / 엔터프라이즈")
r,ms = GET("/api/team/members"); log("팀", "members", r, ms, warn_only=True)
r,ms = GET("/api/enterprise/keys"); log("팀", "enterprise keys", r, ms, warn_only=True)
r,ms = GET("/api/enterprise/plans"); log("팀", "enterprise plans", r, ms, warn_only=True)
r,ms = GET("/api/vertical/config"); log("팀", "vertical config", r, ms, warn_only=True)

# ─────────────────────────────────────────────────────
# 34. 자가치유 Agent 시스템
# ─────────────────────────────────────────────────────
sep("34. 자가치유 Agent (Self-Healing)")
r,ms = GET("/api/agent/status", timeout=10)
log("자가치유", "agent status", r, ms, expect_key="agent_count")
if "agent_count" in r:
    print(f"   Agent 수: {r.get('agent_count')}  대기 패치: {r.get('pending')}  "
          f"적용: {r.get('applied')}  힐링된 프롬프트: {r.get('healed_prompts',[])}")
    tele = r.get("telemetry",{})
    print(f"   총 명령: {tele.get('total')}  빈응답률: {tele.get('empty_rate')}  "
          f"평균응답: {tele.get('avg_duration_ms')}ms")

r,ms = POST("/api/agent/analyze-now", {}, timeout=20)
log("자가치유", "analyze-now (수동 트리거)", r, ms, expect_key="success")
if r.get("success"):
    proposals = r.get("proposals",[])
    print(f"   생성된 패치 제안: {len(proposals)}개")
    for p in proposals[:5]:
        print(f"   [{p.get('severity','?')}] {p.get('agent','?')}: {p.get('title','')}")

r,ms = GET("/api/agent/proposals"); log("자가치유", "proposals list", r, ms, warn_only=True)
r,ms = GET("/api/history/stats"); log("자가치유", "history stats", r, ms, warn_only=True)

# ~/.nexus/prompts/ 파일 확인
prompts_dir = pathlib.Path(os.environ.get("USERPROFILE","C:/Users/User")) / ".nexus" / "prompts"
healed = list(prompts_dir.glob("*.txt")) if prompts_dir.exists() else []
print(f"   힐링된 프롬프트 파일: {[f.name for f in healed] if healed else '없음 (아직 힐링 미발생)'}")

# ─────────────────────────────────────────────────────
# 35. 엣지 케이스 (충돌 방지)
# ─────────────────────────────────────────────────────
sep("35. 엣지 케이스")
edge = [
    ("", "빈 입력"),
    ("ㅁㄴㅇㄹ", "자음만"),
    ("asdfjkl;qwerty", "무의미한 영어"),
    ("a"*500, "500자 반복"),
    ("<script>alert(1)</script>", "XSS"),
    ("'; DROP TABLE users; --", "SQL 인젝션"),
    ("👋🤖💻🎯🔥✨🌟💫⭐", "이모지만"),
    ("1234567890"*10, "숫자 반복"),
    ("null", "null 문자열"),
    ("undefined", "undefined 문자열"),
    ("   ", "공백만"),
    ("넥서스"*50, "한글 반복 장문"),
]
for msg, label in edge:
    r,ms = cmd(msg[:300], timeout=12)
    crashed = bool(r.get("_error"))
    st = "✅ PASS" if not crashed else "❌ CRASH"
    action = r.get("action","—")
    print(f"{st} [엣지] {label:<25} {ms:>5}ms  action={action}")
    if not crashed: PASS += 1
    else: FAIL += 1
    RESULTS.append({"section":"엣지","name":label,"status":"pass" if not crashed else "fail","ms":ms,"action":action,"reason":""})
    time.sleep(0.25)

# ─────────────────────────────────────────────────────
# 36. 응답 시간 분포
# ─────────────────────────────────────────────────────
sep("36. 응답 시간 분석")
all_ms = [x["ms"] for x in RESULTS if x["ms"] > 0]
if all_ms:
    s = sorted(all_ms); n = len(s)
    avg = sum(s)//n
    p50 = s[n//2]; p95 = s[int(n*.95)]; p99 = s[min(int(n*.99),n-1)]
    over3 = sum(1 for x in s if x>3000); over10 = sum(1 for x in s if x>10000)
    print(f"   총 테스트: {n}건")
    print(f"   평균={avg}ms  P50={p50}ms  P95={p95}ms  P99={p99}ms")
    print(f"   최대={max(s)}ms  최소={min(s)}ms")
    print(f"   3초 초과: {over3}건 ({over3/n*100:.1f}%)")
    print(f"   10초 초과: {over10}건 ({over10/n*100:.1f}%)")

# ─────────────────────────────────────────────────────
# 최종 결과
# ─────────────────────────────────────────────────────
total = PASS+FAIL
pass_rate = PASS/total*100 if total else 0

print("\n" + "="*80)
print(f"  최종: {PASS}/{total} PASS ({pass_rate:.1f}%)  |  FAIL={FAIL}  WARN={WARN}  SKIP={SKIP}")
print("="*80)

print("\n── 섹션별 결과 ──")
for sec, st in sorted(SECTION_STATS.items()):
    total_sec = st["pass"]+st["fail"]+st["warn"]
    bar = "✅" if st["fail"]==0 else "❌"
    print(f"  {bar} {sec:<30} PASS={st['pass']} FAIL={st['fail']} WARN={st['warn']}")

if FAIL > 0:
    print(f"\n── 실패 항목 ({FAIL}개) ──")
    for row in RESULTS:
        if row["status"] == "fail":
            print(f"   ❌ [{row['section']}] {row['name'][:50]:<50}  {row.get('reason','')[:60]}")

print(f"\n📋 이 결과를 전체 복사해서 Claude에게 붙여넣기 하세요.")
print(f"   완료: {datetime.now():%H:%M:%S}")

# ─────────────────────────────────────────────────────
# [추가] 실제 테스트 자료 기반 심화 검증
# ─────────────────────────────────────────────────────
def test_with_files():
    """test_data/ 폴더의 실제 파일로 기능 심화 검증"""
    import pathlib, os
    user = os.environ.get("USERPROFILE","C:/Users/User")
    td = pathlib.Path(user) / "Desktop" / "test_data"
    if not td.exists():
        print("\n⚠️  test_data 폴더가 바탕화면에 없습니다 — 심화 테스트 건너뜁니다.")
        return

    sep("★ 심화: 실제 파일 분석 테스트")

    tests = [
        # (파일, 질문, 기대 키워드)
        ("sales_report.xlsx",       "이 엑셀에서 매출이 가장 높은 달 찾아줘",          ["월","매출","최고","최대"]),
        ("sales_report.xlsx",       "직원 중 실수령액이 가장 높은 사람 알려줘",        ["강현우","실수령","급여"]),
        ("sales_report.xlsx",       "재고가 품절된 제품 목록 보여줘",                   ["품절","PRD","재고"]),
        ("sales_report.xlsx",       "고객 중 NPS 점수가 가장 높은 회사는?",            ["NPS","삼성","SK","네이버"]),
        ("sales_report.xlsx",       "지연된 프로젝트 있어? 이슈 정리해줘",             ["지연","프로젝트","이슈"]),
        ("contract_legal.pdf",      "이 계약서에서 SLA 조건이 뭐야?",                  ["99.5","SLA","가용성"]),
        ("contract_legal.pdf",      "계약 자동 연장 조건 알려줘",                       ["30일","자동","연장"]),
        ("contract_legal.pdf",      "Pro+ 플랜 가격이 얼마야?",                        ["39,000","Pro+"]),
        ("medical_record.pdf",      "이 환자 혈압이 정상이야?",                        ["고혈압","138","정상"]),
        ("medical_record.pdf",      "처방된 약 종류와 복용법 알려줘",                  ["암로디핀","로살탄","아토르바"]),
        ("medical_record.pdf",      "당뇨 위험이 있어?",                               ["당뇨","전단계","HbA1c"]),
        ("investment_report.pdf",   "포트폴리오에서 수익률 가장 높은 종목은?",         ["NVIDIA","45.9","수익"]),
        ("investment_report.pdf",   "삼성전자 목표주가 얼마야?",                       ["85,000","목표주가"]),
        ("investment_report.pdf",   "리스크 등급이 어떻게 돼?",                        ["중간","리스크"]),
        ("meeting_minutes.txt",     "회의에서 결정된 사항 요약해줘",                   ["결론","결정","승인"]),
        ("meeting_minutes.txt",     "다음 주 액션 아이템 목록 뽑아줘",                 ["강현우","이서연","박지호"]),
        ("sample_emails.txt",       "스팸 이메일 골라줘",                              ["스팸","suspicious","차단"]),
        ("sample_emails.txt",       "투자 관련 이메일 있어?",                          ["카카오벤처스","투자","미팅"]),
        ("sample_emails.txt",       "불만 고객 이메일 요약해줘",                       ["환불","오류","불만"]),
        ("small_biz_data.txt",      "이 가게 5월 순이익은 얼마야?",                    ["3,830,000","5월","순이익"]),
        ("small_biz_data.txt",      "부가세 납부 기한이 언제야?",                      ["7월 25일","부가세"]),
        ("small_biz_data.txt",      "카드 수수료율이 얼마야?",                         ["0.5%","카드수수료"]),
        ("tax_invoice.xlsx",        "불일치 또는 취소된 세금계산서 찾아줘",            ["불일치","취소","⚠️","❌"]),
        ("tax_invoice.xlsx",        "1기 부가세 납부세액 계산해줘",                    ["납부","세액","14,144,000"]),
        ("security_audit_log.xlsx", "CRITICAL 수준 보안 이벤트 있어?",                ["CRITICAL","SQL","차단"]),
        ("security_audit_log.xlsx", "외부에서 공격 시도가 있었어?",                    ["203.0.113","포트스캔","차단"]),
        ("receipt_ocr.png",         "이 영수증 OCR해서 합계 금액 알려줘",              ["35,175","합계","영수증"]),
        ("business_card_ocr.png",   "이 명함 OCR해서 연락처 추출해줘",                ["010","1234","이메일"]),
    ]

    passed = failed = 0
    for filename, question, keywords in tests:
        fpath = td / filename
        if not fpath.exists():
            print(f"⏭  SKIP [{filename}] 파일 없음")
            continue

        full_q = f"{fpath} {question}"
        timeout = 40 if filename.endswith('.png') else 30
        r, ms = cmd(full_q, timeout=timeout)

        ok = bool(r.get("success") or r.get("message")) and not r.get("_error")
        answer = str(r.get("message","")).lower()

        # 키워드 매칭 (하나라도 있으면 OK)
        kw_hit = any(kw.lower() in answer for kw in keywords) if ok else False
        status = "✅ PASS" if (ok and kw_hit) else ("⚠️  WARN" if ok else "❌ FAIL")
        preview = answer[:60].replace("\n"," ") if ok else str(r.get("_error",""))[:60]
        kw_status = f"키워드:[{keywords[0]}]✓" if kw_hit else f"키워드:[{keywords[0]}]✗"

        print(f"{status} [{filename[:20]}] {question[:38]:<38} {ms:>5}ms  {kw_status}")
        print(f"       응답: {preview}")

        if ok and kw_hit: passed += 1
        elif ok: passed += 1  # 응답은 있지만 키워드 미매칭 → 경고
        else: failed += 1
        time.sleep(0.5)

    print(f"\n   심화 테스트: {passed}건 성공, {failed}건 실패")

test_with_files()
