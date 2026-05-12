//go:build windows

package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)


func main() {
	mux := http.NewServeMux()

	// 헬스체크
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		json200(w, map[string]string{"status": "ok"})
	})

	// PC 전체 진단
	mux.HandleFunc("POST /api/scan", handleScan)

	// 수리 실행
	mux.HandleFunc("POST /api/repair", handleRepair)

	// 파일 정리
	mux.HandleFunc("POST /api/clean", handleClean)

	// 라이선스 활성화
	mux.HandleFunc("POST /api/license/activate", handleLicenseActivate)

	// 라이선스 확인
	mux.HandleFunc("GET /api/license/check", handleLicenseCheck)

	// 실시간 통계
	mux.HandleFunc("GET /api/stats", handleStats)

	// 자동 정리
	mux.HandleFunc("POST /api/autoclean", handleAutoClean)

	// 프라이버시
	mux.HandleFunc("POST /api/privacy", handlePrivacy)

	// 데일리 리포트
	mux.HandleFunc("GET /api/daily-report", handleDailyReport)

	// 폴더 열기
	mux.HandleFunc("POST /api/folder/open", handleFolderOpen)

	// ── 보안 상세 ──────────────────────────────
	mux.HandleFunc("GET /api/security/remote", handleRemoteAccess)
	mux.HandleFunc("GET /api/security/processes", handleProcessSecurity)
	mux.HandleFunc("GET /api/security/hosts", handleHostsCheck)
	mux.HandleFunc("GET /api/security/startup", handleStartupItems)
	mux.HandleFunc("GET /api/security/defender", handleDefender)
	mux.HandleFunc("GET /api/security/accounts", handleAccountCheck)

	// ── 시스템 제어 ────────────────────────────
	mux.HandleFunc("POST /api/system/volume", handleVolume)
	mux.HandleFunc("POST /api/system/brightness", handleBrightness)
	mux.HandleFunc("POST /api/system/wifi", handleWifi)
	mux.HandleFunc("POST /api/system/power", handlePower)
	mux.HandleFunc("POST /api/system/launch", handleLaunchApp)
	mux.HandleFunc("GET /api/processes/top", handleProcessTop)

	// ── 고급 기능 ──────────────────────────────
	mux.HandleFunc("GET /api/drivers", handleDrivers)
	mux.HandleFunc("POST /api/registry/clean", handleRegistryClean)
	mux.HandleFunc("GET /api/power/plans", handlePowerPlans)
	mux.HandleFunc("POST /api/power/plan", handleSetPowerPlan)
	mux.HandleFunc("GET /api/network/analysis", handleNetworkAnalysis)
	mux.HandleFunc("POST /api/restore/create", handleRestoreCreate)
	mux.HandleFunc("POST /api/disk/check", handleDiskCheck)
	mux.HandleFunc("POST /api/browser/clean", handleBrowserClean)
	mux.HandleFunc("GET /api/programs", handleProgramsList)
	mux.HandleFunc("GET /api/boot/analysis", handleBootAnalysis)

	// ── 파일 관리 ──────────────────────────────
	mux.HandleFunc("POST /api/files/search", handleFilesSearch)
	mux.HandleFunc("POST /api/files/organize", handleFilesOrganize)
	mux.HandleFunc("POST /api/files/duplicates", handleFilesDuplicates)

	// ── 생산성 ─────────────────────────────────
	mux.HandleFunc("POST /api/productivity/focus", handleFocusMode)
	mux.HandleFunc("GET /api/productivity/clipboard", handleClipboard)
	mux.HandleFunc("GET /api/notes", handleNotes)
	mux.HandleFunc("POST /api/notes", handleSaveNote)

	// ── 문서 비교 & 검색 ───────────────────────
	mux.HandleFunc("POST /api/docs/compare", handleDocCompare)
	mux.HandleFunc("POST /api/docs/find", handleDocFind)
	mux.HandleFunc("POST /api/search/deep", handleDeepSearch)

	// ── Vision & OCR ───────────────────────────
	mux.HandleFunc("POST /api/vision/screenshot", handleScreenshot)
	mux.HandleFunc("GET /api/vision/active-window", handleActiveWindow)
	mux.HandleFunc("POST /api/vision/ocr-clipboard", handleOCRClipboard)

	// ── 업무 일지 ────────────────────────────
	mux.HandleFunc("GET /api/journal/today", handleJournalToday)
	mux.HandleFunc("POST /api/journal/generate", handleJournalGenerate)
	mux.HandleFunc("GET /api/journal/history", handleJournalHistory)

	// ── 자동화 매크로 ─────────────────────────
	mux.HandleFunc("GET /api/macros", handleMacroList)
	mux.HandleFunc("POST /api/macros", handleMacroCreate)
	mux.HandleFunc("POST /api/macros/run", handleMacroRun)
	mux.HandleFunc("POST /api/macros/delete", handleMacroDelete)
	mux.HandleFunc("POST /api/macros/parse", handleMacroParse)

	// ── PC 건강 리포트 + 이메일 ───────────────
	mux.HandleFunc("GET /api/report/generate", handleReportGenerate)
	mux.HandleFunc("POST /api/report/email", handleReportEmail)
	mux.HandleFunc("POST /api/report/schedule", handleReportSchedule)
	mux.HandleFunc("GET /api/email/config", handleEmailConfig)
	mux.HandleFunc("POST /api/email/config", handleEmailConfig)

	// ── 문서 요약 ────────────────────────────
	mux.HandleFunc("POST /api/docs/summary", handleDocSummary)
	mux.HandleFunc("POST /api/docs/export-report", handleDocExportReport)

	// ── LLM (Groq + Claude fallback) ─────────────────────────
	mux.HandleFunc("GET /api/llm/config", handleLLMConfig)
	mux.HandleFunc("POST /api/llm/config", handleLLMConfig)
	mux.HandleFunc("POST /api/llm/chat", handleLLMChat)
	mux.HandleFunc("POST /api/llm/vision", handleLLMVision)
	mux.HandleFunc("POST /api/llm/doc-summary", handleLLMDocSummary)
	mux.HandleFunc("POST /api/llm/doc-compare", handleLLMDocCompare)
	mux.HandleFunc("POST /api/llm/deep-search", handleLLMDeepSearch)
	mux.HandleFunc("POST /api/llm/deep-search-web", handleLLMDeepSearchWeb)

	// ── Browser Agent (chromedp + Stealth) ──────────────────────
	mux.HandleFunc("GET /api/browser/status", handleBrowserStatus)
	mux.HandleFunc("POST /api/browser/navigate", handleBrowserNavigate)
	mux.HandleFunc("POST /api/browser/extract", handleBrowserExtract)
	mux.HandleFunc("POST /api/browser/click", handleBrowserClick)
	mux.HandleFunc("POST /api/browser/fill", handleBrowserFill)
	mux.HandleFunc("POST /api/browser/screenshot", handleBrowserScreenshot)
	mux.HandleFunc("POST /api/browser/agent", handleBrowserAgent)
	mux.HandleFunc("POST /api/browser/close", handleBrowserClose)
	// 신규 고급 Browser Agent
	mux.HandleFunc("POST /api/browser/smart-agent", handleBrowserSmartAgent)
	mux.HandleFunc("POST /api/browser/collect-price", handleBrowserCollectPrice)
	mux.HandleFunc("POST /api/browser/news-collect", handleBrowserNewsCollect)
	mux.HandleFunc("POST /api/browser/login-session", handleBrowserLoginSession)
	mux.HandleFunc("POST /api/browser/video-download", handleVideoDownload)
	// ★ 핵심: 검색 → PDF 자동 생성 (동적 결과물)
	mux.HandleFunc("POST /api/browser/search-and-pdf", handleBrowserSearchAndPDF)
	mux.HandleFunc("GET /api/browser/open-file", handleOpenFile)

	// ── Excel 내보내기 ────────────────────────────────────────
	mux.HandleFunc("POST /api/excel/save", handleExcelSave)
	mux.HandleFunc("GET /api/excel/list", handleExcelList)

	// ── AI 문서 편집 (excelize 기반) ────────────────────────
	mux.HandleFunc("POST /api/docs/upload", handleDocUpload)
	mux.HandleFunc("POST /api/docs/ai-edit", handleDocAIEdit)
	mux.HandleFunc("GET /api/excel/read", handleReadExcel)

	// ── 자연어 스케줄러 ───────────────────────────────────────
	mux.HandleFunc("POST /api/scheduler/add", handleSchedulerAdd)
	mux.HandleFunc("GET /api/scheduler/list", handleSchedulerList)
	mux.HandleFunc("DELETE /api/scheduler/delete", handleSchedulerDelete)
	mux.HandleFunc("POST /api/scheduler/run-now", handleSchedulerRunNow)
	mux.HandleFunc("POST /api/scheduler/parse", handleSchedulerParse)

	// ── 에이전트 장기 메모리 ──────────────────────────────────
	mux.HandleFunc("GET /api/memory/list", handleMemoryList)
	mux.HandleFunc("POST /api/memory/search", handleMemorySearch)
	mux.HandleFunc("DELETE /api/memory/clear", handleMemoryClear)
	mux.HandleFunc("GET /api/memory/stats", handleMemoryStats)

	// ── 자연어 명령 라우터 (핵심: 말만 하면 알아서 처리) ──────
	mux.HandleFunc("POST /api/command", handleCommand)

	// ── 사이트 직접 검색 (LLM 우회, 항상 링크 반환) ─────────
	mux.HandleFunc("POST /api/site-search", handleSiteSearch)
	mux.HandleFunc("POST /api/file/process", handleFileProcess)

	// ── Proactive AI: 실시간 알림 스트림 (SSE) ────────────────
	mux.HandleFunc("GET /api/alerts/stream", handleAlertStream)
	mux.HandleFunc("GET /api/alerts/latest", handleAlertLatest)

	// ── 📅 캘린더 ────────────────────────────────────────────
	mux.HandleFunc("GET /api/calendar/today", handleCalendarToday)
	mux.HandleFunc("GET /api/calendar/week", handleCalendarWeek)
	mux.HandleFunc("POST /api/calendar/add", handleCalendarAdd)

	// ── 📧 이메일 ─────────────────────────────────────────────
	mux.HandleFunc("GET /api/email/inbox", handleEmailInbox)
	mux.HandleFunc("POST /api/email/send", handleEmailSend)
	mux.HandleFunc("POST /api/email/summarize", handleEmailSummarize)

	// ── 🦠 VirusTotal ────────────────────────────────────────
	mux.HandleFunc("POST /api/security/virustotal", handleVirusTotal)

	// ── 📊 성능 이력 ─────────────────────────────────────────
	mux.HandleFunc("POST /api/history/snapshot", handleHistorySnapshot)
	mux.HandleFunc("GET /api/history/stats", handleHistoryStats)
	mux.HandleFunc("GET /api/history/anomalies", handleHistoryAnomalies)

	// ── 🔧 시스템 확장 ───────────────────────────────────────
	mux.HandleFunc("POST /api/process/kill", handleProcessKill)
	mux.HandleFunc("GET /api/app/permissions", handleAppPermissions)
	mux.HandleFunc("GET /api/system/updates", handleWindowsUpdates)
	mux.HandleFunc("GET /api/gpu/stats", handleGPUStats)

	// ── 🖥️ Windows Recall ────────────────────────────────────────
	mux.HandleFunc("POST /api/recall/capture", handleRecallCapture)
	mux.HandleFunc("POST /api/recall/search", handleRecallSearch)

	// ── 🎙️ 회의 어시스턴트 ───────────────────────────────────────
	mux.HandleFunc("POST /api/meeting/start", handleMeetingStart)
	mux.HandleFunc("POST /api/meeting/stop", handleMeetingStop)
	mux.HandleFunc("POST /api/meeting/transcribe", handleMeetingTranscribe)
	mux.HandleFunc("GET /api/meeting/list", handleMeetingList)
	mux.HandleFunc("POST /api/meeting/summarize", handleMeetingSummarize)

	// ── ⌨️ 음성 받아쓰기 ─────────────────────────────────────────
	mux.HandleFunc("POST /api/dictation/type", handleDictationType)
	mux.HandleFunc("POST /api/dictation/paste", handleDictationPaste)


	// ── 🌤️ 날씨 + 교통 ──────────────────────────────────────────
	mux.HandleFunc("GET /api/weather", handleWeather)
	mux.HandleFunc("POST /api/travel/time", handleTravelTime)

	// ── 🎭 AI 멀티 페르소나 ──────────────────────────────────────
	mux.HandleFunc("GET /api/persona/list", handlePersonaList)
	mux.HandleFunc("POST /api/persona/set", handlePersonaSet)
	mux.HandleFunc("GET /api/persona/current", handlePersonaCurrent)

	// ── 🧠 Second Brain ──────────────────────────────────────────
	mux.HandleFunc("POST /api/brain/index", handleBrainIndex)
	mux.HandleFunc("POST /api/brain/search", handleBrainSearch)
	mux.HandleFunc("POST /api/brain/rebuild", handleBrainRebuild)
	mux.HandleFunc("GET /api/brain/stats", handleBrainStats)

	// ── ⚡ Auto Workflow Agent ────────────────────────────────────
	mux.HandleFunc("POST /api/workflow/plan", handleWorkflowPlan)
	mux.HandleFunc("POST /api/workflow/run", handleWorkflowRun)

	// ── ☀️ Proactive Briefing ─────────────────────────────────────
	mux.HandleFunc("POST /api/briefing/now", handleBriefingNow)
	mux.HandleFunc("GET /api/briefing/config", handleBriefingConfig)
	mux.HandleFunc("POST /api/briefing/config", handleBriefingConfig)

	// ── 🖥️ Desktop Computer Use Agent ────────────────────────────
	mux.HandleFunc("POST /api/agent/desktop/run", handleDesktopAgentRun)
	mux.HandleFunc("POST /api/agent/desktop/click", handleDesktopClick)
	mux.HandleFunc("POST /api/agent/desktop/type", handleDesktopType)
	mux.HandleFunc("POST /api/agent/desktop/key", handleDesktopKey)
	mux.HandleFunc("POST /api/agent/desktop/scroll", handleDesktopScroll)
	mux.HandleFunc("POST /api/agent/desktop/drag", handleDesktopDrag)
	mux.HandleFunc("GET /api/agent/desktop/screenshot", handleDesktopScreenshot)
	mux.HandleFunc("GET /api/agent/desktop/status", handleDesktopStatus)
	mux.HandleFunc("POST /api/agent/desktop/approve", handleDesktopApprove)
	// aliases for frontend
	mux.HandleFunc("GET /api/desktop/screenshot", handleDesktopScreenshot)
	mux.HandleFunc("GET /api/desktop/status", handleDesktopStatus)
	mux.HandleFunc("POST /api/desktop/approve", handleDesktopApprove)
	mux.HandleFunc("POST /api/desktop/agent/run", handleDesktopAgentRun)
	mux.HandleFunc("POST /api/desktop/agent/cancel", handleDesktopAgentCancel)

	// ── 📋 Task Queue ──────────────────────────────────────────────
	mux.HandleFunc("GET /api/tasks/stream", handleTaskStream)
	mux.HandleFunc("GET /api/tasks/list", handleTaskList)
	mux.HandleFunc("POST /api/tasks/cancel", handleTaskCancel)

	// ── 🤖 Multi-Agent ────────────────────────────────────────────
	mux.HandleFunc("POST /api/agent/multi/run", handleMultiAgentRun)
	mux.HandleFunc("POST /api/agent/multi/plan", handleMultiAgentPlan)
	mux.HandleFunc("GET /api/agent/multi/agents", handleAgentList)
	mux.HandleFunc("GET /api/multi-agent/stream/", handleMultiAgentStream)
	mux.HandleFunc("POST /api/multi-agent/run", handleMultiAgentRunV2)

	// ── 📧 Email Deep Agency ──────────────────────────────────────
	mux.HandleFunc("POST /api/email/classify", handleEmailClassify)
	mux.HandleFunc("POST /api/email/draft-reply", handleEmailDraftReply)
	mux.HandleFunc("POST /api/email/extract-events", handleEmailExtractEvents)
	mux.HandleFunc("POST /api/calendar/find-slot", handleCalendarFindSlot)
	mux.HandleFunc("POST /api/calendar/smart-add", handleCalendarSmartAdd)

	// ── 🔧 Visual Workflow Builder ────────────────────────────────
	mux.HandleFunc("GET /api/workflow/list", handleWorkflowList)
	mux.HandleFunc("POST /api/workflow/save", handleWorkflowSave)
	mux.HandleFunc("DELETE /api/workflow/delete", handleWorkflowDelete)
	mux.HandleFunc("POST /api/workflow/run-now", handleWorkflowRunNow)
	mux.HandleFunc("POST /api/workflow/from-text", handleWorkflowFromText)
	mux.HandleFunc("GET /api/workflow/templates", handleWorkflowTemplates)

	// ── 📬 IMAP/SMTP 이메일 (Naver/Daum/Kakao) ───────────────────
	mux.HandleFunc("GET /api/imap/accounts", handleIMAPAccountList)
	mux.HandleFunc("POST /api/imap/accounts", handleIMAPAccountAdd)
	mux.HandleFunc("DELETE /api/imap/accounts", handleIMAPAccountDelete)
	mux.HandleFunc("GET /api/imap/inbox", handleIMAPInbox)
	mux.HandleFunc("POST /api/imap/send", handleIMAPSend)
	mux.HandleFunc("GET /api/imap/reply-suggestions", handleIMAPReplySuggestions)
	mux.HandleFunc("POST /api/imap/classify", handleIMAPClassify)

	// ── 🔒 Privacy & Sandbox ──────────────────────────────────────
	mux.HandleFunc("GET /api/security/audit", handleAuditLog)
	mux.HandleFunc("POST /api/security/check-path", handleCheckPath)
	mux.HandleFunc("GET /api/ollama/config", handleOllamaConfig)
	mux.HandleFunc("POST /api/ollama/config", handleOllamaConfig)
	mux.HandleFunc("POST /api/ollama/test", handleOllamaTest)
	mux.HandleFunc("GET /api/ollama/models", handleOllamaModels)

	// ── 🎬 Live Caption ──────────────────────────────────────────
	mux.HandleFunc("POST /api/caption/start", handleCaptionStart)
	mux.HandleFunc("POST /api/caption/stop", handleCaptionStop)
	mux.HandleFunc("GET /api/caption/stream", handleCaptionStream)
	mux.HandleFunc("GET /api/caption/latest", handleCaptionLatest)

	// ── 💳 Paddle 구독 웹훅 ──────────────────────────────────────
	mux.HandleFunc("POST /api/paddle/webhook", handlePaddleWebhook)

	// 스케줄러 + 메모리 + LLM 키 초기화
	initScheduler()
	initMemory()
	initTaskQueue()
	loadLLMConfig()
	loadPersonaConfig()
	loadBrainIndex()
	go startProactiveMonitor()   // Proactive AI 백그라운드 모니터링
	go startBriefingScheduler()  // 아침 브리핑 + 미팅 사전 알림
	go startWorkflowScheduler()  // 비주얼 워크플로우 스케줄 실행
	loadOllamaConfig()           // 로컬 LLM 설정 로드
	go startHistoryCollector()   // 성능 이력 자동 수집 (5분 간격)
	go startRecallCollector()    // Windows Recall 자동 캡처 (60초 간격)
	go rebuildBrainIndex()       // Second Brain 초기 인덱싱

	srv := &http.Server{
		Addr:    "127.0.0.1:17891",
		Handler: cors(mux),
	}

	go func() {
		log.Println("[Nexus Backend] 시작 :17891")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("[Nexus Backend] 종료")
}
