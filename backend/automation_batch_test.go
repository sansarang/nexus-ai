package main

import "testing"

// webFormTemplate — "웹 폼 반복 입력" 유스케이스의 한 행 처리 템플릿(placeholder 포함).
func webFormTemplate() []AutoStep {
	return []AutoStep{
		{Kind: ActSetText, Selector: AutoSelector{Name: "이름", Role: "edit"}, Value: "{{name}}"},
		{Kind: ActSetText, Selector: AutoSelector{Name: "이메일", Role: "edit"}, Value: "{{email}}"},
		{Kind: ActClick, Selector: AutoSelector{Name: "제출", Role: "button"}},
		{Kind: ActVerify, Selector: AutoSelector{Name: "결과"}, Expect: "{{status}}"},
	}
}

func TestExpandPlaceholders(t *testing.T) {
	row := map[string]string{"name": "홍길동", "email": "a@b.com"}
	cases := map[string]string{
		"{{name}}":               "홍길동",
		"이름: {{name}} / {{email}}": "이름: 홍길동 / a@b.com",
		"{{missing}}":            "",       // 미존재 키 → 빈 문자열
		"plain":                  "plain",  // placeholder 없음
		"{{ name }}":             "홍길동",    // 공백 트림
		"{{unclosed":            "{{unclosed", // 닫힘 없음 → 원문
	}
	for in, want := range cases {
		if got := expandPlaceholders(in, row); got != want {
			t.Errorf("expandPlaceholders(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestExpandSteps_FillsTemplate(t *testing.T) {
	row := map[string]string{"name": "김철수", "email": "k@c.com", "status": "완료"}
	steps := expandSteps(webFormTemplate(), row)
	if steps[0].Value != "김철수" {
		t.Errorf("step0 value = %q", steps[0].Value)
	}
	if steps[1].Value != "k@c.com" {
		t.Errorf("step1 value = %q", steps[1].Value)
	}
	if steps[3].Expect != "완료" {
		t.Errorf("verify expect = %q", steps[3].Expect)
	}
	// 원본 템플릿은 변형되지 않아야 함(순수 함수).
	if webFormTemplate()[0].Value != "{{name}}" {
		t.Errorf("template mutated")
	}
}

func threeRows() []map[string]string {
	return []map[string]string{
		{"name": "A", "email": "a@x.com", "status": "완료"},
		{"name": "B", "email": "b@x.com", "status": "완료"},
		{"name": "C", "email": "c@x.com", "status": "완료"},
	}
}

func TestBatchRun_AllSucceed(t *testing.T) {
	m := &mockAutomator{available: true, verifyOK: true}
	res := BatchRun(m, webFormTemplate(), threeRows(), false)
	if res.Total != 3 || res.Succeeded != 3 {
		t.Fatalf("expected 3/3, got %d/%d", res.Succeeded, res.Total)
	}
	if res.SuccessRate != 1.0 {
		t.Errorf("expected rate 1.0, got %v", res.SuccessRate)
	}
}

// 핵심: 일시적 실패가 닫힌 루프 재시도로 복구되어 배치 전체 100% 유지 (신뢰성).
func TestBatchRun_TransientFailureRecovers(t *testing.T) {
	m := &mockAutomator{available: true, verifyOK: true, verifyFailFirst: 1} // 첫 verify 1회 실패
	res := BatchRun(m, webFormTemplate(), threeRows(), false)
	if res.SuccessRate != 1.0 {
		t.Fatalf("retry should recover transient failure → 100%%, got %v (%+v)", res.SuccessRate, res.Rows[0].Run.Steps)
	}
	// 첫 행의 verify 단계는 2회 시도(1실패+1성공)였어야 함.
	last := res.Rows[0].Run.Steps[len(res.Rows[0].Run.Steps)-1]
	if last.Attempts != 2 {
		t.Errorf("expected row0 verify 2 attempts, got %d", last.Attempts)
	}
}

// 혼합: 특정 행만 영구 실패 → 성공률이 정확히 집계되고, 나머지 행은 계속 진행.
func TestBatchRun_MixedSuccessRate(t *testing.T) {
	rows := []map[string]string{
		{"name": "A", "status": "완료"},
		{"name": "B", "status": "BAD"}, // 이 행만 실패
		{"name": "C", "status": "완료"},
	}
	m := &mockAutomator{available: true, verifyOK: true, failVerifyIfExpect: "BAD"}
	res := BatchRun(m, webFormTemplate(), rows, false) // stopOnError=false → 전 행 시도
	if res.Total != 3 || res.Succeeded != 2 {
		t.Fatalf("expected 2/3, got %d/%d", res.Succeeded, res.Total)
	}
	if res.Rows[1].OK {
		t.Errorf("row B should have failed")
	}
	if !res.Rows[2].OK {
		t.Errorf("row C should still run and succeed after B failed (stopOnError=false)")
	}
}

func TestBatchRun_StopOnError(t *testing.T) {
	// stopOnError=true: 첫 실패 행에서 중단(이후 행 미실행). Total은 전체 행수 유지.
	rows := []map[string]string{
		{"name": "A", "status": "완료"},
		{"name": "B", "status": "BAD"}, // 실패 → 여기서 중단
		{"name": "C", "status": "완료"}, // 실행되지 않아야 함
	}
	m := &mockAutomator{available: true, verifyOK: true, failVerifyIfExpect: "BAD"}
	res := BatchRun(m, webFormTemplate(), rows, true)
	if res.Total != 3 {
		t.Errorf("Total should remain 3 (행수), got %d", res.Total)
	}
	if len(res.Rows) != 2 {
		t.Errorf("expected stop after 2 rows (A ok, B fail), got %d", len(res.Rows))
	}
	if res.Succeeded != 1 {
		t.Errorf("expected 1 success before stop, got %d", res.Succeeded)
	}
}

func TestBatchRun_RefusedWhenUnavailable(t *testing.T) {
	m := &mockAutomator{available: false}
	res := BatchRun(m, webFormTemplate(), threeRows(), false)
	if len(res.Rows) != 0 || res.SuccessRate != 0 {
		t.Fatalf("unavailable engine must run 0 rows, got %d rows rate %v", len(res.Rows), res.SuccessRate)
	}
}
