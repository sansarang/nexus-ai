package main

import (
	"fmt"
	"strings"
)

// classifyItemType: URL과 소스명으로 콘텐츠 타입 판별
func classifyItemType(url, source string) string {
	lower := strings.ToLower(url)
	switch {
	case strings.Contains(lower, "youtube.com") || strings.Contains(lower, "youtu.be"):
		return "video"
	case strings.Contains(lower, "tv.naver.com") || strings.Contains(lower, "vod.mbc.co.kr") ||
		strings.Contains(lower, "vodmall.imbc.com") || strings.Contains(lower, "sbs.co.kr/news/video") ||
		strings.Contains(lower, "tving.com") || strings.Contains(lower, "wavve.com") ||
		strings.Contains(lower, "netflix.com") || strings.Contains(lower, "laftel.net") ||
		strings.Contains(lower, "vimeo.com") || strings.Contains(lower, "dailymotion.com"):
		return "video"
	case source == "youtube" || source == "video":
		return "video"
	case strings.Contains(lower, "news") || strings.Contains(lower, "/article/") ||
		strings.Contains(lower, "yna.co.kr") || strings.Contains(lower, "chosun.com") ||
		strings.Contains(lower, "joongang.co.kr") || strings.Contains(lower, "hani.co.kr") ||
		strings.Contains(lower, "donga.com") || strings.Contains(lower, "mbc.co.kr/news") ||
		strings.Contains(lower, "kbs.co.kr/news") || strings.Contains(lower, "sbs.co.kr/news"):
		return "news"
	case strings.Contains(lower, ".pdf"):
		return "document"
	default:
		return "article"
	}
}

// queryKeywords: 쿼리에서 2글자 이상 유의미한 단어 추출
func queryKeywords(query string) []string {
	stopWords := map[string]bool{
		"에서": true, "에서의": true, "가장": true, "제일": true, "좀": true,
		"찾아줘": true, "보여줘": true, "알려줘": true, "검색해줘": true,
		"싼": true, "비싼": true, "좋은": true, "추천": true, "최저": true,
		"저렴한": true, "저렴": true, "구매": true, "구입": true, "어디서": true,
		"중에서": true, "중": true, "것": true, "거": true, "걸": true,
	}
	words := strings.Fields(query)
	var keywords []string
	for _, w := range words {
		w = strings.TrimSpace(w)
		if len([]rune(w)) >= 2 && !stopWords[w] {
			keywords = append(keywords, strings.ToLower(w))
		}
	}
	return keywords
}

// titleMatchesQuery: 제목이 쿼리 키워드 중 하나 이상 포함하는지 확인
func titleMatchesQuery(title string, keywords []string) bool {
	if len(keywords) == 0 {
		return true
	}
	titleLower := strings.ToLower(title)
	for _, kw := range keywords {
		if strings.Contains(titleLower, kw) {
			return true
		}
	}
	return false
}

// normalizeSite: 사이트 이름을 도메인 형식으로 정규화
func normalizeSite(site string) string {
	aliases := map[string]string{
		"youtube":   "youtube.com",
		"tiktok":    "tiktok.com",
		"temu":      "temu.com",
		"coupang":   "coupang.com",
		"naver":     "naver.com",
		"google":    "google.com",
		"danawa":    "danawa.com",
		"gmarket":   "gmarket.co.kr",
		"11st":      "11st.co.kr",
		"11번가":      "11st.co.kr",
		"auction":   "auction.co.kr",
		"옥션":        "auction.co.kr",
		"auto":      "coupang.com",
		"":          "coupang.com",
	}
	if normalized, ok := aliases[site]; ok {
		return normalized
	}
	return site
}

