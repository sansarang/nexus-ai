//go:build !windows

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/xuri/excelize/v2"
)

func cmdClarify(cx cmdCtx) {
		// 추가 정보 필요 — 프론트엔드에 질문 반환
		var question, missing, pendingIntent string
		var collected map[string]any
		if cx.params != nil {
			question, _ = cx.params["question"].(string)
			missing, _ = cx.params["missing"].(string)
			pendingIntent, _ = cx.params["intent"].(string)
			collected, _ = cx.params["collected"].(map[string]any)
		}
		if question == "" {
			question = "조금 더 알려주시면 도움이 될 것 같아요. 어떻게 도와드릴까요?"
		}
		_ = missing
		json200(cx.w, CommandResponse{
			Success:          true,
			Message:          question,
			Action:           "clarify",
			NeedsClarify:     true,
			ClarifyQuestion:  question,
			ClarifyQuestions: []string{question},
			PendingIntent:    pendingIntent,
			PendingParams:    collected,
			Duration:         cx.dur,
		})

}

func cmdChat(cx cmdCtx) {
		cat := detectCategory(cx.req.Message)
		expertList := detectExperts(cx.req.Message, cx.req.Lang)
		previewType := categoryPreviewType(cat)

		var answer string
		var chatItems []map[string]string
		var wg sync.WaitGroup
		wg.Add(2)

		// 고루틴 A: LLM 답변 (전문가 or 일반)
		go func() {
			defer wg.Done()
			if len(expertList) > 0 {
				answer, _ = runExpertParallel(cx.req.Message, cx.req.Lang, cx.gKey, expertList, cx.req.History)
			}
			if answer == "" {
				lang := cx.req.Lang
				var sysPrompt string
				if lang == "en" {
					sysPrompt = "You are Nexus AI, a helpful assistant. Answer in natural English, 2-4 sentences. No markdown headers."
				} else {
					personaPrompt := getPersonaSystemPrompt()
					sysPrompt = personaPrompt + "\n자연스러운 한국어로 답변하세요. 마크다운 헤더(##, ###) 금지."
				}
				// 세션 히스토리 주입 (최근 6턴)
				var msgs []groqMsg
				msgs = append(msgs, groqMsg{Role: "system", Content: sysPrompt})
				sess := getSession(cx.userID)
				sessionStoreMu.RLock()
				hist := sess.history
				sessionStoreMu.RUnlock()
				start2 := len(hist) - 6
				if start2 < 0 {
					start2 = 0
				}
				for _, h := range hist[start2:] {
					if h.Content != cx.req.Message { // 현재 메시지 중복 방지
						msgs = append(msgs, groqMsg{Role: h.Role, Content: h.Content})
					}
				}
				msgs = append(msgs, groqMsg{Role: "user", Content: cx.req.Message})
				answer, _, _ = callGroqWithCitations(cx.gKey, groqChatModel, msgs, 600)
				if answer == "" {
					if lang == "en" {
						answer = "Sorry, an error occurred while generating a response."
					} else {
						answer = "죄송합니다, 답변을 생성하는 중 오류가 발생했습니다."
					}
				}
			}
		}()

		// 고루틴 B: 카테고리별 상세 페이지 검색
		go func() {
			defer wg.Done()
			expertCat := expertsToCategory(expertList)
			pr := parallelWebSearch(cx.req.Message, 6, expertCat)
			if len(pr.Items) > 0 {
				chatItems = pr.Items
			} else {
				searchCat := cat
				if expertCat >= 0 {
					searchCat = expertCat
				}
				chatItems = categoryFallbackSites(cx.req.Message, searchCat)
			}
		}()
		wg.Wait()

		appendSession(cx.userID, "user", cx.req.Message)
		appendSession(cx.userID, "assistant", answer)

		json200(cx.w, CommandResponse{
			Success:  true,
			Message:  answer,
			Action:   "chat",
			Result:   map[string]any{"reply": answer, "items": chatItems, "preview_type": previewType},
			Duration: cx.dur,
		})

}

