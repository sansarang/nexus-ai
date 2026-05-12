package main

import (
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"
)

// extractDomain: URL에서 도메인명만 추출
func extractDomain(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" {
		if len(rawURL) > 50 {
			return rawURL[:50] + "..."
		}
		return rawURL
	}
	host := strings.TrimPrefix(u.Host, "www.")
	return host
}

// categoryPreviewType: 카테고리 → 프론트엔드 preview_type 문자열
func categoryPreviewType(cat queryCategory) string {
	switch cat {
	case catWeather:
		return "weather"
	case catNews:
		return "news"
	case catRecipe:
		return "recipe"
	case catShopping:
		return "shopping"
	case catTransit:
		return "transit"
	case catFood:
		return "food"
	case catFinance:
		return "finance"
	case catMedical:
		return "medical"
	case catTravel:
		return "travel"
	case catEntertainment:
		return "entertainment"
	case catTech:
		return "tech"
	case catEducation:
		return "education"
	case catRealEstate:
		return "realestate"
	case catLegal:
		return "legal"
	default:
		return "general"
	}
}

// ── 카테고리 감지 ──────────────────────────────────────────────────

type queryCategory int

const (
	catGeneral queryCategory = iota
	catTransit       // 교통 (버스/기차/지하철/항공)
	catFood          // 맛집/음식/배달
	catShopping      // 쇼핑/최저가
	catFinance       // 금융 (환율/주가/코인/보험)
	catWeather       // 날씨
	catNews          // 뉴스/시사
	catMedical       // 의료/건강/병원/약
	catLegal         // 법률/행정/세금
	catEntertainment // 영화/공연/스포츠/게임
	catRecipe        // 요리/레시피
	catTravel        // 여행/숙박/관광
	catTech          // IT/기술/소프트웨어
	catEducation     // 교육/자격증/시험
	catRealEstate    // 부동산/아파트/전세
)

func detectCategory(q string) queryCategory {
	lower := strings.ToLower(q)
	has := func(words ...string) bool {
		for _, w := range words {
			if strings.Contains(lower, w) {
				return true
			}
		}
		return false
	}
	switch {
	case has(
		// 한국어
		"버스", "고속버스", "시외버스", "ktx", "기차", "열차", "지하철", "전철", "항공", "비행기", "시간표", "노선", "예매", "승차권", "터미널", "공항", "택시요금", "어떻게가", "길찾기", "경로",
		// 영어
		"how to get", "directions from", "directions to", "route from", "route to",
		"bus from", "bus to", "subway", "metro", "transit", "commute", "train to",
		"flight to", "fly to", "airport", "travel to", "get to",
	):
		return catTransit
	case has(
		// 한국어
		"맛집", "식당", "카페", "음식점", "배달", "메뉴", "맛있는", "뭐먹", "점심", "저녁", "아침메뉴", "밥집", "분식", "치킨", "피자", "초밥", "삼겹살",
		// 영어
		"restaurant", "food near", "eat", "dining", "cafe", "lunch", "dinner", "breakfast",
		"pizza", "sushi", "burger", "yelp", "tripadvisor food", "delivery food", "takeout",
	):
		return catFood
	case has(
		// 한국어
		"최저가", "가격비교", "쿠팡", "네이버쇼핑", "11번가", "지마켓", "옥션", "테무", "알리", "사고싶", "얼마", "싸게", "할인", "쇼핑",
		// 영어
		"buy", "purchase", "shop", "cheapest", "price", "amazon", "ebay", "etsy", "walmart",
		"discount", "sale", "deal", "coupon", "cheap", "where to buy", "how much",
	):
		return catShopping
	case has(
		// 한국어
		"환율", "주가", "코스피", "코스닥", "비트코인", "코인", "달러", "엔화", "유로", "금리", "적금", "예금", "펀드", "주식", "etf", "나스닥", "다우",
		// 영어
		"stock", "bitcoin", "crypto", "exchange rate", "dollar", "euro", "investment",
		"nasdaq", "dow jones", "s&p", "forex", "interest rate", "inflation", "bank rate",
		"portfolio", "dividend", "ipo", "market cap", "hedge fund",
	):
		return catFinance
	case has(
		// 한국어
		"날씨", "기온", "강수", "황사", "미세먼지",
		// 영어
		"weather", "temperature", "rain", "snow", "sunny", "cloudy", "forecast",
		"humidity", "wind speed", "uv index", "storm", "hurricane",
	):
		return catWeather
	case has(
		// 한국어
		"뉴스", "속보", "이슈", "사건", "사고", "정치", "경제뉴스", "오늘뉴스", "최신뉴스",
		// 영어
		"news", "breaking news", "latest news", "politics", "headline", "bbc", "cnn",
		"reuters", "today's news", "current events",
	):
		return catNews
	case has(
		// 한국어
		"병원", "약국", "증상", "진료", "건강", "의사", "처방", "아파", "질환", "치료", "수술",
		// 영어
		"symptom", "hospital", "doctor", "medicine", "health", "pain", "treatment",
		"disease", "diagnosis", "pharmacy", "clinic", "headache", "fever", "webmd",
	):
		return catMedical
	case has(
		// 한국어
		"세금", "신고", "홈택스", "법률", "법원", "법적", "규정", "벌금", "과태료", "주민등록", "정부24", "민원", "행정", "소송", "판결", "변호사", "법무",
		// 영어
		"tax", "law ", "regulation", "fine penalty", "government service", "visa application", "immigration",
		"passport application", "irs", "tax filing", "legal advice", "lawsuit", "attorney",
	):
		return catLegal
	case has(
		// 한국어
		"영화", "cgv", "롯데시네마", "메가박스", "공연", "뮤지컬", "콘서트", "야구", "축구", "농구", "kbo", "스포츠", "경기결과", "게임",
		// 영어
		"movie", "film", "cinema", "theater", "concert", "sports", "game", "nba", "nfl",
		"mlb", "soccer", "basketball", "imdb", "rotten tomatoes", "netflix original",
		"streaming", "show", "series", "episode",
	):
		return catEntertainment
	case has(
		// 한국어
		"레시피", "만드는법", "요리법", "재료", "칼로리", "조리", "맛있게", "만드는 방법", "끓이는", "볶는", "굽는", "요리하는", "음식만들기", "집밥", "반찬",
		// 영어
		"recipe", "how to cook", "how to make", "ingredients", "calories", "cooking",
		"bake", "grill", "fry", "meal prep", "delicious", "homemade",
	):
		return catRecipe
	case has(
		// 한국어
		"여행", "호텔", "숙박", "관광", "펜션", "에어비앤비", "여행지", "해외여행", "국내여행",
		// 영어
		"travel", "hotel", "vacation", "airbnb", "tourism", "flight booking", "trip",
		"resort", "booking.com", "expedia", "tourist", "sightseeing", "itinerary",
	):
		return catTravel
	case has(
		// 한국어
		"설치방법", "오류", "에러", "버그", "프로그램", "앱", "소프트웨어", "윈도우", "맥", "리눅스", "코딩", "개발", "api",
		// 영어
		"error", "install", "software", "app", "code", "programming", "api", "windows",
		"macos", "linux", "debug", "fix", "github", "stackoverflow", "javascript", "python",
		"how to code", "tutorial", "documentation",
	):
		return catTech
	case has(
		// 한국어
		"자격증", "시험", "공부", "강의", "학원", "토익", "토플", "수능", "대학교", "입시",
		// 영어
		"study", "exam", "course", "university", "ielts", "toefl", "learn", "tutorial",
		"coursera", "udemy", "online course", "certification", "degree", "college",
	):
		return catEducation
	case has(
		// 한국어
		"아파트", "전세", "월세", "매매", "부동산", "청약", "분양", "집값", "임대",
		// 영어
		"apartment", "rent", "buy house", "real estate", "property", "mortgage",
		"zillow", "realtor", "condo", "studio", "lease",
	):
		return catRealEstate
	default:
		return catGeneral
	}
}

