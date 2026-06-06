// phase_b_rag.go — Phase B1: RAG 엔진 (cross-platform MVP)
// 사장님 요구: "사장님 PC의 모든 것을 안다 — 사용자 문서/이메일/노트 자동 인덱싱"
//
// 흐름:
//   1) 백그라운드: 바탕화면/문서 폴더 파일 → 인덱스 (TF-IDF 기반 lightweight)
//   2) 모든 LLM 호출 전: 관련 문서 top-3 가져와서 prompt에 첨부
//   3) 검색 — "어디 있어"가 아니라 "무슨 내용이야"
//
// 임베딩 모델 없이도 동작 — 텍스트 매칭 + 키워드 가중치
// 실제 임베딩(BGE-M3 등)은 향후 추가

package main

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// IndexedDoc — 인덱스 단일 문서
type IndexedDoc struct {
	Path     string
	Name     string
	Excerpt  string   // 첫 1000자
	Keywords []string // 추출 키워드 (대문자 단어 + 한글 명사 추정)
	Size     int64
	ModTime  int64
	Ext      string
}

var (
	ragIndexMu sync.RWMutex
	ragIndex   = make(map[string]*IndexedDoc) // path → doc
	ragLastBuild time.Time
)

// buildRAGIndex — 사용자 문서 폴더 인덱싱
// 호출: 시작 시 1회 + 1시간마다 1회
func buildRAGIndex() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	targets := []string{
		filepath.Join(home, "Desktop"),
		filepath.Join(home, "Documents"),
		filepath.Join(home, "Downloads"),
	}
	allowed := map[string]bool{
		".txt": true, ".md": true, ".csv": true,
		".docx": true, ".xlsx": true, ".pdf": true,
		".pptx": true, ".html": true, ".json": true,
	}
	newIndex := make(map[string]*IndexedDoc)
	for _, root := range targets {
		filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil || info == nil || info.IsDir() {
				return nil
			}
			ext := strings.ToLower(filepath.Ext(path))
			if !allowed[ext] {
				return nil
			}
			if info.Size() > 10*1024*1024 { // 10MB 초과 스킵
				return nil
			}
			doc := &IndexedDoc{
				Path:    path,
				Name:    info.Name(),
				Size:    info.Size(),
				ModTime: info.ModTime().Unix(),
				Ext:     ext,
			}
			// 텍스트 파일만 본문 추출 (PDF/DOCX 등은 향후 sidecar 활용)
			if ext == ".txt" || ext == ".md" || ext == ".csv" || ext == ".json" || ext == ".html" {
				if data, err := os.ReadFile(path); err == nil {
					content := string(data)
					if len(content) > 4000 {
						content = content[:4000]
					}
					doc.Excerpt = content
					doc.Keywords = extractRAGKeywords(content + " " + doc.Name)
				}
			} else {
				doc.Keywords = extractRAGKeywords(doc.Name)
			}
			newIndex[path] = doc
			return nil
		})
	}
	ragIndexMu.Lock()
	ragIndex = newIndex
	ragLastBuild = time.Now()
	ragIndexMu.Unlock()
}

// extractRAGKeywords — 텍스트에서 의미 키워드 추출
func extractRAGKeywords(text string) []string {
	text = strings.ToLower(text)
	// 단순 토큰화: 공백/특수문자 분리
	tokens := strings.FieldsFunc(text, func(r rune) bool {
		return !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || (r >= '가' && r <= '힣'))
	})
	// 불용어 제거 + 길이 2 이상
	stopwords := map[string]bool{
		"the": true, "and": true, "is": true, "to": true, "of": true, "in": true, "a": true,
		"이": true, "그": true, "저": true, "것": true, "수": true, "은": true, "는": true, "이다": true,
	}
	freq := make(map[string]int)
	for _, t := range tokens {
		if len(t) < 2 || stopwords[t] {
			continue
		}
		freq[t]++
	}
	// 빈도 상위 30개
	type kw struct {
		w string
		c int
	}
	list := make([]kw, 0, len(freq))
	for w, c := range freq {
		list = append(list, kw{w, c})
	}
	sort.Slice(list, func(i, j int) bool { return list[i].c > list[j].c })
	if len(list) > 30 {
		list = list[:30]
	}
	out := make([]string, 0, len(list))
	for _, k := range list {
		out = append(out, k.w)
	}
	return out
}

