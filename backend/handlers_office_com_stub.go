//go:build !windows

package main

import "net/http"

// Office COM은 Windows + 설치된 Office 필요. Mac/Linux에서는 windows_only 응답.

func windowsOnlyOffice(w http.ResponseWriter, feature string) {
	writeJSON(w, 503, map[string]any{
		"success": false,
		"code":    "windows_only",
		"message": "Office COM 자동화는 Windows + Microsoft Office 가 필요합니다 (" + feature + ").",
	})
}

func handleExcelComWorkbooks(w http.ResponseWriter, _ *http.Request) { windowsOnlyOffice(w, "Excel workbooks") }
func handleExcelComSetCell(w http.ResponseWriter, _ *http.Request)   { windowsOnlyOffice(w, "Excel set-cell") }
func handleExcelComFormula(w http.ResponseWriter, _ *http.Request)   { windowsOnlyOffice(w, "Excel formula") }
func handleExcelComReadRange(w http.ResponseWriter, _ *http.Request) { windowsOnlyOffice(w, "Excel read-range") }
func handleExcelComMacro(w http.ResponseWriter, _ *http.Request)     { windowsOnlyOffice(w, "Excel macro") }
func handleExcelComChart(w http.ResponseWriter, _ *http.Request)     { windowsOnlyOffice(w, "Excel chart") }
func handleExcelComSave(w http.ResponseWriter, _ *http.Request)      { windowsOnlyOffice(w, "Excel save") }
func handleWordComDocs(w http.ResponseWriter, _ *http.Request)       { windowsOnlyOffice(w, "Word documents") }
func handleWordComReplace(w http.ResponseWriter, _ *http.Request)    { windowsOnlyOffice(w, "Word replace") }
func handleWordComInsert(w http.ResponseWriter, _ *http.Request)     { windowsOnlyOffice(w, "Word insert") }
