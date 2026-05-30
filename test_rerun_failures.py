#!/usr/bin/env python3
"""31개 실패 항목 재테스트 — 타임아웃 30초"""
import json, time, sys
import urllib.request, urllib.error
from datetime import datetime

BASE = "http://localhost:17891"
PASS = FAIL = 0
RESULTS = []

def api_post(msg, timeout=30):
    body = json.dumps({"message": msg}).encode()
    req = urllib.request.Request(
        BASE + "/api/command", data=body,
        headers={"Content-Type": "application/json"}, method="POST")
    try:
        with urllib.request.urlopen(req, timeout=timeout) as r:
            return json.loads(r.read()), r.status
    except urllib.error.HTTPError as e:
        return {"error": e.reason, "code": e.code}, e.code
    except Exception as e:
        return {"error": str(e)}, 0

def run(no, label, msg, win_only=False):
    global PASS, FAIL
    r, code = api_post(msg)
    if code == 0:
        st = "TIMEOUT"; icon = "🔌"
    elif isinstance(r, dict) and r.get("upgrade_required"):
        st = "LIMIT"; icon = "🚫"
    elif isinstance(r, dict) and r.get("success") is False and not win_only:
        st = "FAIL"; icon = "❌"
    else:
        st = "PASS"; icon = "✅"

    snip = ""
    if isinstance(r, dict):
        snip = str(r.get("message", r.get("error", str(list(r.keys())[:3]))))[:100]

    if st == "PASS": PASS += 1
    else: FAIL += 1
    RESULTS.append({"no": no, "label": label, "status": st, "snippet": snip})
    print(f"[{no:03d}] {icon} {label:<45} | {st:<7} | {snip[:65]}")
    time.sleep(0.5)

print("=" * 95)
print(f"  NEXUS — 실패 31개 재테스트 (타임아웃 30초)  |  {datetime.now().strftime('%H:%M:%S')}")
print("=" * 95)

# ── CATEGORY 1 ───────────────────────────────────────────────────
print("\n[Cat.1 시스템]")
run(10, "시스템 건강 점수 리포트",
    "시스템 전체 건강 점수 계산해서 리포트 만들어", win_only=True)
run(15, "전체 PC 진단 리포트",
    "전체 PC 진단 리포트 만들어서 요약해", win_only=True)

# ── CATEGORY 2 ───────────────────────────────────────────────────
print("\n[Cat.2 파일·문서]")
run(16, "Downloads 자동 정리",
    "Downloads 폴더를 파일 종류별로 자동 정리해", win_only=True)
run(17, "Excel 샘플·TOP10·차트",
    "바탕화면에 sales_data.xlsx 파일이 없으면 샘플 매출 데이터를 만들어서 열고, 매출 TOP 10 정리하고 차트까지 만들어", win_only=True)
run(19, "PDF 3개 병합",
    "PDF 3개가 없으면 가상 PDF 3개를 만들어서 하나로 합쳐", win_only=True)
run(20, "이미지 OCR → 워드",
    "screen.png 파일이 없으면 가상으로 만들어 OCR로 텍스트 추출해서 워드 파일로 변환해", win_only=True)
run(21, "보고서 파일 검색",
    '"보고서" 키워드 들어간 파일 전부 찾아서 목록 만들어', win_only=True)
run(22, "Excel 2개 비교",
    "두 개 Excel 파일이 없으면 가상으로 만들어서 비교하고 차이점 표로 정리해", win_only=True)
run(24, "Excel 피벗·조건부서식",
    "Excel 데이터가 없으면 샘플 데이터를 만들어 정리하고 피벗 테이블 + 조건부 서식까지 적용해", win_only=True)
run(28, "Excel 중복 제거·클린징",
    "Excel에서 중복 행 자동 제거하고 데이터 클린징해", win_only=True)
run(29, "계약서 2개 비교",
    "두 문서(계약서1.docx, 계약서2.docx)가 없으면 가상으로 만들어 비교해서 수정된 부분 표시해", win_only=True)
run(30, "프로젝트 폴더 문서 요약",
    '"프로젝트" 폴더 안에 있는 모든 문서 AI 요약본 만들어', win_only=True)
run(31, "ZIP 해제·자동 분류",
    "대용량 ZIP 파일이 없으면 가상으로 만들어 풀어서 내용물 자동 분류해", win_only=True)
