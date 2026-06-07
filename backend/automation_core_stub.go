//go:build !windows

package main

// unsupportedAutomator — 비-Windows(개발/Mac) 빌드용 스텁.
// 데스크탑 UIA 자동화는 Windows 전용이므로 이 플랫폼에서는 항상 거부한다.
// (코어의 RunSteps 로직 자체는 mock UIAutomator로 macOS에서 단위 검증 가능.)
type unsupportedAutomator struct{}

func newPlatformAutomator() UIAutomator { return &unsupportedAutomator{} }

func (u *unsupportedAutomator) Available() bool { return false }

func (u *unsupportedAutomator) FindElement(_ AutoSelector) (AutoElement, error) {
	return AutoElement{}, ErrAutomationUnsupported
}
func (u *unsupportedAutomator) Click(_ AutoElement) error             { return ErrAutomationUnsupported }
func (u *unsupportedAutomator) SetText(_ AutoElement, _ string) error { return ErrAutomationUnsupported }
func (u *unsupportedAutomator) SendKeys(_ string) error               { return ErrAutomationUnsupported }
func (u *unsupportedAutomator) Verify(_ AutoSelector, _ string) (bool, error) {
	return false, ErrAutomationUnsupported
}
