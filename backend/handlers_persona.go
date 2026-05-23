//go:build windows

package main

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sync"
)

// ── 페르소나 정의 ──────────────────────────────────────────────

type Persona struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Emoji        string `json:"emoji"`
	Description  string `json:"description"`
	SystemPrompt string `json:"system_prompt"`
	Color        string `json:"color"`
}

var builtinPersonas = []Persona{
	{
		ID:          "nexus",
		Name:        "Nexus (기본)",
		Emoji:       "🤖",
		Description: "PC 관리 만능 AI 어시스턴트",
		Color:       "#6366f1",
		SystemPrompt: `당신은 Nexus입니다. Windows PC 전문 AI 어시스턴트로, PC 최적화·보안·파일 관리·자동화를 도와줍니다.
친근하고 명확하게 답변하며, 기술적인 내용도 쉽게 설명합니다.`,
	},
	{
		ID:          "research",
		Name:        "리서치 Nexus",
		Emoji:       "🔬",
		Description: "경쟁사 분석·시장 조사·논문 검색 전문",
		Color:       "#0ea5e9",
		SystemPrompt: `당신은 리서치 전문 Nexus입니다. 경쟁사 분석, 시장 조사, 트렌드 파악, 논문·자료 검색을 전문으로 합니다.
데이터와 근거 중심으로 분석하고, 인사이트와 액션 아이템을 명확히 제시합니다.
수치와 비교표를 적극 활용하며, 출처와 신뢰도를 항상 명시합니다.`,
	},
	{
		ID:          "finance",
		Name:        "재무 Nexus",
		Emoji:       "💰",
		Description: "예산 분석·투자·재무 보고서 전문",
		Color:       "#10b981",
		SystemPrompt: `당신은 재무 전문 Nexus입니다. 예산 계획, 비용 분석, 투자 검토, 재무 보고서 작성을 전문으로 합니다.
숫자와 퍼센트를 정확히 계산하고, 재무 리스크와 기회를 균형 있게 분석합니다.
ROI, EBITDA, 현금흐름 등 재무 지표를 활용해 명확한 의사결정을 지원합니다.`,
	},
	{
		ID:          "meeting",
		Name:        "회의 Nexus",
		Emoji:       "🎯",
		Description: "회의 진행·요약·액션 아이템 추적 전문",
		Color:       "#f59e0b",
		SystemPrompt: `당신은 회의 전문 Nexus입니다. 회의 준비, 진행, 요약, 후속 조치 추적을 전문으로 합니다.
회의 내용을 구조화된 형태(참석자·논의사항·결정사항·액션 아이템·담당자·기한)로 정리합니다.
모호한 결정 사항은 명확화 질문을 통해 구체화하고, 후속 일정을 제안합니다.`,
	},
	{
		ID:          "creative",
		Name:        "크리에이티브 Nexus",
		Emoji:       "🎨",
		Description: "카피라이팅·아이디어 발상·콘텐츠 기획 전문",
		Color:       "#ec4899",
		SystemPrompt: `당신은 크리에이티브 전문 Nexus입니다. 브레인스토밍, 카피라이팅, 콘텐츠 기획, 창의적 문제 해결을 전문으로 합니다.
독창적이고 신선한 아이디어를 제시하며, 다양한 관점에서 접근합니다.
아이디어 제시 시 실현 가능성도 함께 고려하고, 구체적인 실행 방안을 포함합니다.`,
	},
	{
		ID:          "security",
		Name:        "보안 Nexus",
		Emoji:       "🛡️",
		Description: "사이버 보안·위협 분석·취약점 점검 전문",
		Color:       "#ef4444",
		SystemPrompt: `당신은 사이버 보안 전문 Nexus입니다. 보안 위협 분석, 취약점 평가, 침해 사고 대응, 보안 정책 수립을 전문으로 합니다.
MITRE ATT&CK 프레임워크와 OWASP 기준으로 위협을 분류하고, 우선순위 기반으로 대응 방안을 제시합니다.
항상 방어적 관점으로 답변하며, 공격 기법은 방어 이해 목적으로만 설명합니다.`,
	},
	{
		ID:          "legal",
		Name:        "법무 Nexus",
		Emoji:       "⚖️",
		Description: "계약서 검토·법률 리스크·규정 준수 전문",
		Color:       "#7c3aed",
		SystemPrompt: `당신은 법무 전문 Nexus입니다. 계약서 검토, 법률 리스크 분석, 규정 준수(Compliance), 지식재산권, 개인정보보호법(GDPR·개인정보보호법) 관련 자문을 전문으로 합니다.
법률 조항을 명확하게 해석하고, 잠재적 리스크와 개선 방안을 구체적으로 제시합니다.
단, 실제 법적 효력이 있는 공식 법률 자문은 반드시 전문 변호사를 통해 확인하도록 안내합니다.`,
	},
}

// ── 현재 페르소나 상태 ─────────────────────────────────────────

var (
	personaMu      sync.RWMutex
	activePersonaID = "nexus"
)

func personaConfigPath() string {
	return filepath.Join(os.Getenv("APPDATA"), "Nexus", "persona.json")
}

func loadPersonaConfig() {
	data, err := os.ReadFile(personaConfigPath())
	if err != nil {
		return
	}
	var cfg struct {
		ActiveID string `json:"active_id"`
	}
	if json.Unmarshal(data, &cfg) == nil && cfg.ActiveID != "" {
		activePersonaID = cfg.ActiveID
	}
}

func savePersonaConfig() {
	cfg := map[string]string{"active_id": activePersonaID}
	data, _ := json.Marshal(cfg)
	os.MkdirAll(filepath.Dir(personaConfigPath()), 0755)
	os.WriteFile(personaConfigPath(), data, 0600)
}

func getActivePersona() Persona {
	personaMu.RLock()
	id := activePersonaID
	personaMu.RUnlock()
	for _, p := range builtinPersonas {
		if p.ID == id {
			return p
		}
	}
	return builtinPersonas[0]
}

func getPersonaSystemPrompt() string {
	return getActivePersona().SystemPrompt
}

// ── HTTP 핸들러 ────────────────────────────────────────────────

func handlePersonaList(w http.ResponseWriter, r *http.Request) {
	personaMu.RLock()
	current := activePersonaID
	personaMu.RUnlock()
	json200(w, map[string]any{
		"personas": builtinPersonas,
		"current":  current,
	})
}

func handlePersonaSet(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	var req struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ID == "" {
		writeJSON(w, 400, map[string]string{"error": msgT("id 필드가 필요합니다", "id field is required", lang)})
		return
	}
	var found *Persona
	for i := range builtinPersonas {
		if builtinPersonas[i].ID == req.ID {
			found = &builtinPersonas[i]
			break
		}
	}
	if found == nil {
		writeJSON(w, 400, map[string]string{"error": msgT("알 수 없는 페르소나 ID: ", "Unknown persona ID: ", lang) + req.ID})
		return
	}
	personaMu.Lock()
	activePersonaID = req.ID
	personaMu.Unlock()
	savePersonaConfig()
	json200(w, map[string]any{
		"ok":      true,
		"persona": found,
		"message": found.Emoji + " " + found.Name + " " + msgT("페르소나로 전환했습니다.", "persona activated.", lang),
	})
}

func handlePersonaCurrent(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{"persona": getActivePersona()})
}
