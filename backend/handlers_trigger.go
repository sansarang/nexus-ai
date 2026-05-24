//go:build windows

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type TriggerCondition struct {
	Type      string  `json:"type"`      // cpu_above | memory_above | battery_below | time_at | interval
	Threshold float64 `json:"threshold"` // % 값 또는 분
	TimeStr   string  `json:"time_str"`  // "09:00"
}

type AlertTrigger struct {
	ID        string           `json:"id"`
	Name      string           `json:"name"`
	Condition TriggerCondition `json:"condition"`
	Message   string           `json:"message"`
	Active    bool             `json:"active"`
	Fired     bool             `json:"fired"`
	LastFired time.Time        `json:"last_fired,omitempty"`
	CreatedAt time.Time        `json:"created_at"`
}

type triggerBroadcaster struct {
	mu      sync.RWMutex
	clients map[chan string]struct{}
}

var globalTrigger = &triggerBroadcaster{clients: make(map[chan string]struct{})}
var (
	triggerStoreMu sync.RWMutex
	triggerStore   = make(map[string]*AlertTrigger)
)

func triggerStorePath() string {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		home, _ := os.UserHomeDir()
		appData = filepath.Join(home, "AppData", "Roaming")
	}
	dir := filepath.Join(appData, "Nexus")
	os.MkdirAll(dir, 0755)
	return filepath.Join(dir, "triggers.json")
}

func loadTriggers() {
	data, err := os.ReadFile(triggerStorePath())
	if err != nil {
		return
	}
	var triggers []*AlertTrigger
	if json.Unmarshal(data, &triggers) == nil {
		triggerStoreMu.Lock()
		for _, t := range triggers {
			if t.Active {
				triggerStore[t.ID] = t
			}
		}
		triggerStoreMu.Unlock()
	}
}

func saveTriggers() {
	triggerStoreMu.RLock()
	list := make([]*AlertTrigger, 0, len(triggerStore))
	for _, t := range triggerStore {
		list = append(list, t)
	}
	triggerStoreMu.RUnlock()
	data, _ := json.MarshalIndent(list, "", "  ")
	os.WriteFile(triggerStorePath(), data, 0600)
}

func (b *triggerBroadcaster) subscribe() chan string {
	ch := make(chan string, 8)
	b.mu.Lock()
	b.clients[ch] = struct{}{}
	b.mu.Unlock()
	return ch
}

func (b *triggerBroadcaster) unsubscribe(ch chan string) {
	b.mu.Lock()
	delete(b.clients, ch)
	b.mu.Unlock()
}

func (b *triggerBroadcaster) broadcast(msg string) {
	b.mu.RLock()
	for ch := range b.clients {
		select {
		case ch <- msg:
		default:
		}
	}
	b.mu.RUnlock()
}

// getCPUPercentWindows: PowerShell로 실제 CPU 사용률
func getCPUPercentWindows() float64 {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive",
		"-Command", "(Get-WmiObject -Query 'SELECT LoadPercentage FROM Win32_Processor' | Measure-Object LoadPercentage -Average).Average").Output()
	if err != nil {
		return 0
	}
	var v float64
	fmt.Sscanf(strings.TrimSpace(string(out)), "%f", &v)
	return v
}

func initTriggerEngine() {
	loadTriggers()
	go runTriggerLoop()
	log.Println("[Trigger] 조건부 알림 엔진 시작됨")
}

func runTriggerLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		checkTriggers()
	}
}

func checkTriggers() {
	now := time.Now()
	triggerStoreMu.RLock()
	list := make([]*AlertTrigger, 0, len(triggerStore))
	for _, t := range triggerStore {
		if t.Active {
			list = append(list, t)
		}
	}
	triggerStoreMu.RUnlock()

	for _, t := range list {
		fired := false
		switch t.Condition.Type {
		case "cpu_above":
			cpu := getCPUPercentWindows()
			if cpu >= t.Condition.Threshold {
				fired = true
				t.Message = fmt.Sprintf("⚠️ CPU 사용률 %.1f%% — %.0f%% 초과!", cpu, t.Condition.Threshold)
			}
		case "memory_above":
			mem := float64(getMemoryUsage())
			if mem >= t.Condition.Threshold {
				fired = true
				t.Message = fmt.Sprintf("⚠️ 메모리 사용률 %.1f%% — %.0f%% 초과!", mem, t.Condition.Threshold)
			}
		case "time_at":
			target := t.Condition.TimeStr
			current := now.Format("15:04")
			if current == target && (t.LastFired.IsZero() || now.Sub(t.LastFired) > 23*time.Hour) {
				fired = true
			}
		case "interval":
			if t.LastFired.IsZero() || now.Sub(t.LastFired) >= time.Duration(t.Condition.Threshold)*time.Minute {
				fired = true
			}
		}

		if fired {
			payload, _ := json.Marshal(map[string]any{
				"type":    "trigger_alert",
				"id":      t.ID,
				"name":    t.Name,
				"message": t.Message,
				"time":    now.Format("15:04:05"),
			})
			globalTrigger.broadcast(string(payload))
			log.Printf("[Trigger] 발화: %s — %s", t.Name, t.Message)

			triggerStoreMu.Lock()
			if st, ok := triggerStore[t.ID]; ok {
				st.LastFired = now
				if t.Condition.Type == "time_at" && t.Fired {
					st.Active = false
				}
			}
			triggerStoreMu.Unlock()
			saveTriggers()
		}
	}
}