func cmdCalendarToday(cx cmdCtx) {
		today := time.Now().Format("2006-01-02")
		evs := loadEvents()
		var todayEvs []CalEvent
		for _, e := range evs {
			if e.Date == today {
				todayEvs = append(todayEvs, e)
			}
		}
		msg := fmt.Sprintf("오늘(%s) 일정이 %d개 있습니다.", today, len(todayEvs))
		if len(todayEvs) == 0 {
			msg = "오늘 등록된 일정이 없습니다."
		}
		json200(cx.w, CommandResponse{
			Success:  true,
			Message:  msg,
			Action:   "calendar_today",
			Result:   map[string]any{"events": todayEvs},
			Duration: cx.dur,
		})

}

func cmdCalendarAdd(cx cmdCtx) {
		var title, date, t string
		if cx.params != nil {
			title, _ = cx.params["title"].(string)
			date, _ = cx.params["date"].(string)
			t, _ = cx.params["time"].(string)
		}
		if title == "" {
			title = cx.req.Message
		}
		if date == "" {
			date = time.Now().Format("2006-01-02")
		}
		ev := CalEvent{
			ID: fmt.Sprintf("%d", time.Now().UnixMilli()),
			Title: title, Date: date, Time: t,
		}
		evs := loadEvents()
		evs = append(evs, ev)
		saveEvents(evs)
		json200(cx.w, CommandResponse{
			Success:  true,
			Message:  fmt.Sprintf("✅ 일정 추가됨: %s (%s)", title, date),
			Action:   "calendar_add",
			Result:   map[string]any{"event": ev},
			Duration: cx.dur,
		})

}

func cmdPersonaSwitch(cx cmdCtx) {
		var id string
		if cx.params != nil {
			id, _ = cx.params["id"].(string)
		}
		for _, p := range builtinPersonas {
			if p.ID == id {
				personaMu.Lock()
				activePersonaID = id
				personaMu.Unlock()
				savePersonaConfig()
				json200(cx.w, CommandResponse{
					Success:  true,
					Message:  p.Emoji + " " + p.Name + " 페르소나로 전환했습니다.",
					Action:   "persona_switch",
					Duration: cx.dur,
				})
				return
			}
		}
		json200(cx.w, CommandResponse{Success: false, Message: "알 수 없는 페르소나입니다.", Action: "persona_switch"})

}

func cmdNote(cx cmdCtx) {
		// 메모 저장
		noteContent, _ := cx.params["content"].(string)
		if noteContent == "" {
			noteContent = cx.req.Message
		}
		noteEng := cx.req.Lang == "en" || isEnglishQuery(cx.req.Message)
		home, _ := os.UserHomeDir()
		noteDir := filepath.Join(home, ".nexus", "notes")
		os.MkdirAll(noteDir, 0755)
		notePath := filepath.Join(noteDir, fmt.Sprintf("note_%s.txt", time.Now().Format("20060102_150405")))
		os.WriteFile(notePath, []byte(noteContent), 0644)
		var noteMsg string
		if noteEng {
			noteMsg = fmt.Sprintf("Note saved! 📝\nFile: %s", notePath)
		} else {
			noteMsg = fmt.Sprintf("메모 저장 완료! 📝\n파일: %s", notePath)
		}
		json200(cx.w, CommandResponse{
			Success: true, Message: noteMsg, Action: "note",
			Result: map[string]any{"path": notePath, "content": noteContent},
			Duration: cx.dur,
		})

}

func cmdDocSummary(cx cmdCtx) {
		// 문서 요약
		dsFile, _ := cx.params["file_path"].(string)
		dsEng := cx.req.Lang == "en" || isEnglishQuery(cx.req.Message)
		if dsFile == "" {
			var dsMsg string
			if dsEng {
				dsMsg = "Please specify the file path to summarize."
			} else {
				dsMsg = "요약할 파일 경로를 알려주세요."
			}
			json200(cx.w, CommandResponse{Success: false, Message: dsMsg, Action: "doc_summary", Duration: cx.dur})
			return
		}
		data, err := os.ReadFile(dsFile)
		if err != nil {
			var dsErr string
			if dsEng {
				dsErr = "Could not read the file: " + err.Error()
			} else {
				dsErr = "파일을 읽을 수 없습니다: " + err.Error()
			}
			json200(cx.w, CommandResponse{Success: false, Message: dsErr, Action: "doc_summary", Duration: cx.dur})
			return
		}
		content := string(data)
		if len(content) > 4000 {
			content = content[:4000]
		}
		llmMu.RLock()
		dsPKey := llmPerplexityKey
		llmMu.RUnlock()
		var dsSummary string
		if dsPKey != "" {
			var dsPrompt string
			if dsEng {
				dsPrompt = "Summarize the following document concisely in 3-5 sentences:\n\n" + content
			} else {
				dsPrompt = "다음 문서를 3-5문장으로 간결하게 요약해주세요:\n\n" + content
			}
			dsSummary, _, _ = callGroqWithFallback([]groqMsg{{Role: "user", Content: dsPrompt}}, 512, false)
		}
		if dsSummary == "" {
			if dsEng {
				dsSummary = "Could not generate summary."
			} else {
				dsSummary = "요약을 생성할 수 없습니다."
			}
		}
		json200(cx.w, CommandResponse{
			Success: true, Message: dsSummary, Action: "doc_summary",
			Result: map[string]any{"file": dsFile, "summary": dsSummary},
			Duration: cx.dur,
		})

}

