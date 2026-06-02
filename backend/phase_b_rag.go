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
func ragContextForLLM(query string) string {
	docs := ragSearch(query, 3)
	if len(docs) == 0 {
		return ""
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
