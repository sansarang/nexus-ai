#!/usr/bin/env python3
"""
e2e_truth.py — 진짜 검증 (사장님 의심 해소용)
- 100+ 명령 실측
- 응답 시간 측정
- 빈 응답 카운트
- 멀티턴 대화 5턴
- 카드 타입 매칭률
- 페르소나 자동 매칭 정확도
- 평균/최악 응답 시간

거짓 없이 raw 측정값만 출력
"""

import json
import time
import urllib.request
import urllib.error
from collections import defaultdict
from pathlib import Path

BASE = "http://127.0.0.1:17891"


def call(path, body=None, timeout=20):
    url = BASE + path
    data = json.dumps(body).encode() if body else None
    req = urllib.request.Request(url, data=data, method="POST")
    req.add_header("Content-Type", "application/json")
    start = time.time()
    try:
        with urllib.request.urlopen(req, timeout=timeout) as resp:
            elapsed = time.time() - start
            return json.loads(resp.read().decode()), elapsed
    except urllib.error.HTTPError as e:
        return {"_error": f"HTTP {e.code}"}, time.time() - start
    except Exception as e:
        return {"_error": str(e)}, time.time() - start


def cmd(msg, lang="ko"):
    return call("/api/command", {"message": msg, "lang": lang, "user_email": "truth@validation"})


# ════════════════════════════════════════════════
# 100+ 명령 카탈로그 (사장님이 진짜 쓸 만한 거)
# ════════════════════════════════════════════════