// buildSearchURL: 사이트별 검색 URL 생성
func buildSearchURL(site, query string) string {
	site = normalizeSite(site)
	encoded := strings.ReplaceAll(query, " ", "+")
	searchURLs := map[string]string{
		// 쇼핑몰
		"coupang.com":        fmt.Sprintf("https://www.coupang.com/np/search?q=%s", encoded),
		"naver.com":          fmt.Sprintf("https://search.naver.com/search.naver?query=%s", encoded),
		"shopping.naver.com": fmt.Sprintf("https://search.shopping.naver.com/search/all?query=%s", encoded),
		"google.com":         fmt.Sprintf("https://www.google.com/search?q=%s&hl=ko", encoded),
		"danawa.com":         fmt.Sprintf("https://search.danawa.com/dsearch.php?query=%s", encoded),
		"gmarket.co.kr":      fmt.Sprintf("https://browse.gmarket.co.kr/search?keyword=%s", encoded),
		"youtube.com":        fmt.Sprintf("https://www.youtube.com/results?search_query=%s", encoded),
		"tiktok.com":         fmt.Sprintf("https://www.tiktok.com/search?q=%s", encoded),
		"temu.com":           fmt.Sprintf("https://www.temu.com/search_result.html?search_key=%s&refer_page_name=home", encoded),
		"11st.co.kr":         fmt.Sprintf("https://search.11st.co.kr/Search.tmall?kwd=%s", encoded),
		"auction.co.kr":      fmt.Sprintf("https://www.auction.co.kr/search/list.aspx?keyword=%s", encoded),
		"musinsa.com":        fmt.Sprintf("https://www.musinsa.com/search/musinsa/integration?q=%s", encoded),
		"a-bly.com":          fmt.Sprintf("https://a-bly.com/search?keyword=%s", encoded),
		"zigzag.kr":          fmt.Sprintf("https://zigzag.kr/search?q=%s", encoded),
		"ohou.se":            fmt.Sprintf("https://ohou.se/search?query=%s", encoded),
		"aliexpress.com":     fmt.Sprintf("https://www.aliexpress.com/wholesale?SearchText=%s", encoded),
		"amazon.com":         fmt.Sprintf("https://www.amazon.com/s?k=%s", encoded),
		// 중고차
		"heydealer.com":   fmt.Sprintf("https://www.heydealer.com/car/search?keyword=%s", encoded),
		"encar.com":       fmt.Sprintf("https://www.encar.com/search/car?searchKey=%s", encoded),
		"kbchachacha.com": fmt.Sprintf("https://www.kbchachacha.com/public/car/list.kbc?keyword=%s", encoded),
		"bobaedream.co.kr": fmt.Sprintf("https://www.bobaedream.co.kr/search?search_params=%s", encoded),
		// 중고거래
		"daangn.com":   fmt.Sprintf("https://www.daangn.com/search/%s", strings.ReplaceAll(query, " ", "%20")),
		"bunjang.co.kr": fmt.Sprintf("https://m.bunjang.co.kr/search/products?q=%s", encoded),
		"joongna.com":  fmt.Sprintf("https://web.joongna.com/search/%s", encoded),
		// 부동산
		"zigbang.com":    fmt.Sprintf("https://www.zigbang.com/search?q=%s", encoded),
		"dabangapp.com":  fmt.Sprintf("https://www.dabangapp.com/map/oneroom?search_type=keyword&keyword=%s", encoded),
		"land.naver.com": fmt.Sprintf("https://land.naver.com/search/search.nhn?query=%s", encoded),
		// 여행/숙박
		"yanolja.com":   fmt.Sprintf("https://www.yanolja.com/keyword/%s", encoded),
		"goodchoice.kr": fmt.Sprintf("https://www.goodchoice.kr/product/search?keyword=%s", encoded),
		"airbnb.com":    fmt.Sprintf("https://www.airbnb.co.kr/s/%s/homes", encoded),
		// 배달
		"baemin.com":    fmt.Sprintf("https://www.baemin.com/search?query=%s", encoded),
		"yogiyo.co.kr":  fmt.Sprintf("https://www.yogiyo.co.kr/search?keyword=%s", encoded),
	}
	if url, ok := searchURLs[site]; ok {
		return url
	}
	return fmt.Sprintf("https://www.google.com/search?q=%s&hl=ko", encoded)
}