// optimizeQuery: 카테고리에 맞게 검색 쿼리 최적화
func optimizeQuery(original string, cat queryCategory) string {
	today := time.Now().Format("2006년 1월")
	q := original
	switch cat {
	case catTransit:
		if !strings.Contains(q, "시간표") && !strings.Contains(q, "요금") && !strings.Contains(q, "예매") {
			q += " 시간표 요금"
		}
		q += " " + time.Now().Format("2006")
	case catFinance:
		q += " " + today + " 실시간"
	case catNews:
		q += " " + today
	case catFood:
		if !strings.Contains(q, "추천") && !strings.Contains(q, "맛집") {
			q += " 추천"
		}
	case catMedical:
		q += " 원인 증상 치료"
	case catRealEstate:
		q += " " + today
	case catEntertainment:
		if strings.Contains(q, "야구") || strings.Contains(q, "축구") || strings.Contains(q, "농구") {
			q += " " + time.Now().Format("2006") + " 일정 결과"
		}
	}
	return q
}

// categoryFallbackSites: 카테고리별 공식 사이트 링크 DB
func categoryFallbackSites(query string, cat queryCategory) []map[string]string {
	enc := urlEncode(query)
	isEng := isEnglishQuery(query)
	naverSearch := map[string]string{"title": "네이버 검색: " + query, "url": "https://search.naver.com/search.naver?query=" + enc}
	googleSearch := map[string]string{"title": "Google Search: " + query, "url": "https://www.google.com/search?q=" + enc}

	lower := strings.ToLower(query)

	if isEng {
		switch cat {
		case catTransit:
			isAir := strings.Contains(lower, "flight") || strings.Contains(lower, "airport") || strings.Contains(lower, "airline")
			isSubway := strings.Contains(lower, "subway") || strings.Contains(lower, "metro") || strings.Contains(lower, "tube")
			if isAir {
				return []map[string]string{
					{"title": "Google Flights: " + query, "url": "https://www.google.com/flights?q=" + enc},
					{"title": "Skyscanner", "url": "https://www.skyscanner.com/"},
					{"title": "Kayak Flights", "url": "https://www.kayak.com/flights"},
					{"title": "Expedia Flights", "url": "https://www.expedia.com/Flights-Search?trip=roundTrip&leg1=from:" + enc},
					googleSearch,
				}
			}
			if isSubway {
				return []map[string]string{
					{"title": "Google Maps Transit: " + query, "url": "https://www.google.com/maps/dir/?api=1&travelmode=transit&destination=" + enc},
					{"title": "Rome2rio: " + query, "url": "https://www.rome2rio.com/s/" + enc},
					googleSearch,
				}
			}
			return []map[string]string{
				{"title": "Google Maps Directions: " + query, "url": "https://www.google.com/maps/dir/?api=1&destination=" + enc},
				{"title": "Rome2rio: " + query, "url": "https://www.rome2rio.com/s/" + enc},
				{"title": "Greyhound Bus", "url": "https://www.greyhound.com/"},
				{"title": "Amtrak Train", "url": "https://www.amtrak.com/"},
				googleSearch,
			}
		case catFood:
			return []map[string]string{
				{"title": "Yelp: " + query, "url": "https://www.yelp.com/search?find_desc=" + enc},
				{"title": "Google Maps Restaurants: " + query, "url": "https://www.google.com/maps/search/restaurants+" + enc},
				{"title": "TripAdvisor: " + query, "url": "https://www.tripadvisor.com/Search?q=" + enc},
				{"title": "DoorDash", "url": "https://www.doordash.com/"},
				{"title": "UberEats", "url": "https://www.ubereats.com/"},
			}
		case catShopping:
			return []map[string]string{
				{"title": "Amazon: " + query, "url": "https://www.amazon.com/s?k=" + enc},
				{"title": "eBay: " + query, "url": "https://www.ebay.com/sch/i.html?_nkw=" + enc},
				{"title": "Walmart: " + query, "url": "https://www.walmart.com/search?q=" + enc},
				{"title": "Target: " + query, "url": "https://www.target.com/s?searchTerm=" + enc},
				{"title": "Etsy: " + query, "url": "https://www.etsy.com/search?q=" + enc},
			}
		case catFinance:
			isCrypto := strings.Contains(lower, "bitcoin") || strings.Contains(lower, "crypto") || strings.Contains(lower, "ethereum") || strings.Contains(lower, "coin")
			isStock := strings.Contains(lower, "stock") || strings.Contains(lower, "nasdaq") || strings.Contains(lower, "s&p") || strings.Contains(lower, "dow")
			isForex := strings.Contains(lower, "exchange rate") || strings.Contains(lower, "forex") || strings.Contains(lower, "usd") || strings.Contains(lower, "eur")
			if isCrypto {
				return []map[string]string{
					{"title": "CoinMarketCap: " + query, "url": "https://coinmarketcap.com/"},
					{"title": "CoinGecko: " + query, "url": "https://www.coingecko.com/en/search?query=" + enc},
					{"title": "Binance", "url": "https://www.binance.com/en/markets/overview"},
					googleSearch,
				}
			}
			if isStock {
				return []map[string]string{
					{"title": "Yahoo Finance: " + query, "url": "https://finance.yahoo.com/quote/" + enc},
					{"title": "Google Finance: " + query, "url": "https://www.google.com/finance/quote/" + enc},
					{"title": "Bloomberg: " + query, "url": "https://www.bloomberg.com/search?query=" + enc},
					{"title": "NASDAQ", "url": "https://www.nasdaq.com/market-activity/stocks"},
				}
			}
			if isForex {
				return []map[string]string{
					{"title": "XE Currency: " + query, "url": "https://www.xe.com/currencyconverter/"},
					{"title": "Google Finance Forex", "url": "https://www.google.com/finance/"},
					{"title": "Bloomberg FX", "url": "https://www.bloomberg.com/markets/currencies"},
				}
			}
			return []map[string]string{
				{"title": "Yahoo Finance: " + query, "url": "https://finance.yahoo.com/lookup?s=" + enc},
				{"title": "Bloomberg: " + query, "url": "https://www.bloomberg.com/search?query=" + enc},
				googleSearch,
			}
		case catWeather:
			return []map[string]string{
				{"title": "Weather.com: " + query, "url": "https://weather.com/weather/today/l/" + enc},
				{"title": "AccuWeather: " + query, "url": "https://www.accuweather.com/en/search-locations?query=" + enc},
				{"title": "National Weather Service", "url": "https://www.weather.gov/"},
				googleSearch,
			}
		case catNews:
			return []map[string]string{
				{"title": "Google News: " + query, "url": "https://news.google.com/search?q=" + enc + "&hl=en"},
				{"title": "BBC News: " + query, "url": "https://www.bbc.com/search?q=" + enc},
				{"title": "Reuters: " + query, "url": "https://www.reuters.com/search/news?blob=" + enc},
				{"title": "AP News: " + query, "url": "https://apnews.com/search?q=" + enc},
				{"title": "CNN: " + query, "url": "https://edition.cnn.com/search?q=" + enc},
			}
		case catMedical:
			return []map[string]string{
				{"title": "WebMD: " + query, "url": "https://www.webmd.com/search/search_results/default.aspx?query=" + enc},
				{"title": "Mayo Clinic: " + query, "url": "https://www.mayoclinic.org/search/search-results?q=" + enc},
				{"title": "Healthline: " + query, "url": "https://www.healthline.com/search?q1=" + enc},
				{"title": "NHS Health A-Z", "url": "https://www.nhs.uk/conditions/"},
				googleSearch,
			}
		case catLegal:
			return []map[string]string{
				{"title": "IRS (US Tax): " + query, "url": "https://www.irs.gov/search?query=" + enc},
				{"title": "USA.gov: " + query, "url": "https://www.usa.gov/search?query=" + enc},
				{"title": "LegalZoom", "url": "https://www.legalzoom.com/"},
				{"title": "FindLaw: " + query, "url": "https://www.findlaw.com/search/?q=" + enc},
				googleSearch,
			}
		case catEntertainment:
			isMovie := strings.Contains(lower, "movie") || strings.Contains(lower, "film") || strings.Contains(lower, "cinema") || strings.Contains(lower, "theater")
			isNBA := strings.Contains(lower, "nba") || strings.Contains(lower, "basketball")
			isNFL := strings.Contains(lower, "nfl") || strings.Contains(lower, "football")
			isConcert := strings.Contains(lower, "concert") || strings.Contains(lower, "ticket") || strings.Contains(lower, "show")
			if isMovie {
				return []map[string]string{
					{"title": "IMDb: " + query, "url": "https://www.imdb.com/find?q=" + enc},
					{"title": "Fandango: " + query, "url": "https://www.fandango.com/search?q=" + enc},
					{"title": "Rotten Tomatoes: " + query, "url": "https://www.rottentomatoes.com/search?search=" + enc},
					{"title": "Netflix", "url": "https://www.netflix.com/search?q=" + enc},
				}
			}
			if isNBA {
				return []map[string]string{
					{"title": "NBA Official: " + query, "url": "https://www.nba.com/search?q=" + enc},
					{"title": "ESPN NBA: " + query, "url": "https://www.espn.com/nba/"},
					googleSearch,
				}
			}
			if isNFL {
				return []map[string]string{
					{"title": "NFL Official: " + query, "url": "https://www.nfl.com/"},
					{"title": "ESPN NFL: " + query, "url": "https://www.espn.com/nfl/"},
					googleSearch,
				}
			}
			if isConcert {
				return []map[string]string{
					{"title": "Ticketmaster: " + query, "url": "https://www.ticketmaster.com/search?q=" + enc},
					{"title": "StubHub: " + query, "url": "https://www.stubhub.com/find/s/?q=" + enc},
					{"title": "SeatGeek: " + query, "url": "https://seatgeek.com/search?search_query=" + enc},
				}
			}
			return []map[string]string{
				{"title": "ESPN: " + query, "url": "https://www.espn.com/search/results?q=" + enc},
				{"title": "IMDb: " + query, "url": "https://www.imdb.com/find?q=" + enc},
				googleSearch,
			}
		case catRecipe:
			return []map[string]string{
				{"title": "AllRecipes: " + query, "url": "https://www.allrecipes.com/search?q=" + enc},
				{"title": "Food Network: " + query, "url": "https://www.foodnetwork.com/search/" + enc + "-"},
				{"title": "Epicurious: " + query, "url": "https://www.epicurious.com/search/" + enc},
				{"title": "YouTube recipes: " + query, "url": "https://www.youtube.com/results?search_query=" + enc + "+recipe"},
				googleSearch,
			}
		case catTravel:
			return []map[string]string{
				{"title": "Booking.com: " + query, "url": "https://www.booking.com/searchresults.html?ss=" + enc},
				{"title": "Airbnb: " + query, "url": "https://www.airbnb.com/s/" + enc + "/homes"},
				{"title": "Expedia: " + query, "url": "https://www.expedia.com/Hotel-Search?destination=" + enc},
				{"title": "TripAdvisor: " + query, "url": "https://www.tripadvisor.com/Search?q=" + enc},
				{"title": "Lonely Planet: " + query, "url": "https://www.lonelyplanet.com/search?q=" + enc},
			}
		case catTech:
			return []map[string]string{
				{"title": "Stack Overflow: " + query, "url": "https://stackoverflow.com/search?q=" + enc},
				{"title": "GitHub: " + query, "url": "https://github.com/search?q=" + enc},
				{"title": "MDN Web Docs: " + query, "url": "https://developer.mozilla.org/en-US/search?q=" + enc},
				{"title": "Reddit r/programming: " + query, "url": "https://www.reddit.com/r/programming/search/?q=" + enc},
				googleSearch,
			}
		case catEducation:
			return []map[string]string{
				{"title": "Coursera: " + query, "url": "https://www.coursera.org/search?query=" + enc},
				{"title": "Udemy: " + query, "url": "https://www.udemy.com/courses/search/?q=" + enc},
				{"title": "Khan Academy: " + query, "url": "https://www.khanacademy.org/search?page_search_query=" + enc},
				{"title": "edX: " + query, "url": "https://www.edx.org/search?q=" + enc},
				{"title": "YouTube: " + query, "url": "https://www.youtube.com/results?search_query=" + enc},
			}
		case catRealEstate:
			return []map[string]string{
				{"title": "Zillow: " + query, "url": "https://www.zillow.com/homes/" + enc + "_rb/"},
				{"title": "Realtor.com: " + query, "url": "https://www.realtor.com/realestateandhomes-search/" + enc},
				{"title": "Trulia: " + query, "url": "https://www.trulia.com/for_sale/" + enc + "/"},
				{"title": "Apartments.com: " + query, "url": "https://www.apartments.com/" + enc + "/"},
				googleSearch,
			}
		default:
			return []map[string]string{
				googleSearch,
				{"title": "Bing Search: " + query, "url": "https://www.bing.com/search?q=" + enc},
				{"title": "Reddit: " + query, "url": "https://www.reddit.com/search/?q=" + enc},
			}
		}
	}

	// ── 한국어 fallback ──────────────────────────────────────────────
	switch cat {
	case catTransit:
		isAir := strings.Contains(lower, "항공") || strings.Contains(lower, "비행기")
		isTrain := strings.Contains(lower, "ktx") || strings.Contains(lower, "기차") || strings.Contains(lower, "열차") || strings.Contains(lower, "무궁화") || strings.Contains(lower, "새마을")
		isSubway := strings.Contains(lower, "지하철") || strings.Contains(lower, "전철")
		isTaxi := strings.Contains(lower, "택시")
		if isAir {
			return []map[string]string{
				{"title": "네이버 항공권 검색", "url": "https://flight.naver.com/"},
				{"title": "스카이스캐너", "url": "https://www.skyscanner.co.kr/"},
				{"title": "카약 항공권", "url": "https://www.kayak.co.kr/flights"},
				naverSearch,
			}
		}
		if isTrain {
			return []map[string]string{
				{"title": "코레일 기차 예매", "url": "https://www.letskorail.com/"},
				{"title": "SRT 수서고속철도", "url": "https://etk.srail.kr/"},
				naverSearch,
			}
		}
		if isSubway {
			return []map[string]string{
				{"title": "네이버 지도 경로", "url": "https://map.naver.com/v5/directions/-/-/-/transit?query=" + enc},
				{"title": "카카오맵 대중교통", "url": "https://map.kakao.com/?q=" + enc},
				{"title": "서울 지하철 노선도", "url": "https://www.seoulmetro.co.kr/kr/cyberStation.do"},
			}
		}
		if isTaxi {
			return []map[string]string{
				{"title": "카카오T 택시", "url": "https://www.kakaomobility.com/"},
				{"title": "네이버 지도 경로", "url": "https://map.naver.com/v5/directions/-/-/-/car?query=" + enc},
			}
		}
		return []map[string]string{
			{"title": "버스타고 (시외/고속버스)", "url": "https://www.bustago.or.kr/"},
			{"title": "코버스 고속버스 예매", "url": "https://www.kobus.co.kr/"},
			{"title": "네이버 지도 경로", "url": "https://map.naver.com/v5/directions/-/-/-/transit?query=" + enc},
			{"title": "카카오맵 경로", "url": "https://map.kakao.com/?q=" + enc},
			{"title": "코레일 기차 예매", "url": "https://www.letskorail.com/"},
		}
	case catFood:
		return []map[string]string{
			{"title": "네이버 지도 맛집: " + query, "url": "https://map.naver.com/v5/search/" + enc, "type": "web"},
			{"title": "망고플레이트: " + query, "url": "https://www.mangoplate.com/search/" + enc, "type": "web"},
			{"title": "카카오맵 음식점: " + query, "url": "https://map.kakao.com/?q=" + enc, "type": "web"},
			{"title": "유튜브 맛집: " + query, "url": "https://www.youtube.com/results?search_query=" + enc + "+맛집", "type": "video"},
			{"title": "배달의민족", "url": "https://www.baemin.com/", "type": "web"},
		}
	case catShopping:
		return []map[string]string{
			{"title": "쿠팡: " + query, "url": "https://www.coupang.com/np/search?q=" + enc, "type": "web"},
			{"title": "네이버쇼핑: " + query, "url": "https://shopping.naver.com/search/all?query=" + enc, "type": "web"},
			{"title": "다나와 가격비교: " + query, "url": "https://search.danawa.com/dsearch.php?query=" + enc, "type": "web"},
			{"title": "유튜브 리뷰: " + query, "url": "https://www.youtube.com/results?search_query=" + enc + "+리뷰", "type": "video"},
		}
	case catFinance:
		isCrypto := strings.Contains(lower, "비트코인") || strings.Contains(lower, "코인") || strings.Contains(lower, "이더리움")
		isExchange := strings.Contains(lower, "환율") || strings.Contains(lower, "달러") || strings.Contains(lower, "엔화") || strings.Contains(lower, "유로")
		if isCrypto {
			return []map[string]string{
				{"title": "업비트 코인 시세", "url": "https://upbit.com/exchange?code=CRIX.UPBIT.KRW-BTC", "type": "web"},
				{"title": "빗썸 코인 시세", "url": "https://www.bithumb.com/trade/order/BTC_KRW", "type": "web"},
				{"title": "코인마켓캡", "url": "https://coinmarketcap.com/", "type": "web"},
				{"title": "유튜브 코인분석: " + query, "url": "https://www.youtube.com/results?search_query=" + enc + "+코인+분석", "type": "video"},
			}
		}
		if isExchange {
			return []map[string]string{
				{"title": "네이버 환율", "url": "https://finance.naver.com/marketindex/", "type": "web"},
				{"title": "한국은행 환율", "url": "https://www.bok.or.kr/portal/main/contents.do?menuNo=200644", "type": "web"},
				{"title": "유튜브 환율분석: " + query, "url": "https://www.youtube.com/results?search_query=" + enc + "+환율", "type": "video"},
			}
		}
		return []map[string]string{
			{"title": "네이버 증권: " + query, "url": "https://finance.naver.com/item/main.naver?code=005930", "type": "web"},
			{"title": "한국거래소(KRX)", "url": "https://www.krx.co.kr/main/main.jsp", "type": "web"},
			{"title": "인베스팅닷컴", "url": "https://kr.investing.com/", "type": "web"},
			{"title": "유튜브 주식분석: " + query, "url": "https://www.youtube.com/results?search_query=" + enc + "+주식+분석", "type": "video"},
		}
	case catWeather:
		return []map[string]string{
			{"title": "기상청 날씨", "url": "https://www.weather.go.kr/w/index.do", "type": "web"},
			{"title": "케이웨더", "url": "https://www.kweather.co.kr/", "type": "web"},
			{"title": "네이버 날씨: " + query, "url": "https://weather.naver.com/", "type": "web"},
		}
	case catNews:
		return []map[string]string{
			{"title": "JTBC 뉴스", "url": "https://news.jtbc.co.kr/", "type": "web"},
			{"title": "KBS 뉴스", "url": "https://news.kbs.co.kr/", "type": "web"},
			{"title": "MBC 뉴스", "url": "https://imnews.imbc.com/", "type": "web"},
			{"title": "유튜브 뉴스: " + query, "url": "https://www.youtube.com/results?search_query=" + enc + "+뉴스", "type": "video"},
		}
	case catMedical:
		return []map[string]string{
			{"title": "서울아산병원 건강정보", "url": "https://www.amc.seoul.kr/asan/healthinfo/body/bodyDetail.do", "type": "web"},
			{"title": "건강보험공단 건강정보", "url": "https://www.nhis.or.kr/nhis/healthin/retrieveHealthInfoMain.do", "type": "web"},
			{"title": "약학정보원", "url": "https://www.health.kr/", "type": "web"},
			{"title": "유튜브 건강정보: " + query, "url": "https://www.youtube.com/results?search_query=" + enc + "+건강+의학", "type": "video"},
		}
	case catLegal:
		return []map[string]string{
			{"title": "법제처 국가법령정보", "url": "https://www.law.go.kr/LSW/lsSc.do?menuId=1&query=" + enc, "type": "doc"},
			{"title": "정부24 민원", "url": "https://www.gov.kr/search?srchWord=" + enc, "type": "web"},
			{"title": "국세청 홈택스", "url": "https://www.hometax.go.kr/", "type": "web"},
			{"title": "유튜브 법률상담: " + query, "url": "https://www.youtube.com/results?search_query=" + enc + "+법률+상담", "type": "video"},
		}
	case catEntertainment:
		isMovie := strings.Contains(lower, "영화") || strings.Contains(lower, "cgv") || strings.Contains(lower, "롯데시네마") || strings.Contains(lower, "메가박스")
		isBaseball := strings.Contains(lower, "야구") || strings.Contains(lower, "kbo")
		isSoccer := strings.Contains(lower, "축구") || strings.Contains(lower, "k리그") || strings.Contains(lower, "월드컵")
		isConcert := strings.Contains(lower, "공연") || strings.Contains(lower, "뮤지컬") || strings.Contains(lower, "콘서트")
		if isMovie {
			return []map[string]string{
				{"title": "CGV 상영시간표", "url": "https://www.cgv.co.kr/movies/", "type": "web"},
				{"title": "롯데시네마 시간표", "url": "https://www.lottecinema.co.kr/NLCHS/Movie/MovieList", "type": "web"},
				{"title": "메가박스 시간표", "url": "https://www.megabox.co.kr/movie", "type": "web"},
				{"title": "유튜브 영화리뷰: " + query, "url": "https://www.youtube.com/results?search_query=" + enc + "+리뷰", "type": "video"},
			}
		}
		if isBaseball {
			return []map[string]string{
				{"title": "KBO 야구 일정/결과", "url": "https://www.koreabaseball.com/Schedule/Schedule.aspx", "type": "web"},
				{"title": "네이버 스포츠 야구", "url": "https://sports.naver.com/kbaseball/index", "type": "web"},
				{"title": "유튜브 야구하이라이트", "url": "https://www.youtube.com/results?search_query=" + enc + "+야구+하이라이트", "type": "video"},
			}
		}
		if isSoccer {
			return []map[string]string{
				{"title": "K리그 공식사이트", "url": "https://www.kleague.com/", "type": "web"},
				{"title": "네이버 스포츠 축구", "url": "https://sports.naver.com/football/index", "type": "web"},
				{"title": "유튜브 축구하이라이트", "url": "https://www.youtube.com/results?search_query=" + enc + "+축구+하이라이트", "type": "video"},
			}
		}
		if isConcert {
			return []map[string]string{
				{"title": "인터파크 티켓: " + query, "url": "https://tickets.interpark.com/search?keyword=" + enc, "type": "web"},
				{"title": "YES24 공연: " + query, "url": "https://ticket.yes24.com/Pages/Perf/PerfList.aspx", "type": "web"},
				{"title": "유튜브 공연영상: " + query, "url": "https://www.youtube.com/results?search_query=" + enc, "type": "video"},
			}
		}
		return []map[string]string{
			{"title": "네이버 스포츠", "url": "https://sports.naver.com/", "type": "web"},
			{"title": "유튜브: " + query, "url": "https://www.youtube.com/results?search_query=" + enc, "type": "video"},
		}
	case catRecipe:
		return []map[string]string{
			{"title": "만개의레시피: " + query, "url": "https://www.10000recipe.com/recipe/list.html?q=" + enc, "type": "web"},
			{"title": "해먹남녀", "url": "https://haemuknam.com/", "type": "web"},
			{"title": "유튜브 요리영상: " + query, "url": "https://www.youtube.com/results?search_query=" + enc + "+레시피", "type": "video"},
		}
	case catTravel:
		return []map[string]string{
			{"title": "한국관광공사", "url": "https://www.visitkorea.or.kr/", "type": "web"},
			{"title": "야놀자", "url": "https://www.yanolja.com/", "type": "web"},
			{"title": "여기어때", "url": "https://www.yeogi.com/", "type": "web"},
			{"title": "유튜브 여행브이로그: " + query, "url": "https://www.youtube.com/results?search_query=" + enc + "+여행+브이로그", "type": "video"},
		}
	case catTech:
		return []map[string]string{
			{"title": "스택오버플로우: " + query, "url": "https://stackoverflow.com/search?q=" + enc, "type": "doc"},
			{"title": "GitHub: " + query, "url": "https://github.com/search?q=" + enc, "type": "doc"},
			{"title": "나무위키: " + query, "url": "https://namu.wiki/w/" + enc, "type": "web"},
			{"title": "유튜브 튜토리얼: " + query, "url": "https://www.youtube.com/results?search_query=" + enc + "+tutorial", "type": "video"},
		}
	case catEducation:
		return []map[string]string{
			{"title": "EBS 강의: " + query, "url": "https://www.ebs.co.kr/search?query=" + enc, "type": "web"},
			{"title": "큐넷 자격증", "url": "https://www.q-net.or.kr/crf005.do?id=crf00503", "type": "web"},
			{"title": "유튜브 강의: " + query, "url": "https://www.youtube.com/results?search_query=" + enc + "+강의", "type": "video"},
		}
	case catRealEstate:
		return []map[string]string{
			{"title": "네이버 부동산", "url": "https://land.naver.com/", "type": "web"},
			{"title": "직방", "url": "https://www.zigbang.com/", "type": "web"},
			{"title": "호갱노노", "url": "https://hogangnono.com/", "type": "web"},
			{"title": "유튜브 부동산: " + query, "url": "https://www.youtube.com/results?search_query=" + enc + "+부동산", "type": "video"},
		}
	default:
		return []map[string]string{
			{"title": "유튜브: " + query, "url": "https://www.youtube.com/results?search_query=" + enc, "type": "video"},
			{"title": "나무위키: " + query, "url": "https://namu.wiki/w/" + enc, "type": "web"},
			{"title": "위키피디아: " + query, "url": "https://ko.wikipedia.org/w/index.php?search=" + enc, "type": "web"},
		}
	}
}

