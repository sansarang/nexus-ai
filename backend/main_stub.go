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

	// ── 사용량 관리 ──────────────────────────────────────────
	mux.HandleFunc("GET /api/usage", handleUsageStatus)

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
	mux.HandleFunc("POST /api/video/quick-search", handleVideoQuickSearch)
	mux.HandleFunc("POST /api/browser/search-and-pdf", handleBrowserSearchAndPDF)
	mux.HandleFunc("GET /api/browser/open-file", handleOpenFile)

	// ── 날씨 ─────────────────────────────────────────────────
	mux.HandleFunc("GET /api/weather", handleWeather)
	mux.HandleFunc("POST /api/travel/time", handleTravelTime)

	// ── 캘린더 ───────────────────────────────────────────────
	mux.HandleFunc("GET /api/calendar/today", handleCalendarToday)
	mux.HandleFunc("GET /api/calendar/week", handleCalendarWeek)
	mux.HandleFunc("POST /api/calendar/add", handleCalendarAdd)

	// ── 이메일 ───────────────────────────────────────────────
	mux.HandleFunc("GET /api/email/inbox", handleEmailInbox)
	mux.HandleFunc("POST /api/email/send", handleEmailSend)
	mux.HandleFunc("POST /api/email/summarize", handleEmailSummarize)

	// ── 메모리 / Second Brain ─────────────────────────────────
	mux.HandleFunc("GET /api/memory/list", handleMemoryList)
	mux.HandleFunc("POST /api/memory/search", handleMemorySearch)
	mux.HandleFunc("GET /api/memory/stats", handleMemoryStats)
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
	mux.HandleFunc("POST /api/email/classify", handleEmailClassify)
	mux.HandleFunc("POST /api/email/draft-reply", handleEmailDraftReply)
	mux.HandleFunc("POST /api/email/extract-events", handleEmailExtractEvents)
	mux.HandleFunc("POST /api/calendar/find-slot", handleCalendarFindSlot)
	mux.HandleFunc("POST /api/calendar/smart-add", handleCalendarSmartAdd)

	// ── Briefing ──────────────────────────────────────────────
	mux.HandleFunc("POST /api/briefing/now", handleBriefingNow)
	mux.HandleFunc("GET /api/briefing/config", handleBriefingConfig)
	mux.HandleFunc("POST /api/briefing/config", handleBriefingConfig)

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
	mux.HandleFunc("POST /api/recall/capture", winOnly)
	mux.HandleFunc("POST /api/recall/search", winOnly)
	mux.HandleFunc("POST /api/vision/screenshot", winOnly)
	mux.HandleFunc("POST /api/dictation/type", winOnly)
	mux.HandleFunc("POST /api/dictation/paste", winOnly)
	mux.HandleFunc("POST /api/meeting/start", winOnly)
	mux.HandleFunc("POST /api/meeting/stop", winOnly)
	mux.HandleFunc("POST /api/caption/start", winOnly)
	mux.HandleFunc("POST /api/caption/stop", winOnly)
	mux.HandleFunc("GET /api/caption/latest", winOnly)

	// ── 파일 관리 (cross-platform) ──────────────────────────
	mux.HandleFunc("POST /api/files/search", handleFilesSearch)
	mux.HandleFunc("POST /api/files/organize", handleFilesOrganize)
	mux.HandleFunc("POST /api/files/duplicates", handleFilesDuplicates)

	// ── 이메일 IMAP (cross-platform) ────────────────────────
	mux.HandleFunc("GET /api/imap/accounts", handleIMAPAccountList)
	mux.HandleFunc("POST /api/imap/accounts", handleIMAPAccountAdd)
	mux.HandleFunc("DELETE /api/imap/accounts", handleIMAPAccountDelete)
	mux.HandleFunc("GET /api/imap/inbox", handleIMAPInbox)
	mux.HandleFunc("POST /api/imap/send", handleIMAPSend)
	mux.HandleFunc("POST /api/imap/classify", handleIMAPClassify)
	mux.HandleFunc("GET /api/imap/reply-suggestions", handleIMAPReplySuggestions)

	// ── 비전·OCR stub ────────────────────────────────────────
	mux.HandleFunc("POST /api/vision/ocr-clipboard", handleOCRClipboard)

	// ── 주식 (cross-platform) ─────────────────────────────────
	mux.HandleFunc("GET /api/stock/quote", handleStockQuote)
	mux.HandleFunc("GET /api/stock/watchlist", handleStockWatchlist)
	mux.HandleFunc("POST /api/stock/watchlist", handleStockWatchlistAdd)
	mux.HandleFunc("DELETE /api/stock/watchlist", handleStockWatchlistDelete)

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
	mux.HandleFunc("GET /api/youtube/subscriptions", handleYouTubeSubscriptions)
	mux.HandleFunc("POST /api/youtube/playlist/add", handleYouTubePlaylistAdd)
	mux.HandleFunc("POST /api/youtube/playlist/batch", handleYouTubePlaylistBatch)
	mux.HandleFunc("POST /api/youtube/search", handleYouTubeSearch)

	// ── TikTok + YouTube Music ────────────────────────────────
	mux.HandleFunc("GET /api/tiktok/hot-songs", handleTikTokHotSongs)
	mux.HandleFunc("POST /api/ytmusic/search", handleYTMusicSearch)
	mux.HandleFunc("POST /api/ytmusic/playlist/add", handleYTMusicPlaylistAdd)
	mux.HandleFunc("POST /api/tiktok/songs-to-ytmusic", handleTikTokSongsToYTMusic)

	// ── 콘텐츠 추천 ───────────────────────────────────────────
	mux.HandleFunc("POST /api/recommend/content", handleContentRecommend)
	mux.HandleFunc("GET /api/netflix/trending", handleNetflixTrending)
	mux.HandleFunc("GET /api/recall/keywords", handleRecallKeywordsMac)
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
	mux.HandleFunc("POST /api/video/search-enhanced", handleVideoSearchEnhanced)
	mux.HandleFunc("POST /api/video/download-with-cookie", handleVideoDownloadCookie)
	mux.HandleFunc("POST /api/video/set-cookie", handleVideoSetCookie)
	mux.HandleFunc("GET /api/video/cookie-status", handleVideoCookieStatus)
	mux.HandleFunc("POST /api/video/ytdlp-info", handleVideoInfo)

	initMemory()
	initScheduler()
	initCronEngine()
	initTriggerEngine()
	loadLLMConfig()
	loadPersonaConfig()
	loadBrainIndex()
	go startBriefingScheduler()
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