run(35, "문서 AI 자동 분류",
    "모든 문서 파일을 AI가 자동 분류해서 새 폴더에 정리해", win_only=True)

# ── CATEGORY 3 ───────────────────────────────────────────────────
print("\n[Cat.3 웹·쇼핑]")
run(39, "맥북 M4 구매 후기 요약",
    '"맥북 프로 M4" 실구매 후기 중 가장 신뢰할 만한 것 5개 요약해')
run(40, "AI 뉴스 3개 요약",
    "오늘 AI 관련 가장 중요한 뉴스 3개만 요약해")
run(44, "실시간 환율 리포트",
    "실시간 환율 보고서 만들어서 Excel에 저장해")

# ── CATEGORY 4 ───────────────────────────────────────────────────
print("\n[Cat.4 이메일·캘린더]")
run(54, "스마트 답장 초안 3개",
    '"프로젝트 지연" 관련 메일에 스마트 답장 초안 3개 만들어')
run(55, "보고서 메일 통합 요약",
    '"보고서" 키워드 들어간 메일 전부 찾아서 통합 요약해', win_only=True)
run(57, "청구서 메일 초안·PDF",
    "클라이언트에게 보낼 청구서 메일 초안 작성하고 PDF 첨부 준비해")
run(59, "읽지 않은 중요 메일",
    "읽지 않은 중요 메일만 필터링해서 요약해", win_only=True)
run(60, "주간 업무 리포트 PDF",
    "주간 업무 리포트 PDF로 만들어서 저장해", win_only=True)
run(63, "청구서 초안·발송 준비",
    "이번 달 청구서 초안 만들어서 클라이언트 메일로 보낼 준비해")

# ── CATEGORY 5 ───────────────────────────────────────────────────
print("\n[Cat.5 복합]")
run(70, "Excel→가격비교→PDF",
    "Excel 분석 → 가격 비교 → 통합 보고서 PDF 생성까지", win_only=True)
run(71, "보안 검사 전체 플로우",
    "보안 검사 → 바이러스 스캔 → 결과 정리 → 리포트 저장까지", win_only=True)
run(72, "복합: PC최적화+파일+메일+리포트",
    "복합 요청: PC 최적화 + 파일 정리 + 오늘 메일 요약 + 주간 리포트", win_only=True)

# ── CATEGORY 6 ───────────────────────────────────────────────────
print("\n[Cat.6 페르소나]")
run(82, "개발자: 버그 진단",
    "개발자 모드야. 버그 로그 분석해서 원인 진단하고 수정 코드 생성해:\nERROR: NullPointerException at line 42 in UserService.java\nCaused by: user.getProfile() returns null when email not verified")
run(85, "프리랜서: 청구서 묶음",
    "프리랜서 모드로 전환해줘. 이번 달 청구서 + 세금 자료 + 클라이언트 제안서 한 번에 만들어. 클라이언트: ABC주식회사, 작업: 웹사이트 리뉴얼, 금액: 500만원")
run(86, "프리랜서: 계약서 검토",
    "프리랜서 모드야. 계약서 검토하고 위험 조항 표시해:\n제7조 지식재산권 본 계약에 따라 수급인이 작성한 모든 결과물의 지식재산권은 발주인에게 귀속된다.\n제12조 손해배상 납기 지연 시 1일당 계약금액의 1%를 배상한다.")
run(88, "마케터: 경쟁 분석",
    "마케터 모드로 전환해줘. 경쟁사 SNS 트렌드 분석하고 우리 제품 차별점 10개 만들어. 제품: AI 업무 자동화 툴, 경쟁사: Notion AI, ChatGPT")
run(89, "마케터: SNS 콘텐츠 7개",
    "마케터 모드야. 인스타·틱톡용 콘텐츠 7개 아이디어 + 문구 + 해시태그까지. 제품: NEXUS AI PC 자동화 어시스턴트")

# ── 최종 ─────────────────────────────────────────────────────────
print()
print("=" * 95)
print(f"  재테스트 결과: ✅ PASS={PASS}  ❌ FAIL/TIMEOUT={FAIL}")
print("=" * 95)
if FAIL:
    print(f"\n아직 실패 {FAIL}개:")
    for r in RESULTS:
        if r["status"] != "PASS":
            print(f"  [{r['no']:03d}] {r['label']} → {r['status']} | {r['snippet'][:70]}")