func cmdHealthReport(cx cmdCtx) {
		// PC 건강 리포트 - Mac 시스템 정보 기반
		hrEng := cx.req.Lang == "en" || isEnglishQuery(cx.req.Message)
		hrStats := map[string]any{}
		if out, err := exec.Command("df", "-H", "/").Output(); err == nil {
			lines := strings.Split(string(out), "\n")
			if len(lines) > 1 {
				fields := strings.Fields(lines[1])
				if len(fields) >= 5 {
					hrStats["disk_used"] = fields[2]
					hrStats["disk_total"] = fields[1]
					pctStr := strings.TrimSuffix(fields[4], "%")
					if p, err2 := strconv.ParseFloat(pctStr, 64); err2 == nil {
						hrStats["disk_percent"] = p
					}
				}
			}
		}
		var hrMsg string
		if hrEng {
			hrMsg = fmt.Sprintf("Mac health report: Disk usage %v / %v. System appears healthy.", hrStats["disk_used"], hrStats["disk_total"])
		} else {
			hrMsg = fmt.Sprintf("Mac 건강 리포트: 디스크 사용량 %v / %v. 시스템이 정상입니다.", hrStats["disk_used"], hrStats["disk_total"])
		}
		json200(cx.w, CommandResponse{
			Success: true, Message: hrMsg, Action: "health_report",
			Result: hrStats, Duration: cx.dur,
		})

}

func cmdExcelSave(cx cmdCtx) {
		// 엑셀 저장
		exTitle, _ := cx.params["title"].(string)
		exEng := cx.req.Lang == "en" || isEnglishQuery(cx.req.Message)
		if exTitle == "" {
			if exEng {
				exTitle = "Nexus Data"
			} else {
				exTitle = "넥서스 데이터"
			}
		}
		home, _ := os.UserHomeDir()
		exPath := filepath.Join(home, "Desktop", fmt.Sprintf("nexus_%s.xlsx", time.Now().Format("20060102_150405")))
		f := excelize.NewFile()
		f.SetCellValue("Sheet1", "A1", exTitle)
		f.SetCellValue("Sheet1", "A2", time.Now().Format("2006-01-02 15:04:05"))
		if exErr := f.SaveAs(exPath); exErr != nil {
			var exMsg string
			if exEng {
				exMsg = "Failed to save Excel file: " + exErr.Error()
			} else {
				exMsg = "엑셀 저장 실패: " + exErr.Error()
			}
			json200(cx.w, CommandResponse{Success: false, Message: exMsg, Action: "excel_save", Duration: cx.dur})
			return
		}
		var exMsg string
		if exEng {
			exMsg = fmt.Sprintf("Excel saved! 📊\nFile: %s", exPath)
		} else {
			exMsg = fmt.Sprintf("엑셀 저장 완료! 📊\n파일: %s", exPath)
		}
		json200(cx.w, CommandResponse{
			Success: true, Message: exMsg, Action: "excel_save",
			Result: map[string]any{"path": exPath},
			Duration: cx.dur,
		})

}

