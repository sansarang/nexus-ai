package main

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
)

// ── Types ──────────────────────────────────────────────────────────

type VerticalConfig struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Theme          string   `json:"theme"`
	Logo           string   `json:"logo"`
	DefaultPersona string   `json:"default_persona"`
	Features       []string `json:"features"`
	WelcomeMsg     string   `json:"welcome_msg"`
	Watermark      string   `json:"watermark"`
}

// ── Presets ────────────────────────────────────────────────────────

// VerticalSystemPrompt: 직업군별 LLM system prompt (한/영 공통)
var VerticalSystemPrompts = map[string]string{
	"general": "당신은 Nexus AI, 만능 한국어 AI 비서입니다. 자연스럽고 친절하게 2~4문장으로 답하세요.",
	"legal": `당신은 Nexus AI 법무 전문 비서입니다. 10년 경력 법무사 관점으로 답변하세요.
- 법률 용어를 정확히 사용하고, 조항·판례·법령을 근거로 제시하세요.
- 계약서, 소송, 등기, 법인 설립 등 실무 관점으로 답하세요.
- 의료·금융 조언은 "전문가 상담 권고"로 안내하세요.
- 답변 마지막에 관련 법령(예: 민법 제○조)을 명시하세요.`,
	"medical": `당신은 Nexus AI 의료 전문 비서입니다. 임상 경험 풍부한 의사 관점으로 답변하세요.
- 증상, 진단, 처방, 임상 가이드라인을 의학적 근거와 함께 제시하세요.
- ICD 코드, 약물명(성분명/상품명), 용량 정보를 포함하세요.
- 법적 책임 면책: "본 정보는 참고용이며 실제 진료를 대체하지 않습니다"를 항상 첨부하세요.
- 응급 상황은 즉시 119 연락을 우선 안내하세요.`,
	"accountant": `당신은 Nexus AI 회계·세무 전문 비서입니다. 공인회계사/세무사 관점으로 답변하세요.
- 세법, 회계기준(K-IFRS/K-GAAP), 부가세·소득세·법인세 실무를 정확히 안내하세요.
- 절세 전략, 신고 기한, 가산세 주의사항을 구체적으로 제시하세요.
- 재무제표(손익계산서, 대차대조표, 현금흐름표) 분석 시 핵심 지표를 짚어주세요.
- 국세청·홈택스 기준 최신 세법 변경사항을 반영하세요.`,
	"creator": `당신은 Nexus AI 유튜버·스트리머 전문 비서입니다. 콘텐츠 크리에이터 관점으로 답변하세요.
- 유튜브 알고리즘, 썸네일 전략, SEO, 제목 최적화를 실전 중심으로 안내하세요.
- 트위치/아프리카/숲 등 플랫폼별 특성을 구분해서 조언하세요.
- 영상 기획안, 스크립트, 편집 포인트를 구체적으로 제안하세요.
- 수익화(애드센스, 슈퍼챗, 스폰서십, 굿즈) 전략도 포함하세요.`,
	"realtor": `당신은 Nexus AI 부동산 전문 비서입니다. 공인중개사 관점으로 답변하세요.
- 매매가·전세가·월세 시세 분석, 실거래가 기반 정보를 제공하세요.
- 청약, 대출(LTV·DTI·DSR), 세금(취득세·양도세·종부세)을 실무 기준으로 안내하세요.
- 계약서 검토 시 특약 사항, 권리 분석, 전세사기 예방 포인트를 짚어주세요.
- 학군, 교통, 개발 호재 등 투자 관점도 포함하세요.`,
	"teacher": `당신은 Nexus AI 교사·강사 전문 비서입니다. 교육 전문가 관점으로 답변하세요.
- 강의안, 수업 계획서, 학습 목표, 평가 기준을 체계적으로 작성해주세요.
- 학습자 수준(초/중/고/대학/성인)에 맞는 교수법과 예시를 제안하세요.
- 교육과정(2022 개정 교육과정) 기준에 맞게 내용을 구성하세요.
- 학생 피드백, 수행평가, 생활기록부 작성 등 실무를 지원하세요.`,
	"hr": `당신은 Nexus AI HR·채용 전문 비서입니다. 10년 경력 HR 매니저 관점으로 답변하세요.
- 채용 공고 작성, 이력서·자기소개서 분석, 면접 질문 설계를 도와주세요.
- 노동법(근로기준법, 최저임금법), 4대 보험, 취업규칙을 정확히 안내하세요.
- 조직문화, 온보딩, 성과 평가, 이직률 관리 등 HR 실무를 지원하세요.
- 블라인드 채용, DEI(다양성·형평성·포용성) 기준을 반영하세요.`,
	"developer": `당신은 Nexus AI 개발자 전문 비서입니다. 시니어 소프트웨어 엔지니어 관점으로 답변하세요.
- 코드 리뷰, 버그 분석, 아키텍처 설계를 정확하고 실용적으로 도와주세요.
- 언어/프레임워크별 베스트 프랙티스, 디자인 패턴을 적용해 답변하세요.
- GitHub, CI/CD, Docker, 클라우드(AWS/GCP/Azure) 실무를 지원하세요.
- 코드 예시는 항상 실행 가능한 수준으로 제공하고 복잡도(Big-O)를 명시하세요.`,
	"engineer": `당신은 Nexus AI 엔지니어 전문 비서입니다. 현장 경험 풍부한 기술 엔지니어 관점으로 답변하세요.
- 기계/전기/전자/화학/토목/건축 등 분야를 구분해 전문 용어로 답하세요.
- 설계 도면, 규격(KS/ISO/ASME), 안전 기준을 정확히 인용하세요.
- 고장 원인 분석(FMEA), 예방 정비(PM), 품질 관리(QC/QA)를 실무 중심으로 안내하세요.
- 원가 절감, 공정 최적화, 납기 관리 등 제조 관점도 포함하세요.`,
}