COMMANDS = [
    # === 시스템 (15) ===
    ("PC 상태 보여줘", "stats"),
    ("메모리 어때", "stats"),
    ("CPU 사용률", "stats"),
    ("디스크 공간 얼마", "stats"),
    ("내 컴퓨터 상태", "stats"),
    ("느려졌어 왜이래", "scan"),
    ("렉 걸려", "scan"),
    ("보안 스캔", "scan"),
    ("악성코드 검사", "scan"),
    ("PC 진단해줘", "scan"),
    ("캐시 비워줘", "clean"),
    ("임시파일 정리", "clean"),
    ("디스크 정리", "clean"),
    ("청소해", "clean"),
    ("공간 확보해줘", "clean"),

    # === 위험 액션 (8) — 확인 카드 필수 ===
    ("PC 재시작해", "restart"),
    ("컴퓨터 꺼", "shutdown"),
    ("전원 종료", "shutdown"),
    ("리부팅", "restart"),
    ("절전 모드로", "sleep"),
    ("디스크 포맷", "format_disk"),
    ("결제해줘", "payment"),
    ("파일 삭제해", "file_delete"),

    # === 자동 문서 생성 (10) ===
    ("엑셀로 매출 정리해", "excel_auto_create"),
    ("엑셀 만들어 분기 실적", "excel_auto_create"),
    ("스프레드시트 생성", "excel_auto_create"),
    ("PDF 보고서 만들어", "pdf_auto_create"),
    ("PDF로 작성해줘 시장 분석", "pdf_auto_create"),
    ("회의록 작성", "doc_auto_create"),
    ("메모 만들어", "doc_auto_create"),
    ("보고서 작성", "doc_auto_create"),
    ("엑셀 분석해", "excel_analyze"),
    ("이 엑셀 요약해", "excel_analyze"),

    # === 검색/뉴스 (10) ===
    ("쿠팡에서 무선이어폰 최저가", "price_compare"),
    ("아이폰 16 가격 비교", "price_compare"),
    ("AI 뉴스", "news_search"),
    ("오늘 속보", "news_search"),
    ("유튜브에서 코딩 강의", "youtube_search"),
    ("틱톡 트렌드", "video_search"),
    ("뮤직비디오 추천", "video_search"),
    ("플레이리스트 찾아", "video_search"),
    ("번역해 안녕하세요", "translate"),
    ("영어로 번역", "translate"),

    # === 일정/회의 (8) ===
    ("오늘 일정", "calendar_today"),
    ("이번 주 스케줄", "calendar_week"),
    ("빈 시간 찾아", "calendar_find_slot"),
    ("미팅 잡을 시간", "calendar_find_slot"),
    ("회의 목록", "meeting_list"),
    ("회의 요약해", "meeting_summary"),
    ("받은 메일", "email_inbox"),
    ("메일 요약", "email_summarize"),

    # === 직업 페르소나 (10) ===
    ("계약서 검토해", "contract_review"),
    ("NDA 검토 포인트", "contract_review"),
    ("삼성전자 주가 분석", "stock_analysis"),
    ("비트코인 전망", "stock_analysis"),
    ("판례 검색 임대차", "legal_search"),
    ("개인정보보호법 조항", "chat"),
    ("메트포르민 부작용", "medical_search"),
    ("당뇨 가이드라인", "medical_search"),
    ("PRD 작성 도와줘", "chat"),
    ("리서치 분석", "chat"),

    # === 멀티 액션 (10) — 핵심 차별점 ===
    ("엑셀로 매출 정리하고 PDF로도 저장해", "workflow_run"),
    ("AI 뉴스 찾고 노트에 저장", "workflow_run"),
    ("오늘 일정 보고 빈 시간에 회의 잡아", "workflow_run"),
    ("메일 확인하고 중요한 거 요약", "workflow_run"),
    ("뉴스 검색하고 엑셀로 정리", "workflow_run"),
    ("PC 진단하고 정리도 해", "workflow_run"),
    ("쿠팡 가격 찾고 PDF로 저장", "workflow_run"),
    ("유튜브 영상 찾고 자막 떼", "workflow_run"),
    ("날씨 보고 일정 잡아", "workflow_run"),
    ("환율 확인하고 메모에 적어", "workflow_run"),

    # === Vision (3) ===
    ("화면 분석해줘", "vision"),
    ("스크린샷 분석", "vision"),
    ("이 화면 뭐야", "vision"),

    # === 날씨 (5) ===
    ("서울 날씨", "weather"),
    ("부산 날씨 어때", "weather"),
    ("도쿄 기온", "weather"),
    ("미세먼지 어때", "weather"),
    ("내일 비 와", "chat"),

    # === Windows 전용 (10) — Mac에선 stub ===
    ("디펜더 상태", "defender_status"),
    ("바이러스 검사", "virus_check"),
    ("윈도우 업데이트", "windows_updates"),
    ("드라이버 확인", "driver_check"),
    ("시작 프로그램", "startup_items"),
    ("설치 프로그램 목록", "programs_list"),
    ("GPU 상태", "gpu_stats"),
    ("프로세스 TOP", "process_top"),
    ("의심 프로세스", "process_security"),
    ("네트워크 분석", "network_analysis"),

    # === RAG (3) — 사용자 문서 검색 ===
    ("박부장 계약서 손해배상 한도", "chat"),
    ("매출 보고서 어디 있어", "chat"),
    ("최근 작성한 메모 찾아", "recall_search"),

    # === 자유 채팅 (5) ===
    ("안녕", "chat"),
    ("오늘 기분 어때", "chat"),
    ("고마워", "chat"),
    ("넌 누구야", "chat"),
    ("뭐 잘해?", "chat"),
]


print(f"━━━ 진짜 검증: {len(COMMANDS)}개 명령 실측 ━━━\n")

# RAG 테스트 문서 생성
test_doc = Path.home() / "Desktop" / "_truth_rag.md"
test_doc.write_text("# 박부장 계약서\n손해배상 한도: 1억원\n위약금: 매출 5%\n", encoding="utf-8")

# ════════════════════════════════════════════════
# 실측
# ════════════════════════════════════════════════
total = 0
hit_action = 0      # 정확 액션 일치
hit_card = 0        # 카드 있음 (action ≠ "")
empty_msg = 0       # 빈 응답
err_count = 0       # 에러
clarify_count = 0   # clarify (의도 불명)
times = []
slow = []           # >5초
category_stats = defaultdict(lambda: {"total": 0, "hit_action": 0, "hit_card": 0, "empty": 0})

# 카테고리 매핑
def cat_of(idx):
    if idx < 15: return "시스템"
    if idx < 23: return "위험액션"
    if idx < 33: return "자동문서"
    if idx < 43: return "검색뉴스"
    if idx < 51: return "일정회의"
    if idx < 61: return "직업페르소나"
    if idx < 71: return "멀티액션"
    if idx < 74: return "Vision"
    if idx < 79: return "날씨"
    if idx < 89: return "Windows전용"
    if idx < 92: return "RAG"
    return "자유채팅"