// parallelSearchResult: 병렬 수집 결과 묶음
type parallelSearchResult struct {
	Summary string
	Items   []map[string]string // {title, url, source}
}

// ── 5분 TTL 인메모리 캐시 ────────────────────────────────────────
type searchCacheEntry struct {
	result    parallelSearchResult
	expiresAt time.Time
}

var (
	searchCacheMu sync.RWMutex
	searchCache   = map[string]searchCacheEntry{}
)

func getCachedSearch(query string) (parallelSearchResult, bool) {
	searchCacheMu.RLock()
	defer searchCacheMu.RUnlock()
	e, ok := searchCache[query]
	if !ok || time.Now().After(e.expiresAt) {
		return parallelSearchResult{}, false
	}
	return e.result, true
}

func setCachedSearch(query string, r parallelSearchResult) {
	searchCacheMu.Lock()
	defer searchCacheMu.Unlock()
	searchCache[query] = searchCacheEntry{result: r, expiresAt: time.Now().Add(5 * time.Minute)}
	// 캐시 항목이 100개 초과 시 만료된 것 정리
	if len(searchCache) > 100 {
		now := time.Now()
		for k, v := range searchCache {
			if now.After(v.expiresAt) {
				delete(searchCache, k)
			}
		}
	}
}

// isSearchResultURL: 검색 결과 페이지 URL 여부 (상세 페이지 아님)
func isSearchResultURL(u string) bool {
	searchHosts := []string{
		"search.naver.com", "search.daum.net", "google.com/search",
		"bing.com/search", "yahoo.com/search", "duckduckgo.com",
	}
	for _, h := range searchHosts {
		if strings.Contains(u, h) {
			return true
		}
	}
	return isSectionPageURL(u)
}

