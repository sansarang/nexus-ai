// phase_b_domain_apis.go — Phase B3: 도메인 DB 실연동 (cross-platform)
// 사장님 요구: "법무/의료/투자/개발 페르소나별 신뢰 도메인 API 우선"
//
// API 목록:
//   - 의료: PubMed E-utilities (NCBI, 무료)
//   - 법무: 국가법령정보센터 (law.go.kr, 무료 API)
//   - 투자: DART OpenAPI (한국 공시, 무료)
//   - 개발: GitHub Code Search API (무료 rate limit)
//
// 호출 흐름:
//   detectPersonaForQuery → 매칭 페르소나 → 해당 API 우선 검색 → LLM 컨텍스트로 첨부

package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// DomainSearchResult — 도메인 API 통일 결과 구조
type DomainSearchResult struct {
	Title   string `json:"title"`
	Snippet string `json:"snippet"`
	URL     string `json:"url"`
	Source  string `json:"source"`
	Date    string `json:"date,omitempty"`
	Score   int    `json:"score,omitempty"`
}

var domainHTTPClient = &http.Client{Timeout: 8 * time.Second}

// ── 1. PubMed (의료) ──────────────────────────────────────────────
// E-utilities: https://eutils.ncbi.nlm.nih.gov/entrez/eutils/
// esearch.fcgi → ID 리스트 → esummary.fcgi → 요약 fetch

type pubmedESearchResp struct {
	XMLName  xml.Name `xml:"eSearchResult"`
	IDList   struct {
		IDs []string `xml:"Id"`
	} `xml:"IdList"`
}

