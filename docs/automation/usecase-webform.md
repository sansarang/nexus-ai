# 유스케이스 #1 — 엑셀 N행 → 폼 반복 입력 (Web Form Repeat-Fill)

> 데스크탑 자동화 전문가 피벗의 **첫 번째 wedge**. "가장 아픈 것 하나"로 선정.

## 1. 왜 이것인가 (고통)

사무직·자영업자·운영팀이 **같은 폼에 수십~수백 행을 손으로 입력**한다:

- 관공서/공공 포털 신청·신고 (반복 양식)
- CRM / ERP / 재고 / 주문 입력
- 엑셀에 모아둔 데이터를 레거시 웹·데스크탑 앱에 옮겨치기

특징: **반복적 · 실수 잦음 · 시간 소모 · PC에 갇힘**. ChatGPT/Copilot은 "어떻게 하라"고 말해줄 뿐, **대신 입력해주지는 못한다.** 우리가 이긴다.

## 2. 타깃

- 🇰🇷 자영업/중소기업 사무, 관공서 반복 신고, 엑셀 노가다
- 🇺🇸 data-entry / back-office / "personal RPA" 수요 (r/automation 등)

## 3. 성공 기준 (이게 제품의 전부)

- **N행 중 95%+** 가 자동으로 입력 → 제출 → 검증 성공
- **행당 사람 개입 0** (시작 한 번 + 끝 확인)
- 실패한 행은 **명확히 표시** (어느 행, 어느 단계, 왜) — 사람이 그것만 처리

## 4. 한 행 처리 = AutoStep 템플릿

좌표가 아닌 **요소 이름(접근성)** 으로 타겟팅. `{{col}}` 은 행 데이터로 치환.

```json
{
  "name": "회원가입 폼 반복",
  "steps": [
    { "kind": "set_text", "selector": { "name": "이름", "role": "edit" },  "value": "{{name}}" },
    { "kind": "set_text", "selector": { "name": "이메일", "role": "edit" }, "value": "{{email}}" },
    { "kind": "set_text", "selector": { "name": "전화", "role": "edit" },   "value": "{{phone}}" },
    { "kind": "click",    "selector": { "name": "제출", "role": "button" } },
    { "kind": "verify",   "selector": { "name": "결과" }, "expect": "{{status}}" }
  ]
}
```

샘플 데이터(rows):

```json
[
  { "name": "홍길동", "email": "hong@a.com", "phone": "010-1111-2222", "status": "완료" },
  { "name": "김철수", "email": "kim@b.com",  "phone": "010-3333-4444", "status": "완료" },
  { "name": "이영희", "email": "lee@c.com",  "phone": "010-5555-6666", "status": "완료" }
]
```

## 5. 닫힌 루프 + 배치 (신뢰성)

- **닫힌 루프** (`RunSteps`, `automation_core.go`): 각 단계 실행 → `verify`로 "진짜 됐나?" 확인 → 실패 시 **최대 3회 재시도**. 한 단계 실패하면 그 행은 중단(잘못된 자동화 폭주 방지).
- **배치** (`BatchRun`, `automation_batch.go`): 템플릿을 행별로 확장(`expandSteps`)해 실행하고 **행별 성공/실패 + 전체 successRate** 집계.
- **안전 게이트**: 엔진(`GetAutomator().Available()`)이 준비 안 됐으면 **아무 것도 실행 안 함**.

## 6. 검증 경계 (정직)

| 부분 | 상태 | 검증 |
|---|---|---|
| placeholder 치환 / 배치 집계 / 재시도 오케스트레이션 | ✅ 구현·검증 | `automation_batch_test.go` (mock으로 전이실패→재시도→100% 등) |
| HTTP 라우트 / 안전 게이트(501) / 워크플로 저장·재생 | ✅ 구현·검증 | `handlers_automation_test.go` + 라이브 |
| **pywinauto 실제 폼 클릭/입력 동작** | ⚠️ **Windows QA 필요** | `scripts/uia_poc_webform.py` 를 Windows에서 실행 (런북 참고) |
| **실제 폼 95% 신뢰성 실측** | ⚠️ **Windows QA 필요** | PoC 스크립트의 N회 반복 성공률 출력 |
| 데모 영상 / 커뮤니티 반응 | ⛔ 범위 외(비코드 + Windows 영상) | — |

→ macOS에서 만들 수 있는 **오케스트레이션·안전·데이터모델**은 전부 섰다.
   남은 건 **실제 Windows 폼에서의 UIA 동작 검증**뿐이며, 이는 `windows-qa-runbook.md` 로 패키징했다.