// isSectionPageURL: 기사/상세페이지가 아닌 카테고리 섹션 홈 URL 감지
// 예) news.daum.net/economy, news.naver.com/main/main.naver
func isSectionPageURL(u string) bool {
	parsed, err := url.Parse(u)
	if err != nil {
		return false
	}
	host := parsed.Hostname()
	path := strings.Trim(parsed.Path, "/")

	// 루트 또는 빈 경로
	if path == "" || path == "index.html" || path == "index.php" {
		return true
	}

	// 알려진 섹션 경로 패턴 (뉴스 사이트 카테고리 홈)
	knownSections := []string{
		"economy", "world", "politics", "society", "sports", "entertainment",
		"it", "culture", "science", "health", "local", "global",
		"경제", "국제", "정치", "사회", "스포츠", "연예", "문화",
		"main/main", "section/main", "home",
	}
	pathLower := strings.ToLower(path)
	for _, sec := range knownSections {
		// 경로가 섹션 이름과 정확히 일치하거나 섹션 이름으로 끝나는 경우
		if pathLower == sec || strings.HasSuffix(pathLower, "/"+sec) {
			return true
		}
	}

	// 뉴스 도메인에서 숫자 없는 짧은 단일 경로는 섹션 홈으로 간주
	newsDomains := []string{
		"news.daum.net", "news.naver.com", "news.kbs.co.kr",
		"news.jtbc.co.kr", "imbc.com", "yonhapnews.co.kr",
		"apnews.com", "reuters.com", "bbc.com",
	}
	for _, nd := range newsDomains {
		if strings.Contains(host, nd) || host == nd {
			segments := strings.Split(path, "/")
			hasDigit := false
			for _, seg := range segments {
				for _, ch := range seg {
					if ch >= '0' && ch <= '9' {
						hasDigit = true
					}
				}
			}
			// 숫자도 없고 경로가 1~2단계인 경우 → 섹션 홈
			if !hasDigit && len(segments) <= 2 && parsed.RawQuery == "" {
				return true
			}
		}
	}
	return false
}

