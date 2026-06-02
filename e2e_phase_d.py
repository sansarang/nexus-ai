#!/usr/bin/env python3
"""
e2e_phase_d.py — Phase D 자가 치유 Agent 시뮬레이션
- 의도적으로 다양한 명령 100+개 호출 → 텔레메트리 쌓기
- 5분 ticker 대신 즉시 분석 트리거 (수동)
- Agent가 어떤 패치 제안하는지 확인
"""

import json
import time
import urllib.request

BASE = "http://127.0.0.1:17891"

def call(path, body=None, method="POST"):
    url = BASE + path
    data = json.dumps(body).encode() if body else None
    req = urllib.request.Request(url, data=data, method=method)
    req.add_header("Content-Type", "application/json")
    try:
        with urllib.request.urlopen(req, timeout=25) as resp:
            return json.loads(resp.read().decode())
    except Exception as e:
        return {"_error": str(e)}

def cmd(msg):
    return call("/api/command", {"message": msg, "lang": "ko", "user_email": "phased@test"})

# 1) 다양한 명령 50+개 (텔레메트리 쌓기)
print("━━━ 텔레메트리 수집 (50건) ━━━")
COMMANDS = [
    "PC 상태", "메모리", "디스크", "느려졌어", "캐시 비워",
    "오늘 일정", "이번 주", "회의 잡아", "받은 메일", "메일 요약",
    "엑셀로 매출", "PDF 보고서", "회의록 작성",
    "쿠팡 무선이어폰", "AI 뉴스", "유튜브 코딩 강의",
    "서울 날씨", "부산 날씨",
    "계약서 검토", "삼성전자 분석", "메트포르민",
    "PC 재시작", "전원 꺼", "포맷",
    "엑셀로 매출 정리하고 PDF로 저장",
    "박부장 계약서 손해배상",
    "코드 리뷰", "인스타 광고 카피",
    "안녕", "고마워",
    "화면 분석",
    # 빈 의도 / clarify 유발
    "어 잠깐", "그거", "음...", "글쎄", "모르겠어",
    "asdfghjk", "12345", "?", "ㅋㅋ",
    # 미인식 한국어
    "투두 정리해주삼", "월렛 잔액", "테이블 만들어줘",
    "코파일럿이랑 비교해", "노션이랑 연동", "지라 티켓",
    # 도메인 페르소나
    "임상 가이드라인 당뇨",
    "판례 검색 임대차",
    "비트코인 차트",
]
for i, c in enumerate(COMMANDS):
    r = cmd(c)
    if (i+1) % 10 == 0:
        print(f"  진행: {i+1}/{len(COMMANDS)}")

# 2) Agent 즉시 분석 트리거 (수동, 5분 ticker 대기 X)
print("\n━━━ Agent 분석 즉시 트리거 ━━━")
analyzed = call("/api/agent/analyze-now")
print(f"  생성된 제안: {analyzed.get('generated', 0)}건")

# 3) Agent 상태 조회
print("\n━━━ Agent 상태 ━━━")
status = call("/api/agent/status", method="GET")
print(json.dumps(status, ensure_ascii=False, indent=2)[:1500])

# 4) Pending proposals 조회
print("\n━━━ 패치 제안 조회 ━━━")
props = call("/api/agent/proposals", method="GET")
print(f"  대기 중: {props.get('count', 0)}건")
for p in props.get("proposals", [])[:10]:
    print(f"  [{p['severity']}] {p['agent']}: {p['title']}")
    print(f"    → {p['description'][:100]}")

print("\n━━━ 종료 ━━━")
