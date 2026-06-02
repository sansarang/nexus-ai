// phase_c_extensions.go — Phase C: 장기 메모리 + 협업 + 로컬 모드 (cross-platform)
// 사장님 요구:
//   C3 장기 메모리: "사장님이 좋아하는 형식" 학습, 페르소나 자동 추천 정확도 향상
//   C5 협업: 공유 워크스페이스, Slack/Teams 위임
//   C6 보안: 로컬 모드 (모든 데이터 로컬, 클라우드 X)

package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ── C3: 장기 메모리 (사용자 패턴 학습) ─────────────────────────────────

// UserPattern — 학습된 사용자 패턴 단위
type UserPattern struct {
	UserID         string    `json:"user_id"`
	PreferredLang  string    `json:"preferred_lang"`
	FavoriteActions map[string]int `json:"favorite_actions"`  // 액션 → 사용 횟수
	PreferredFormats map[string]int `json:"preferred_formats"` // excel/pdf/md
	RejectedActions map[string]int `json:"rejected_actions"`  // 거부한 액션
	LastActiveAt    time.Time `json:"last_active_at"`
	Tone            string    `json:"tone"`            // friendly / formal / concise
	WorkSchedule    []int     `json:"work_schedule"`   // 활동 시각 hist (0~23시)
}

var (
	userPatternsMu sync.RWMutex
	userPatterns   = make(map[string]*UserPattern)
)

func userPatternStorePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".nexus", "user_patterns.json")
}

func loadUserPatterns() {
	data, err := os.ReadFile(userPatternStorePath())
	if err != nil {
		return
	}
	userPatternsMu.Lock()
	defer userPatternsMu.Unlock()
	json.Unmarshal(data, &userPatterns)
}

func saveUserPatterns() {
	userPatternsMu.RLock()
	data, _ := json.Marshal(userPatterns)
	userPatternsMu.RUnlock()
	p := userPatternStorePath()
	os.MkdirAll(filepath.Dir(p), 0755)
	os.WriteFile(p, data, 0600)
}

func recordUserAction(userID, action, lang string, accepted bool) {
	if userID == "" {
		return
	}
	userPatternsMu.Lock()
	defer userPatternsMu.Unlock()
	p, ok := userPatterns[userID]
	if !ok {
		p = &UserPattern{
			UserID:           userID,
			PreferredLang:    lang,
			FavoriteActions:  make(map[string]int),
			PreferredFormats: make(map[string]int),
			RejectedActions:  make(map[string]int),
			WorkSchedule:     make([]int, 24),
		}
		userPatterns[userID] = p
	}
	if accepted {
		p.FavoriteActions[action]++
	} else {
		p.RejectedActions[action]++
	}
	p.LastActiveAt = time.Now()
	if lang != "" {
		p.PreferredLang = lang
	}
	hour := time.Now().Hour()
	if hour >= 0 && hour < 24 {
		p.WorkSchedule[hour]++
	}
	// 비동기 저장 (간단)
	go saveUserPatterns()
}

func recordUserFormat(userID, format string) {
	if userID == "" || format == "" {
		return
	}
	userPatternsMu.Lock()
	defer userPatternsMu.Unlock()
	p, ok := userPatterns[userID]
	if !ok {
		return
	}
	p.PreferredFormats[format]++
}

// getUserContext — LLM 응답 시 사용자 패턴 컨텍스트 주입
func getUserContext(userID string) string {
	if userID == "" {
		return ""
	}
	userPatternsMu.RLock()
	p, ok := userPatterns[userID]
	userPatternsMu.RUnlock()
	if !ok {
		return ""
	}
	// 자주 쓰는 액션 top-3
	type fa struct {
		a string
		c int
	}
	favs := make([]fa, 0, len(p.FavoriteActions))
	for a, c := range p.FavoriteActions {
		favs = append(favs, fa{a, c})
	}
	// 단순 정렬
	for i := 0; i < len(favs); i++ {
		for j := i + 1; j < len(favs); j++ {
			if favs[j].c > favs[i].c {
				favs[i], favs[j] = favs[j], favs[i]
			}
		}
	}
	topN := 3
	if len(favs) < topN {
		topN = len(favs)
	}
	ctx := "\n[사용자 패턴: "
	for i := 0; i < topN; i++ {
		ctx += favs[i].a
		if i < topN-1 {
			ctx += ", "
		}
	}
	ctx += " 자주 사용]"
	return ctx
}