func pubmedSearch(query string, maxResults int) []DomainSearchResult {
	if maxResults <= 0 || maxResults > 10 {
		maxResults = 5
	}
	esearchURL := fmt.Sprintf(
		"https://eutils.ncbi.nlm.nih.gov/entrez/eutils/esearch.fcgi?db=pubmed&retmode=xml&retmax=%d&term=%s",
		maxResults, url.QueryEscape(query),
	)
	resp, err := domainHTTPClient.Get(esearchURL)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var es pubmedESearchResp
	if err := xml.Unmarshal(body, &es); err != nil || len(es.IDList.IDs) == 0 {
		return nil
	}
	// 요약 fetch
	idStr := strings.Join(es.IDList.IDs, ",")
	summaryURL := fmt.Sprintf("https://eutils.ncbi.nlm.nih.gov/entrez/eutils/esummary.fcgi?db=pubmed&retmode=json&id=%s", idStr)
	sresp, serr := domainHTTPClient.Get(summaryURL)
	if serr != nil {
		// 요약 실패해도 URL만이라도 반환
		results := make([]DomainSearchResult, 0, len(es.IDList.IDs))
		for _, id := range es.IDList.IDs {
			results = append(results, DomainSearchResult{
				Title:  "PubMed " + id,
				URL:    "https://pubmed.ncbi.nlm.nih.gov/" + id,
				Source: "pubmed.ncbi.nlm.nih.gov",
			})
		}
		return results
	}
	defer sresp.Body.Close()
	sbody, _ := io.ReadAll(sresp.Body)
	var raw struct {
		Result map[string]json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal(sbody, &raw); err != nil {
		return nil
	}
	results := make([]DomainSearchResult, 0, len(es.IDList.IDs))
	for _, id := range es.IDList.IDs {
		if v, ok := raw.Result[id]; ok {
			var doc struct {
				Title    string `json:"title"`
				Source   string `json:"source"`
				PubDate  string `json:"pubdate"`
			}
			if err := json.Unmarshal(v, &doc); err == nil {
				results = append(results, DomainSearchResult{
					Title:   doc.Title,
					Snippet: doc.Source,
					URL:     "https://pubmed.ncbi.nlm.nih.gov/" + id,
					Source:  "pubmed.ncbi.nlm.nih.gov",
					Date:    doc.PubDate,
				})
			}
		}
	}
	return results
}

// ── 2. law.go.kr (법무) ───────────────────────────────────────────
// 국가법령정보센터 — 무료 API (OC 코드 등록 필요 사항 없음, 기본 검색은 공개)
// 실제 API는 키 필요할 수 있어 fallback 으로 검색 결과 페이지 URL 반환

func lawSearch(query string, maxResults int) []DomainSearchResult {
	if maxResults <= 0 || maxResults > 10 {
		maxResults = 5
	}
	// 국가법령정보센터 검색 페이지 (실제 API 키 등록 시 더 정교한 결과)
	apiURL := fmt.Sprintf("https://www.law.go.kr/DRF/lawSearch.do?OC=guest&target=law&query=%s&type=XML&display=%d",
		url.QueryEscape(query), maxResults)
	resp, err := domainHTTPClient.Get(apiURL)
	if err != nil {
		// 폴백: 검색 페이지 URL만
		return []DomainSearchResult{{
			Title:   "법령 검색: " + query,
			URL:     "https://www.law.go.kr/lsSc.do?menuId=1&subMenuId=15&query=" + url.QueryEscape(query),
			Source:  "law.go.kr",
			Snippet: "국가법령정보센터에서 검색하세요.",
		}}
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	// XML 파싱 — 간단한 구조만 추출
	type lawItem struct {
		Name     string `xml:"법령명한글"`
		ID       string `xml:"법령ID"`
		Date     string `xml:"공포일자"`
		Category string `xml:"법령구분명"`
	}
	type lawResp struct {
		XMLName xml.Name  `xml:"LawSearch"`
		Laws    []lawItem `xml:"law"`
	}
	var lr lawResp
	if err := xml.Unmarshal(body, &lr); err != nil || len(lr.Laws) == 0 {
		return []DomainSearchResult{{
			Title:   "법령 검색: " + query,
			URL:     "https://www.law.go.kr/lsSc.do?menuId=1&subMenuId=15&query=" + url.QueryEscape(query),
			Source:  "law.go.kr",
			Snippet: "국가법령정보센터 검색 결과 페이지.",
		}}
	}
	results := make([]DomainSearchResult, 0, len(lr.Laws))
	for _, l := range lr.Laws {
		results = append(results, DomainSearchResult{
			Title:   l.Name,
			Snippet: l.Category,
			URL:     "https://www.law.go.kr/lsInfoP.do?lsiSeq=" + l.ID,
			Source:  "law.go.kr",
			Date:    l.Date,
		})
	}
	return results
}

// ── 3. DART (한국 공시, 투자/재무) ────────────────────────────────
// DART OpenAPI: https://opendart.fss.or.kr — API 키 필요 (무료, 일 10000건)
// 사장님 등록 안 했을 가능성 → fallback URL 반환

func dartSearch(corpName string, maxResults int) []DomainSearchResult {
	if maxResults <= 0 || maxResults > 10 {
		maxResults = 5
	}
	// DART 키 환경변수 또는 빌드 임베딩 확인 (없으면 검색 페이지로)
	dartKey := getEnvKey("DART_API_KEY")
	if dartKey == "" {
		return []DomainSearchResult{{
			Title:   "DART 검색: " + corpName,
			URL:     "https://dart.fss.or.kr/dsab007/main.do?textCrpNm=" + url.QueryEscape(corpName),
			Source:  "dart.fss.or.kr",
			Snippet: "전자공시 시스템에서 검색하세요. (API 키 미등록)",
		}}
	}
	// API 호출 (실제 운영 시)
	apiURL := fmt.Sprintf("https://opendart.fss.or.kr/api/list.json?crtfc_key=%s&corp_code=&bgn_de=&end_de=&page_count=%d", dartKey, maxResults)
	_ = apiURL // 실제 호출은 corp_code 매핑 필요 (별도 단계)
	return []DomainSearchResult{{
		Title:   "DART 공시: " + corpName,
		URL:     "https://dart.fss.or.kr/dsab007/main.do?textCrpNm=" + url.QueryEscape(corpName),
		Source:  "dart.fss.or.kr",
	}}
}

// ── 4. GitHub Code Search (개발) ──────────────────────────────────

func githubCodeSearch(query string, maxResults int) []DomainSearchResult {
	if maxResults <= 0 || maxResults > 10 {
		maxResults = 5
	}
	apiURL := fmt.Sprintf("https://api.github.com/search/code?q=%s&per_page=%d", url.QueryEscape(query), maxResults)
	req, _ := http.NewRequest("GET", apiURL, nil)
	req.Header.Set("Accept", "application/vnd.github+json")
	if tok := getEnvKey("GITHUB_TOKEN"); tok != "" {
		req.Header.Set("Authorization", "token "+tok)
	}
	resp, err := domainHTTPClient.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var gr struct {
		Items []struct {
			Name       string `json:"name"`
			HTMLURL    string `json:"html_url"`
			Repository struct {
				FullName string `json:"full_name"`
			} `json:"repository"`
		} `json:"items"`
	}
	if err := json.Unmarshal(body, &gr); err != nil {
		return nil
	}
	results := make([]DomainSearchResult, 0, len(gr.Items))
	for _, it := range gr.Items {
		results = append(results, DomainSearchResult{
			Title:   it.Name,
			Snippet: it.Repository.FullName,
			URL:     it.HTMLURL,
			Source:  "github.com",
		})
	}
	return results
}

// getEnvKey — 환경변수 헬퍼 (없으면 빈 문자열)
func getEnvKey(key string) string {
	return strings.TrimSpace(osGetenv(key))
}

func osGetenv(k string) string { return os.Getenv(k) }

// domainSearchForPersona — 페르소나 기반 자동 도메인 검색 디스패치
// 호출: LLM 응답 생성 전 컨텍스트 보강용
func domainSearchForPersona(personaID, query string) []DomainSearchResult {
	switch personaID {
	case "medical":
		return pubmedSearch(query, 5)
	case "legal":
		return lawSearch(query, 5)
	case "investor", "finance":
		return dartSearch(query, 5)
	case "developer":
		return githubCodeSearch(query, 5)
	}
	return nil
}

// domainContextForLLM — LLM 프롬프트에 주입할 도메인 컨텍스트
func domainContextForLLM(personaID, query string) string {
	results := domainSearchForPersona(personaID, query)
	if len(results) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("\n━━━ 도메인 신뢰 출처 (자동 검색) ━━━\n")
	for i, r := range results {
		sb.WriteString(fmt.Sprintf("%d. [%s] %s — %s\n", i+1, r.Source, r.Title, r.URL))
		if r.Snippet != "" {
			sb.WriteString("   ")
			sb.WriteString(r.Snippet)
			sb.WriteString("\n")
		}
	}
	sb.WriteString("━━━ 위 출처 인용해서 답변하세요. 환각 금지. ━━━\n")
	return sb.String()
}
