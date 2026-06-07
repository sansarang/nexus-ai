package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func automationMux() *http.ServeMux {
	m := http.NewServeMux()
	registerAutomationRoutes(m)
	return m
}

func doJSON(t *testing.T, mux *http.ServeMux, method, path, body string) (int, map[string]any) {
	t.Helper()
	var r *http.Request
	if body != "" {
		r = httptest.NewRequest(method, path, strings.NewReader(body))
	} else {
		r = httptest.NewRequest(method, path, nil)
	}
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	var out map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	return w.Code, out
}

func TestAutomationStatusHandler(t *testing.T) {
	code, out := doJSON(t, automationMux(), "GET", "/api/automation/status", "")
	if code != 200 {
		t.Fatalf("status code = %d", code)
	}
	// 비-Windows(darwin) 단위 테스트 환경 → 엔진 미가용이어야 함.
	if out["available"] != false {
		t.Errorf("expected available=false, got %v", out["available"])
	}
	if p, _ := out["platform"].(string); p == "" {
		t.Errorf("platform missing")
	}
}

func TestAutomationRunDryRun(t *testing.T) {
	body := `{"steps":[{"kind":"click","selector":{"name":"a"}},{"kind":"wait","value":"100"}],"dry_run":true}`
	code, out := doJSON(t, automationMux(), "POST", "/api/automation/run", body)
	if code != 200 {
		t.Fatalf("dry_run code = %d", code)
	}
	if out["dry_run"] != true {
		t.Errorf("expected dry_run=true, got %v", out["dry_run"])
	}
	if c, _ := out["count"].(float64); c != 2 {
		t.Errorf("expected count=2, got %v", out["count"])
	}
}

func TestAutomationRunRefusedWhenUnavailable(t *testing.T) {
	// 엔진 미가용 시 실제 실행은 절대 안 되고 501로 거부돼야 한다 (안전 게이트).
	body := `{"steps":[{"kind":"click","selector":{"name":"a"}}]}`
	code, out := doJSON(t, automationMux(), "POST", "/api/automation/run", body)
	if code != http.StatusNotImplemented {
		t.Fatalf("expected 501 when engine unavailable, got %d", code)
	}
	if out["code"] != "automation_unavailable" {
		t.Errorf("expected code=automation_unavailable, got %v", out["code"])
	}
}

func TestAutomationRunNoStepsBadRequest(t *testing.T) {
	code, _ := doJSON(t, automationMux(), "POST", "/api/automation/run", `{}`)
	if code != 400 {
		t.Fatalf("expected 400 for empty request, got %d", code)
	}
}

func TestAutomationWorkflowSaveListGetReplay(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	mux := automationMux()

	// save
	code, out := doJSON(t, mux, "POST", "/api/automation/workflows",
		`{"name":"my-task","steps":[{"kind":"wait","value":"50"},{"kind":"verify","selector":{"name":"ok"},"expect":"ok"}]}`)
	if code != 200 || out["success"] != true {
		t.Fatalf("save failed: %d %v", code, out)
	}
	if s, _ := out["steps"].(float64); s != 2 {
		t.Errorf("expected 2 steps saved, got %v", out["steps"])
	}

	// list
	code, out = doJSON(t, mux, "GET", "/api/automation/workflows", "")
	if code != 200 {
		t.Fatalf("list code = %d", code)
	}
	wfs, _ := out["workflows"].([]any)
	if len(wfs) != 1 || wfs[0] != "my-task" {
		t.Fatalf("expected [my-task], got %v", out["workflows"])
	}

	// get by name (path param via mux)
	code, out = doJSON(t, mux, "GET", "/api/automation/workflows/my-task", "")
	if code != 200 || out["success"] != true {
		t.Fatalf("get by name failed: %d %v", code, out)
	}

	// replay → 미가용이므로 501 (무단 실행 금지)
	code, _ = doJSON(t, mux, "POST", "/api/automation/workflows/my-task/replay", `{}`)
	if code != http.StatusNotImplemented {
		t.Errorf("expected replay 501 when unavailable, got %d", code)
	}

	// get 없는 워크플로 → 404
	code, _ = doJSON(t, mux, "GET", "/api/automation/workflows/nope", "")
	if code != 404 {
		t.Errorf("expected 404 for missing workflow, got %d", code)
	}
}
