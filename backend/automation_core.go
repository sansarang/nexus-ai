// automation_core.go — 데스크탑 자동화 엔진 코어 (cross-platform 계약)
//
// 피벗 핵심: "말하는 AI"가 아니라 "실제로 PC를 조작하는 AI".
// 신뢰성의 열쇠 = 픽셀 좌표가 아닌 UI Automation(접근성 트리) 기반 '의미' 타겟팅 +
//
//	perceive → act → verify → retry 닫힌 루프.
//
// ⚠️ 실제 Windows UIA 구현(pywinauto/uiautomation 연동)은 Windows 런타임에서 완성한다.
//
//	이 파일은 플랫폼 독립 계약/타입/닫힌 루프 실행기만 정의한다 (macOS에서 단위 검증 가능).
package main

import (
	"errors"
	"strconv"
	"strings"
	"time"
)

var (
	// ErrAutomationNotImplemented — Windows UIA 엔진 미완성(런타임 연동 대기).
	ErrAutomationNotImplemented = errors.New("automation: UIA engine not yet implemented (Windows runtime pending)")
	// ErrAutomationUnsupported — 비-Windows 플랫폼.
	ErrAutomationUnsupported = errors.New("automation: not supported on this platform")
	// ErrAutomationUnknownStep — 알 수 없는 단계 종류.
	ErrAutomationUnknownStep = errors.New("automation: unknown step kind")
)

// AutoSelector — 좌표가 아닌 '의미'로 요소를 지목 (레이아웃/해상도 변경에 강함).
type AutoSelector struct {
	Name         string `json:"name,omitempty"`          // 접근성 이름 ("로그인", "이메일")
	Role         string `json:"role,omitempty"`          // 컨트롤 타입 ("button","edit","checkbox")
	AutomationID string `json:"automation_id,omitempty"` // UIA AutomationId
	Index        int    `json:"index,omitempty"`         // 동일 매칭 중 N번째 (0-base)
	Window       string `json:"window,omitempty"`        // 최상위 창 제목 힌트 (멀티윈도우/팝업 재생용, 부분 일치)
}

// AutoActionKind — 한 단계가 수행할 동작.
type AutoActionKind string

const (
	ActFind        AutoActionKind = "find"
	ActClick       AutoActionKind = "click"
	ActDoubleClick AutoActionKind = "double_click"
	ActRightClick  AutoActionKind = "right_click"
	ActSetText     AutoActionKind = "set_text"
	ActKey         AutoActionKind = "key"
	ActScrollStep  AutoActionKind = "scroll"
	ActWait        AutoActionKind = "wait"
	ActVerify      AutoActionKind = "verify"
)

// AutoStep — 녹화/재생되는 단일 자동화 단계 (JSON 직렬화 = 녹화 포맷).
type AutoStep struct {
	Kind     AutoActionKind `json:"kind"`
	Selector AutoSelector   `json:"selector,omitempty"`
	Value    string         `json:"value,omitempty"`  // set_text 값 / key 조합 / wait(ms)
	Expect   string         `json:"expect,omitempty"` // verify 기대값(부분 일치)
}

// AutoElement — 찾은 요소 핸들 (플랫폼 구현이 채움).
type AutoElement struct {
	Found  bool
	Name   string
	Role   string
	handle any // 플랫폼 네이티브 핸들 (UIA element 등) — 코어는 들여다보지 않음
}

// UIAutomator — 신뢰성 있는 데스크탑 자동화 계약.
// 구현: windowsAutomator(UIA, 실구현 예정) / unsupportedAutomator(non-windows).
type UIAutomator interface {
	FindElement(sel AutoSelector) (AutoElement, error)
	Click(el AutoElement) error
	DoubleClick(el AutoElement) error
	RightClick(el AutoElement) error
	SetText(el AutoElement, text string) error
	SendKeys(combo string) error
	// Scroll — 휠 스크롤 (amount: 양수=위, 음수=아래; selector 비면 현재 위치).
	Scroll(sel AutoSelector, amount int) error
	// Verify — 액션 후 기대 상태 확인 (닫힌 루프의 핵심: "진짜 됐나?").
	Verify(sel AutoSelector, expect string) (bool, error)
	// Available — 실제 동작 가능 여부 (스텁은 false).
	Available() bool
}

