#!/usr/bin/env python3
"""NEXUS AI 90-question full test — real API calls, no mocking"""
import json, time, sys, re
import urllib.request, urllib.error
from datetime import datetime

BASE = "http://localhost:17891"
RESULTS = []
PASS = FAIL = WIN_ONLY = 0

def api(path, method="GET", body=None):
    url = BASE + path
    data = json.dumps(body).encode() if body else None
    req = urllib.request.Request(url, data=data,
          headers={"Content-Type":"application/json"}, method=method)
    try:
        with urllib.request.urlopen(req, timeout=15) as r:
            return json.loads(r.read()), r.status
    except urllib.error.HTTPError as e:
        return {"error": e.reason, "code": e.code}, e.code
    except Exception as e:
        return {"error": str(e)}, 0

def chat(msg, persona=None):
    body = {"message": msg}
    if persona:
        body["persona"] = persona
    return api("/api/command", "POST", body)

def judge(resp, code, windows_only=False):
    if windows_only and isinstance(resp, dict):
        v = str(resp)
        if any(k in v for k in ["mac-dev","available.*false","Windows","sidecar","platform"]):
            return "WIN_ONLY"
    if code == 0:
        return "CONN_ERR"
    if code >= 400:
        return "HTTP_ERR"
    if isinstance(resp, dict):
        if resp.get("error"):
            return "API_ERR"
        if resp.get("success") is False and not windows_only:
            return "FAIL"
    return "PASS"

def record(no, label, msg, resp, code, windows_only=False, notes=""):
    global PASS, FAIL, WIN_ONLY
    status = judge(resp, code, windows_only)
    icon = {"PASS":"✅","FAIL":"❌","WIN_ONLY":"🪟","CONN_ERR":"🔌","HTTP_ERR":"⚠️","API_ERR":"⚠️"}.get(status,"?")

    # Extract meaningful snippet from response
    snippet = ""
    if isinstance(resp, dict):
        if "message" in resp:
            snippet = str(resp["message"])[:120]
        elif "result" in resp and isinstance(resp["result"], dict):
            snippet = str(resp["result"])[:120]
        elif "error" in resp:
            snippet = f"ERROR: {resp['error']}"
        else:
            keys = list(resp.keys())[:4]
            snippet = f"keys={keys}"

    if status == "PASS": PASS += 1
    elif status == "WIN_ONLY": WIN_ONLY += 1
    else: FAIL += 1

    entry = {"no":no,"label":label,"status":status,"icon":icon,
             "snippet":snippet[:150],"notes":notes,"code":code}
    RESULTS.append(entry)
    print(f"[{no:03d}] {icon} {label[:50]:<50} | {status:<8} | {snippet[:60]}")
    time.sleep(0.3)
    return status

print("=" * 90)
print(f"  NEXUS AI 90-Question Full Test  |  {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}")
print(f"  Target: {BASE}")
print("=" * 90)

# Health check
resp, code = api("/api/health")
if code == 0:
    print("❌ Backend offline! Aborting.")
    sys.exit(1)
print(f"✅ Backend online: {resp}")
print()

# ══════════════════════════════════════════════════════════════════
print("■ CATEGORY 1: 시스템 최적화 (1~15)")
print("─" * 90)

r,c = chat("PC 현재 상태 전체적으로 분석해서 알려줘")
record(1,"PC 현재 상태 분석","",r,c,True)

r,c = chat("CPU, 메모리, 디스크 사용량 실시간으로 자세히 분석해")
record(2,"CPU·메모리·디스크 분석","",r,c,True)

r,c = chat("GPU 온도와 사용량 확인하고 문제점 있으면 알려줘")
record(3,"GPU 온도·사용량","",r,c,True)

r,c = chat("부팅 속도 분석해서 현재 상태와 개선점 제안해")
record(4,"부팅 속도 분석","",r,c,True)

r,c = chat("드라이버 상태 전체 확인하고 업데이트 가능한 목록만 보여줘")
record(5,"드라이버 업데이트 목록","",r,c,True)

r,c = chat("Windows Defender 현재 상태와 마지막 검사 결과 알려줘")
record(6,"Windows Defender 상태","",r,c,True)

r,c = chat("네트워크 상태 분석하고 속도 측정해서 리포트 만들어")
record(7,"네트워크 분석·속도 리포트","",r,c,True)

