//go:build !windows

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
		json200(w, map[string]string{"status": "ok", "platform": "mac-dev"})
	})

	// ── 설치 후 의존성 상태 체크 ─────────────────────────────
	mux.HandleFunc("GET /api/setup/status", handleSetupStatus)

	// ── 사용자 언어 설정 (영속) ──────────────────────────────
	mux.HandleFunc("GET /api/settings/lang", handleSettingsLang)
	mux.HandleFunc("POST /api/settings/lang", handleSettingsLang)

	// ── 💼 Pro Persona 전용 기능 ──────────────────────────────
	mux.HandleFunc("POST /api/finance/stock", handleStockAnalysis)
	mux.HandleFunc("POST /api/medical/search", handleMedicalSearch)
	mux.HandleFunc("POST /api/legal/review", handleContractReview)
	mux.HandleFunc("POST /api/legal/search", handleLegalSearch)
	mux.HandleFunc("POST /api/content/script", handleContentScript)

	// ── LLM ──────────────────────────────────────────────────
	mux.HandleFunc("GET /api/llm/config", handleLLMConfig)
	mux.HandleFunc("POST /api/llm/config", handleLLMConfig)
	mux.HandleFunc("POST /api/llm/chat", handleLLMChat)
	mux.HandleFunc("POST /api/llm/deep-search-web", handleLLMDeepSearchWeb)
	mux.HandleFunc("POST /api/llm/deep-search", handleLLMDeepSearch)
	mux.HandleFunc("GET /api/notes", handleNotes)
	mux.HandleFunc("POST /api/notes", handleSaveNote)
	mux.HandleFunc("POST /api/notes/save", handleSaveNote)

	// ── 자연어 명령 라우터 (핵심) ────────────────────────────
	mux.HandleFunc("POST /api/command", handleCommand)
	mux.HandleFunc("GET /api/agent/proposals", handleAgentPatchProposals)
	mux.HandleFunc("POST /api/agent/approve", handleAgentApprove)
	mux.HandleFunc("POST /api/agent/reject", handleAgentReject)
	mux.HandleFunc("GET /api/agent/status", handleAgentStatus)
	mux.HandleFunc("POST /api/agent/analyze-now", handleAgentAnalyzeNow)
	// Team workspace
	mux.HandleFunc("POST /api/team/create", handleTeamCreate)
	mux.HandleFunc("GET /api/team/members", handleTeamMembers)
	mux.HandleFunc("POST /api/team/invite", handleTeamInvite)
	mux.HandleFunc("POST /api/team/accept", handleTeamAccept)
	mux.HandleFunc("POST /api/team/remove", handleTeamRemove)
	// Gmail OAuth

	// ── 🛒 워크플로우 마켓플레이스 ───────────────────────────────

	// ── 사용량 관리 ──────────────────────────────────────────
	mux.HandleFunc("GET /api/usage", handleUsageStatus)
	mux.HandleFunc("GET /api/usage/ai", handleUsageAI)
	mux.HandleFunc("POST /api/usage/ai", handleUsageAI)

	// ── Windows-only 엔드포인트 호환 shim (Mac 개발환경) ─────
	mux.HandleFunc("GET /api/daily-report", func(w http.ResponseWriter, r *http.Request) {
		json200(w, map[string]any{"available": false, "platform": "mac-dev", "message": "일일 리포트는 Windows 환경에서만 지원됩니다."})
	})
	mux.HandleFunc("GET /api/network/analysis", func(w http.ResponseWriter, r *http.Request) {
		json200(w, map[string]any{"available": false, "platform": "mac-dev", "message": "네트워크 분석은 Windows 환경에서만 지원됩니다."})
	})
	// /api/news shim
	mux.HandleFunc("GET /api/news", func(w http.ResponseWriter, r *http.Request) {
		json200(w, map[string]any{"items": []any{}, "message": "뉴스는 /api/command 채팅으로 검색하세요."})
	})

	// ── 사이트 직접 검색 (LLM 우회, 항상 링크 반환) ─────────
	mux.HandleFunc("POST /api/site-search", handleSiteSearch)
	mux.HandleFunc("POST /api/file/process", handleFileProcess)
	mux.HandleFunc("POST /api/directions", handleDirections)
	mux.HandleFunc("POST /api/place-view", handlePlaceView)

	// ── Browser / 크롤링 ─────────────────────────────────────
	mux.HandleFunc("GET /api/browser/status", handleBrowserStatus)
	mux.HandleFunc("POST /api/browser/navigate", handleBrowserNavigate)
	mux.HandleFunc("POST /api/browser/extract", handleBrowserExtract)
	mux.HandleFunc("POST /api/browser/click", handleBrowserClick)
	mux.HandleFunc("POST /api/browser/fill", handleBrowserFill)
	mux.HandleFunc("POST /api/browser/screenshot", handleBrowserScreenshot)
	mux.HandleFunc("POST /api/browser/agent", handleBrowserAgent)
	mux.HandleFunc("POST /api/browser/close", handleBrowserClose)
	mux.HandleFunc("POST /api/browser/smart-agent", handleBrowserSmartAgent)
	mux.HandleFunc("POST /api/browser/collect-price", handleBrowserCollectPrice)
	mux.HandleFunc("POST /api/browser/news-collect", handleBrowserNewsCollect)
	mux.HandleFunc("POST /api/browser/search-and-pdf", handleBrowserSearchAndPDF)
	mux.HandleFunc("GET /api/browser/open-file", handleOpenFile)

	// ── 날씨 ─────────────────────────────────────────────────

	// ── 캘린더 ───────────────────────────────────────────────

	// ── 이메일 ───────────────────────────────────────────────

	// ── 메모리 / Second Brain ─────────────────────────────────
	mux.HandleFunc("POST /api/brain/search", handleBrainSearch)
	mux.HandleFunc("GET /api/brain/stats", handleBrainStats)
	mux.HandleFunc("POST /api/brain/rebuild", handleBrainRebuild)
	mux.HandleFunc("POST /api/brain/index", handleBrainIndex)

	// ── 페르소나 ──────────────────────────────────────────────
	mux.HandleFunc("GET /api/persona/list", handlePersonaList)
	mux.HandleFunc("POST /api/persona/set", handlePersonaSet)
	mux.HandleFunc("GET /api/persona/current", handlePersonaCurrent)

	// ── 스케줄러 ─────────────────────────────────────────────
	mux.HandleFunc("POST /api/scheduler/add", handleSchedulerAdd)
	mux.HandleFunc("GET /api/scheduler/list", handleSchedulerList)
	mux.HandleFunc("DELETE /api/scheduler/delete", handleSchedulerDelete)

	// ── Excel ────────────────────────────────────────────────
	mux.HandleFunc("POST /api/excel/save", handleExcelSave)

	// ── 워크플로우 ────────────────────────────────────────────
	mux.HandleFunc("POST /api/workflow/plan", handleWorkflowPlan)
	mux.HandleFunc("POST /api/workflow/run", handleWorkflowRun)
	registerAutomationRoutes(mux) // 🤖 데스크탑 자동화 엔진 라우트
	mux.HandleFunc("GET /api/workflow/list", handleWorkflowList)
	mux.HandleFunc("POST /api/workflow/save", handleWorkflowSave)
	mux.HandleFunc("DELETE /api/workflow/delete", handleWorkflowDelete)
	mux.HandleFunc("POST /api/workflow/run-now", handleWorkflowRunNow)
	mux.HandleFunc("POST /api/workflow/from-text", handleWorkflowFromText)
	mux.HandleFunc("GET /api/workflow/templates", handleWorkflowTemplates)

	// ── VirusTotal ───────────────────────────────────────────
	mux.HandleFunc("POST /api/security/virustotal", handleVirusTotal)

	// ── 성능 이력 ─────────────────────────────────────────────
	mux.HandleFunc("GET /api/history/stats", handleHistoryStats)
	mux.HandleFunc("GET /api/history/anomalies", handleHistoryAnomalies)

	// ── Proactive 알림 + SSE ───────────────────────────────────
	mux.HandleFunc("GET /api/alerts/stream", handleAlertStream)
	mux.HandleFunc("GET /api/alerts/latest", handleAlertLatest)

	// ── Task Queue ────────────────────────────────────────────
	mux.HandleFunc("GET /api/tasks/stream", handleTaskStream)
	mux.HandleFunc("GET /api/tasks/list", handleTaskList)
	mux.HandleFunc("POST /api/tasks/cancel", handleTaskCancel)

	// ── Multi-Agent ───────────────────────────────────────────
	mux.HandleFunc("POST /api/agent/multi/run", handleMultiAgentRun)
	mux.HandleFunc("POST /api/agent/multi/plan", handleMultiAgentPlan)
	mux.HandleFunc("GET /api/agent/multi/agents", handleAgentList)

	// ── Email Deep Agency ─────────────────────────────────────

	// ── Briefing ──────────────────────────────────────────────

	// ── Desktop Agent ─────────────────────────────────────────
	mux.HandleFunc("POST /api/agent/desktop/run", handleDesktopAgentRun)
	mux.HandleFunc("GET /api/agent/desktop/status", handleDesktopStatus)
	mux.HandleFunc("GET /api/agent/desktop/screenshot", handleDesktopScreenshot)
	mux.HandleFunc("POST /api/agent/desktop/approve", handleDesktopApprove)

	// ── Productivity (추가) ───────────────────────────────────
	mux.HandleFunc("GET /api/productivity/clipboard", handleClipboard)
	mux.HandleFunc("GET /api/scheduler/tasks", handleSchedulerList)
	mux.HandleFunc("POST /api/scheduler/run-now", handleSchedulerRunNow)
	mux.HandleFunc("POST /api/scheduler/parse", handleSchedulerParse)


	// ── Privacy & Sandbox ─────────────────────────────────────
	mux.HandleFunc("GET /api/security/audit", handleAuditLog)
	mux.HandleFunc("POST /api/security/check-path", handleCheckPath)
	mux.HandleFunc("GET /api/ollama/config", handleOllamaConfig)
	mux.HandleFunc("POST /api/ollama/config", handleOllamaConfig)
	mux.HandleFunc("POST /api/ollama/test", handleOllamaTest)
	mux.HandleFunc("GET /api/ollama/models", handleOllamaModels)

	// Windows 전용 기능 → "지원 안 됨" 응답
	winOnly := func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]any{"success": false, "message": "이 기능은 Windows에서만 사용 가능합니다."})
	}
	mux.HandleFunc("POST /api/scan", winOnly)
	mux.HandleFunc("POST /api/repair", winOnly)
	mux.HandleFunc("POST /api/clean", winOnly)
	mux.HandleFunc("GET /api/stats", handleStats)
	mux.HandleFunc("POST /api/system/volume", winOnly)
	mux.HandleFunc("POST /api/system/brightness", winOnly)
	mux.HandleFunc("POST /api/system/wifi", winOnly)
	mux.HandleFunc("POST /api/system/power", winOnly)
	mux.HandleFunc("POST /api/system/launch", winOnly)
	mux.HandleFunc("GET /api/processes/top", handleProcessTop)
	mux.HandleFunc("GET /api/security/remote", winOnly)
	mux.HandleFunc("GET /api/security/processes", winOnly)
	mux.HandleFunc("GET /api/security/startup", winOnly)
	mux.HandleFunc("GET /api/security/defender", winOnly)
	mux.HandleFunc("GET /api/security/accounts", winOnly)
	mux.HandleFunc("GET /api/drivers", winOnly)
	mux.HandleFunc("POST /api/registry/clean", winOnly)
	mux.HandleFunc("POST /api/vision/screenshot", winOnly)
	mux.HandleFunc("POST /api/dictation/type", winOnly)
	mux.HandleFunc("POST /api/dictation/paste", winOnly)
	mux.HandleFunc("POST /api/caption/start", winOnly)
	mux.HandleFunc("POST /api/caption/stop", winOnly)
	mux.HandleFunc("GET /api/caption/latest", winOnly)

	// ── 파일 관리 (cross-platform) ──────────────────────────
	mux.HandleFunc("POST /api/files/search", handleFilesSearch)
	mux.HandleFunc("POST /api/files/organize", handleFilesOrganize)
	mux.HandleFunc("POST /api/files/duplicates", handleFilesDuplicates)

	// ── 이메일 IMAP (cross-platform) ────────────────────────

	// ── 비전·OCR stub ────────────────────────────────────────
	mux.HandleFunc("POST /api/vision/ocr-clipboard", handleOCRClipboard)

	// ── 주식 (cross-platform) ─────────────────────────────────

	// ── cron 실행 엔진 ────────────────────────────────────────
	mux.HandleFunc("POST /api/cron/add", handleCronAdd)
	mux.HandleFunc("GET /api/cron/list", handleCronList)
	mux.HandleFunc("DELETE /api/cron/delete", handleCronDelete)
	mux.HandleFunc("POST /api/cron/run-now", handleCronRunNow)

	// ── 🔴 환율/주가 실시간 API ──────────────────────────────
	mux.HandleFunc("POST /api/exchange-rate", handleExchangeRate)

	// ── 🟠 파일 시스템 조작 ───────────────────────────────────
	mux.HandleFunc("POST /api/file/organize", handleFileOrganize)
	mux.HandleFunc("POST /api/file/duplicates", handleFileDuplicates)
	mux.HandleFunc("POST /api/file/large", handleFileLarge)

	// ── 🟠 조건부 알림 트리거 ─────────────────────────────────
	mux.HandleFunc("POST /api/trigger/add", handleTriggerAdd)
	mux.HandleFunc("GET /api/trigger/list", handleTriggerList)
	mux.HandleFunc("DELETE /api/trigger/delete", handleTriggerDelete)
	mux.HandleFunc("GET /api/trigger/events", handleTriggerEvents)

	// ── 🟡 화면 캡처 + Vision ─────────────────────────────────
	mux.HandleFunc("POST /api/screenshot/analyze", handleScreenshotAnalyze)
	mux.HandleFunc("POST /api/screenshot/translate", handleScreenshotTranslate)

	// ── 브라우저 히스토리 ─────────────────────────────────────
	mux.HandleFunc("GET /api/history/tiktok", handleTikTokHistory)
	mux.HandleFunc("GET /api/history/youtube", handleYouTubeHistory)
	mux.HandleFunc("GET /api/history/keywords", handleHistoryKeywords)
	mux.HandleFunc("GET /api/history/summary", handleHistorySummary)

	// ── YouTube 자동화 ────────────────────────────────────────

	// ── TikTok + YouTube Music ────────────────────────────────

	// ── 콘텐츠 추천 ───────────────────────────────────────────
	mux.HandleFunc("POST /api/recommend/content", handleContentRecommend)
	mux.HandleFunc("GET /api/wishlist/content", handleContentWishlist)
	mux.HandleFunc("POST /api/wishlist/content", handleContentWishlistAdd)

	// ── 메타데이터 분석 ───────────────────────────────────────
	mux.HandleFunc("POST /api/file/metadata", handleFileMetadata)

	// ── Wayback Machine ───────────────────────────────────────
	mux.HandleFunc("POST /api/wayback/snapshots", handleWaybackSnapshots)
	mux.HandleFunc("GET /api/wayback/available", handleWaybackAvailable)

	// ── 익명 검색 (SearXNG/DDG) ──────────────────────────────
	mux.HandleFunc("POST /api/search/anonymous", handleAnonymousSearch)

	// ── 보안 감사 (Shodan/ipinfo) ─────────────────────────────
	mux.HandleFunc("POST /api/security/shodan", handleShodanAudit)
	mux.HandleFunc("GET /api/security/myip", handleMyIPAudit)

	// ── 영상 수집 강화 ────────────────────────────────────────

	// Enterprise API Key Management
	mux.HandleFunc("GET /api/enterprise/keys", handleEnterpriseListKeys)
	mux.HandleFunc("POST /api/enterprise/keys", handleEnterpriseCreateKey)
	mux.HandleFunc("DELETE /api/enterprise/keys/{id}", handleEnterpriseRevokeKey)
	mux.HandleFunc("GET /api/enterprise/keys/{id}/usage", handleEnterpriseKeyUsage)
	mux.HandleFunc("GET /api/enterprise/plans", handleEnterprisePlans)

	// External v1 API
	mux.HandleFunc("POST /v1/chat", handleV1Chat)
	mux.HandleFunc("POST /v1/search", handleV1Search)
	mux.HandleFunc("POST /v1/stock", handleV1Stock)
	mux.HandleFunc("POST /v1/legal", handleV1Legal)
	mux.HandleFunc("POST /v1/medical", handleV1Medical)

	// Vertical App Config
	mux.HandleFunc("GET /api/vertical/config", handleVerticalGetConfig)
	mux.HandleFunc("POST /api/vertical/config", handleVerticalSetConfig)
	mux.HandleFunc("GET /api/vertical/presets", handleVerticalPresets)

	initMemory()
	initScheduler()
	initCronEngine()
	initTriggerEngine()
	loadLLMConfig()
	loadPersonaConfig()
	loadBrainIndex()
	loadUserPatterns()
	loadLocalOnlyMode()
	startRAGBackgroundWatcher()
	startSelfHealingLoop()
	loadWorkspaceStore()
	go startMacProactiveMonitor()

	srv := &http.Server{
		Addr:    "127.0.0.1:17891",
		Handler: cors(mux),
	}

	go func() {
		log.Println("[Nexus Backend Mac] 시작 :17891")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("[Nexus Backend Mac] 종료")
}
