//go:build windows

package main

// windowsAutomator — Windows UI Automation(접근성 트리) 기반 자동화 (실구현 예정).
//
// 향후 연동 지점(택1):
//   (A) Python 사이드카에 /desktop/uia/* 엔드포인트 추가 → pywinauto / uiautomation 사용,
//       callPython()으로 호출. (기존 사이드카 인프라 재사용 — 권장)
//   (B) Go UIA 바인딩(go-ole + UIAutomationCore) 직접 사용.
//
// 현재는 명시적 미구현 상태로 둔다 (무테스트 배포 회피 원칙).
// Available()이 false인 동안 RunSteps는 ErrAutomationNotImplemented로 안전하게 단락된다.
type windowsAutomator struct{}

func newPlatformAutomator() UIAutomator { return &windowsAutomator{} }

// Available — UIA 연동 완료 후 true로 전환 (그 전까지 자동화 실행은 안전하게 거부).
func (w *windowsAutomator) Available() bool { return false }

func (w *windowsAutomator) FindElement(_ AutoSelector) (AutoElement, error) {
	return AutoElement{}, ErrAutomationNotImplemented
}
func (w *windowsAutomator) Click(_ AutoElement) error              { return ErrAutomationNotImplemented }
func (w *windowsAutomator) SetText(_ AutoElement, _ string) error  { return ErrAutomationNotImplemented }
func (w *windowsAutomator) SendKeys(_ string) error                { return ErrAutomationNotImplemented }
func (w *windowsAutomator) Verify(_ AutoSelector, _ string) (bool, error) {
	return false, ErrAutomationNotImplemented
}