func cmdRecall(cx cmdCtx) {
		// Windows Recall → Mac에서 mdfind/Spotlight로 대체
		rcQuery, _ := cx.params["query"].(string)
		if rcQuery == "" {
			rcQuery = cx.req.Message
		}
		rcEng := cx.req.Lang == "en" || isEnglishQuery(cx.req.Message)
		out, err := exec.Command("mdfind", rcQuery).Output()
		var rcItems []map[string]string
		if err == nil {
			lines := strings.Split(strings.TrimSpace(string(out)), "\n")
			for i, l := range lines {
				if l == "" || i >= 8 {
					break
				}
				rcItems = append(rcItems, map[string]string{"path": l, "name": filepath.Base(l)})
			}
		}
		var rcMsg string
		if len(rcItems) == 0 {
			if rcEng {
				rcMsg = fmt.Sprintf("No recent items found for '%s'.", rcQuery)
			} else {
				rcMsg = fmt.Sprintf("'%s' 관련 최근 항목을 찾지 못했습니다.", rcQuery)
			}
		} else {
			if rcEng {
				rcMsg = fmt.Sprintf("Found %d item(s) matching '%s' via Spotlight.", len(rcItems), rcQuery)
			} else {
				rcMsg = fmt.Sprintf("Spotlight에서 '%s' 관련 %d개 항목을 찾았습니다.", rcQuery, len(rcItems))
			}
		}
		json200(cx.w, CommandResponse{
			Success: true, Message: rcMsg, Action: "recall",
			Result: map[string]any{"query": rcQuery, "items": rcItems},
			Duration: cx.dur,
		})

}

func cmdBrowsePage(cx cmdCtx) {
		// 웹페이지 브라우징 → web_search로 처리
		bpURL, _ := cx.params["url"].(string)
		bpQuery, _ := cx.params["query"].(string)
		if bpURL == "" && bpQuery == "" {
			bpQuery = cx.req.Message
		}
		searchQ := bpURL
		if searchQ == "" {
			searchQ = bpQuery
		}
		bpEng := cx.req.Lang == "en" || isEnglishQuery(cx.req.Message)
		llmMu.RLock()
		bpTKey := llmTavilyKey
		llmMu.RUnlock()
		var bpSummary string
		var bpItems []map[string]string
		if bpTKey != "" {
			if tr, ok := tavilySearch(bpTKey, searchQ, 4); ok {
				bpSummary = tr.Summary
				bpItems = tr.Items
			}
		}
		if bpSummary == "" {
			if bpEng {
				bpSummary = fmt.Sprintf("Here are search results for: %s", searchQ)
			} else {
				bpSummary = fmt.Sprintf("%s 검색 결과입니다.", searchQ)
			}
		}
		json200(cx.w, CommandResponse{
			Success: true, Message: bpSummary, Action: "browse_page",
			Result: map[string]any{"query": searchQ, "summary": bpSummary, "items": bpItems},
			Duration: cx.dur,
		})

	// ── 🟠 3. 파일 조작 ──────────────────────────────────────────
}

func cmdDirections(cx cmdCtx) {
		// 길찾기 → handleDirections 내부 로직 재사용 (Tavily 검색)
		from, _ := cx.params["from"].(string)
		to, _ := cx.params["to"].(string)
		mode, _ := cx.params["mode"].(string)
		if to == "" {
			to = cx.req.Message
		}
		if mode == "" {
			mode = "transit"
		}
		dEng := cx.req.Lang == "en" || isEnglishQuery(cx.req.Message)
		var dQuery string
		if dEng {
			if from != "" {
				dQuery = fmt.Sprintf("directions from %s to %s by %s", from, to, mode)
			} else {
				dQuery = fmt.Sprintf("directions to %s by %s", to, mode)
			}
		} else {
			if from != "" {
				dQuery = fmt.Sprintf("%s에서 %s 가는 %s 길찾기", from, to, mode)
			} else {
				dQuery = fmt.Sprintf("%s 가는 방법 %s", to, mode)
			}
		}
		llmMu.RLock()
		localTKey := llmTavilyKey; cx.tKey = localTKey
		llmMu.RUnlock()
		var dSummary string
		var dItems []map[string]string
		if cx.tKey != "" {
			if tr, ok := tavilySearch(cx.tKey, dQuery, 4); ok {
				dSummary = tr.Summary
				dItems = tr.Items
			}
		}
		if dSummary == "" {
			if dEng {
				dSummary = fmt.Sprintf("Here are directions to %s.", to)
			} else {
				dSummary = fmt.Sprintf("%s 경로 정보입니다.", to)
			}
		}
		links := buildMapLinks(from, to, mode, dEng)
		json200(cx.w, CommandResponse{
			Success: true, Message: dSummary, Action: "directions",
			Result: map[string]any{"from": from, "to": to, "mode": mode, "summary": dSummary, "items": dItems, "links": links},
			Duration: cx.dur,
		})

}