// parallelWebSearch: Tavily + 브라우저(chromedp) 를 goroutine으로 동시 실행
// catOverride: 전문가 시스템에서 카테고리를 이미 알고 있을 때 전달 (-1이면 자동 감지)
func parallelWebSearch(query string, maxItems int, catOverride ...queryCategory) parallelSearchResult {
	if cached, ok := getCachedSearch(query); ok {
		return cached
	}

	// 카테고리 감지 + 쿼리 최적화
	var cat queryCategory
	if len(catOverride) > 0 && catOverride[0] >= 0 {
		cat = catOverride[0]
	} else {
		cat = detectCategory(query)
	}
	optimized := optimizeQuery(query, cat)
	type srcResult struct {
		source  string
		summary string
		items   []map[string]string
	}

	llmMu.RLock()
	tKey := llmTavilyKey
	gKey := llmPerplexityKey
	llmMu.RUnlock()

	isKorean := !isEnglishQuery(query)

	// 카테고리별 추가 Tavily 도메인 검색 (실제 상세 페이지 + 유튜브 + 문서)
	type domainSearch struct{ domain, src string }
	var extraDomains []domainSearch
	switch cat {
	case catNews:
		if isKorean {
			extraDomains = []domainSearch{
				{"news.naver.com", "news"}, {"news.daum.net", "news"},
				{"jtbc.co.kr", "news"}, {"imbc.com", "news"}, {"kbs.co.kr", "news"},
				{"youtube.com", "youtube"},
			}
		} else {
			extraDomains = []domainSearch{
				{"apnews.com", "news"}, {"reuters.com", "news"}, {"bbc.com", "news"},
				{"youtube.com", "youtube"},
			}
		}
	case catFinance:
		if isKorean {
			extraDomains = []domainSearch{
				{"finance.naver.com", "finance"}, {"investing.com", "finance"},
				{"coinmarketcap.com", "crypto"}, {"youtube.com", "youtube"},
			}
		} else {
			extraDomains = []domainSearch{
				{"finance.yahoo.com", "finance"}, {"bloomberg.com", "finance"},
				{"coinmarketcap.com", "crypto"}, {"youtube.com", "youtube"},
			}
		}
	case catRecipe:
		if isKorean {
			extraDomains = []domainSearch{
				{"10000recipe.com", "recipe"}, {"haemuknam.com", "recipe"},
				{"youtube.com", "youtube"}, {"blog.naver.com", "blog"},
			}
		} else {
			extraDomains = []domainSearch{
				{"allrecipes.com", "recipe"}, {"foodnetwork.com", "recipe"},
				{"youtube.com", "youtube"},
			}
		}
	case catShopping:
		if isKorean {
			extraDomains = []domainSearch{
				{"coupang.com", "shop"}, {"shopping.naver.com", "shop"},
				{"gmarket.co.kr", "shop"}, {"youtube.com", "youtube"},
			}
		} else {
			extraDomains = []domainSearch{
				{"amazon.com", "shop"}, {"ebay.com", "shop"}, {"youtube.com", "youtube"},
			}
		}
	case catFood:
		if isKorean {
			extraDomains = []domainSearch{
				{"place.naver.com", "food"}, {"mangoplate.com", "food"},
				{"youtube.com", "youtube"},
			}
		} else {
			extraDomains = []domainSearch{
				{"yelp.com", "food"}, {"tripadvisor.com", "food"}, {"youtube.com", "youtube"},
			}
		}
	case catMedical:
		if isKorean {
			extraDomains = []domainSearch{
				{"health.naver.com", "medical"}, {"amc.seoul.kr", "medical"},
				{"youtube.com", "youtube"},
			}
		} else {
			extraDomains = []domainSearch{
				{"webmd.com", "medical"}, {"mayoclinic.org", "medical"},
				{"ncbi.nlm.nih.gov", "doc"}, {"youtube.com", "youtube"},
			}
		}
	case catTravel:
		if isKorean {
			extraDomains = []domainSearch{
				{"visitkorea.or.kr", "travel"}, {"yanolja.com", "travel"},
				{"youtube.com", "youtube"},
			}
		} else {
			extraDomains = []domainSearch{
				{"tripadvisor.com", "travel"}, {"lonelyplanet.com", "travel"},
				{"youtube.com", "youtube"},
			}
		}
	case catTech:
		extraDomains = []domainSearch{
			{"stackoverflow.com", "doc"}, {"github.com", "doc"},
			{"dev.to", "doc"}, {"youtube.com", "youtube"},
		}
	case catEducation:
		if isKorean {
			extraDomains = []domainSearch{
				{"ebs.co.kr", "edu"}, {"kin.naver.com", "edu"}, {"youtube.com", "youtube"},
			}
		} else {
			extraDomains = []domainSearch{
				{"coursera.org", "edu"}, {"khanacademy.org", "edu"}, {"youtube.com", "youtube"},
			}
		}
	case catRealEstate:
		if isKorean {
			extraDomains = []domainSearch{
				{"land.naver.com", "realestate"}, {"zigbang.com", "realestate"},
				{"hogangnono.com", "realestate"}, {"youtube.com", "youtube"},
			}
		}
	case catLegal:
		if isKorean {
			extraDomains = []domainSearch{
				{"law.go.kr", "legal"}, {"youtube.com", "youtube"},
			}
		} else {
			extraDomains = []domainSearch{
				{"law.cornell.edu", "legal"}, {"youtube.com", "youtube"},
			}
		}
	case catEntertainment:
		extraDomains = []domainSearch{
			{"youtube.com", "youtube"}, {"imdb.com", "entertainment"},
		}
	case catTransit:
		if isKorean {
			extraDomains = []domainSearch{
				{"letskorail.com", "transit"}, {"kobus.co.kr", "transit"},
				{"youtube.com", "youtube"},
			}
		}
	}

	ch := make(chan srcResult, 4+len(extraDomains))
	var wg sync.WaitGroup

	// ── 소스 1: Tavily 일반 검색 ──────────────────────────────
	if tKey != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if r, ok := tavilySearch(tKey, optimized, maxItems); ok {
				ch <- srcResult{source: "tavily", summary: r.Summary, items: r.Items}
			}
		}()
	}

	// ── 소스 2: 카테고리별 특화 도메인 Tavily 검색 (실제 상세페이지) ──
	// 날짜 붙은 optimized 쿼리는 도메인 검색 시 결과 0 → 원본 query 사용
	if tKey != "" {
		for _, ed := range extraDomains {
			ed := ed
			wg.Add(1)
			go func() {
				defer wg.Done()
				if r, ok := tavilySearchDomain(tKey, query, 4, ed.domain); ok {
					ch <- srcResult{source: ed.src, items: r.Items}
				}
			}()
		}
	}

	// ── 소스 3: 플랫폼별 브라우저 크롤링 ─────────────────────────
	wg.Add(1)
	go func() {
		defer wg.Done()
		items := browserParallelScrape(optimized, maxItems)
		if len(items) > 0 {
			ch <- srcResult{source: "browser", items: items}
		}
	}()

	// ── 모든 goroutine 완료 후 channel 닫기 ─────────────────────
	go func() {
		wg.Wait()
		close(ch)
	}()

	// ── 결과 수집 + URL 중복 제거 ─────────────────────────────
	queryWords := queryKeywords(query)
	seen := map[string]bool{}
	var allRaw []map[string]string
	var summaries []string

	for r := range ch {
		if r.summary != "" {
			summaries = append(summaries, r.summary)
		}
		for _, item := range r.items {
			url := item["url"]
			if url == "" || seen[url] || isSearchResultURL(url) {
				continue
			}
			seen[url] = true
			item["source"] = r.source
			// YouTube 소스 태깅
			if strings.Contains(url, "youtube.com") || strings.Contains(url, "youtu.be") {
				item["type"] = "video"
			}
			allRaw = append(allRaw, item)
		}
	}

	// 키워드 관련성 필터: 비디오/일반 항목 분리 처리
	var merged []map[string]string
	var videoItems []map[string]string
	for _, item := range allRaw {
		if item["type"] == "video" || strings.Contains(item["url"], "youtube.com") || strings.Contains(item["url"], "tiktok.com") {
			videoItems = append(videoItems, item)
		} else {
			merged = append(merged, item)
		}
	}

	// 일반 항목: 키워드 필터 적용
	if len(queryWords) > 0 {
		var filtered []map[string]string
		for _, item := range merged {
			if titleMatchesQuery(item["title"], queryWords) {
				filtered = append(filtered, item)
			}
		}
		// 필터 통과 항목이 절반 이상이면 필터 결과 사용, 아니면 전체 fallback
		if len(filtered) >= len(merged)/2 || len(filtered) > 0 {
			merged = filtered
		}
	}
	if len(merged) > maxItems*2 {
		merged = merged[:maxItems*2]
	}

	// 비디오 항목: 키워드 필터 엄격 적용 (관련 없는 영상 제거)
	if len(queryWords) > 0 {
		var filteredVideo []map[string]string
		for _, item := range videoItems {
			if titleMatchesQuery(item["title"], queryWords) {
				filteredVideo = append(filteredVideo, item)
			}
		}
		videoItems = filteredVideo // 관련 없으면 아예 제거 (fallback 없음)
	}
	merged = append(merged, videoItems...)

	// 제목만 추출 (URL·원문 절대 Groq에 전달 안 함)
	titleLines := make([]string, 0, len(merged))
	for _, it := range merged {
		if t := it["title"]; t != "" {
			titleLines = append(titleLines, "• "+t)
		}
	}
	// Tavily answer가 있으면 우선 사용 (이미 정제된 자연어)
	tavilySummary := ""
	for _, s := range summaries {
		if s != "" {
			tavilySummary = s
			break
		}
	}

	// 카테고리별 공식 사이트 안내 문구
	officialSiteHint := buildOfficialSiteHint(cat)

	// 브라우저 스크래핑(merged)도 비어있고 Tavily content도 없을 때만
	// Perplexity sonar-online 직접 검색으로 폴백
	needsDirectSearch := len(merged) == 0 && tavilySummary == "" &&
		(cat == catShopping || cat == catFood || cat == catEntertainment ||
			cat == catTravel || cat == catNews || cat == catRecipe)

	var summary string
	if gKey != "" {
		kst := time.FixedZone("KST", 9*3600)
		nowKST := time.Now().In(kst)
		today := nowKST.Format("2006-01-02 15:04 KST")

		var msgs []groqMsg

		if needsDirectSearch {
			// Perplexity sonar-online의 자체 웹 검색 활용
			// — Tavily content 전달하지 않고 쿼리만 던져서 직접 검색하게 함
			directSys := `당신은 Nexus AI 한국어 비서입니다. 실시간 웹 검색으로 정확한 정보를 찾아 답하세요.

[규칙]
- 자연스러운 한국어 2~4문장으로 답하세요
- URL, 링크, 출처명 포함 금지
- 마크다운 헤더(##), 과도한 불릿 금지
- 가격·수치 등 구체적 정보 반드시 포함
- "봇 차단", "차단으로 인해", "접근 불가" 등 표현 절대 금지
- 정보를 찾지 못했을 때는 공식 사이트 이용 안내로 마무리`
			directUser := fmt.Sprintf("현재 시각(KST): %s\n\n%s", today, query)
			msgs = []groqMsg{
				{Role: "system", Content: directSys},
				{Role: "user", Content: directUser},
			}
		} else {
			context := strings.Join(titleLines, "\n")
			if context == "" {
				context = "(검색 결과 없음)"
			}
			tavilyHint := ""
			if tavilySummary != "" {
				hint := tavilySummary
				if len([]rune(hint)) > 300 {
					hint = string([]rune(hint)[:300])
				}
				tavilyHint = "\nTavily 요약: " + hint
			}
			sysMsg := fmt.Sprintf(`당신은 Nexus AI 한국어 비서입니다.

[최우선 규칙]
1. 검색 결과를 바탕으로 자연스러운 한국어로 답하세요
2. URL, 링크, 출처명([tavily] 등) 절대 포함 금지
3. 마크다운 헤더(##), 과도한 불릿 금지
4. 검색 결과가 부족하더라도 절대 "모른다", "찾지 못했습니다"로 끝내지 마세요
5. 결과가 없으면 → 카테고리에 맞는 공식 사이트/방법을 안내하세요

[카테고리별 결과 부족 시 안내]
%s

[답변 형식]
- 검색 결과 있을 때: 핵심 정보를 2~4문장으로 자연스럽게 요약
- 검색 결과 없을 때: "~에서 확인하실 수 있습니다" 형식으로 공식 경로 안내
- 구체적 수치(가격, 시간, 요금 등) 있으면 반드시 포함
- 복잡한 질문은 4~6문장까지 허용

[절대 금지 표현]
- "봇 차단", "봇차단", "차단으로 인해", "차단되어", "수집할 수 없", "수집이 불가", "봇을 감지", "자동화된 접근", "bot detected", "access denied"
- 검색이 막혔다거나 크롤링 실패, 정보 수집 불가를 사용자에게 언급하는 모든 표현
- "정확한 정보를 찾지 못했습니다. 미리보기 버튼으로 직접 확인해보세요." 이 문구 사용 금지
- "모릅니다", "알 수 없습니다" 로 끝내는 것 금지`, officialSiteHint)

			userMsg := fmt.Sprintf("현재 시각(KST): %s\n사용자 질문: \"%s\"\n최적화 검색어: \"%s\"\n검색 결과 제목:\n%s%s\n\n⚠️ 시간을 언급할 때 반드시 KST 기준으로 표현하세요. UTC 표기 절대 금지.\n위 정보를 바탕으로 답하되, 결과가 부족하면 공식 사이트 안내로 마무리하세요.", today, query, optimized, context, tavilyHint)
			msgs = []groqMsg{
				{Role: "system", Content: sysMsg},
				{Role: "user", Content: userMsg},
			}
		}

		refined, _, err := callGroq(gKey, groqChatModel, msgs, 600, false)
		if err == nil && refined != "" {
			summary = refined
		}
	}

	// 3단 폴백: Groq 실패 시
	if summary == "" {
		if tavilySummary != "" {
			// 1순위: Tavily raw 요약 그대로 사용
			summary = tavilySummary
		} else if len(merged) > 0 {
			// 2순위: 검색 결과 개수 + 카테고리 안내
			summary = fmt.Sprintf("관련 정보 %d건을 찾았습니다. 아래 링크에서 확인해보세요.", len(merged))
		} else {
			// 3순위: 카테고리별 공식 사이트 안내
			summary = buildNoResultMessage(query, cat, tKey)
		}
	}

	// 검색 결과 없을 때 카테고리별 폴백 사이트 링크 주입
	if len(merged) == 0 {
		merged = categoryFallbackSites(query, cat)
	}

	// 최대 maxItems 개만 반환
	if len(merged) > maxItems {
		merged = merged[:maxItems]
	}

	result := parallelSearchResult{Summary: summary, Items: merged}
	setCachedSearch(query, result)
	return result
}