func parseTriggerFromNL(msg string) *AlertTrigger {
	lower := strings.ToLower(msg)
	t := &AlertTrigger{
		ID: fmt.Sprintf("%d", time.Now().UnixMilli()), Active: true, CreatedAt: time.Now(), Name: msg,
	}
	if len(t.Name) > 40 {
		t.Name = t.Name[:40]
	}
	switch {
	case strings.Contains(lower, "cpu") && (strings.Contains(lower, "넘으면") || strings.Contains(lower, "이상") || strings.Contains(lower, "above") || strings.Contains(lower, "over")):
		threshold := 80.0
		fmt.Sscanf(msg, "%f", &threshold)
		t.Condition = TriggerCondition{Type: "cpu_above", Threshold: threshold}
		t.Message = fmt.Sprintf("CPU %.0f%% 초과 감지!", threshold)
	case (strings.Contains(lower, "메모리") || strings.Contains(lower, "memory") || strings.Contains(lower, "ram")) &&
		(strings.Contains(lower, "넘으면") || strings.Contains(lower, "이상") || strings.Contains(lower, "above")):
		threshold := 80.0
		fmt.Sscanf(msg, "%f", &threshold)
		t.Condition = TriggerCondition{Type: "memory_above", Threshold: threshold}
		t.Message = fmt.Sprintf("메모리 %.0f%% 초과!", threshold)
	case strings.Contains(lower, "시에") || strings.Contains(lower, "시 알") || strings.Contains(lower, "at "):
		var hour, min int
		if n, _ := fmt.Sscanf(msg, "%d시 %d분", &hour, &min); n >= 1 {
			t.Condition = TriggerCondition{Type: "time_at", TimeStr: fmt.Sprintf("%02d:%02d", hour, min)}
			t.Message = fmt.Sprintf("⏰ %02d:%02d 알림: %s", hour, min, msg)
		} else {
			return nil
		}
	case strings.Contains(lower, "분마다") || strings.Contains(lower, "every") || strings.Contains(lower, "minutes"):
		var mins float64 = 60
		fmt.Sscanf(msg, "%f분", &mins)
		t.Condition = TriggerCondition{Type: "interval", Threshold: mins}
		t.Message = fmt.Sprintf("⏱ %.0f분 간격 알림", mins)
	default:
		return nil
	}
	return t
}

// POST /api/trigger/add
func handleTriggerAdd(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name      string           `json:"name"`
		Message   string           `json:"message"`
		Condition TriggerCondition `json:"condition"`
		NL        string           `json:"nl"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	var t *AlertTrigger
	if req.NL != "" {
		t = parseTriggerFromNL(req.NL)
		if t == nil {
			prompt := fmt.Sprintf(`다음 자연어 알림 설정을 JSON으로 변환해줘.
입력: "%s"
JSON만 출력:
{"type":"cpu_above|memory_above|time_at|interval","threshold":80,"time_str":"09:00","name":"알림명"}`, req.NL)
			raw, _, err := callGroqWithFallback([]groqMsg{{Role: "user", Content: prompt}}, 150, true)
			if err == nil {
				var parsed TriggerCondition
				if json.Unmarshal([]byte(strings.TrimSpace(raw)), &parsed) == nil {
					t = &AlertTrigger{
						ID: fmt.Sprintf("%d", time.Now().UnixMilli()), Name: req.NL, Active: true,
						CreatedAt: time.Now(), Condition: parsed, Message: req.NL,
					}
				}
			}
		}
	} else {
		t = &AlertTrigger{
			ID: fmt.Sprintf("%d", time.Now().UnixMilli()), Name: req.Name, Active: true,
			CreatedAt: time.Now(), Condition: req.Condition, Message: req.Message,
		}
	}

	if t == nil {
		writeJSON(w, 400, map[string]any{"success": false, "message": msgT("트리거 조건을 파싱할 수 없습니다. 예: 'CPU 80% 넘으면 알려줘', '매일 9시에 알려줘'", "Cannot parse trigger condition. e.g. 'Alert me when CPU exceeds 80%'", getLang(r))})
		return
	}

	triggerStoreMu.Lock()
	triggerStore[t.ID] = t
	triggerStoreMu.Unlock()
	saveTriggers()
	json200(w, map[string]any{"success": true, "trigger": t, "message": fmt.Sprintf("✅ '%s' 알림 트리거 등록됨", t.Name)})
}

// GET /api/trigger/list
func handleTriggerList(w http.ResponseWriter, r *http.Request) {
	triggerStoreMu.RLock()
	list := make([]*AlertTrigger, 0, len(triggerStore))
	for _, t := range triggerStore {
		list = append(list, t)
	}
	triggerStoreMu.RUnlock()
	json200(w, map[string]any{"success": true, "triggers": list, "count": len(list)})
}

// DELETE /api/trigger/delete?id=xxx
func handleTriggerDelete(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	triggerStoreMu.Lock()
	_, ok := triggerStore[id]
	if ok {
		delete(triggerStore, id)
	}
	triggerStoreMu.Unlock()
	if !ok {
		writeJSON(w, 404, map[string]any{"success": false, "message": msgT("트리거를 찾을 수 없습니다", "Trigger not found", getLang(r))})
		return
	}
	saveTriggers()
	json200(w, map[string]any{"success": true, "message": msgT("트리거 삭제됨", "Trigger deleted", getLang(r))})
}

// GET /api/trigger/events — SSE 스트림
func handleTriggerEvents(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSON(w, 500, map[string]any{"error": "SSE not supported"})
		return
	}

	ch := globalTrigger.subscribe()
	defer globalTrigger.unsubscribe(ch)

	fmt.Fprintf(w, "data: {\"type\":\"connected\"}\n\n")
	flusher.Flush()

	for {
		select {
		case msg := <-ch:
			fmt.Fprintf(w, "data: %s\n\n", msg)
			flusher.Flush()
		case <-r.Context().Done():
			return
		case <-time.After(30 * time.Second):
			fmt.Fprintf(w, ": ping\n\n")
			flusher.Flush()
		}
	}
}