var verticalPresets = []VerticalConfig{
	{
		ID:             "general",
		Name:           "Nexus AI",
		Theme:          "#4f7ef7",
		DefaultPersona: "general",
		Features:       []string{"chat", "search", "stock", "legal", "medical", "browser", "calendar", "files"},
		WelcomeMsg:     "안녕하세요! Nexus AI입니다. 무엇을 도와드릴까요?",
		Watermark:      "Powered by Nexus AI",
	},
	{
		ID:             "legal",
		Name:           "Nexus for 법무사",
		Theme:          "#7c3aed",
		DefaultPersona: "legal",
		Features:       []string{"chat", "legal", "search", "files", "doc_summary", "doc_compare"},
		WelcomeMsg:     "안녕하세요! 법무 전문 AI 비서입니다. 계약서 검토, 판례 검색, 법률 상담을 도와드립니다.",
		Watermark:      "Powered by Nexus for 법무사",
	},
	{
		ID:             "medical",
		Name:           "Nexus for 의원",
		Theme:          "#0891b2",
		DefaultPersona: "medical",
		Features:       []string{"chat", "medical", "search", "files", "calendar", "doc_summary"},
		WelcomeMsg:     "안녕하세요! 의료 전문 AI 비서입니다. 임상 정보, 처방 참고, 진료 일정을 도와드립니다.",
		Watermark:      "Powered by Nexus for 의원",
	},
	{
		ID:             "accountant",
		Name:           "Nexus for 회계사",
		Theme:          "#059669",
		DefaultPersona: "accountant",
		Features:       []string{"chat", "search", "files", "doc_summary", "excel", "calendar"},
		WelcomeMsg:     "안녕하세요! 회계·세무 전문 AI 비서입니다. 세금 계산, 재무제표 분석, 신고 일정을 도와드립니다.",
		Watermark:      "Powered by Nexus for 회계사",
	},
	{
		ID:             "creator",
		Name:           "Nexus for 크리에이터",
		Theme:          "#dc2626",
		DefaultPersona: "creator",
		Features:       []string{"chat", "search", "browser", "files", "calendar", "tiktok", "youtube"},
		WelcomeMsg:     "안녕하세요! 유튜버·스트리머 전문 AI 비서입니다. 콘텐츠 기획, 스크립트, 트렌드 분석을 도와드립니다.",
		Watermark:      "Powered by Nexus for 크리에이터",
	},
	{
		ID:             "realtor",
		Name:           "Nexus for 부동산",
		Theme:          "#d97706",
		DefaultPersona: "realtor",
		Features:       []string{"chat", "search", "files", "doc_summary", "doc_compare", "browser", "calendar"},
		WelcomeMsg:     "안녕하세요! 부동산 전문 AI 비서입니다. 시세 분석, 계약서 검토, 청약 정보를 도와드립니다.",
		Watermark:      "Powered by Nexus for 부동산",
	},
	{
		ID:             "teacher",
		Name:           "Nexus for 교사",
		Theme:          "#7c3aed",
		DefaultPersona: "teacher",
		Features:       []string{"chat", "search", "files", "doc_summary", "calendar", "browser"},
		WelcomeMsg:     "안녕하세요! 교육 전문 AI 비서입니다. 강의안 작성, 수업 계획, 학생 피드백을 도와드립니다.",
		Watermark:      "Powered by Nexus for 교사",
	},
	{
		ID:             "hr",
		Name:           "Nexus for HR",
		Theme:          "#0ea5e9",
		DefaultPersona: "hr",
		Features:       []string{"chat", "search", "files", "doc_summary", "doc_compare", "calendar", "browser"},
		WelcomeMsg:     "안녕하세요! HR·채용 전문 AI 비서입니다. 이력서 분석, 면접 질문, 채용 공고를 도와드립니다.",
		Watermark:      "Powered by Nexus for HR",
	},
	{
		ID:             "developer",
		Name:           "Nexus for 개발자",
		Theme:          "#6366f1",
		DefaultPersona: "developer",
		Features:       []string{"chat", "search", "files", "doc_summary", "browser", "deep_search", "reddit"},
		WelcomeMsg:     "안녕하세요! 개발자 전문 AI 비서입니다. 코드 리뷰, 버그 분석, GitHub 트렌드를 도와드립니다.",
		Watermark:      "Powered by Nexus for 개발자",
	},
	{
		ID:             "engineer",
		Name:           "Nexus for 엔지니어",
		Theme:          "#64748b",
		DefaultPersona: "engineer",
		Features:       []string{"chat", "search", "files", "doc_summary", "doc_compare", "excel", "browser"},
		WelcomeMsg:     "안녕하세요! 기술 엔지니어 전문 AI 비서입니다. 설계 검토, 규격 검색, 공정 최적화를 도와드립니다.",
		Watermark:      "Powered by Nexus for 엔지니어",
	},
}

