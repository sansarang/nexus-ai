package main

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

// mockAutomator — 닫힌 루프 로직을 플랫폼 없이 검증하기 위한 가짜 구현.
type mockAutomator struct {
	available       bool
	findErr         error
	clickErr        error
	setErr          error
	keyErr          error
	verifyOK           bool
	verifyErr          error
	verifyFailFirst    int    // 첫 N회 verify를 false로 (재시도 검증용)
	failVerifyIfExpect string // 이 expect 값이면 항상 false (특정 행 영구 실패 시뮬)

	findCalls, clickCalls, setCalls, keyCalls, verifyCalls int
}

func (m *mockAutomator) Available() bool { return m.available }
func (m *mockAutomator) FindElement(sel AutoSelector) (AutoElement, error) {
	m.findCalls++
	if m.findErr != nil {
		return AutoElement{}, m.findErr
	}
	return AutoElement{Found: true, Name: sel.Name, Role: sel.Role}, nil
}
func (m *mockAutomator) Click(_ AutoElement) error            { m.clickCalls++; return m.clickErr }
func (m *mockAutomator) SetText(_ AutoElement, _ string) error { m.setCalls++; return m.setErr }
func (m *mockAutomator) SendKeys(_ string) error              { m.keyCalls++; return m.keyErr }
func (m *mockAutomator) Verify(_ AutoSelector, expect string) (bool, error) {
	m.verifyCalls++
	if m.verifyErr != nil {
		return false, m.verifyErr
	}
	if m.failVerifyIfExpect != "" && expect == m.failVerifyIfExpect {
		return false, nil // 특정 행 영구 실패
	}
	if m.verifyFailFirst > 0 {
		m.verifyFailFirst--
		return false, nil
	}
	return m.verifyOK, nil
}

func TestRunSteps_HappyPath(t *testing.T) {
	m := &mockAutomator{available: true, verifyOK: true}
	steps := []AutoStep{
		{Kind: ActFind, Selector: AutoSelector{Name: "로그인", Role: "button"}},
		{Kind: ActSetText, Selector: AutoSelector{Name: "이메일", Role: "edit"}, Value: "a@b.com"},
		{Kind: ActClick, Selector: AutoSelector{Name: "로그인", Role: "button"}, Expect: "환영"},
	}
	res := RunSteps(m, steps)
	if !res.OK {
		t.Fatalf("expected OK, got %+v", res)
	}
	if len(res.Steps) != 3 {
		t.Fatalf("expected 3 step results, got %d", len(res.Steps))
	}
	for i, sr := range res.Steps {
		if !sr.OK {
			t.Errorf("step %d not OK: %s", i, sr.Error)
		}
	}
	if m.setCalls != 1 || m.clickCalls != 1 {
		t.Errorf("expected 1 set + 1 click, got set=%d click=%d", m.setCalls, m.clickCalls)
	}
	// click 단계에 Expect가 있으므로 verify가 한 번 호출돼야 함.
	if m.verifyCalls != 1 {
		t.Errorf("expected 1 verify (click had Expect), got %d", m.verifyCalls)
	}
}

func TestRunSteps_VerifyRetriesThenFails(t *testing.T) {
	m := &mockAutomator{available: true, verifyOK: false}
	steps := []AutoStep{{Kind: ActVerify, Selector: AutoSelector{Name: "성공"}, Expect: "성공"}}
	res := RunSteps(m, steps)
	if res.OK {
		t.Fatalf("expected failure when verify never passes")
	}
	if got := res.Steps[0].Attempts; got != autoMaxRetries+1 {
		t.Errorf("expected %d attempts, got %d", autoMaxRetries+1, got)
	}
	if res.Steps[0].Error != "verify failed" {
		t.Errorf("expected 'verify failed', got %q", res.Steps[0].Error)
	}
}

func TestRunSteps_RecoversAfterRetry(t *testing.T) {
	// 첫 verify는 false, 두 번째 true → 재시도로 복구되어야 함.
	m := &mockAutomator{available: true, verifyOK: true, verifyFailFirst: 1}
	steps := []AutoStep{{Kind: ActVerify, Selector: AutoSelector{Name: "x"}, Expect: "x"}}
	res := RunSteps(m, steps)
	if !res.OK {
		t.Fatalf("expected recovery after retry, got %+v", res.Steps[0])
	}
	if !res.Steps[0].Verified {
		t.Errorf("expected Verified=true")
	}
	if res.Steps[0].Attempts != 2 {
		t.Errorf("expected 2 attempts, got %d", res.Steps[0].Attempts)
	}
}

func TestRunSteps_UnavailableAutomator(t *testing.T) {
	m := &mockAutomator{available: false}
	res := RunSteps(m, []AutoStep{{Kind: ActClick}})
	if res.OK {
		t.Fatalf("expected OK=false for unavailable automator")
	}
	if !strings.Contains(res.Steps[0].Error, "not yet implemented") {
		t.Errorf("expected not-implemented error, got %q", res.Steps[0].Error)
	}
}

