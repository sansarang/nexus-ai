// response_cache_shared.go — handleCommand 응답 캐시 (Mac/Windows 공용)
// 자비스 응답 속도 개선: 자주 묻는 쿼리 (PC 상태/날씨/뉴스 등) 60초간 메모리 캐시
package main

import (
	"strings"
	"sync"
	"time"
)

type cmdCacheEntry struct {
	resp    CommandResponse
	expires time.Time
}

var (
	cmdCacheMu sync.RWMutex
	cmdCache   = make(map[string]cmdCacheEntry)
)

const (
	cmdCacheTTL     = 60 * time.Second // 기본 1분 (날씨/뉴스 등 자주 묻는 거)
	cmdCacheMaxSize = 200              // 최대 200개 항목 (메모리 절약)
)

// normalizeQuery: 캐시 키 정규화 (공백 / 대소문자 영향 무시)
func normalizeQuery(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	// 연속 공백 → 단일 공백
	fields := strings.Fields(s)
	return strings.Join(fields, " ")
}

// cmdCacheGet: 캐시 hit 시 응답 반환 (true)
func cmdCacheGet(query, lang string) (CommandResponse, bool) {
	key := lang + "|" + normalizeQuery(query)
	cmdCacheMu.RLock()
	entry, ok := cmdCache[key]
	cmdCacheMu.RUnlock()
	if !ok {
		return CommandResponse{}, false
	}
	if time.Now().After(entry.expires) {
		// 만료 — 비동기 삭제 (lock 짧게)
		go func() {
			cmdCacheMu.Lock()
			delete(cmdCache, key)
			cmdCacheMu.Unlock()
		}()
		return CommandResponse{}, false
	}
	return entry.resp, true
}

// cmdCacheSet: 응답 저장. success=false 인 경우는 저장 안 함 (재시도 가능하게)
func cmdCacheSet(query, lang string, resp CommandResponse) {
	if !resp.Success {
		return
	}
	// 캐시 안 할 action: 시간 의존적 (날짜 명령), 일회성 (저장/삭제)
	noCacheActions := map[string]bool{
		"calendar_add": true, "calendar_today": true, "calendar_week": true,
		"email_send": true, "schedule_add": true, "macro_create": true,
		"note": true, "excel_save": true, "clarify": true,
	}
	if noCacheActions[resp.Action] {
		return
	}
	key := lang + "|" + normalizeQuery(query)
	cmdCacheMu.Lock()
	// 사이즈 제한: 초과 시 가장 오래된 것부터 제거 (간단히 모두 비우는 LRU 근사)
	if len(cmdCache) >= cmdCacheMaxSize {
		// 만료된 것부터 제거
		now := time.Now()
		for k, v := range cmdCache {
			if now.After(v.expires) {
				delete(cmdCache, k)
			}
		}
		// 여전히 가득이면 전부 제거 (간단 LRU)
		if len(cmdCache) >= cmdCacheMaxSize {
			cmdCache = make(map[string]cmdCacheEntry)
		}
	}
	cmdCache[key] = cmdCacheEntry{
		resp:    resp,
		expires: time.Now().Add(cmdCacheTTL),
	}
	cmdCacheMu.Unlock()
}
