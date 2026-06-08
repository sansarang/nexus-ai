#!/usr/bin/env python3
"""
Nexus AI 전체 기능 테스트 하네스 (Windows/Mac)

설치된 앱이 떠 있는 상태(=Go 백엔드 127.0.0.1:17891 + Python 사이드카 127.0.0.1:17893)에서
실행하면 기능을 카테고리별로 두드려 PASS / FAIL / SKIP 을 출력하고 요약한다.

사용법:
  python feature_test.py --setup        # 테스트용 실제 파일/샘플영상 생성 후 종료
  python feature_test.py                 # 전체 기능 테스트
  python feature_test.py --setup --run   # 픽스처 생성 + 바로 테스트
  python feature_test.py --only 자동화,비전   # 특정 카테고리만

SKIP 의미: 플랫폼(Windows 전용) 또는 API 키 미설정 또는 엔진 미준비(UIA Available=false).
가이드: docs/automation/feature-test-guide.md
"""
import argparse
import base64
import json
import os
import platform
import sys
import time
import urllib.error
import urllib.request
from pathlib import Path

GO_BASE = "http://127.0.0.1:17891"
PY_BASE = "http://127.0.0.1:17893"
IS_WIN = platform.system() == "Windows"
FIX_DIR = Path.home() / "nexus_feature_test"
SAMPLE_MP4_URL = "https://download.samplelib.com/mp4/sample-5s.mp4"

# 최소 유효 PNG (1x1 흰색) / PDF — stdlib만으로 실파일 생성용.
_PNG_1x1 = base64.b64decode(
    "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg=="
)
_MIN_PDF = (
    b"%PDF-1.4\n1 0 obj<</Type/Catalog/Pages 2 0 R>>endobj\n"
    b"2 0 obj<</Type/Pages/Kids[3 0 R]/Count 1>>endobj\n"
    b"3 0 obj<</Type/Page/Parent 2 0 R/MediaBox[0 0 200 200]/Contents 4 0 R/Resources<</Font<</F1 5 0 R>>>>>>endobj\n"
    b"4 0 obj<</Length 44>>stream\nBT /F1 18 Tf 20 100 Td (Nexus Test PDF) Tj ET\nendstream endobj\n"
    b"5 0 obj<</Type/Font/Subtype/Type1/BaseFont/Helvetica>>endobj\n"
    b"trailer<</Root 1 0 R>>\n%%EOF"
)

results = []  # (category, name, status, detail)


def http(method, url, body=None, timeout=60):
    data = json.dumps(body).encode() if body is not None else None
    headers = {"Content-Type": "application/json"} if data else {}
    req = urllib.request.Request(url, data=data, method=method, headers=headers)
    try:
        with urllib.request.urlopen(req, timeout=timeout) as r:
            raw = r.read().decode("utf-8", "replace")
            try:
                return r.status, json.loads(raw)
            except Exception:
                return r.status, {"_raw": raw[:200]}
    except urllib.error.HTTPError as e:
        try:
            return e.code, json.loads(e.read().decode("utf-8", "replace"))
        except Exception:
            return e.code, {}
    except Exception as e:
        return 0, {"_error": str(e)}


def record(cat, name, status, detail=""):
    results.append((cat, name, status, detail))
    icon = {"PASS": "✅", "FAIL": "❌", "SKIP": "⏭️"}[status]
    print("  %s [%s] %s — %s" % (icon, cat, name, str(detail)[:90]))


def cmd(message, lang="ko", timeout=70):
    return http("POST", GO_BASE + "/api/command", {"message": message, "lang": lang}, timeout)


def _is_key_missing(msg):
    """API 키 미설정 응답 감지 → 설치본은 번들 키로 PASS하지만 키 없는 환경에선 SKIP 처리."""
    m = (msg or "").lower()
    return any(s in m for s in ["api 키", "api key", "not configured", "키가 설정되지", "키를 입력"])