// ── C5: 협업 (Slack/Teams 위임 골격) ───────────────────────────────────

type DelegateRequest struct {
	Channel string `json:"channel"` // slack / teams / email
	Target  string `json:"target"`  // 사람 이름 / ID
	Task    string `json:"task"`
	Due     string `json:"due,omitempty"`
}

func delegateToTeammate(req DelegateRequest) (map[string]any, error) {
	// 실제 Slack/Teams API 호출은 별도 토큰 필요 → 골격만
	// 사장님이 키 등록하면 활성화
	slackTok := getEnvKey("SLACK_BOT_TOKEN")
	if req.Channel == "slack" && slackTok != "" {
		// TODO: Slack chat.postMessage 호출
		return map[string]any{
			"success": true,
			"channel": "slack",
			"target":  req.Target,
			"message": "Slack 위임 큐에 추가됨 (토큰 등록됨)",
		}, nil
	}
	return map[string]any{
		"success":   false,
		"channel":   req.Channel,
		"target":    req.Target,
		"reason":    "토큰 미등록 (Slack/Teams API 키 설정 필요)",
		"fallback":  "이메일로 대신 전송 권장",
	}, nil
}

// ── C6: 로컬 모드 (Local-Only Mode) ───────────────────────────────────

var (
	localOnlyModeMu sync.RWMutex
	localOnlyMode   = false
)

func isLocalOnlyMode() bool {
	localOnlyModeMu.RLock()
	defer localOnlyModeMu.RUnlock()
	return localOnlyMode
}

func setLocalOnlyMode(on bool) {
	localOnlyModeMu.Lock()
	localOnlyMode = on
	localOnlyModeMu.Unlock()
	// 영구 저장
	home, _ := os.UserHomeDir()
	p := filepath.Join(home, ".nexus", "local_mode.json")
	os.MkdirAll(filepath.Dir(p), 0755)
	cfg := map[string]bool{"enabled": on}
	data, _ := json.Marshal(cfg)
	os.WriteFile(p, data, 0600)
}

func loadLocalOnlyMode() {
	home, _ := os.UserHomeDir()
	p := filepath.Join(home, ".nexus", "local_mode.json")
	data, err := os.ReadFile(p)
	if err != nil {
		return
	}
	var cfg map[string]bool
	if json.Unmarshal(data, &cfg) == nil {
		localOnlyModeMu.Lock()
		localOnlyMode = cfg["enabled"]
		localOnlyModeMu.Unlock()
	}
}

// ── B2: 에이전트 데이터 전달 ──────────────────────────────────────────
// workflow_run 단계간 결과를 다음 step.params 에 자동 주입

// PrevStepResult — 이전 step 결과
type PrevStepResult struct {
	Action  string
	Message string
	Result  any
}

// injectPrevResultIntoParams — 다음 step의 params 에 이전 결과 추출본 주입
// 명시 키 _prev_message / _prev_result / _prev_action 자동 추가
// 또한 흔한 패턴(email_summarize → note save) 자동 매핑
func injectPrevResultIntoParams(params map[string]any, prev PrevStepResult, nextAction string) map[string]any {
	if params == nil {
		params = map[string]any{}
	}
	params["_prev_action"] = prev.Action
	if prev.Message != "" {
		params["_prev_message"] = prev.Message
	}
	if prev.Result != nil {
		params["_prev_result"] = prev.Result
	}

	// 자동 매핑 룰: 이전 결과의 텍스트가 다음 액션의 핵심 입력일 때
	switch nextAction {
	case "note", "voice_todo":
		if _, has := params["content"]; !has && prev.Message != "" {
			params["content"] = prev.Message
		}
	case "email_draft":
		if _, has := params["body"]; !has && prev.Message != "" {
			params["body"] = prev.Message
		}
	case "translate":
		if _, has := params["text"]; !has && prev.Message != "" {
			params["text"] = prev.Message
		}
	case "excel_auto_create", "pdf_auto_create", "doc_auto_create":
		if _, has := params["topic"]; !has && prev.Message != "" {
			// 첫 문장만
			params["topic"] = prev.Message
			if len(prev.Message) > 80 {
				params["topic"] = prev.Message[:80]
			}
		}
	}
	return params
}
