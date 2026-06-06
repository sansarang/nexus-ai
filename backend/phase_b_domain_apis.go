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

var domainHTTPClient = &http.Client{Timeout: 3 * time.Second}

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

// ── 5. 국세청 사업자 상태 조회 (NTS OpenAPI) ──────────────────────
// https://api.odcloud.kr/api/nts-businessman/v1/status
// 무료 API 키: data.go.kr 회원가입 후 발급 (ODCLOUD_API_KEY 환경변수)
// 키 없으면: 홈택스 웹 링크 + 프롬프트 기반 안내로 폴백

type NTSBusinessResult struct {
	BusinessNo     string `json:"b_no"`
	StatusCode     string `json:"b_stt_cd"` // "01"=계속, "02"=휴업, "03"=폐업
	Status         string `json:"b_stt"`    // 계속사업자/휴업자/폐업자
	TaxType        string `json:"tax_type"`
	TaxTypeName    string `json:"tax_type_nm"`
	EndDate        string `json:"end_dt,omitempty"`
	UTCTime        string `json:"utcc_yn"`
}

type ntsAPIResp struct {
	StatusCode string              `json:"status_code"`
	Data       []NTSBusinessResult `json:"data"`
}

// LookupBusinessNumber: 사업자등록번호로 국세청 API 조회
// 반환: status(계속/휴업/폐업), taxType, 오류시 에러
func LookupBusinessNumber(brno string) (*NTSBusinessResult, error) {
	// 숫자만 추출 (하이픈 제거)
	cleaned := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, brno)
	if len(cleaned) != 10 {
		return nil, fmt.Errorf("사업자등록번호는 10자리여야 합니다 (입력: %s)", brno)
	}

	// 키 우선순위: VerticalAPIKeys.DataGOKR → 환경변수 ODCLOUD_API_KEY (공공데이터포털 통합키)
	apiKey := loadVerticalAPIKeys().DataGOKR
	if apiKey == "" {
		apiKey = getEnvKey("ODCLOUD_API_KEY")
	}
	if apiKey == "" {
		// 폴백: 홈택스 링크 안내
		return &NTSBusinessResult{
			BusinessNo: cleaned,
			Status:     "홈택스 확인 필요",
			TaxType:    fmt.Sprintf("https://teht.hometax.go.kr/websquare/websquare.wq?w2xPath=/ui/ab/a/a/UTEABAAA13.xml&qlSltPrno=%s", cleaned),
		}, fmt.Errorf("ODCLOUD_API_KEY 미설정 — 홈택스 직접 조회 필요")
	}

	payload := map[string]any{"b_no": []string{cleaned}}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST",
		"https://api.odcloud.kr/api/nts-businessman/v1/status?serviceKey="+apiKey,
		strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("국세청 API 호출 실패: %w", err)
	}
	defer resp.Body.Close()
	rawBody, _ := io.ReadAll(resp.Body)

	var result ntsAPIResp
	if err := json.Unmarshal(rawBody, &result); err != nil {
		return nil, fmt.Errorf("국세청 API 응답 파싱 실패: %w", err)
	}
	if len(result.Data) == 0 {
		return nil, fmt.Errorf("조회 결과 없음 (사업자번호 %s)", cleaned)
	}

	item := result.Data[0]
	// 상태 코드 한글 변환
	switch item.StatusCode {
	case "01":
		item.Status = "✅ 계속사업자 (정상)"
	case "02":
		item.Status = "⚠️ 휴업자"
	case "03":
		item.Status = "❌ 폐업자"
	}
	return &item, nil
}

// FormatBusinessLookupResult: LLM 응답에 삽입할 포맷 문자열 생성
func FormatBusinessLookupResult(brno string) string {
	result, err := LookupBusinessNumber(brno)
	if err != nil {
		if result != nil && result.TaxType != "" {
			// 키 없는 폴백 — 링크 안내
			return fmt.Sprintf(
				"🏢 사업자번호 %s\n\n국세청 API 키가 설정되지 않았습니다.\n직접 확인: %s",
				brno, result.TaxType)
		}
		return fmt.Sprintf("❌ 사업자 조회 실패: %v", err)
	}

	sb := &strings.Builder{}
	sb.WriteString(fmt.Sprintf("🏢 사업자 조회 결과\n\n"))
	sb.WriteString(fmt.Sprintf("📌 사업자번호: %s\n", result.BusinessNo))
	sb.WriteString(fmt.Sprintf("📊 사업자 상태: %s\n", result.Status))
	if result.TaxTypeName != "" {
		sb.WriteString(fmt.Sprintf("💼 과세유형: %s\n", result.TaxTypeName))
	}
	if result.EndDate != "" && result.StatusCode == "03" {
		sb.WriteString(fmt.Sprintf("📅 폐업일: %s\n", result.EndDate))
	}
	sb.WriteString("\n⚠️ 출처: 국세청 NTS OpenAPI")
	return sb.String()
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
