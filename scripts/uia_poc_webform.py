#!/usr/bin/env python3
"""
uia_poc_webform.py — 웹/데스크탑 폼 반복 입력 UIA PoC (Windows 전용)

GTM 7번 step 2~3 검증 도구:
  - pywinauto로 폼 요소를 '이름(접근성)'으로 잡고(좌표 X) 입력
  - verify로 "진짜 됐나?" 확인 → 실패 시 재시도 (닫힌 루프)
  - N회 반복 성공률을 측정해 95% 기준으로 PASS/FAIL 판정

⚠️ Windows에서 대상 폼(브라우저/데스크탑 앱)을 띄운 뒤 실행해야 한다.
   Mac/Linux에서는 안내만 출력하고 종료한다(실행 불가).

사용법:
  pip install pywinauto pyperclip pyautogui
  python uia_poc_webform.py --target "윈도우 제목 일부" --rows rows.json --runs 20

자세한 절차/판정 기준: docs/automation/windows-qa-runbook.md
"""
import argparse
import json
import sys
import time


def _load_rows(args):
    if args.rows:
        with open(args.rows, encoding="utf-8") as f:
            rows = json.load(f)
    else:
        rows = [
            {"name": "홍길동", "email": "hong@a.com", "status": "완료"},
            {"name": "김철수", "email": "kim@b.com", "status": "완료"},
            {"name": "이영희", "email": "lee@c.com", "status": "완료"},
        ]
    if args.runs and args.runs > len(rows):  # --runs 만큼 데이터 반복 확장
        out = []
        while len(out) < args.runs:
            out.extend(rows)
        return out[:args.runs]
    return rows


def main():
    ap = argparse.ArgumentParser(description="웹/데스크탑 폼 반복 입력 UIA PoC")
    ap.add_argument("--target", default="", help="대상 윈도우 제목(부분 일치). 미지정 시 포그라운드.")
    ap.add_argument("--rows", default="", help="입력 데이터 JSON 경로. 미지정 시 내장 샘플.")
    ap.add_argument("--runs", type=int, default=10, help="반복 횟수(데이터 자동 확장)")
    ap.add_argument("--retries", type=int, default=2, help="단계당 추가 재시도 횟수")
    ap.add_argument("--field-name", default="이름", help="이름 입력칸의 접근성 이름")
    ap.add_argument("--field-email", default="이메일", help="이메일 입력칸의 접근성 이름")
    ap.add_argument("--submit", default="제출", help="제출 버튼의 접근성 이름")
    ap.add_argument("--result", default="결과", help="결과/성공 영역의 접근성 이름")
    args = ap.parse_args()

    # ── 플랫폼/의존성 가드 (Mac에서도 안전하게 종료) ──
    import platform
    if platform.system() != "Windows":
        print("⚠️ 이 PoC는 Windows 전용입니다 (현재: %s)." % platform.system())
        print("   Windows에서 대상 폼을 띄운 뒤 실행하세요.")
        print("   런북: docs/automation/windows-qa-runbook.md")
        return 2
    try:
        from pywinauto import Desktop
    except Exception as e:  # pragma: no cover - Windows 런타임 전용
        print("❌ pywinauto 미설치 → pip install pywinauto pyperclip pyautogui")
        print("   원인:", e)
        return 3

    rows = _load_rows(args)
    if not rows:
        print("입력 데이터가 없습니다.")
        return 1

    desktop = Desktop(backend="uia")

    def win():
        if args.target:
            return desktop.window(title_re=".*%s.*" % args.target)
        return desktop.window(active_only=True)

    def find(label):
        # 이름 '부분 일치'로 컨트롤 검색 (좌표 사용 안 함 → 레이아웃 변화에 강함)
        try:
            for c in win().descendants():
                try:
                    if label in (c.window_text() or ""):
                        return c
                except Exception:
                    continue
        except Exception:
            return None
        return None

    def set_text(label, value):
        el = find(label)
        if el is None:
            return False, "요소 없음: %s" % label
        try:
            el.set_focus()
            try:  # 한글/유니코드 안전 → 클립보드 경유
                import pyperclip
                import pyautogui
                pyperclip.copy(value)
                pyautogui.hotkey("ctrl", "a")
                pyautogui.hotkey("ctrl", "v")
            except Exception:
                el.type_keys(value, with_spaces=True)
            return True, ""
        except Exception as e:
            return False, str(e)

    def click(label):
        el = find(label)
        if el is None:
            return False, "버튼 없음: %s" % label
        try:
            el.click_input()
            return True, ""
        except Exception as e:
            return False, str(e)

    def verify(label, expect):
        el = find(label)
        if el is None:
            return False
        txt = el.window_text() or ""
        return (expect in txt) if expect else True

    def with_retry(fn):
        last = ""
        for _ in range(args.retries + 1):  # 닫힌 루프 재시도
            ok, err = fn()
            if ok:
                return True, ""
            last = err
            time.sleep(0.4)
        return False, last

    succeeded = 0
    for i, row in enumerate(rows):
        ok, why = True, ""
        for label, key in ((args.field_name, "name"), (args.field_email, "email")):
            s, e = with_retry(lambda lbl=label, k=key: set_text(lbl, row.get(k, "")))
            if not s:
                ok, why = False, "%s 입력 실패: %s" % (label, e)
                break
        if ok:
            s, e = with_retry(lambda: click(args.submit))
            if not s:
                ok, why = False, "제출 실패: %s" % e
        if ok:
            vok = False
            for _ in range(args.retries + 1):  # verify 재시도
                if verify(args.result, row.get("status", "")):
                    vok = True
                    break
                time.sleep(0.4)
            if not vok:
                ok, why = False, "검증 실패(결과 미확인)"
        print("  [%d/%d] %s — %s" % (i + 1, len(rows), row.get("name", ""), "✅" if ok else "❌ " + why))
        if ok:
            succeeded += 1
        time.sleep(0.3)

    rate = succeeded / len(rows) * 100.0
    print("\n성공률: %d/%d = %.1f%%" % (succeeded, len(rows), rate))
    print("판정: %s (기준 95%%)" % ("✅ PASS" if rate >= 95 else "❌ FAIL — 셀렉터/대기 튜닝 필요(런북 참고)"))
    return 0 if rate >= 95 else 4


if __name__ == "__main__":
    sys.exit(main())