// buildOfficialSiteHint: Groq 프롬프트에 삽입할 카테고리별 안내 문구
func buildOfficialSiteHint(cat queryCategory) string {
	switch cat {
	case catTransit:
		return `교통/버스/기차/항공:
- 고속버스: "버스타고(bustago.or.kr) 또는 코버스(kobus.co.kr)에서 예매하세요"
- KTX/기차: "코레일(letskorail.com) 또는 SRT(srail.kr)에서 검색하세요"
- 지하철: "네이버 지도나 카카오맵에서 경로를 검색해보세요"
- 항공: "네이버 항공권(flight.naver.com) 또는 스카이스캐너에서 비교하세요"
- 택시: "카카오T 앱에서 요금 확인이 가능합니다"`
	case catFood:
		return `맛집/음식:
- "네이버 지도 또는 카카오맵에서 해당 지역 음식점을 검색해보세요"
- "배달은 배달의민족(baemin.com) 또는 요기요에서 확인하세요"
- "망고플레이트에서 리뷰와 평점을 볼 수 있습니다"`
	case catShopping:
		return `쇼핑/가격비교:
- "쿠팡, 네이버쇼핑, 다나와에서 가격을 비교해보세요"
- "다나와(danawa.com)는 전자제품 최저가 비교에 특화되어 있습니다"`
	case catFinance:
		return `금융/환율/주가:
- 환율: "네이버 금융(finance.naver.com/marketindex)에서 실시간 환율을 확인하세요"
- 주식: "네이버 금융 또는 증권사 앱에서 실시간 주가를 확인하세요"
- 코인: "업비트(upbit.com) 또는 빗썸(bithumb.com)에서 확인하세요"`
	case catWeather:
		return `날씨:
- "기상청(weather.go.kr) 또는 네이버 날씨에서 정확한 예보를 확인하세요"`
	case catNews:
		return `뉴스:
- "네이버 뉴스(news.naver.com) 또는 다음 뉴스에서 최신 기사를 확인하세요"`
	case catMedical:
		return `의료/건강/병원:
- "근처 병원은 네이버 지도에서 '병원' 검색으로 찾으세요"
- "증상 정보는 서울아산병원 건강정보(amc.seoul.kr) 또는 건강보험공단 사이트를 참고하세요"
- "약 정보는 약학정보원(health.kr)에서 확인하세요"`
	case catLegal:
		return `법률/행정/세금:
- "민원 서비스는 정부24(gov.kr)에서 처리 가능합니다"
- "세금 신고는 국세청 홈택스(hometax.go.kr)를 이용하세요"
- "법령 정보는 법제처 국가법령정보센터(law.go.kr)에서 확인하세요"`
	case catEntertainment:
		return `영화/공연/스포츠:
- 영화: "CGV, 롯데시네마, 메가박스 앱에서 시간표와 예매가 가능합니다"
- 공연: "인터파크 티켓(tickets.interpark.com) 또는 YES24에서 예매하세요"
- 야구: "KBO 공식사이트(koreabaseball.com)에서 일정과 결과를 확인하세요"
- 축구: "K리그 공식사이트(kleague.com)에서 확인하세요"`
	case catRecipe:
		return `요리/레시피:
- "만개의레시피(10000recipe.com)에서 레시피를 검색해보세요"
- "유튜브에서 요리 영상도 많이 찾을 수 있습니다"`
	case catTravel:
		return `여행/숙박:
- "야놀자(yanolja.com) 또는 여기어때에서 숙소를 예약하세요"
- "한국관광공사(visitkorea.or.kr)에서 국내 여행지 정보를 확인하세요"`
	case catRealEstate:
		return `부동산:
- "네이버 부동산(land.naver.com), 직방, 호갱노노에서 시세를 확인하세요"
- "청약은 청약홈(applyhome.co.kr)에서 신청 가능합니다"`
	default:
		return `결과 부족 시:
- "네이버 또는 구글에서 더 구체적인 키워드로 검색해보세요"
- 가능하면 지역명, 날짜, 브랜드명 등을 추가해서 다시 질문해주세요`
	}
}