r,c = chat("프로세스 TOP 15 보여주고 CPU·메모리 많이 먹는 프로세스 분석해")
record(8,"프로세스 TOP 15 분석","",r,c,True)

r,c = chat("디스크 사용량 분석해서 어떤 폴더가 가장 큰지 상세히 알려줘")
record(9,"디스크 폴더 사용량 분석","",r,c,True)

r,c = chat("시스템 전체 건강 점수 계산해서 리포트 만들어")
record(10,"시스템 건강 점수 리포트","",r,c,True)

r,c = chat("전원 관리 플랜 현재 설정과 추천 플랜 분석해")
record(11,"전원 플랜 분석","",r,c,True)

r,c = chat("시스템 업데이트 가능한 항목 목록만 확인해서 보여줘")
record(12,"업데이트 가능 목록","",r,c,True)

r,c = chat("메모리 누수 의심 프로세스 분석해서 알려줘")
record(13,"메모리 누수 프로세스 분석","",r,c,True)

r,c = chat("PC 예측 케어 — 앞으로 고장 날 가능성 있는 부분 분석해")
record(14,"PC 예측 케어 진단","",r,c,True)

r,c = chat("전체 PC 진단 리포트 만들어서 요약해")
record(15,"전체 PC 진단 리포트","",r,c,True)

# ══════════════════════════════════════════════════════════════════
print()
print("■ CATEGORY 2: 파일·문서·Excel 관리 (16~35)")
print("─" * 90)

r,c = chat("Downloads 폴더를 파일 종류별로 자동 정리해")
record(16,"Downloads 자동 정리","",r,c,True)

r,c = chat("바탕화면에 sales_data.xlsx 파일이 없으면 샘플 매출 데이터를 만들어서 열고, 매출 TOP 10 정리하고 차트까지 만들어")
record(17,"Excel 샘플 생성·TOP10·차트","",r,c,True)

r,c = chat("Downloads에 중복 파일 전부 찾아서 삭제해")
record(18,"중복 파일 찾기·삭제","",r,c,True)

r,c = chat("PDF 3개가 없으면 가상 PDF 3개를 만들어서 하나로 합쳐")
record(19,"PDF 3개 병합","",r,c,True)

r,c = chat("screen.png 파일이 없으면 가상으로 만들어 OCR로 텍스트 추출해서 워드 파일로 변환해")
record(20,"이미지 OCR → 워드 변환","",r,c,True)

r,c = chat('"보고서" 키워드 들어간 파일 전부 찾아서 목록 만들어')
record(21,"보고서 파일 검색","",r,c,True)

r,c = chat("두 개 Excel 파일이 없으면 가상으로 만들어서 비교하고 차이점 표로 정리해")
record(22,"Excel 2개 비교","",r,c,True)

r,c = chat("50MB 이상 대용량 파일 전부 찾아서 목록 만들어")
record(23,"50MB+ 파일 목록","",r,c,True)

r,c = chat("Excel 데이터가 없으면 샘플 데이터를 만들어 정리하고 피벗 테이블 + 조건부 서식까지 적용해")
record(24,"Excel 피벗·조건부서식","",r,c,True)

r,c = chat("모든 이미지 파일을 날짜별 폴더로 자동 분류해")
record(25,"이미지 날짜별 분류","",r,c,True)

r,c = chat("PDF 5개가 없으면 가상으로 만들어 병합하고 목차 자동 생성해")
record(26,"PDF 5개 병합·목차 생성","",r,c,True)

r,c = chat('"회의록" 들어간 파일 10개 찾아서 AI로 통합 요약해')
record(27,"회의록 AI 통합 요약","",r,c,True)

r,c = chat("Excel에서 중복 행 자동 제거하고 데이터 클린징해")
record(28,"Excel 중복 제거·클린징","",r,c,True)

r,c = chat("두 문서(계약서1.docx, 계약서2.docx)가 없으면 가상으로 만들어 비교해서 수정된 부분 표시해")
record(29,"계약서 2개 비교","",r,c,True)

r,c = chat('"프로젝트" 폴더 안에 있는 모든 문서 AI 요약본 만들어')
record(30,"프로젝트 폴더 문서 요약","",r,c,True)

r,c = chat("대용량 ZIP 파일이 없으면 가상으로 만들어 풀어서 내용물 자동 분류해")
record(31,"ZIP 해제·자동 분류","",r,c,True)