def test_cmd(cat, name, message, lang="ko", want_keywords=None):
    """/api/command 호출 → success!=false & 비어있지 않은 응답이면 PASS.
       API 키 미설정 응답은 SKIP(설치본=번들 키로 동작, 개발/무키 환경=SKIP)."""
    code, d = cmd(message, lang)
    if code == 0:
        record(cat, name, "FAIL", "백엔드 응답 없음: %s" % d.get("_error", ""))
        return
    msg = str(d.get("message") or "")
    if _is_key_missing(msg):
        record(cat, name, "SKIP", "API 키 미설정 (설치본은 번들 키로 동작)")
        return
    body_text = msg + json.dumps(d.get("result") or "", ensure_ascii=False)
    # success는 명시적으로 true여야 PASS (없거나 false면 실패로 간주 — false-PASS 방지).
    ok = code == 200 and d.get("success") is True and len(body_text.strip()) > 0
    if ok and want_keywords:
        ok = any(k in body_text for k in want_keywords)
    detail = "action=%s · %s" % (d.get("action"), msg[:50])
    record(cat, name, "PASS" if ok else "FAIL", detail)


# ──────────────────────────────────────────────────────────────
#  픽스처 생성
# ──────────────────────────────────────────────────────────────
def setup_fixtures():
    print("\n=== 픽스처 생성: %s ===" % FIX_DIR)
    messy = FIX_DIR / "messy"
    messy.mkdir(parents=True, exist_ok=True)
    # 지저분한 폴더 — 여러 확장자
    for fn, content in [
        ("photo1.jpg", _PNG_1x1), ("photo2.png", _PNG_1x1),
        ("doc1.pdf", _MIN_PDF), ("notes.txt", b"nexus test note\n"),
        ("data.csv", b"a,b,c\n1,2,3\n"), ("readme.md", b"# test\n"),
        ("song.mp3", b"ID3"), ("archive.zip", b"PK\x03\x04"),
    ]:
        (messy / fn).write_bytes(content)
    print("  ✓ 지저분한 폴더: %s (%d개 파일)" % (messy, len(list(messy.iterdir()))))

    (FIX_DIR / "test.txt").write_text("넥서스 테스트 문서입니다. 핵심 데이터: 매출 1,200만원, 날짜 2026-06-08.\n", encoding="utf-8")
    (FIX_DIR / "test.pdf").write_bytes(_MIN_PDF)
    (FIX_DIR / "test.png").write_bytes(_PNG_1x1)
    print("  ✓ test.txt / test.pdf / test.png")

    # xlsx — openpyxl 있으면
    try:
        import openpyxl
        wb = openpyxl.Workbook()
        ws = wb.active
        ws.append(["이름", "이메일", "금액"])
        ws.append(["홍길동", "hong@a.com", 1000])
        ws.append(["김철수", "kim@b.com", 2000])
        wb.save(str(FIX_DIR / "test.xlsx"))
        print("  ✓ test.xlsx (openpyxl)")
    except Exception as e:
        print("  ⏭️ test.xlsx SKIP (openpyxl 없음: %s)" % e)

    # 샘플 동영상 1개 다운로드
    mp4 = FIX_DIR / "sample.mp4"
    if mp4.exists() and mp4.stat().st_size > 0:
        print("  ✓ sample.mp4 (이미 존재)")
    else:
        try:
            urllib.request.urlretrieve(SAMPLE_MP4_URL, str(mp4))
            print("  ✓ sample.mp4 다운로드 (%d KB)" % (mp4.stat().st_size // 1024))
        except Exception as e:
            print("  ⏭️ sample.mp4 다운로드 실패(graceful): %s" % e)
    print("=== 픽스처 준비 완료 ===\n")


# ──────────────────────────────────────────────────────────────
#  카테고리별 테스트
# ──────────────────────────────────────────────────────────────
def t_health():
    cat = "헬스"
    code, d = http("GET", GO_BASE + "/api/health", timeout=8)
    record(cat, "Go 백엔드 /api/health", "PASS" if code == 200 else "FAIL", "code=%s" % code)
    code, d = http("GET", PY_BASE + "/health", timeout=8)
    record(cat, "Python 사이드카 /health", "PASS" if code == 200 else "SKIP", "code=%s (사이드카 미기동 가능)" % code)
    code, d = http("GET", GO_BASE + "/api/automation/status", timeout=8)
    record(cat, "자동화 엔진 status", "PASS" if code == 200 else "FAIL", "available=%s" % d.get("available"))
    code, d = http("GET", PY_BASE + "/desktop/uia/status", timeout=8)
    av = d.get("available")
    record(cat, "UIA status", "PASS" if code == 200 else "SKIP", "available=%s platform=%s" % (av, d.get("platform")))


def t_llm():
    test_cmd("LLM", "한국어 지식응답", "대한민국 수도와 인구를 한 문장으로 알려줘", want_keywords=["서울"])
    test_cmd("LLM", "영어 응답", "In one sentence, what is the capital of France?", lang="en", want_keywords=["Paris"])


def t_search():
    test_cmd("검색", "웹검색", "최신 인공지능 뉴스 검색해줘")
    test_cmd("검색", "딥서치", "전기차 배터리 기술 딥서치")
    test_cmd("검색", "뉴스", "오늘 IT 뉴스 알려줘")
    test_cmd("검색", "영상검색", "잔잔한 음악 유튜브에서 찾아줘")


def t_info():
    test_cmd("정보", "날씨", "서울 오늘 날씨 알려줘")
    test_cmd("정보", "주가", "삼성전자 주가 알려줘")
    test_cmd("정보", "환율", "오늘 달러 환율 알려줘")
    test_cmd("정보", "길찾기", "서울역에서 강남역 가는 방법")


def t_system():
    test_cmd("시스템", "PC 상태", "내 PC 메모리랑 디스크 상태 알려줘")
    test_cmd("시스템", "보안 스캔", "보안 스캔 해줘")
    test_cmd("시스템", "번역", "Hello, how are you? 한국어로 번역해줘")


def t_files():
    folder = str(FIX_DIR / "messy")
    test_cmd("파일", "파일 정리", "%s 폴더 파일 종류별로 정리해줘" % folder)
    test_cmd("파일", "엑셀 자동작성", "이번 달 가계부 표 엑셀로 만들어줘")
    test_cmd("파일", "문서 요약", "%s 문서 요약해줘" % str(FIX_DIR / "test.txt"))


def t_productivity():
    test_cmd("생산성", "스케줄/매크로", "매일 아침 9시에 크롬 열어줘")
    test_cmd("생산성", "클립보드 처리", "방금 복사한 내용 번역해줘")
    test_cmd("생산성", "워크플로(멀티액션)", "오늘 IT 뉴스 찾아서 PDF로 저장해줘")
    test_cmd("생산성", "페르소나 전환", "개발자 모드로 전환해줘")


def t_vision():
    cat = "비전"
    png = FIX_DIR / "test.png"
    if not png.exists():
        record(cat, "스크린샷 분석", "SKIP", "픽스처 없음(--setup 먼저)")
        return
    b64 = base64.b64encode(png.read_bytes()).decode()
    code, d = http("POST", PY_BASE + "/screenshot/analyze",
                   {"image_base64": b64, "question": "이 이미지에 뭐가 있어?"}, timeout=40)
    if code == 0:
        record(cat, "스크린샷 분석", "SKIP", "사이드카 미기동")
    elif d.get("success"):
        record(cat, "스크린샷 분석", "PASS", str(d.get("analysis", ""))[:50])
    else:
        msg = d.get("message", "")
        # 키 없음/플랫폼 = SKIP, 그 외 = FAIL
        skip = "키" in msg or "Windows" in msg or "claude" in msg.lower()
        record(cat, "스크린샷 분석", "SKIP" if skip else "FAIL", msg[:70])
    # 명령 경로(Groq Vision) — Windows에서 실제 화면 캡처
    test_cmd(cat, "화면 분석(명령)", "지금 화면 분석해줘")


def t_video():
    cat = "영상"
    mp4 = FIX_DIR / "sample.mp4"
    if mp4.exists() and mp4.stat().st_size > 0:
        record(cat, "샘플 영상 픽스처", "PASS", "%d KB" % (mp4.stat().st_size // 1024))
    else:
        record(cat, "샘플 영상 픽스처", "SKIP", "다운로드 안 됨")
    # 영상 다운로드 기능(yt-dlp) — 짧은 공개 영상
    test_cmd(cat, "영상 다운로드(명령)", "https://www.youtube.com/watch?v=aqz-KE-bpKQ 이 영상 다운로드해줘")


def t_automation():
    cat = "자동화"
    wf = {"name": "feature_test_wf", "steps": [
        {"kind": "set_text", "selector": {"name": "이름"}, "value": "{{name}}"},
        {"kind": "click", "selector": {"name": "제출", "role": "button"}},
        {"kind": "verify", "selector": {"name": "결과"}, "expect": "완료"},
    ]}
    code, d = http("POST", GO_BASE + "/api/automation/workflows", wf, timeout=10)
    record(cat, "워크플로 저장", "PASS" if d.get("success") else "FAIL", "steps=%s" % d.get("steps"))
    code, d = http("GET", GO_BASE + "/api/automation/workflows", timeout=10)
    has = "feature_test_wf" in (d.get("workflows") or [])
    record(cat, "워크플로 목록", "PASS" if has else "FAIL", "count=%s" % d.get("count"))
    code, d = http("GET", GO_BASE + "/api/automation/workflows/feature_test_wf", timeout=10)
    record(cat, "워크플로 조회", "PASS" if d.get("success") else "FAIL", "code=%s" % code)
    # dry-run: 실행 없이 미리보기
    code, d = http("POST", GO_BASE + "/api/automation/run",
                   {"workflow_name": "feature_test_wf", "dry_run": True}, timeout=10)
    record(cat, "run dry-run(미리보기)", "PASS" if d.get("dry_run") else "FAIL", "count=%s" % d.get("count"))
    # 실제 실행: 엔진 미준비면 501(정상), 준비되면 실행 결과
    code, d = http("POST", GO_BASE + "/api/automation/run", {"workflow_name": "feature_test_wf"}, timeout=20)
    if code == 501:
        record(cat, "run 실행", "SKIP", "엔진 미준비(501) — UIA Available=false. Windows+pywinauto 후 동작")
    elif code == 200 and d.get("success"):
        record(cat, "run 실행", "PASS", "엔진 동작 — 전 단계 성공")
    elif code == 207:
        record(cat, "run 실행", "FAIL", "일부 단계 실패(partial) — result 확인 필요")
    else:
        record(cat, "run 실행", "FAIL", "code=%s" % code)


CATEGORIES = {
    "헬스": t_health, "LLM": t_llm, "검색": t_search, "정보": t_info,
    "시스템": t_system, "파일": t_files, "생산성": t_productivity,
    "비전": t_vision, "영상": t_video, "자동화": t_automation,
}


def backend_up():
    code, _ = http("GET", GO_BASE + "/api/health", timeout=5)
    return code == 200


def summarize():
    print("\n" + "=" * 56)
    print("기능별 요약")
    print("=" * 56)
    cats = {}
    for cat, _n, st, _d in results:
        c = cats.setdefault(cat, {"PASS": 0, "FAIL": 0, "SKIP": 0})
        c[st] += 1
    for cat, c in cats.items():
        print("  %-8s  ✅ %2d   ❌ %2d   ⏭️ %2d" % (cat, c["PASS"], c["FAIL"], c["SKIP"]))
    total = {"PASS": 0, "FAIL": 0, "SKIP": 0}
    for _c, _n, st, _d in results:
        total[st] += 1
    print("-" * 56)
    print("  총계      ✅ %d   ❌ %d   ⏭️ %d  (전체 %d)" %
          (total["PASS"], total["FAIL"], total["SKIP"], len(results)))
    if total["FAIL"]:
        print("\n  ❌ 실패 항목:")
        for cat, n, st, d in results:
            if st == "FAIL":
                print("     - [%s] %s — %s" % (cat, n, d))
    return total["FAIL"]


def main():
    ap = argparse.ArgumentParser(description="Nexus AI 전체 기능 테스트")
    ap.add_argument("--setup", action="store_true", help="테스트용 실파일/샘플영상 생성")
    ap.add_argument("--run", action="store_true", help="--setup과 함께 쓰면 생성 후 바로 테스트")
    ap.add_argument("--only", default="", help="특정 카테고리만 (쉼표구분): 헬스,LLM,검색,...")
    args = ap.parse_args()

    if args.setup:
        setup_fixtures()
        if not args.run:
            return 0

    print("플랫폼: %s | 백엔드: %s | 사이드카: %s" % (platform.system(), GO_BASE, PY_BASE))
    if not backend_up():
        print("\n❌ Go 백엔드(127.0.0.1:17891)에 연결할 수 없습니다.")
        print("   설치된 Nexus 앱을 먼저 실행한 뒤 다시 시도하세요.")
        print("   (픽스처만 만들려면: python feature_test.py --setup)")
        return 2

    only = [c.strip() for c in args.only.split(",") if c.strip()]
    print()
    for name, fn in CATEGORIES.items():
        if only and name not in only:
            continue
        try:
            fn()
        except Exception as e:
            record(name, "(카테고리 실행 오류)", "FAIL", str(e))

    fails = summarize()
    return 1 if fails else 0


if __name__ == "__main__":
    sys.exit(main())