// ragSearch — 쿼리로 인덱스 검색 → 관련 문서 top-N
func ragSearch(query string, topN int) []*IndexedDoc {
	if topN <= 0 {
		topN = 3
	}
	qTokens := extractRAGKeywords(query)
	if len(qTokens) == 0 {
		return nil
	}
	ragIndexMu.RLock()
	defer ragIndexMu.RUnlock()
	type scored struct {
		doc   *IndexedDoc
		score float64
	}
	results := make([]scored, 0)
	for _, doc := range ragIndex {
		score := 0.0
		for _, q := range qTokens {
			for _, k := range doc.Keywords {
				if k == q {
					score += 1.0
				} else if strings.Contains(k, q) || strings.Contains(q, k) {
					score += 0.4
				}
			}
			// 파일명 매칭 가중치
			if strings.Contains(strings.ToLower(doc.Name), q) {
				score += 2.0
			}
		}
		// 최근성 가중치 (30일 이내 +0.5)
		if time.Now().Unix()-doc.ModTime < 30*86400 {
			score += 0.5
		}
		if score > 0 {
			results = append(results, scored{doc, score})
		}
	}
	sort.Slice(results, func(i, j int) bool { return results[i].score > results[j].score })
	if len(results) > topN {
		results = results[:topN]
	}
	out := make([]*IndexedDoc, 0, len(results))
	for _, r := range results {
		out = append(out, r.doc)
	}
	return out
}

// ragContextForLLM — LLM 프롬프트에 주입할 컨텍스트 문자열 생성
// 호출: 의도 분류 / chat 응답 전에 prompt 앞에 첨부
// ── 내장 지식 베이스 (FAISS 인덱스 없어도 항상 동작) ─────────────────────
// 키워드 매칭 → 관련 항목을 시스템 프롬프트에 자동 주입
var builtinKnowledge = []struct {
	Keywords []string
	Content  string
}{
	{
		Keywords: []string{"nexus", "뭐 해", "뭘 할", "기능", "사용법", "어떻게", "뭐가 가능"},
		Content:  "Nexus가 할 수 있는 것: PC 상태 진단·정리·보안 점검, 웹 검색·가격 비교·뉴스, 파일 찾기·정리, 날씨·번역·유튜브 검색, 캘린더·이메일·메모, 앱 실행·볼륨·와이파이 제어.",
	},
	{
		Keywords: []string{"느려", "느림", "버벅", "렉", "속도", "무거워", "답답"},
		Content:  "PC 속도 개선 순서: 1) '임시파일 정리해줘' → 2) '메모리 많이 쓰는 프로세스 보여줘' → 3) '시작 프로그램 목록 보여줘' → 4) '드라이버 확인해줘'. 대부분 1~2단계에서 해결돼요.",
	},
	{
		Keywords: []string{"보안", "해킹", "바이러스", "악성코드", "이상해", "침입"},
		Content:  "'보안 점검해줘' 또는 '해킹 탐지해줘'라고 하면 바로 스캔해요. 의심스러운 프로세스나 원격 접속 흔적도 확인할 수 있어요.",
	},
	{
		Keywords: []string{"파일", "찾아", "어디 있", "문서", "사진", "엑셀"},
		Content:  "파일 찾기는 '파일 이름 or 키워드 + 찾아줘'로 하면 PC 전체에서 검색해요. 예: '지난달 보고서 찾아줘', '계약서 PDF 찾아줘'.",
	},
	{
		Keywords: []string{"가격", "최저가", "얼마", "쿠팡", "네이버", "쇼핑", "사고 싶"},
		Content:  "가격 비교는 '상품명 + 최저가' 또는 '쿠팡에서 찾아줘'로 검색해요. 다나와·쿠팡·네이버쇼핑 동시 비교도 가능해요.",
	},
	{
		Keywords: []string{"청소", "정리", "용량", "디스크", "공간", "가득"},
		Content:  "'임시파일 정리해줘' 또는 'PC 청소해줘'라고 하면 불필요한 파일을 자동으로 삭제해요. 보통 수백 MB~수 GB 공간을 확보할 수 있어요.",
	},
	{
		Keywords: []string{"일정", "캘린더", "스케줄", "오늘", "미팅", "회의"},
		Content:  "일정 관련: '오늘 일정 알려줘', '이번 주 일정', '미팅 추가해줘 + 날짜/시간' 형태로 말씀해 주세요.",
	},
	{
		Keywords: []string{"번역", "영어로", "한국어로", "일본어", "중국어"},
		Content:  "번역은 '텍스트 + 영어로 번역해줘' 형태로 말씀해 주세요. 한→영, 영→한, 일→한 등 주요 언어 모두 지원해요.",
	},
	{
		Keywords: []string{"요금제", "플랜", "구독", "pro", "업그레이드", "유료"},
		Content:  "Nexus 요금제: Free(기본), Pro($19/월·₩14,900) — AI 요청 무제한·고급 분석·우선 처리. 업그레이드는 설정 > 구독 관리에서 가능해요.",
	},
	{
		Keywords: []string{"api", "groq", "openai", "perplexity", "키", "설정"},
		Content:  "AI 성능 향상: 설정 > AI 키 관리에서 Groq(무료)·OpenAI·Perplexity API 키를 입력하면 더 빠르고 정확해져요.",
	},
}