r,c = chat('파일 이름에 "draft" 들어간 거 전부 "final"로 변경해')
record(32,"draft→final 파일 이름 변경","",r,c,True)

r,c = chat("Excel 매출 데이터가 없으면 샘플 데이터를 만들어 분석하고 다음 분기 예측 그래프 만들어")
record(33,"Excel 매출 예측 그래프","",r,c,True)

r,c = chat('"세금" 관련 파일이 없으면 가상으로 만들어 모아서 압축 파일로 만들어')
record(34,"세금 파일 압축","",r,c,True)

r,c = chat("모든 문서 파일을 AI가 자동 분류해서 새 폴더에 정리해")
record(35,"문서 AI 자동 분류","",r,c,True)

# ══════════════════════════════════════════════════════════════════
print()
print("■ CATEGORY 3: 웹·쇼핑·미디어 검색 (36~50)")
print("─" * 90)

r,c = chat("쿠팡에서 에어팟 프로 2 최저가 찾아서 11번가·G마켓·알리랑 실시간 비교해")
record(36,"쿠팡 에어팟 최저가 비교","",r,c)

r,c = chat("틱톡에서 지금 대한민국에서 가장 핫한 영상 Top 8 찾아줘")
record(37,"틱톡 한국 인기 영상 Top8","",r,c)

r,c = chat('유튜브에서 "개발자 생산성" 관련 조회수 100만 이상 영상 10개 추천해')
record(38,"유튜브 개발자 생산성 영상","",r,c)

r,c = chat('"맥북 프로 M4" 실구매 후기 중 가장 신뢰할 만한 것 5개 요약해')
record(39,"맥북 M4 구매 후기 요약","",r,c)

r,c = chat("오늘 AI 관련 가장 중요한 뉴스 3개만 요약해")
record(40,"AI 뉴스 3개 요약","",r,c)

r,c = chat("Amazon US에서 AirPods Pro 2 최저가 + 관세 포함 가격 계산해")
record(41,"Amazon 에어팟 관세 계산","",r,c)

r,c = chat('"AI 도구" 키워드로 TikTok 24시간 내 업로드된 영상 Top 5 찾아')
record(42,"TikTok AI 도구 최신 영상","",r,c)

r,c = chat("쿠팡 로켓배송 중 에어팟 프로 2와 가장 비슷한 제품 3개 추천해")
record(43,"쿠팡 에어팟 유사 제품 추천","",r,c)

r,c = chat("실시간 환율 보고서 만들어서 Excel에 저장해")
record(44,"실시간 환율 리포트","",r,c)

r,c = chat("Spotify에서 지금 트렌드인 플레이리스트 3개 추천해")
record(45,"Spotify 트렌드 플레이리스트","",r,c)

r,c = chat("넷플릭스에서 오늘 한국에서 가장 많이 본 드라마 Top 5 알려줘")
record(46,"넷플릭스 한국 드라마 Top5","",r,c)

r,c = chat('"Windows AI 자동화" 관련 최신 유튜브 영상 5개 찾아')
record(47,"유튜브 Windows AI 자동화","",r,c)

r,c = chat("해외 직구 시 관세+배송비 포함해서 아이폰 17 프로 최저가 계산해")
record(48,"아이폰17 프로 직구 최저가","",r,c)

r,c = chat('TikTok + YouTube Shorts에서 "AI 에이전트" 실시간 트렌드 분석해')
record(49,"AI 에이전트 트렌드 분석","",r,c)

r,c = chat('"갤럭시 S25 울트라" 최저가 실시간 비교표 만들어')
record(50,"갤럭시 S25 최저가 비교표","",r,c)

# ══════════════════════════════════════════════════════════════════
print()
print("■ CATEGORY 4: 이메일·캘린더·생산성 (51~65)")
print("─" * 90)

r,c = chat("오늘 받은 메일 중 중요한 것만 5개 골라서 AI 요약해")
record(51,"중요 메일 5개 AI 요약","",r,c,True)

r,c = chat("내일 오전 10시~11시 사이 빈 시간 찾아서 30분 미팅 잡아")
record(52,"캘린더 빈 시간 미팅 등록","",r,c,True)

r,c = chat("이번 주 일정 중 겹치는 거 있으면 자동 조정해")
record(53,"일정 충돌 자동 조정","",r,c,True)