for i, (q, expected) in enumerate(COMMANDS):
    r, elapsed = cmd(q)
    total += 1
    times.append(elapsed)
    cat = cat_of(i)
    category_stats[cat]["total"] += 1
    if elapsed > 5:
        slow.append((q, elapsed))

    action = r.get("action", "")
    msg = r.get("message", "")

    if "_error" in r:
        err_count += 1
        category_stats[cat]["empty"] += 1
        continue
    if not msg or len(msg.strip()) == 0:
        empty_msg += 1
        category_stats[cat]["empty"] += 1
    if action == "clarify":
        clarify_count += 1
    if action and action != "clarify":
        hit_card += 1
        category_stats[cat]["hit_card"] += 1
    if action == expected or (expected == "workflow_run" and action in ("workflow_run", "multi_action")):
        hit_action += 1
        category_stats[cat]["hit_action"] += 1

    # 짧은 진행 표시
    if (i + 1) % 20 == 0:
        print(f"  진행: {i+1}/{len(COMMANDS)} ({elapsed:.2f}s 평균)")

# ════════════════════════════════════════════════
# 멀티턴 대화 (5턴)
# ════════════════════════════════════════════════
print("\n━━━ 멀티턴 대화 5턴 ━━━")
turns = [
    "AI 뉴스 찾아줘",
    "그중 가장 흥미로운 거 요약해",
    "그거 PDF로 만들어줘",
    "방금 그 PDF 어디 있어?",
    "삭제해줘",
]
multi_turn_ok = 0
for i, q in enumerate(turns):
    r, e = cmd(q)
    msg = r.get("message", "")
    action = r.get("action", "")
    has_response = len(msg) > 0
    if has_response:
        multi_turn_ok += 1
    print(f"  Turn {i+1}: '{q[:40]:40}' → [{action}] {'✓' if has_response else '✗'} ({len(msg)}자, {e:.2f}s)")

# 정리
try: test_doc.unlink()
except: pass

# ════════════════════════════════════════════════
# 최종 리포트
# ════════════════════════════════════════════════
avg = sum(times) / len(times)
sorted_times = sorted(times)
p50 = sorted_times[len(sorted_times) // 2]
p95 = sorted_times[int(len(sorted_times) * 0.95)]
p99 = sorted_times[int(len(sorted_times) * 0.99)]

print("\n" + "═" * 60)
print("  🎯 최종 결과")
print("═" * 60)
print(f"  총 명령: {total}개")
print(f"  정확 액션 매칭: {hit_action}/{total} = {hit_action/total*100:.1f}%")
print(f"  카드 표시 (action ≠ ''): {hit_card}/{total} = {hit_card/total*100:.1f}%")
print(f"  빈 응답 (message=''): {empty_msg}/{total} = {empty_msg/total*100:.1f}%")
print(f"  에러: {err_count}/{total}")
print(f"  Clarify (의도 불명): {clarify_count}/{total}")
print()
print(f"  ⏱️ 응답 시간")
print(f"  평균: {avg:.2f}s")
print(f"  P50:  {p50:.2f}s")
print(f"  P95:  {p95:.2f}s")
print(f"  P99:  {p99:.2f}s")
print(f"  최대: {max(times):.2f}s")
print(f"  >5초: {len(slow)}건")
print()
print(f"  🔄 멀티턴: {multi_turn_ok}/5 응답")
print()
print(f"  📊 카테고리별 정확도:")
for cat, s in category_stats.items():
    action_rate = s["hit_action"]/s["total"]*100 if s["total"] else 0
    card_rate = s["hit_card"]/s["total"]*100 if s["total"] else 0
    empty_rate = s["empty"]/s["total"]*100 if s["total"] else 0
    print(f"    {cat:12s} 액션 {action_rate:5.1f}% / 카드 {card_rate:5.1f}% / 빈응답 {empty_rate:5.1f}% ({s['total']}건)")
print()
if slow:
    print(f"  🐌 느린 응답 TOP {min(5, len(slow))}:")
    for q, e in sorted(slow, key=lambda x: -x[1])[:5]:
        print(f"    {e:.2f}s — '{q}'")
print("═" * 60)
