# Windows QA 런북 — 웹폼 반복입력 UIA PoC (GTM 7번 step 2~3)

> macOS에서는 **실행 자체가 불가능**한 부분입니다. 아래는 **Windows에서 사장님(또는 QA)이 직접** 실행해 "실제로 폼이 채워지는가 / 95% 신뢰성이 나오는가"를 확인하는 절차입니다.

## 0. 무엇을 증명하나
- **step 2**: pywinauto가 폼 요소를 *이름*으로 잡아 좌표 없이 입력한다.
- **step 3**: 입력→검증→실패 시 재시도(닫힌 루프)로 **N회 반복 성공률 95%+** 를 만든다.

## 1. 준비 (Windows 10/11)
```powershell
pip install pywinauto pyperclip pyautogui
```
- 대상 폼을 연다 (예: 사내 웹 양식, 회원가입 페이지, 데스크탑 입력 앱).
- 폼 필드의 **접근성 이름**을 확인한다. 모르면 `inspect.exe`(Windows SDK) 또는
  `python -c "from pywinauto import Desktop; Desktop(backend='uia').window(active_only=True).print_control_identifiers()"`.

## 2. 실행
```powershell
# 기본 샘플 3행, 20회 반복
python scripts\uia_poc_webform.py --target "회원가입" --runs 20

# 실제 데이터로
python scripts\uia_poc_webform.py --target "회원가입" --rows rows.json --runs 50 ^
  --field-name "이름" --field-email "이메일" --submit "제출" --result "결과"
```
`rows.json` 예:
```json
[
  {"name": "홍길동", "email": "hong@a.com", "status": "완료"},
  {"name": "김철수", "email": "kim@b.com",  "status": "완료"}
]
```

## 3. 기대 출력
```
  [1/20] 홍길동 — ✅
  [2/20] 김철수 — ✅
  ...
성공률: 19/20 = 95.0%
판정: ✅ PASS (기준 95%)
```

## 4. 판정 기준
- **성공률 ≥ 95% → step 2~3 합격.** 다음(step 4 녹화→재생 실연결, step 5 데모영상)으로.
- **< 95% → 다음으로 가지 않는다.** 아래 튜닝 후 재측정.

## 5. 실패 시 셀렉터/대기 튜닝
| 증상 | 원인 | 조치 |
|---|---|---|
| "요소 없음: 이름" | 접근성 이름 불일치 | `print_control_identifiers()`로 실제 이름 확인 후 `--field-*` 조정 |
| 입력은 되는데 검증 실패 | 결과 영역이 늦게 렌더 | `--retries` 증가, 결과 라벨/`status` 값 확인 |
| 가끔만 실패(전이적) | 렌더 타이밍 | 재시도(닫힌 루프)가 흡수 — `--retries 3`까지 |
| 전부 실패 | backend=uia 미지원 앱(구형 Win32) | `backend="win32"` 변형 필요 — 별도 검토 |

## 6. 앱 통합과의 관계
이 PoC가 합격하면, 동일 로직을 앱의 Python 사이드카(`/desktop/uia/*`, `nexus_python/main.py`)에서
`Available()=true`로 전환 → Go `windowsAutomator` → `RunSteps`/`BatchRun` 닫힌 루프가 그대로 사용한다.
오케스트레이션(placeholder 치환·N행 배치·재시도·성공률)은 이미 구현·검증됨(`automation_batch_test.go`).

## 7. 범위 밖 (이 런북에서 다루지 않음)
- **step 5 데모 영상**: 위 PoC가 PASS한 화면을 녹화 → 커뮤니티 배포. (Windows 화면 녹화 + 편집, 비코드)
- **step 6 커뮤니티 반응 / 돈길(인증·결제) 테스트**: 반응 측정 후 착수하는 사업·코드 활동.
- 이들은 "실제 동작하는 Windows 자동화"가 전제라, **본 PoC 합격이 선행 조건**입니다.