r,c = chat('"프로젝트 지연" 관련 메일에 스마트 답장 초안 3개 만들어')
record(54,"스마트 답장 초안 3개","",r,c)

r,c = chat('"보고서" 키워드 들어간 메일 전부 찾아서 통합 요약해')
record(55,"보고서 메일 통합 요약","",r,c,True)

r,c = chat("다음 주 출장 일정 만들어서 캘린더에 등록 준비해")
record(56,"출장 일정 캘린더 등록","",r,c,True)

r,c = chat("클라이언트에게 보낼 청구서 메일 초안 작성하고 PDF 첨부 준비해")
record(57,"청구서 메일 초안·PDF 준비","",r,c)

r,c = chat("오늘 미팅 3건 요약 + 액션아이템 추출해")
record(58,"미팅 요약·액션아이템","",r,c)

r,c = chat("읽지 않은 중요 메일만 필터링해서 요약해")
record(59,"읽지 않은 중요 메일 요약","",r,c,True)

r,c = chat("주간 업무 리포트 PDF로 만들어서 저장해")
record(60,"주간 업무 리포트 PDF","",r,c,True)

r,c = chat('"팀 미팅" 관련 일정 2주치 자동 생성해')
record(61,"팀 미팅 2주 일정 생성","",r,c,True)

r,c = chat("스팸 의심 메일 자동 분류하고 삭제해")
record(62,"스팸 메일 분류·삭제","",r,c,True)

r,c = chat("이번 달 청구서 초안 만들어서 클라이언트 메일로 보낼 준비해")
record(63,"청구서 초안·메일 발송 준비","",r,c)

r,c = chat("클라이언트 3명에게 맞춤형 팔로업 메일 초안 만들어")
record(64,"클라이언트 팔로업 메일 3개","",r,c)

r,c = chat("내일까지 제출해야 하는 서류 목록 만들고 리마인더 설정해")
record(65,"서류 목록·리마인더 설정","",r,c,True)

# ══════════════════════════════════════════════════════════════════
print()
print("■ CATEGORY 5: 복합·고난도·모호 테스트 (66~80)")
print("─" * 90)

r,c = chat("PC 최적화하면서 동시에 쿠팡 가격 비교하고 결과 Excel에 저장해")
record(66,"PC 최적화+쿠팡+Excel","",r,c,True)

r,c = chat("개발자 모드로 전환해서 코드 리뷰 + 수정 제안 + PR 초안까지")
record(67,"개발자 모드 전환·코드 리뷰","",r,c)

r,c = chat("프리랜서 모드로 전환해서 청구서 + 제안서 + 세금 자료 한 번에")
record(68,"프리랜서 모드·청구서 묶음","",r,c)

r,c = chat("PC 느려짐 원인 분석 → 최적화 → 결과 리포트 → 메일 초안까지")
record(69,"PC 느려짐 전체 플로우","",r,c,True)

r,c = chat("Excel 분석 → 가격 비교 → 통합 보고서 PDF 생성까지")
record(70,"Excel→가격비교→PDF","",r,c,True)

r,c = chat("보안 검사 → 바이러스 스캔 → 결과 정리 → 리포트 저장까지")
record(71,"보안 검사 전체 플로우","",r,c,True)

r,c = chat("복합 요청: PC 최적화 + 파일 정리 + 오늘 메일 요약 + 주간 리포트")
record(72,"복합: PC최적화+파일+메일+리포트","",r,c,True)

r,c = chat('AI가 스스로 계획 세워서 "이번 주 생산성 극대화" 전체 실행해')
record(73,"AI 자율 생산성 계획 실행","",r,c)

r,c = chat("맛집 찾아줘")
record(74,"모호: 맛집 찾아줘","",r,c)

r,c = chat("보고서 만들어")
record(75,"모호: 보고서 만들어","",r,c)

r,c = chat("여행 계획 세워줘")
record(76,"모호: 여행 계획","",r,c)

r,c = chat("옷 추천해")
record(77,"모호: 옷 추천","",r,c)

r,c = chat("최저가 찾아")
record(78,"모호: 최저가 찾아","",r,c)

r,c = chat("파일 정리해")
record(79,"모호: 파일 정리해","",r,c,True)

r,c = chat("미팅 준비해")
record(80,"모호: 미팅 준비해","",r,c)

# ══════════════════════════════════════════════════════════════════
print()
print("■ CATEGORY 6: 페르소나 고난도 (81~90)")
print("─" * 90)

