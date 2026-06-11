//go:build windows

package main

import "errors"

// windowsAutomator — Windows UI Automation 클라이언트.
// 실제 요소 탐색/조작은 Python 사이드카(/desktop/uia/*, pywinauto)에 위임한다.
//
// ⚠️ [Windows QA 필요] pywinauto 실제 클릭/입력 동작은 Windows 머신에서 검증해야 한다.
//
//	RunSteps 닫힌 루프는 Available()(=Python /desktop/uia/status) 게이트를 통과해야만
//	실행되므로, UIA 미설치/비-Windows 환경에서는 자동으로 거부된다(무단 실행 없음).
type windowsAutomator struct{}

func newPlatformAutomator() UIAutomator { return &windowsAutomator{} }

func pyOK(res map[string]any) bool { ok, _ := res["success"].(bool); return ok }

func pyMsg(res map[string]any) string {
	if res != nil {
		if m, _ := res["message"].(string); m != "" {
			return m
		}
		if m, _ := res["reason"].(string); m != "" {
			return m
		}
	}
	return "automation error"
}

// Available — 런타임 조회: Python UIA 엔드포인트가 available:true일 때만 true.
// Python 미기동/사이드카 부재 시 callPython 오류 → false (안전 거부).
func (w *windowsAutomator) Available() bool {
	res, err := callPython("GET", "/desktop/uia/status", nil)
	if err != nil || res == nil {
		return false
	}
	a, _ := res["available"].(bool)
	return a
}

func (w *windowsAutomator) FindElement(sel AutoSelector) (AutoElement, error) {
	res, err := callPython("POST", "/desktop/uia/find", map[string]any{"selector": sel})
	if err != nil {
		return AutoElement{}, err
	}
	if !pyOK(res) {
		return AutoElement{}, errors.New(pyMsg(res))
	}
	name, _ := res["name"].(string)
	role, _ := res["role"].(string)
	// 셀렉터를 핸들에 보관 → Click/SetText가 Python에 재-위임할 때 재사용.
	// (재-find는 stale element에 강한 robust 패턴이라 의도적.)
	return AutoElement{Found: true, Name: name, Role: role, handle: sel}, nil
}

func (w *windowsAutomator) Click(el AutoElement) error {
	sel, _ := el.handle.(AutoSelector)
	res, err := callPython("POST", "/desktop/uia/click", map[string]any{"selector": sel})
	if err != nil {
		return err
	}
	if !pyOK(res) {
		return errors.New(pyMsg(res))
	}
	return nil
}

func (w *windowsAutomator) SetText(el AutoElement, text string) error {
	sel, _ := el.handle.(AutoSelector)
	res, err := callPython("POST", "/desktop/uia/set_text", map[string]any{"selector": sel, "text": text})
	if err != nil {
		return err
	}
	if !pyOK(res) {
		return errors.New(pyMsg(res))
	}
	return nil
}

func (w *windowsAutomator) DoubleClick(el AutoElement) error {
	sel, _ := el.handle.(AutoSelector)
	res, err := callPython("POST", "/desktop/uia/dclick", map[string]any{"selector": sel})
	if err != nil {
		return err
	}
	if !pyOK(res) {
		return errors.New(pyMsg(res))
	}
	return nil
}

func (w *windowsAutomator) RightClick(el AutoElement) error {
	sel, _ := el.handle.(AutoSelector)
	res, err := callPython("POST", "/desktop/uia/rclick", map[string]any{"selector": sel})
	if err != nil {
		return err
	}
	if !pyOK(res) {
		return errors.New(pyMsg(res))
	}
	return nil
}

func (w *windowsAutomator) Scroll(sel AutoSelector, amount int) error {
	res, err := callPython("POST", "/desktop/uia/scroll", map[string]any{"selector": sel, "amount": amount})
	if err != nil {
		return err
	}
	if !pyOK(res) {
		return errors.New(pyMsg(res))
	}
	return nil
}

func (w *windowsAutomator) SendKeys(combo string) error {
	res, err := callPython("POST", "/desktop/uia/send_keys", map[string]any{"keys": combo})
	if err != nil {
		return err
	}
	if !pyOK(res) {
		return errors.New(pyMsg(res))
	}
	return nil
}

func (w *windowsAutomator) Verify(sel AutoSelector, expect string) (bool, error) {
	res, err := callPython("POST", "/desktop/uia/verify", map[string]any{"selector": sel, "expect": expect})
	if err != nil {
		return false, err
	}
	v, _ := res["verified"].(bool)
	return v, nil
}
