// automation_workflow.go — 녹화 → 재생 워크플로 데이터 모델 (cross-platform)
//
// "한 번 가르치면 알아서 반복" = RPA 해자. 사용자가 시연/녹화한 자동화 시퀀스를
// AutoWorkflow(=[]AutoStep)로 저장하고, Replay()로 닫힌 루프 재생한다.
// 실제 입력(액션)은 GetAutomator()(플랫폼 구현)로 위임 → 데이터모델/재생순서는
// macOS에서 단위 검증 가능, Windows UIA 실행만 런타임 의존.
package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const autoWorkflowSchemaVersion = 1

// AutoWorkflow — 재사용·재생 단위의 자동화 시퀀스.
type AutoWorkflow struct {
	Name      string     `json:"name"`
	Steps     []AutoStep `json:"steps"`
	CreatedAt int64      `json:"created_at"`
	Version   int        `json:"version"`
}

// NewAutoWorkflow — 녹화 결과로 워크플로 생성.
func NewAutoWorkflow(name string, steps []AutoStep) AutoWorkflow {
	return AutoWorkflow{
		Name:      name,
		Steps:     steps,
		CreatedAt: time.Now().Unix(),
		Version:   autoWorkflowSchemaVersion,
	}
}

// Marshal / UnmarshalAutoWorkflow — 녹화 포맷 직렬화(저장/전송 공용).
func (wf AutoWorkflow) Marshal() ([]byte, error) { return json.MarshalIndent(wf, "", "  ") }

func UnmarshalAutoWorkflow(data []byte) (AutoWorkflow, error) {
	var wf AutoWorkflow
	err := json.Unmarshal(data, &wf)
	return wf, err
}

// sanitizeWorkflowName — 파일명 안전화 (경로 순회/특수문자 차단).
func sanitizeWorkflowName(name string) string {
	name = strings.TrimSpace(name)
	repl := func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			return r
		case r == '-' || r == '_':
			return r
		// 한글 등 유니코드 글자는 허용 (파일명 가능), 그 외 구분자류는 '_'
		case r > 0x7F:
			return r
		default:
			return '_'
		}
	}
	out := strings.Map(repl, name)
	out = strings.Trim(out, "_")
	if out == "" {
		out = "workflow"
	}
	if len(out) > 80 {
		out = out[:80]
	}
	return out
}

func autoWorkflowDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".nexus", "automations")
}

// Save — ~/.nexus/automations/<name>.json 으로 저장, 저장 경로 반환.
func (wf AutoWorkflow) Save() (string, error) {
	dir := autoWorkflowDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, sanitizeWorkflowName(wf.Name)+".json")
	data, err := wf.Marshal()
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return "", err
	}
	return path, nil
}

// LoadAutoWorkflow — 이름으로 저장된 워크플로 로드.
func LoadAutoWorkflow(name string) (AutoWorkflow, error) {
	path := filepath.Join(autoWorkflowDir(), sanitizeWorkflowName(name)+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return AutoWorkflow{}, err
	}
	return UnmarshalAutoWorkflow(data)
}

// ListAutoWorkflows — 저장된 워크플로 이름 목록.
func ListAutoWorkflows() ([]string, error) {
	entries, err := os.ReadDir(autoWorkflowDir())
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.HasSuffix(e.Name(), ".json") {
			names = append(names, strings.TrimSuffix(e.Name(), ".json"))
		}
	}
	return names, nil
}

// Replay — 저장된 워크플로를 닫힌 루프로 재생 (실제 액션은 플랫폼 자동화로 위임).
// non-Windows / UIA 미완성 환경에서는 RunResult.OK=false + not-implemented로 안전하게 거부.
func (wf AutoWorkflow) Replay() RunResult {
	return RunSteps(GetAutomator(), wf.Steps)
}