func TestRunSteps_FindErrorRetried(t *testing.T) {
	m := &mockAutomator{available: true, findErr: errors.New("element not found")}
	res := RunSteps(m, []AutoStep{{Kind: ActClick, Selector: AutoSelector{Name: "없는버튼"}}})
	if res.OK {
		t.Fatalf("expected failure on find error")
	}
	if res.Steps[0].Attempts != autoMaxRetries+1 {
		t.Errorf("expected %d attempts, got %d", autoMaxRetries+1, res.Steps[0].Attempts)
	}
	if !strings.Contains(res.Steps[0].Error, "element not found") {
		t.Errorf("expected find error surfaced, got %q", res.Steps[0].Error)
	}
}

func TestRunSteps_UnknownStep(t *testing.T) {
	m := &mockAutomator{available: true}
	res := RunSteps(m, []AutoStep{{Kind: AutoActionKind("bogus")}})
	if res.OK {
		t.Fatalf("expected failure on unknown step kind")
	}
	if res.Steps[0].Error != ErrAutomationUnknownStep.Error() {
		t.Errorf("expected unknown-step error, got %q", res.Steps[0].Error)
	}
}

func TestRunSteps_StopsAtFirstFailure(t *testing.T) {
	// 2번째 단계 실패 시 3번째는 실행되지 않아야 함 (잘못된 자동화 폭주 방지).
	m := &mockAutomator{available: true, verifyOK: true, clickErr: errors.New("boom")}
	steps := []AutoStep{
		{Kind: ActFind, Selector: AutoSelector{Name: "a"}},
		{Kind: ActClick, Selector: AutoSelector{Name: "b"}},
		{Kind: ActSetText, Selector: AutoSelector{Name: "c"}, Value: "x"},
	}
	res := RunSteps(m, steps)
	if res.OK {
		t.Fatalf("expected overall failure")
	}
	if len(res.Steps) != 2 {
		t.Errorf("expected to stop after 2 steps, got %d", len(res.Steps))
	}
	if m.setCalls != 0 {
		t.Errorf("3rd step (set_text) must not run after failure, setCalls=%d", m.setCalls)
	}
}

func TestWorkflow_MarshalRoundTrip(t *testing.T) {
	wf := NewAutoWorkflow("로그인 자동화", []AutoStep{
		{Kind: ActSetText, Selector: AutoSelector{Name: "이메일"}, Value: "a@b.com"},
		{Kind: ActClick, Selector: AutoSelector{Name: "로그인", Role: "button"}, Expect: "환영"},
	})
	data, err := wf.Marshal()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got, err := UnmarshalAutoWorkflow(data)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Name != wf.Name || got.Version != autoWorkflowSchemaVersion {
		t.Errorf("name/version mismatch: %+v", got)
	}
	if !reflect.DeepEqual(got.Steps, wf.Steps) {
		t.Errorf("steps not preserved:\n got=%+v\nwant=%+v", got.Steps, wf.Steps)
	}
}

func TestWorkflow_SaveLoadList(t *testing.T) {
	t.Setenv("HOME", t.TempDir()) // os.UserHomeDir → 임시 디렉터리
	wf := NewAutoWorkflow("내 작업 1", []AutoStep{{Kind: ActWait, Value: "500"}})
	path, err := wf.Save()
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	if !strings.HasSuffix(path, ".json") {
		t.Errorf("unexpected save path %q", path)
	}
	got, err := LoadAutoWorkflow("내 작업 1")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if !reflect.DeepEqual(got.Steps, wf.Steps) {
		t.Errorf("loaded steps mismatch: %+v", got.Steps)
	}
	names, err := ListAutoWorkflows()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(names) != 1 {
		t.Errorf("expected 1 workflow listed, got %v", names)
	}
}

func TestSanitizeWorkflowName_BlocksTraversal(t *testing.T) {
	out := sanitizeWorkflowName("../../etc/passwd")
	if strings.Contains(out, "/") || strings.Contains(out, "..") {
		t.Errorf("path traversal not sanitized: %q", out)
	}
	if out == "" {
		t.Errorf("sanitized name must not be empty")
	}
	if sanitizeWorkflowName("   ") == "" {
		t.Errorf("blank name should fall back to default, got empty")
	}
}

func TestReplay_RefusedWhenEngineUnavailable(t *testing.T) {
	// 현재 어떤 플랫폼에서도 자동화 엔진 미완성(Available=false) → Replay는 안전하게 거부.
	wf := NewAutoWorkflow("x", []AutoStep{{Kind: ActWait, Value: "10"}})
	res := wf.Replay()
	if res.OK {
		t.Fatalf("expected Replay to be refused while engine unavailable")
	}
}