// buildNoResultMessage: 검색 결과 완전 없을 때 카테고리별 안내 메시지
func buildNoResultMessage(query string, cat queryCategory, tKey string) string {
	eng := isEnglishQuery(query)
	if tKey == "" {
		if eng {
			return "Tavily API key is not configured. Go to Settings → API Keys and add your Tavily key for real-time search results."
		}
		return "Tavily API 키가 설정되지 않아 실시간 검색이 제한됩니다. 설정 → API 키에서 Tavily 키를 입력하면 훨씬 정확한 검색이 가능합니다."
	}
	lower := strings.ToLower(query)
	if eng {
		switch cat {
		case catTransit:
			if strings.Contains(lower, "flight") || strings.Contains(lower, "airport") || strings.Contains(lower, "airline") {
				return "Search for flights on Google Flights (google.com/flights), Skyscanner, or Kayak."
			}
			if strings.Contains(lower, "subway") || strings.Contains(lower, "metro") {
				return "Check transit routes on Google Maps — enter your origin and destination for step-by-step directions."
			}
			return "Use Google Maps (maps.google.com) or Rome2rio (rome2rio.com) to find bus, train, and transit routes."
		case catFood:
			return fmt.Sprintf("Search for '%s' restaurants on Yelp (yelp.com) or Google Maps to find reviews and locations nearby.", query)
		case catShopping:
			return fmt.Sprintf("Compare prices for '%s' on Amazon, eBay, or Google Shopping.", query)
		case catFinance:
			return "Check real-time financial data on Yahoo Finance (finance.yahoo.com) or Bloomberg (bloomberg.com)."
		case catWeather:
			return fmt.Sprintf("Check the weather for '%s' on weather.com or AccuWeather (accuweather.com).", query)
		case catNews:
			return fmt.Sprintf("Search for '%s' news on Google News (news.google.com), BBC, Reuters, or AP News.", query)
		case catMedical:
			return fmt.Sprintf("For information on '%s', consult WebMD (webmd.com) or Mayo Clinic (mayoclinic.org). Always see a doctor for personal medical advice.", query)
		case catLegal:
			return "For legal and tax questions, check IRS.gov (US tax), USA.gov, or consult a licensed attorney."
		case catEntertainment:
			if strings.Contains(lower, "movie") || strings.Contains(lower, "film") {
				return "Check movie showtimes and reviews on IMDb (imdb.com), Fandango, or Rotten Tomatoes."
			}
			if strings.Contains(lower, "concert") || strings.Contains(lower, "ticket") {
				return "Buy event tickets on Ticketmaster (ticketmaster.com) or StubHub."
			}
			return fmt.Sprintf("Search for '%s' on ESPN, IMDb, or Google for the latest results.", query)
		case catRecipe:
			return fmt.Sprintf("Find '%s' recipes on AllRecipes (allrecipes.com), Food Network, or YouTube.", query)
		case catTravel:
			return fmt.Sprintf("Search for '%s' travel info on Booking.com, Airbnb, or TripAdvisor.", query)
		case catRealEstate:
			return "Browse real estate listings on Zillow (zillow.com), Realtor.com, or Apartments.com."
		default:
			return fmt.Sprintf("No results found for '%s'. Try searching on Google or Bing with more specific keywords.", query)
		}
	}
	switch cat {
	case catTransit:
		if strings.Contains(lower, "ktx") || strings.Contains(lower, "기차") || strings.Contains(lower, "열차") {
			return "기차/KTX 시간표와 예매는 코레일(letskorail.com) 또는 SRT(srail.kr) 앱에서 확인하세요."
		}
		if strings.Contains(lower, "항공") || strings.Contains(lower, "비행기") {
			return "항공권은 네이버 항공권(flight.naver.com) 또는 스카이스캐너에서 비교 검색하세요."
		}
		if strings.Contains(lower, "지하철") || strings.Contains(lower, "전철") {
			return "지하철 경로는 카카오맵 또는 네이버 지도 앱에서 출발지와 도착지를 입력하면 정확한 시간표와 환승 정보를 볼 수 있습니다."
		}
		return "고속버스/시외버스 시간표와 예매는 버스타고(bustago.or.kr) 또는 코버스(kobus.co.kr)에서 출발지와 도착지를 검색하세요."
	case catFood:
		return fmt.Sprintf("'%s' 관련 맛집은 네이버 지도 또는 카카오맵에서 검색하시면 리뷰와 위치를 함께 볼 수 있습니다.", query)
	case catShopping:
		return fmt.Sprintf("'%s' 가격 비교는 쿠팡, 네이버쇼핑, 다나와(danawa.com)에서 확인해보세요.", query)
	case catFinance:
		return "실시간 금융 정보는 네이버 금융(finance.naver.com), 주식은 각 증권사 앱, 코인은 업비트·빗썸에서 확인하세요."
	case catWeather:
		return fmt.Sprintf("'%s' 날씨는 기상청(weather.go.kr) 또는 네이버 날씨에서 정확한 예보를 확인하세요.", query)
	case catNews:
		return fmt.Sprintf("'%s' 관련 뉴스는 네이버 뉴스(news.naver.com) 또는 구글 뉴스에서 검색해보세요.", query)
	case catMedical:
		return fmt.Sprintf("'%s' 관련 의료 정보는 서울아산병원 건강정보(amc.seoul.kr) 또는 건강보험공단 사이트를 참고하고, 정확한 진단은 전문의와 상담하세요.", query)
	case catLegal:
		return "행정 민원은 정부24(gov.kr), 세금은 홈택스(hometax.go.kr), 법령 확인은 법제처(law.go.kr)를 이용하세요."
	case catEntertainment:
		if strings.Contains(lower, "영화") {
			return "영화 시간표와 예매는 CGV, 롯데시네마, 메가박스 앱에서 확인하세요."
		}
		if strings.Contains(lower, "공연") || strings.Contains(lower, "뮤지컬") || strings.Contains(lower, "콘서트") {
			return "공연/콘서트 예매는 인터파크 티켓(tickets.interpark.com) 또는 YES24에서 확인하세요."
		}
		return fmt.Sprintf("'%s' 관련 정보는 네이버 스포츠 또는 해당 리그 공식 사이트에서 확인하세요.", query)
	case catRecipe:
		return fmt.Sprintf("'%s' 레시피는 만개의레시피(10000recipe.com)에서 검색하거나 유튜브에서 요리 영상을 찾아보세요.", query)
	case catTravel:
		return fmt.Sprintf("'%s' 여행 정보는 한국관광공사(visitkorea.or.kr), 숙소 예약은 야놀자 또는 여기어때에서 확인하세요.", query)
	case catRealEstate:
		return "부동산 시세는 네이버 부동산(land.naver.com), 직방, 호갱노노에서 확인하고, 청약은 청약홈(applyhome.co.kr)을 이용하세요."
	default:
		return fmt.Sprintf("'%s'에 대한 검색 결과를 찾지 못했습니다. 네이버나 구글에서 더 구체적인 키워드로 검색해보세요.", query)
	}
}
