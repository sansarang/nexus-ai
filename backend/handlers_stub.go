//go:build !windows

package main

import "net/http"

// handlers.go stubs
func handleScan(w http.ResponseWriter, r *http.Request)            {}
func handleRepair(w http.ResponseWriter, r *http.Request)          {}
func handleClean(w http.ResponseWriter, r *http.Request)           {}
func handleLicenseActivate(w http.ResponseWriter, r *http.Request) {}
func handleLicenseCheck(w http.ResponseWriter, r *http.Request)    {}
func handleStats(w http.ResponseWriter, r *http.Request)           {}
func handleAutoClean(w http.ResponseWriter, r *http.Request)       {}
func handlePrivacy(w http.ResponseWriter, r *http.Request)         {}
func handleDailyReport(w http.ResponseWriter, r *http.Request)     {}
func handleFolderOpen(w http.ResponseWriter, r *http.Request)      {}

// handlers_security.go stubs
func handleRemoteAccess(w http.ResponseWriter, r *http.Request)    {}
func handleProcessSecurity(w http.ResponseWriter, r *http.Request) {}
func handleHostsCheck(w http.ResponseWriter, r *http.Request)      {}
func handleStartupItems(w http.ResponseWriter, r *http.Request)    {}
func handleDefender(w http.ResponseWriter, r *http.Request)        {}
func handleAccountCheck(w http.ResponseWriter, r *http.Request)    {}

// handlers_system.go stubs
func handleVolume(w http.ResponseWriter, r *http.Request)          {}
func handleBrightness(w http.ResponseWriter, r *http.Request)      {}
func handleWifi(w http.ResponseWriter, r *http.Request)            {}
func handlePower(w http.ResponseWriter, r *http.Request)           {}
func handleLaunchApp(w http.ResponseWriter, r *http.Request)       {}
func handleProcessTop(w http.ResponseWriter, r *http.Request)      {}

// handlers_advanced.go stubs
func handleDrivers(w http.ResponseWriter, r *http.Request)         {}
func handleRegistryClean(w http.ResponseWriter, r *http.Request)   {}
func handlePowerPlans(w http.ResponseWriter, r *http.Request)      {}
func handleSetPowerPlan(w http.ResponseWriter, r *http.Request)    {}
func handleNetworkAnalysis(w http.ResponseWriter, r *http.Request) {}
func handleRestoreCreate(w http.ResponseWriter, r *http.Request)   {}
func handleDiskCheck(w http.ResponseWriter, r *http.Request)       {}
func handleBrowserClean(w http.ResponseWriter, r *http.Request)    {}
func handleProgramsList(w http.ResponseWriter, r *http.Request)    {}
func handleBootAnalysis(w http.ResponseWriter, r *http.Request)    {}
func handleFilesSearch(w http.ResponseWriter, r *http.Request)     {}
func handleFilesOrganize(w http.ResponseWriter, r *http.Request)   {}
func handleFilesDuplicates(w http.ResponseWriter, r *http.Request) {}
func handleFocusMode(w http.ResponseWriter, r *http.Request)       {}
func handleClipboard(w http.ResponseWriter, r *http.Request)       {}
func handleNotes(w http.ResponseWriter, r *http.Request)           {}
func handleSaveNote(w http.ResponseWriter, r *http.Request)        {}

// handlers_docs.go stubs
func handleDocCompare(w http.ResponseWriter, r *http.Request)      {}
func handleDocFind(w http.ResponseWriter, r *http.Request)         {}

// handlers_vision.go stubs
func handleDeepSearch(w http.ResponseWriter, r *http.Request)      {}
func handleScreenshot(w http.ResponseWriter, r *http.Request)      {}
func handleActiveWindow(w http.ResponseWriter, r *http.Request)    {}
func handleOCRClipboard(w http.ResponseWriter, r *http.Request)    {}

// handlers_journal.go stubs
func handleJournalToday(w http.ResponseWriter, r *http.Request)    {}
func handleJournalGenerate(w http.ResponseWriter, r *http.Request) {}
func handleJournalHistory(w http.ResponseWriter, r *http.Request)  {}

// handlers_macro.go stubs
func handleMacroList(w http.ResponseWriter, r *http.Request)       {}
func handleMacroCreate(w http.ResponseWriter, r *http.Request)     {}
func handleMacroRun(w http.ResponseWriter, r *http.Request)        {}
func handleMacroDelete(w http.ResponseWriter, r *http.Request)     {}
func handleMacroParse(w http.ResponseWriter, r *http.Request)      {}

// handlers_report.go stubs
func handleReportGenerate(w http.ResponseWriter, r *http.Request)  {}
func handleReportEmail(w http.ResponseWriter, r *http.Request)     {}
func handleReportSchedule(w http.ResponseWriter, r *http.Request)  {}
func handleEmailConfig(w http.ResponseWriter, r *http.Request)     {}

// handlers_docsummary.go stubs
func handleDocSummary(w http.ResponseWriter, r *http.Request)      {}
func handleDocExportReport(w http.ResponseWriter, r *http.Request) {}

// handlers_proactive.go stubs (SSE alert stream)
func handleAlertStream(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Write([]byte("data: {\"type\":\"connected\"}\n\n"))
	if f, ok := w.(http.Flusher); ok { f.Flush() }
	<-r.Context().Done()
}
func publishAlert(a Alert) {}

type Alert struct {
	ID        string `json:"id"`
	Level     string `json:"level"`
	Title     string `json:"title"`
	Message   string `json:"message"`
	Action    string `json:"action,omitempty"`
	Dismissed bool   `json:"dismissed"`
}