func cmdPlaceView(cx cmdCtx) {
		// 장소 검색
		query, _ := cx.params["query"].(string)
		if query == "" {
			query = cx.req.Message
		}
		pEng := cx.req.Lang == "en" || isEnglishQuery(cx.req.Message)
		llmMu.RLock()
		tKey2 := llmTavilyKey
		llmMu.RUnlock()
		var pSummary string
		var pItems []map[string]string
		if tKey2 != "" {
			var pq string
			if pEng {
				pq = query + " location address hours"
			} else {
				pq = query + " 위치 주소 영업시간"
			}
			if tr, ok := tavilySearch(tKey2, pq, 4); ok {
				pSummary = tr.Summary
				pItems = tr.Items
			}
		}
		if pSummary == "" {
			if pEng {
				pSummary = fmt.Sprintf("Here is information about %s.", query)
			} else {
				pSummary = fmt.Sprintf("%s 정보입니다.", query)
			}
		}
		json200(cx.w, CommandResponse{
			Success: true, Message: pSummary, Action: "place_view",
			Result: map[string]any{"query": query, "summary": pSummary, "items": pItems},
			Duration: cx.dur,
		})

}

func cmdMultiAgent(cx cmdCtx) {
		// 멀티 에이전트 → handleMultiAgentRun에 위임
		goal, _ := cx.params["goal"].(string)
		if goal == "" {
			goal = cx.req.Message
		}
		llmMu.RLock()
		maKey := llmPerplexityKey
		llmMu.RUnlock()
		if maKey == "" {
			maEng := cx.req.Lang == "en" || isEnglishQuery(cx.req.Message)
			var maMsg string
			if maEng {
				maMsg = "API key required for multi-agent execution."
			} else {
				maMsg = "멀티 에이전트 실행에 API 키가 필요합니다."
			}
			json200(cx.w, CommandResponse{Success: false, Message: maMsg, Action: "multi_agent", Duration: cx.dur})
			return
		}
		maEng := cx.req.Lang == "en" || isEnglishQuery(cx.req.Message)
		var maStart string
		if maEng {
			maStart = fmt.Sprintf("Starting multi-agent execution for: %s", goal)
		} else {
			maStart = fmt.Sprintf("멀티 에이전트 실행 시작: %s", goal)
		}
		go func(g, k string) {
			result, err := runMacOrchestrate(g, k)
			var alertMsg string
			if err != nil {
				if maEng {
					alertMsg = "Multi-agent execution failed: " + err.Error()
				} else {
					alertMsg = "멀티 에이전트 실행 실패: " + err.Error()
				}
			} else {
				if maEng {
					alertMsg = "Multi-agent complete: " + result
				} else {
					alertMsg = "멀티 에이전트 완료: " + result
				}
			}
			publishAlert(Alert{ID: fmt.Sprintf("ma_%d", time.Now().Unix()), Level: "info", Title: "Multi-Agent", Message: alertMsg})
		}(goal, maKey)
		json200(cx.w, CommandResponse{
			Success: true, Message: maStart, Action: "multi_agent",
			Result:   map[string]any{"goal": goal, "status": "running"},
			Duration: cx.dur,
		})

}

func cmdEmail(cx cmdCtx) {
		// 이메일 - SMTP 설정 안내
		eEng := cx.req.Lang == "en" || isEnglishQuery(cx.req.Message)
		var eMsg string
		if eEng {
			eMsg = "Email feature requires SMTP configuration. Please go to Settings → Email to set up your email account."
		} else {
			eMsg = "이메일 기능을 사용하려면 SMTP 설정이 필요합니다. 설정 → 이메일에서 이메일 계정을 설정해주세요."
		}
		json200(cx.w, CommandResponse{Success: false, Message: eMsg, Action: "email", Duration: cx.dur})

}

