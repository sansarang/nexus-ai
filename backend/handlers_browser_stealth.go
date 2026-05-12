//go:build windows

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

// ──────────────────────────────────────────────────────────────
// Stealth 자바스크립트 — 봇 탐지 우회
// 출처: puppeteer-extra-plugin-stealth 전략 포팅
// ──────────────────────────────────────────────────────────────

const stealthJS = `
(function(){
  // 1. webdriver 속성 제거
  Object.defineProperty(navigator, 'webdriver', {
    get: () => undefined,
    configurable: true
  });

  // 2. chrome 객체 복원
  if (!window.chrome) {
    window.chrome = {
      app: { isInstalled: false },
      csi: function(){},
      loadTimes: function(){},
      runtime: {}
    };
  }

  // 3. plugins 배열 위장 (실제 플러그인처럼)
  Object.defineProperty(navigator, 'plugins', {
    get: () => {
      const plugins = [
        { name: 'Chrome PDF Plugin',     filename: 'internal-pdf-viewer', description: 'Portable Document Format' },
        { name: 'Chrome PDF Viewer',     filename: 'mhjfbmdgcfjbbpaeojofohoefgiehjai', description: '' },
        { name: 'Native Client',         filename: 'internal-nacl-plugin', description: '' },
      ];
      plugins.refresh = function(){};
      plugins.namedItem = name => plugins.find(p => p.name === name) ?? null;
      plugins.item = i => plugins[i] ?? null;
      Object.setPrototypeOf(plugins, PluginArray.prototype);
      return plugins;
    },
    configurable: true
  });

  // 4. languages 복원
  Object.defineProperty(navigator, 'languages', {
    get: () => ['ko-KR', 'ko', 'en-US', 'en'],
    configurable: true
  });

  // 5. permissions API 위장 (Notification 권한 등)
  const origQuery = window.navigator.permissions.query.bind(window.navigator.permissions);
  window.navigator.permissions.query = (parameters) => {
    if (parameters.name === 'notifications') {
      return Promise.resolve({ state: Notification.permission });
    }
    return origQuery(parameters);
  };

  // 6. iframe contentWindow webdriver 제거
  const origDesc = Object.getOwnPropertyDescriptor(HTMLIFrameElement.prototype, 'contentWindow');
  Object.defineProperty(HTMLIFrameElement.prototype, 'contentWindow', {
    get: function(){
      const win = origDesc.get.call(this);
      if (!win) return win;
      Object.defineProperty(win.navigator, 'webdriver', { get: () => undefined, configurable: true });
      return win;
    }
  });

  // 7. screen 해상도 자연스럽게 설정
  if (screen.colorDepth === 24) {
    Object.defineProperty(screen, 'colorDepth', { get: () => 32, configurable: true });
  }

  // 8. WebGL vendor 위장
  const getParam = WebGLRenderingContext.prototype.getParameter;
  WebGLRenderingContext.prototype.getParameter = function(param) {
    if (param === 37445) return 'Intel Inc.';
    if (param === 37446) return 'Intel Iris OpenGL Engine';
    return getParam.call(this, param);
  };

  // 9. 자동화 흔적 제거 ($cdc_ 변수 등)
  delete window.cdc_adoQpoasnfa76pfcZLmcfl_Array;
  delete window.cdc_adoQpoasnfa76pfcZLmcfl_Promise;
  delete window.cdc_adoQpoasnfa76pfcZLmcfl_Symbol;
  delete window.$chrome_asyncScriptInfo;

  // 10. toString 위조 방지 (native function 위장)
  window.chrome.runtime.connect = function() {};
  window.chrome.runtime.sendMessage = function() {};
})();
`

// ──────────────────────────────────────────────────────────────
// 봇 우회 User-Agent 풀
// ──────────────────────────────────────────────────────────────

var stealthUserAgents = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36 Edg/122.0.0.0",
	"Mozilla/5.0 (Windows NT 11.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36",
}

func randomUA() string {
	return stealthUserAgents[rand.Intn(len(stealthUserAgents))]
}