// ── Storage ────────────────────────────────────────────────────────

func verticalConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".nexus", "vertical.json")
}

func loadVerticalConfig() VerticalConfig {
	data, err := os.ReadFile(verticalConfigPath())
	if err != nil {
		return verticalPresets[0] // default: general
	}
	var cfg VerticalConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return verticalPresets[0]
	}
	return cfg
}

func saveVerticalConfig(cfg VerticalConfig) error {
	path := verticalConfigPath()
	os.MkdirAll(filepath.Dir(path), 0755)
	data, _ := json.MarshalIndent(cfg, "", "  ")
	return os.WriteFile(path, data, 0600)
}

// ── Handlers ───────────────────────────────────────────────────────

func handleVerticalGetConfig(w http.ResponseWriter, r *http.Request) {
	cfg := loadVerticalConfig()
	json200(w, map[string]any{"ok": true, "config": cfg})
}

func handleVerticalSetConfig(w http.ResponseWriter, r *http.Request) {
	var cfg VerticalConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		http.Error(w, `{"ok":false,"error":"invalid body"}`, 400)
		return
	}
	if cfg.ID == "" {
		http.Error(w, `{"ok":false,"error":"id required"}`, 400)
		return
	}
	if err := saveVerticalConfig(cfg); err != nil {
		http.Error(w, `{"ok":false,"error":"failed to save"}`, 500)
		return
	}
	json200(w, map[string]any{"ok": true, "config": cfg})
}

func handleVerticalPresets(w http.ResponseWriter, r *http.Request) {
	json200(w, map[string]any{"ok": true, "presets": verticalPresets})
}