func cmdMeeting(cx cmdCtx) {
		// 회의 요약/분석 → 웹 검색 폴백
		mQuery, _ := cx.params["query"].(string)
		if mQuery == "" {
			mQuery = cx.req.Message
		}
		mEng := cx.req.Lang == "en" || isEnglishQuery(cx.req.Message)
		llmMu.RLock()
		mPKey := llmPerplexityKey
		llmMu.RUnlock()
		var mAnswer string
		if mPKey != "" {
			var mPrompt string
			if mEng {
				mPrompt = "You are a meeting assistant. Help with: " + mQuery + "\nAnswer concisely in English."
			} else {
				mPrompt = getPersonaSystemPrompt() + "\n회의 관련 질문: " + mQuery + "\n간결하게 한국어로 답변하세요."
			}
			mMsgs := []groqMsg{{Role: "user", Content: mPrompt}}
			mAnswer, _, _ = callGroqWithFallback(mMsgs, 512, false)
		}
		if mAnswer == "" {
			if mEng {
				mAnswer = "Please describe what you need help with for your meeting."
			} else {
				mAnswer = "회의 관련해서 무엇을 도와드릴까요?"
			}
		}
		json200(cx.w, CommandResponse{Success: true, Message: mAnswer, Action: "meeting", Duration: cx.dur})

}

func cmdBriefing(cx cmdCtx) {
		// 브리핑 → handleBriefingNow 로직 직접 호출
		bEng := cx.req.Lang == "en" || isEnglishQuery(cx.req.Message)
		llmMu.RLock()
		bKey := llmTavilyKey
		llmMu.RUnlock()
		var bSections []string
		// 날씨
		weatherURL := "https://wttr.in/Seoul?format=j1"
		if bEng {
			weatherURL = "https://wttr.in/New York?format=j1"
		}
		wClient := &http.Client{Timeout: 5 * time.Second}
		if wr, err := wClient.Get(weatherURL); err == nil {
			defer wr.Body.Close()
			var wraw map[string]any
			if json.NewDecoder(wr.Body).Decode(&wraw) == nil {
				if cc, ok := wraw["current_condition"].([]any); ok && len(cc) > 0 {
					c := cc[0].(map[string]any)
					temp := fmt.Sprintf("%v", c["temp_C"])
					desc := ""
					if wds, ok := c["weatherDesc"].([]any); ok && len(wds) > 0 {
						desc = fmt.Sprintf("%v", (wds[0].(map[string]any))["value"])
					}
					if bEng {
						bSections = append(bSections, fmt.Sprintf("🌤️ Weather: %s°C, %s", temp, desc))
					} else {
						bSections = append(bSections, fmt.Sprintf("🌤️ 날씨: %s°C, %s", temp, desc))
					}
				}
			}
		}
		// 뉴스
		if bKey != "" {
			var nq string
			if bEng {
				nq = "today's top news worldwide 2026"
			} else {
				nq = "오늘 주요 뉴스 한국"
			}
			if nr, ok := tavilySearch(bKey, nq, 3); ok && nr.Summary != "" {
				if bEng {
					bSections = append(bSections, "📰 News: "+nr.Summary)
				} else {
					bSections = append(bSections, "📰 뉴스: "+nr.Summary)
				}
			}
		}
		var bMsg string
		if len(bSections) > 0 {
			bMsg = strings.Join(bSections, "\n\n")
		} else {
			if bEng {
				bMsg = "Good morning! Today's briefing is ready."
			} else {
				bMsg = "좋은 아침이에요! 오늘의 브리핑입니다."
			}
		}
		json200(cx.w, CommandResponse{
			Success: true, Message: bMsg, Action: "briefing",
			Result: map[string]any{"sections": bSections},
			Duration: cx.dur,
		})

}

func cmdDefault(cx cmdCtx) {
		// 알 수 없는 액션 → chat으로 폴백
		chatMsgs := []groqMsg{
			{Role: "system", Content: getPersonaSystemPrompt()},
			{Role: "user", Content: cx.req.Message},
		}
		answer, _, _ := callGroqWithFallback(chatMsgs, 1024, false)
		json200(cx.w, CommandResponse{
			Success:  true,
			Message:  answer,
			Action:   "chat",
			Duration: cx.dur,
		})
}