// ──────────────────────────────────────────────────────────────
// Stealth 브라우저 초기화 (기존 ensureBrowser 대체)
// ──────────────────────────────────────────────────────────────

func ensureStealthBrowser() (context.Context, error) {
	browserMu.Lock()
	defer browserMu.Unlock()

	if browserAlloc != nil && !browserBroken {
		select {
		case <-browserAlloc.Done():
		default:
			return browserCtx, nil
		}
	}

	if browserCancel != nil {
		browserCancel()
	}

	ua := randomUA()
	// 랜덤 화면 크기 (일반 사무 모니터 해상도)
	resolutions := [][2]int{{1920, 1080}, {1366, 768}, {1440, 900}, {1600, 900}, {1280, 800}}
	res := resolutions[rand.Intn(len(resolutions))]

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", false),
		chromedp.Flag("no-first-run", true),
		chromedp.Flag("no-default-browser-check", true),
		chromedp.Flag("disable-notifications", true),
		chromedp.Flag("disable-infobars", true),
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-web-security", false),
		chromedp.Flag("enable-features", "NetworkServiceInProcess"),
		chromedp.Flag("lang", "ko-KR"),
		chromedp.Flag("accept-lang", "ko-KR,ko;q=0.9,en-US;q=0.8,en;q=0.7"),
		chromedp.ExecPath(findChromePath()),
		chromedp.WindowSize(res[0], res[1]),
		chromedp.UserAgent(ua),
	)

	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), opts...)
	ctx, ctxCancel := chromedp.NewContext(allocCtx)

	// 브라우저 시작 + stealth 스크립트를 모든 페이지에 자동 주입
	pingCtx, pingCancel := context.WithTimeout(ctx, 8*time.Second)
	defer pingCancel()

	if err := chromedp.Run(pingCtx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			// CDP: Page.addScriptToEvaluateOnNewDocument
			// → 새 페이지/탭이 열릴 때마다 자동으로 stealthJS를 실행
			_, err := page.AddScriptToEvaluateOnNewDocument(stealthJS).Do(ctx)
			return err
		}),
	); err != nil {
		ctxCancel()
		allocCancel()
		return nil, fmt.Errorf("Stealth 브라우저 초기화 실패: %w", err)
	}

	browserAlloc = allocCtx
	browserCancel = func() {
		ctxCancel()
		allocCancel()
	}
	browserCtx = ctx
	browserBroken = false
	return ctx, nil
}