// builtinKnowledgeContext: 쿼리 키워드와 내장 지식을 매칭해 컨텍스트 문자열 반환
func builtinKnowledgeContext(query string) string {
	queryLower := strings.ToLower(query)
	var matches []string
	seen := map[string]bool{}
	for _, kb := range builtinKnowledge {
		for _, kw := range kb.Keywords {
			if strings.Contains(queryLower, kw) {
				if !seen[kb.Content] {
					matches = append(matches, "• "+kb.Content)
					seen[kb.Content] = true
				}
				break
			}
		}
		if len(matches) >= 2 {
			break // 최대 2개만 주입 (프롬프트 길이 관리)
		}
	}
	if len(matches) == 0 {
		return ""
	}
	return "\n━━━ Nexus 참고 정보 ━━━\n" + strings.Join(matches, "\n") + "\n━━━━━━━━━━━━━━━━━━━━━━\n"
}

func ragContextForLLM(query string) string {
	docs := ragSearch(query, 3)
	if len(docs) == 0 {
		// FAISS 인덱스 비어있음 → 내장 지식 베이스로 폴백
		return builtinKnowledgeContext(query)
	}
	var sb strings.Builder
	sb.WriteString("\n━━━ 관련 사용자 문서 (RAG 컨텍스트) ━━━\n")
	for i, d := range docs {
		excerpt := d.Excerpt
		if len(excerpt) > 500 {
			excerpt = excerpt[:500] + "..."
		}
		sb.WriteString("\n[")
		sb.WriteString(d.Name)
		sb.WriteString("]\n")
		if excerpt != "" {
			sb.WriteString(excerpt)
			sb.WriteString("\n")
		}
		if i >= 2 {
			break
		}
	}
	sb.WriteString("\n━━━ 위 문서를 참고해서 답변하세요. 없으면 일반 지식 사용. ━━━\n")
	return sb.String()
}

// ragStats — 인덱스 통계
func ragStats() map[string]any {
	ragIndexMu.RLock()
	defer ragIndexMu.RUnlock()
	return map[string]any{
		"total_docs":  len(ragIndex),
		"last_build":  ragLastBuild.Unix(),
		"build_age_s": int(time.Since(ragLastBuild).Seconds()),
	}
}

// startRAGBackgroundWatcher — 시작 시 인덱스 빌드 + 1시간마다 재빌드
func startRAGBackgroundWatcher() {
	go func() {
		buildRAGIndex()
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			buildRAGIndex()
		}
	}()
}