// GetAutomator — 플랫폼별 자동화 구현 반환 (factory).
// newPlatformAutomator는 빌드태그별 파일(_windows / _stub)에 정의.
func GetAutomator() UIAutomator { return newPlatformAutomator() }

// StepResult / RunResult — 닫힌 루프 실행 결과.
type StepResult struct {
	Step     AutoStep `json:"step"`
	OK       bool     `json:"ok"`
	Verified bool     `json:"verified"`
	Attempts int      `json:"attempts"`
	Error    string   `json:"error,omitempty"`
}

type RunResult struct {
	OK    bool         `json:"ok"`
	Steps []StepResult `json:"steps"`
}

// autoMaxRetries — verify/액션 실패 시 추가 재시도 횟수 (self-heal 훅 자리).
const autoMaxRetries = 2

// autoRetryBackoff — 재시도 사이 대기 (페이지 로딩/렌더 지연 흡수). 테스트는 0으로 재정의.
var autoRetryBackoff = 700 * time.Millisecond

// autoMaxWaitMs — wait 단계 상한 (폭주 방지).
const autoMaxWaitMs = 15000

// RunSteps — perceive → act → verify → retry 닫힌 루프 실행기 (플랫폼 독립).
// 실제 액션은 주입된 UIAutomator로 위임 → 코어 로직은 macOS에서도 mock으로 검증 가능.
func RunSteps(a UIAutomator, steps []AutoStep) RunResult {
	if a == nil || !a.Available() {
		return RunResult{OK: false, Steps: []StepResult{{OK: false, Error: ErrAutomationNotImplemented.Error()}}}
	}
	res := RunResult{OK: true}
	for _, st := range steps {
		sr := runOneStep(a, st)
		res.Steps = append(res.Steps, sr)
		if !sr.OK {
			res.OK = false
			break // 한 단계 실패 시 중단 (잘못된 자동화 계속 진행 방지)
		}
	}
	return res
}

func runOneStep(a UIAutomator, st AutoStep) StepResult {
	sr := StepResult{Step: st}
	for attempt := 1; attempt <= autoMaxRetries+1; attempt++ {
		sr.Attempts = attempt
		if attempt > 1 && autoRetryBackoff > 0 {
			time.Sleep(autoRetryBackoff) // 로딩/렌더 지연 흡수 후 재시도
		}
		if err := execStep(a, st); err != nil {
			sr.Error = err.Error()
			continue // 재시도 (향후 self-heal: 셀렉터 보정/대기 후 재시도)
		}
		// verify 단계이거나 expect가 지정되면 기대 상태 확인.
		if st.Kind == ActVerify || st.Expect != "" {
			ok, vErr := a.Verify(st.Selector, st.Expect)
			if vErr != nil {
				sr.Error = vErr.Error()
				continue
			}
			sr.Verified = ok
			if !ok {
				sr.Error = "verify failed"
				continue
			}
		}
		sr.OK = true
		sr.Error = ""
		return sr
	}
	return sr
}

func execStep(a UIAutomator, st AutoStep) error {
	switch st.Kind {
	case ActWait:
		// Value = 대기 ms (녹화 시 사용자 행동 간격에서 캡처). 상한으로 폭주 방지.
		ms, _ := strconv.Atoi(strings.TrimSpace(st.Value))
		if ms > autoMaxWaitMs {
			ms = autoMaxWaitMs
		}
		if ms > 0 {
			time.Sleep(time.Duration(ms) * time.Millisecond)
		}
		return nil
	case ActKey:
		return a.SendKeys(st.Value)
	case ActScrollStep:
		amt, _ := strconv.Atoi(strings.TrimSpace(st.Value))
		if amt == 0 {
			return nil
		}
		return a.Scroll(st.Selector, amt)
	case ActFind, ActVerify:
		_, err := a.FindElement(st.Selector)
		return err
	case ActClick:
		el, err := a.FindElement(st.Selector)
		if err != nil {
			return err
		}
		return a.Click(el)
	case ActDoubleClick:
		el, err := a.FindElement(st.Selector)
		if err != nil {
			return err
		}
		return a.DoubleClick(el)
	case ActRightClick:
		el, err := a.FindElement(st.Selector)
		if err != nil {
			return err
		}
		return a.RightClick(el)
	case ActSetText:
		el, err := a.FindElement(st.Selector)
		if err != nil {
			return err
		}
		return a.SetText(el, st.Value)
	default:
		return ErrAutomationUnknownStep
	}
}