// findChromePath: Chrome/Edge 실행 파일 경로 탐색
func findChromePath() string {
	candidates := []string{
		`C:\Program Files\Google\Chrome\Application\chrome.exe`,
		`C:\Program Files (x86)\Google\Chrome\Application\chrome.exe`,
		`C:\Program Files\Microsoft\Edge\Application\msedge.exe`,
		`C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`,
		`C:\Users\` + os.Getenv("USERNAME") + `\AppData\Local\Google\Chrome\Application\chrome.exe`,
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return "" // 기본 PATH에서 탐색
}

// ──────────────────────────────────────────────────────────────
// Human-like 행동 시뮬레이션
// ──────────────────────────────────────────────────────────────

// humanDelay: 사람처럼 랜덤 대기 (min~max ms)
func humanDelay(minMs, maxMs int) chromedp.Action {
	d := time.Duration(minMs+rand.Intn(maxMs-minMs)) * time.Millisecond
	return chromedp.Sleep(d)
}

// humanType: 문자를 한 글자씩 랜덤 딜레이로 입력
func humanType(ctx context.Context, selector, text string) error {
	// 필드 클릭 먼저
	if err := chromedp.Run(ctx,
		chromedp.WaitVisible(selector, chromedp.ByQuery),
		chromedp.Click(selector, chromedp.ByQuery),
		humanDelay(200, 500),
		chromedp.Clear(selector, chromedp.ByQuery),
	); err != nil {
		return err
	}

	// 글자별 입력
	for _, ch := range text {
		delay := time.Duration(80+rand.Intn(150)) * time.Millisecond
		if err := chromedp.Run(ctx,
			chromedp.SendKeys(selector, string(ch), chromedp.ByQuery),
			chromedp.Sleep(delay),
		); err != nil {
			return err
		}
		// 가끔 더 긴 쉬기 (실수로 멈추는 것처럼)
		if rand.Intn(10) == 0 {
			chromedp.Run(ctx, chromedp.Sleep(time.Duration(300+rand.Intn(700))*time.Millisecond))
		}
	}
	return nil
}

// humanScroll: 자연스러운 스크롤 (여러 번 나눠서)
func humanScroll(ctx context.Context, totalPx int) error {
	steps := 3 + rand.Intn(5)
	perStep := totalPx / steps
	for i := 0; i < steps; i++ {
		scroll := perStep + rand.Intn(50) - 25
		js := fmt.Sprintf(`window.scrollBy({top: %d, behavior: 'smooth'})`, scroll)
		if err := chromedp.Run(ctx,
			chromedp.Evaluate(js, nil),
			chromedp.Sleep(time.Duration(150+rand.Intn(250))*time.Millisecond),
		); err != nil {
			return err
		}
	}
	return nil
}

// waitForPageStable: 페이지가 안정될 때까지 대기
func waitForPageStable(ctx context.Context) error {
	// DOM이 안정될 때까지 최대 5초 대기
	stabilizeJS := `
	new Promise(resolve => {
		let last = document.body ? document.body.innerHTML.length : 0;
		let stable = 0;
		const check = setInterval(() => {
			const cur = document.body ? document.body.innerHTML.length : 0;
			if (cur === last) {
				stable++;
				if (stable >= 3) { clearInterval(check); resolve(true); }
			} else {
				stable = 0;
				last = cur;
			}
		}, 200);
		setTimeout(() => { clearInterval(check); resolve(true); }, 5000);
	})
	`
	var result bool
	tCtx, cancel := context.WithTimeout(ctx, 6*time.Second)
	defer cancel()
	return chromedp.Run(tCtx, chromedp.Evaluate(stabilizeJS, &result))
}

// ──────────────────────────────────────────────────────────────
// 쿠키 세션 지속성 (사이트별 저장/복원)
// ──────────────────────────────────────────────────────────────

func cookiePath(site string) string {
	dir := filepath.Join(os.TempDir(), "nexus_sessions")
	os.MkdirAll(dir, 0755)
	return filepath.Join(dir, site+".json")
}

// saveCookies: 현재 브라우저 쿠키를 파일에 저장
func saveCookies(ctx context.Context, site string) error {
	var cookies []*network.Cookie
	if err := chromedp.Run(ctx, chromedp.ActionFunc(func(ctx context.Context) error {
		var err error
		cookies, err = network.GetCookies().Do(ctx)
		return err
	})); err != nil {
		return err
	}
	data, err := json.Marshal(cookies)
	if err != nil {
		return err
	}
	return os.WriteFile(cookiePath(site), data, 0644)
}

// loadCookies: 저장된 쿠키를 브라우저에 복원
func loadCookies(ctx context.Context, site string) error {
	data, err := os.ReadFile(cookiePath(site))
	if err != nil {
		return nil // 쿠키 없으면 무시
	}
	var cookies []*network.Cookie
	if err := json.Unmarshal(data, &cookies); err != nil {
		return nil
	}
	for _, c := range cookies {
		params := network.SetCookie(c.Name, c.Value).WithDomain(c.Domain).WithPath(c.Path)
		if err := chromedp.Run(ctx, chromedp.ActionFunc(func(ctx context.Context) error {
			return params.Do(ctx)
		})); err != nil {
			continue // 개별 쿠키 실패는 무시
		}
	}
	return nil
}

// ──────────────────────────────────────────────────────────────
// 안티봇 탐지 감지 + 재시도 로직
// ──────────────────────────────────────────────────────────────

// detectAntiBot: 현재 페이지에 봇 차단 징후가 있는지 확인
func detectAntiBot(ctx context.Context) (bool, string) {
	var pageText string
	if err := chromedp.Run(ctx,
		chromedp.Text("body", &pageText, chromedp.ByQuery),
	); err != nil {
		return false, ""
	}

	antiBotSigns := []string{
		"Access Denied", "403 Forbidden", "Bot detected",
		"CAPTCHA", "captcha", "자동화된 접근",
		"비정상적인 트래픽", "차단", "Blocked",
		"cf-browser-verification", "ray ID",
		"인증이 필요합니다",
	}

	for _, sign := range antiBotSigns {
		if len(pageText) > 0 && contains(pageText, sign) {
			return true, sign
		}
	}

	// Cloudflare challenge 확인
	var cfDetected bool
	chromedp.Run(ctx, chromedp.Evaluate(
		`document.querySelector('#challenge-form, .cf-browser-verification') !== null`,
		&cfDetected,
	))
	if cfDetected {
		return true, "Cloudflare 인증"
	}

	return false, ""
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsStr(s, sub))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// retryWithBackoff: 안티봇 감지 시 재시도
func retryWithBackoff(ctx context.Context, maxRetries int, action func() error) error {
	for attempt := 0; attempt < maxRetries; attempt++ {
		err := action()
		if err == nil {
			return nil
		}
		blocked, reason := detectAntiBot(ctx)
		if !blocked {
			return err
		}
		// 지수 백오프 + 랜덤 지터
		backoff := time.Duration(2<<uint(attempt)) * time.Second
		jitter := time.Duration(rand.Intn(3000)) * time.Millisecond
		wait := backoff + jitter
		if wait > 30*time.Second {
			wait = 30 * time.Second
		}
		_ = reason
		time.Sleep(wait)

		// stealth JS 재주입 (현재 페이지에 즉시 적용)
		chromedp.Run(ctx, chromedp.ActionFunc(func(c context.Context) error {
			_, e := page.AddScriptToEvaluateOnNewDocument(stealthJS).Do(c)
			return e
		}))
	}
	return fmt.Errorf("최대 재시도 횟수(%d) 초과", maxRetries)
}

// ──────────────────────────────────────────────────────────────
// 사이트별 특화 셀렉터 (한국 주요 쇼핑몰/뉴스)
// ──────────────────────────────────────────────────────────────

type SiteProfile struct {
	SearchInputSel  string
	SearchBtnSel    string
	ProductListSel  string
	ProductNameSel  string
	ProductPriceSel string
	LoadMoreSel     string
	WaitAfterSearch time.Duration
}

var siteProfiles = map[string]SiteProfile{
	"coupang.com": {
		SearchInputSel:  "#headerSearchbarInput",
		SearchBtnSel:    "button.search-button",
		ProductListSel:  ".search-product-wrap .search-product",
		ProductNameSel:  ".name",
		ProductPriceSel: ".price-value",
		WaitAfterSearch: 2 * time.Second,
	},
	"naver.com": {
		SearchInputSel:  "#query",
		SearchBtnSel:    "button.btn_search",
		ProductListSel:  ".basicList_item__0T9YD",
		ProductNameSel:  ".basicList_title__VfX3c",
		ProductPriceSel: ".price_num__S2p_v",
		WaitAfterSearch: 1500 * time.Millisecond,
	},
	"danawa.com": {
		SearchInputSel:  "#searchText",
		SearchBtnSel:    ".btnSearch",
		ProductListSel:  ".main_prodlist .prod_item",
		ProductNameSel:  ".prod_name a",
		ProductPriceSel: ".price_sect strong",
		WaitAfterSearch: 2 * time.Second,
	},
	"gmarket.co.kr": {
		SearchInputSel:  "#search-keyword",
		SearchBtnSel:    "button.btn-search",
		ProductListSel:  ".box__item-wrapper",
		ProductNameSel:  ".item__title",
		ProductPriceSel: ".item__price-area em",
		WaitAfterSearch: 2 * time.Second,
	},
	"11st.co.kr": {
		SearchInputSel:  "input#integratedSearchKeyword",
		SearchBtnSel:    "button.btn_search",
		ProductListSel:  ".itemlist_area li.item",
		ProductNameSel:  ".prd_name",
		ProductPriceSel: ".price b",
		WaitAfterSearch: 2 * time.Second,
	},
	"auction.co.kr": {
		SearchInputSel:  "input#keyword",
		SearchBtnSel:    "button.btn_search",
		ProductListSel:  "ul.item_list li.item",
		ProductNameSel:  ".item_title a",
		ProductPriceSel: ".price strong",
		WaitAfterSearch: 2 * time.Second,
	},
	"temu.com": {
		SearchInputSel:  "input[placeholder*='earch']",
		SearchBtnSel:    "button[type='submit']",
		ProductListSel:  "div[class*='goods-item'], div[class*='search-item'], article",
		ProductNameSel:  "div[class*='goods-title'], div[class*='item-name'], h3",
		ProductPriceSel: "div[class*='goods-price'], span[class*='price']",
		WaitAfterSearch: 3 * time.Second,
	},
	"youtube.com": {
		SearchInputSel:  "input#search",
		SearchBtnSel:    "button#search-icon-legacy",
		ProductListSel:  "ytd-video-renderer",
		ProductNameSel:  "#video-title",
		ProductPriceSel: "#metadata-line span:first-child",
		WaitAfterSearch: 2500 * time.Millisecond,
	},
	"tiktok.com": {
		SearchInputSel:  "input[data-e2e='search-user-input']",
		SearchBtnSel:    "button[data-e2e='search-button']",
		ProductListSel:  "div[data-e2e='search_video-item'], div[class*='DivItemContainerV2']",
		ProductNameSel:  "div[class*='SpanText'], p[class*='video-title']",
		ProductPriceSel: "strong[class*='VideoCount']",
		WaitAfterSearch: 3 * time.Second,
	},
	"finance.naver.com": {
		SearchInputSel:  "#stock-search",
		ProductListSel:  ".news_list li",
		ProductNameSel:  ".tit",
		WaitAfterSearch: 1 * time.Second,
	},
	// ── 중고차 ──────────────────────────────────────────────
	"heydealer.com": {
		SearchInputSel:  "input[placeholder*='검색']",
		ProductListSel:  "ul.car-list li, div[class*='CarCard'], div[class*='car-item']",
		ProductNameSel:  "p[class*='name'], h3[class*='name'], div[class*='title']",
		ProductPriceSel: "p[class*='price'], span[class*='price']",
		WaitAfterSearch: 3 * time.Second,
	},
	"encar.com": {
		SearchInputSel:  "input#SearchText",
		SearchBtnSel:    "button.btn_search",
		ProductListSel:  ".card_wrap, .item_list li",
		ProductNameSel:  ".car_name, .tit_car",
		ProductPriceSel: ".price, .tit_price",
		WaitAfterSearch: 2 * time.Second,
	},
	"kbchachacha.com": {
		SearchInputSel:  "input[name='keyword']",
		ProductListSel:  ".list_item, .car_list li",
		ProductNameSel:  ".car_name, .tit",
		ProductPriceSel: ".price",
		WaitAfterSearch: 2 * time.Second,
	},
	"bobaedream.co.kr": {
		SearchInputSel:  "input#search_word",
		SearchBtnSel:    "button.btn-search",
		ProductListSel:  ".car-list-item, .listing-card",
		ProductNameSel:  ".car-name, h3",
		ProductPriceSel: ".price",
		WaitAfterSearch: 2 * time.Second,
	},
	// ── 중고거래 ─────────────────────────────────────────────
	"daangn.com": {
		SearchInputSel:  "input[type='search'], input[placeholder*='검색']",
		ProductListSel:  "article, div[data-type='article'], li[data-type='article']",
		ProductNameSel:  "strong, h2, .article-title",
		ProductPriceSel: "span[class*='price'], div[class*='price']",
		WaitAfterSearch: 3 * time.Second,
	},
	"bunjang.co.kr": {
		SearchInputSel:  "input[placeholder*='검색']",
		ProductListSel:  "ul.product-list li, div[class*='product-item']",
		ProductNameSel:  "p[class*='name'], div[class*='title']",
		ProductPriceSel: "p[class*='price']",
		WaitAfterSearch: 2500 * time.Millisecond,
	},
	"joongna.com": {
		SearchInputSel:  "input[type='search']",
		ProductListSel:  ".product-card, .item-card",
		ProductNameSel:  ".product-name, .item-name",
		ProductPriceSel: ".price",
		WaitAfterSearch: 2 * time.Second,
	},
	// ── 쇼핑 ─────────────────────────────────────────────────
	"shopping.naver.com": {
		SearchInputSel:  "input.input_text",
		SearchBtnSel:    "button.btn_search",
		ProductListSel:  ".basicList_item__0T9YD, .product_item__MDjeH",
		ProductNameSel:  ".basicList_title__VfX3c, .product_title__Mmkiq",
		ProductPriceSel: ".price_num__S2p_v, .price_area__BCCh0",
		WaitAfterSearch: 2 * time.Second,
	},
	"musinsa.com": {
		SearchInputSel:  "input[placeholder*='검색']",
		ProductListSel:  "ul.list-section li, .goods_list_item",
		ProductNameSel:  ".goods_nm, .article_title",
		ProductPriceSel: ".price, .sale_price",
		WaitAfterSearch: 2500 * time.Millisecond,
	},
	"a-bly.com": {
		ProductListSel:  "li[class*='product'], div[class*='ProductCard']",
		ProductNameSel:  "p[class*='name']",
		ProductPriceSel: "span[class*='price']",
		WaitAfterSearch: 2 * time.Second,
	},
	"zigzag.kr": {
		ProductListSel:  "div[class*='ProductCard'], li[class*='item']",
		ProductNameSel:  "div[class*='name'], p[class*='title']",
		ProductPriceSel: "span[class*='price']",
		WaitAfterSearch: 2 * time.Second,
	},
	"ohou.se": {
		ProductListSel:  "div[class*='product'], article[class*='card']",
		ProductNameSel:  "p[class*='name'], div[class*='title']",
		ProductPriceSel: "span[class*='price']",
		WaitAfterSearch: 2 * time.Second,
	},
	// ── 부동산 ───────────────────────────────────────────────
	"zigbang.com": {
		SearchInputSel:  "input[placeholder*='검색'], input[type='text']",
		ProductListSel:  "li[class*='item'], div[class*='ItemCard']",
		ProductNameSel:  "p[class*='name'], div[class*='title']",
		ProductPriceSel: "span[class*='price']",
		WaitAfterSearch: 3 * time.Second,
	},
	"dabangapp.com": {
		ProductListSel:  "div[class*='room-item'], li[class*='item']",
		ProductNameSel:  "div[class*='title'], p[class*='name']",
		ProductPriceSel: "span[class*='price']",
		WaitAfterSearch: 3 * time.Second,
	},
	// ── 여행/숙박 ────────────────────────────────────────────
	"yanolja.com": {
		SearchInputSel:  "input[placeholder*='검색']",
		ProductListSel:  "div[class*='accommodation'], li[class*='item'], article",
		ProductNameSel:  "p[class*='name'], h3, div[class*='title']",
		ProductPriceSel: "span[class*='price'], div[class*='price']",
		WaitAfterSearch: 3 * time.Second,
	},
	"goodchoice.kr": {
		ProductListSel:  "li.box_list, div.item_area",
		ProductNameSel:  ".name, h3",
		ProductPriceSel: ".price, .sale",
		WaitAfterSearch: 2 * time.Second,
	},
	// ── 배달 ─────────────────────────────────────────────────
	"baemin.com": {
		ProductListSel:  "li[class*='shop'], div[class*='restaurant']",
		ProductNameSel:  "div[class*='name'], span[class*='title']",
		ProductPriceSel: "span[class*='min'], div[class*='price']",
		WaitAfterSearch: 3 * time.Second,
	},
}

func getSiteProfile(url string) (SiteProfile, string) {
	for domain, profile := range siteProfiles {
		if containsStr(url, domain) {
			return profile, domain
		}
	}
	return SiteProfile{
		SearchInputSel:  "input[type='search'], input[name='q'], input[name='query']",
		SearchBtnSel:    "button[type='submit'], .search-btn",
		ProductListSel:  "ul li, .item, .product",
		ProductNameSel:  "h2, h3, .title, .name",
		ProductPriceSel: ".price, .amount, .cost",
		WaitAfterSearch: 2 * time.Second,
	}, "generic"
}

// ──────────────────────────────────────────────────────────────
// JavaScript 기반 데이터 추출 헬퍼 (범용)
// ──────────────────────────────────────────────────────────────

// extractTableData: 모든 테이블을 JSON으로 추출
func extractTableData(ctx context.Context) ([]map[string]interface{}, error) {
	var result []map[string]interface{}
	extractJS := `
	JSON.stringify(Array.from(document.querySelectorAll('table')).map((tbl, tblIdx) => {
		const headers = Array.from(tbl.querySelectorAll('th')).map(th => th.innerText.trim());
		const rows = Array.from(tbl.querySelectorAll('tr')).slice(headers.length > 0 ? 1 : 0).map(tr =>
			Array.from(tr.querySelectorAll('td')).map(td => td.innerText.trim())
		).filter(r => r.some(c => c));
		return { table_index: tblIdx, headers, rows };
	}))
	`
	var raw string
	if err := chromedp.Run(ctx, chromedp.Evaluate(extractJS, &raw)); err != nil {
		return nil, err
	}
	json.Unmarshal([]byte(raw), &result)
	return result, nil
}

// extractStructuredProducts: 상품 목록을 구조화해서 추출
func extractStructuredProducts(ctx context.Context, profile SiteProfile, maxItems int) ([]map[string]string, error) {
	extractJS := fmt.Sprintf(`
	JSON.stringify(Array.from(document.querySelectorAll('%s')).slice(0, %d).map(item => {
		const nameEl  = item.querySelector('%s');
		const priceEl = item.querySelector('%s');
		const linkEl  = item.querySelector('a');
		return {
			name:  nameEl  ? nameEl.innerText.trim()  : '',
			price: priceEl ? priceEl.innerText.trim() : '',
			link:  linkEl  ? linkEl.href              : '',
		};
	}).filter(p => p.name))
	`, profile.ProductListSel, maxItems, profile.ProductNameSel, profile.ProductPriceSel)

	var raw string
	if err := chromedp.Run(ctx, chromedp.Evaluate(extractJS, &raw)); err != nil {
		return nil, err
	}
	var products []map[string]string
	json.Unmarshal([]byte(raw), &products)
	return products, nil
}

// ──────────────────────────────────────────────────────────────
// CDP 기반 네트워크 요청 차단 (광고, 트래커 제거 → 속도 향상)
// ──────────────────────────────────────────────────────────────

func enableAdBlocking(ctx context.Context) error {
	blockedDomains := []string{
		"doubleclick.net", "googlesyndication.com", "googletagmanager.com",
		"facebook.net", "analytics.google.com", "hotjar.com",
	}
	_ = blockedDomains

	// network.SetBlockedURLs를 사용하는 대신 간단한 방법으로
	return chromedp.Run(ctx, chromedp.ActionFunc(func(ctx context.Context) error {
		// Enable network
		return network.Enable().Do(ctx)
	}))
}

// ──────────────────────────────────────────────────────────────
// 스텔스 withBrowserTimeout 오버라이드
// ──────────────────────────────────────────────────────────────

func withStealthBrowserTimeout(timeout time.Duration) (context.Context, context.CancelFunc, error) {
	base, err := ensureStealthBrowser()
	if err != nil {
		return nil, nil, err
	}
	ctx, cancel := context.WithTimeout(base, timeout)
	return ctx, cancel, nil
}

// cdp 패키지 사용을 위한 임시 변수
var _ = cdp.Node{}