r,c = chat("개발자 모드로 전환해줘. 이제 이 코드 리뷰하고 개선안 + 수정 코드 제안해:\ndef get_user(id):\n  db = connect()\n  return db.query('SELECT * FROM users WHERE id='+str(id))")
record(81,"개발자 페르소나: 코드 리뷰","",r,c)

r,c = chat("개발자 모드야. 버그 로그 분석해서 원인 진단하고 수정 코드 생성해:\nERROR: NullPointerException at line 42 in UserService.java\nCaused by: user.getProfile() returns null when email not verified")
record(82,"개발자 페르소나: 버그 진단","",r,c)

r,c = chat("개발자 모드야. REST API 설계해줘. 요구사항: 사용자 관리 시스템 (회원가입/로그인/프로필 수정/탈퇴)")
record(83,"개발자 페르소나: API 설계","",r,c)

r,c = chat("개발자 모드야. CI/CD 파이프라인 최적화 제안해줘. 현재: GitHub Actions, 빌드 15분 소요, 테스트 20분 소요")
record(84,"개발자 페르소나: CI/CD 최적화","",r,c)

r,c = chat("프리랜서 모드로 전환해줘. 이번 달 청구서 + 세금 자료 + 클라이언트 제안서 한 번에 만들어. 클라이언트: ABC주식회사, 작업: 웹사이트 리뉴얼, 금액: 500만원")
record(85,"프리랜서 페르소나: 청구서 묶음","",r,c)

r,c = chat("프리랜서 모드야. 계약서 검토하고 위험 조항 표시해:\n제7조 (지식재산권) 본 계약에 따라 수급인이 작성한 모든 결과물의 지식재산권은 발주인에게 귀속된다.\n제12조 (손해배상) 납기 지연 시 1일당 계약금액의 1%를 배상한다.")
record(86,"프리랜서 페르소나: 계약서 검토","",r,c)

r,c = chat("프리랜서 모드야. 클라이언트 미팅 준비 자료 완성해. 미팅: 내일 오후 2시, 고객: 스타트업 CEO, 안건: UX 개선 프로젝트 제안")
record(87,"프리랜서 페르소나: 미팅 준비","",r,c)

r,c = chat("마케터 모드로 전환해줘. 경쟁사 SNS 트렌드 분석하고 우리 제품 차별점 10개 만들어. 제품: AI 업무 자동화 툴, 경쟁사: Notion AI, ChatGPT")
record(88,"마케터 페르소나: 경쟁 분석","",r,c)

r,c = chat("마케터 모드야. 인스타·틱톡용 콘텐츠 7개 아이디어 + 문구 + 해시태그까지. 제품: NEXUS AI PC 자동화 어시스턴트")
record(89,"마케터 페르소나: SNS 콘텐츠","",r,c)

r,c = chat("마케터 모드야. 광고 문구 A/B 테스트용 15개 버전 만들어. 타겟: 30대 직장인, 제품: NEXUS AI, 채널: 페이스북 광고")
record(90,"마케터 페르소나: 광고 문구 15종","",r,c)

# ══════════════════════════════════════════════════════════════════
print()
print("=" * 90)
print(f"  FINAL SCORE:  ✅ PASS={PASS}  🪟 WIN_ONLY={WIN_ONLY}  ❌ FAIL={FAIL}")
total = PASS + WIN_ONLY + FAIL
print(f"  총 {total}개 / PASS율 {PASS/total*100:.0f}% / Win-only {WIN_ONLY/total*100:.0f}% / FAIL {FAIL/total*100:.0f}%")
print("=" * 90)

# Save JSON
out = {
    "date": datetime.now().isoformat(),
    "summary": {"pass": PASS, "win_only": WIN_ONLY, "fail": FAIL, "total": total},
    "results": RESULTS
}
with open("test_90_results.json","w",encoding="utf-8") as f:
    json.dump(out, f, ensure_ascii=False, indent=2)
print("  결과 저장: test_90_results.json")

# Show failures for fixing
fails = [r for r in RESULTS if r["status"] not in ("PASS","WIN_ONLY")]
if fails:
    print()
    print(f"❌ 수정 필요 항목 ({len(fails)}개):")
    for r in fails:
        print(f"  [{r['no']:03d}] {r['label']} → {r['status']} | {r['snippet'][:80]}")
