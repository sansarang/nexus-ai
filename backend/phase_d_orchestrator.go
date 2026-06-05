// phase_d_orchestrator.go — Phase D Orchestrator
// 12개 Agent 조율 + 패치 제안 큐 + 수준별 자동 대수 결정
// 사장님 정책: low → 자동 / medium → 10분 후 자동 / high+critical → 승인 필수

package main

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	patchQueueMu       sync.RWMutex
	patchProposalQueue = make([]PatchProposal, 0, 100)
	patchHistory       = make([]PatchProposal, 0, 500) // 적용/거부 이력
)

func patchHistoryPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".nexus", "patch_history.json")
}

func loadPatchHistory() {
	data, err := os.ReadFile(patchHistoryPath())
	if err != nil {
		return
	}
	patchQueueMu.Lock()
	json.Unmarshal(data, &patchHistory)
	patchQueueMu.Unlock()
}

func savePatchHistory() {
	patchQueueMu.RLock()
	data, _ := json.Marshal(patchHistory)
	patchQueueMu.RUnlock()
	p := patchHistoryPath()
	os.MkdirAll(filepath.Dir(p), 0755)
	os.WriteFile(p, data, 0600)
}

// addToPatchQueue — 중복 제거 + 큐 추가
func addToPatchQueue(proposals []PatchProposal) {
	patchQueueMu.Lock()
	defer patchQueueMu.Unlock()
	// 중복 키 (agent + title) 24시간 내 재제안 차단
	now := time.Now().Unix()
	for _, p := range proposals {
		dup := false
		for _, existing := range patchProposalQueue {
			if existing.AgentName == p.AgentName && existing.Title == p.Title {
				dup = true
				break
			}
		}
		// 히스토리 24h 내 동일 제안 차단
		for _, h := range patchHistory {
			if h.AgentName == p.AgentName && h.Title == p.Title && now-h.ProposedAt < 86400 {
				dup = true
				break
			}
		}
		if !dup {
			patchProposalQueue = append(patchProposalQueue, p)
		}
	}
	if len(patchProposalQueue) > 50 {
		patchProposalQueue = patchProposalQueue[len(patchProposalQueue)-50:]
	}
}

// getPendingProposals — 프론트 폴링용
func getPendingProposals() []PatchProposal {
	patchQueueMu.RLock()
	defer patchQueueMu.RUnlock()
	out := make([]PatchProposal, 0, len(patchProposalQueue))
	for _, p := range patchProposalQueue {
		if p.Status == "proposed" {
			out = append(out, p)
		}
	}
	return out
}

// approvePatch — 사용자 승인 → 상태 변경 + executeHeal() 실행
func approvePatch(id string) (PatchProposal, bool) {
	patchQueueMu.Lock()
	defer patchQueueMu.Unlock()
	for i, p := range patchProposalQueue {
		if p.ID == id {
			p.Status = "approved"
			patchProposalQueue[i] = p
			patchHistory = append(patchHistory, p)
			go savePatchHistory()
			// ★ 실제 힐링 실행 — LLM 호출 + 프롬프트 파일 저장
			go executeHeal(p)
			return p, true
		}
	}
	return PatchProposal{}, false
}

// rejectPatch — 사용자 거부
func rejectPatch(id string) bool {
	patchQueueMu.Lock()
	defer patchQueueMu.Unlock()
	for i, p := range patchProposalQueue {
		if p.ID == id {
			p.Status = "rejected"
			patchProposalQueue[i] = p
			patchHistory = append(patchHistory, p)
			go savePatchHistory()
			return true
		}
	}
	return false
}

// autoApplyTick — 1분마다 호출, low/medium 자동 승인
func autoApplyTick() {
	patchQueueMu.Lock()
	defer patchQueueMu.Unlock()
	now := time.Now().Unix()
	for i, p := range patchProposalQueue {
		if p.Status != "proposed" {
			continue
		}
		if p.AutoApplyAfter > 0 && now >= p.AutoApplyAfter {
			// 위험도가 high/critical 이면 자동 적용 X
			if p.Severity == SevHigh || p.Severity == SevCritical {
				continue
			}
			p.Status = "approved"
			patchProposalQueue[i] = p
			patchHistory = append(patchHistory, p)
			// ★ 자동 승인도 실제 힐링 실행
			go executeHeal(p)
		}
	}
	go savePatchHistory()
}

// startSelfHealingLoop — 시작 시 1회
//   - 5분마다 모든 Agent 분석 → 패치 제안 큐 추가
//   - 1분마다 자동 적용 체크 (low/medium)
func startSelfHealingLoop() {
	go func() {
		loadPatchHistory()
		analyzeTicker := time.NewTicker(5 * time.Minute)
		applyTicker := time.NewTicker(1 * time.Minute)
		defer analyzeTicker.Stop()
		defer applyTicker.Stop()
		for {
			select {
			case <-analyzeTicker.C:
				proposals := runAllAgents()
				if len(proposals) > 0 {
					addToPatchQueue(proposals)
				}
			case <-applyTicker.C:
				autoApplyTick()
			}
		}
	}()
}

// ── HTTP 엔드포인트 (프론트가 폴링) ─────────────────────────

// handleAgentPatchProposals — GET /api/agent/proposals
func handleAgentPatchProposals(w http.ResponseWriter, r *http.Request) {
	props := getPendingProposals()
	json200(w, map[string]any{
		"success":   true,
		"proposals": props,
		"count":     len(props),
	})
}

// handleAgentApprove — POST /api/agent/approve
func handleAgentApprove(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID string `json:"id"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	p, ok := approvePatch(req.ID)
	if !ok {
		writeJSON(w, 404, map[string]any{"success": false, "message": "proposal not found"})
		return
	}
	json200(w, map[string]any{"success": true, "patch": p})
}

// handleAgentReject — POST /api/agent/reject
func handleAgentReject(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID string `json:"id"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if !rejectPatch(req.ID) {
		writeJSON(w, 404, map[string]any{"success": false, "message": "proposal not found"})
		return
	}
	json200(w, map[string]any{"success": true})
}

// handleAgentStatus — GET /api/agent/status (대시보드)
func handleAgentStatus(w http.ResponseWriter, r *http.Request) {
	entries := getTelemetrySnapshot(500)
	stats := computeTelemetryStats(entries)
	patchQueueMu.RLock()
	pending := 0
	applied := 0
	rejected := 0
	for _, p := range patchProposalQueue {
		if p.Status == "proposed" {
			pending++
		}
	}
	for _, p := range patchHistory {
		if p.Status == "approved" {
			applied++
		} else if p.Status == "rejected" {
			rejected++
		}
	}
	patchQueueMu.RUnlock()
	// 힐링된 프롬프트 파일 목록 (실제 적용 여부 확인용)
	healedPrompts := []string{}
	for _, name := range []string{"system", "intent", "persona_base"} {
		if _, err := os.Stat(promptFilePath(name)); err == nil {
			healedPrompts = append(healedPrompts, name)
		}
	}

	json200(w, map[string]any{
		"success":        true,
		"telemetry":      stats,
		"pending":        pending,
		"applied":        applied,
		"rejected":       rejected,
		"agent_count":    len(phaseDAgents),
		"healed_prompts": healedPrompts,
	})
}


// markPatchApplied — executeHeal 성공 후 상태 업데이트
func markPatchApplied(id, promptName string) {
	patchQueueMu.Lock()
	defer patchQueueMu.Unlock()
	for i, p := range patchProposalQueue {
		if p.ID == id {
			p.Status = "applied"
			if p.Evidence == nil {
				p.Evidence = map[string]any{}
			}
			p.Evidence["healed_prompt"] = promptName
			patchProposalQueue[i] = p
			break
		}
	}
	go savePatchHistory()
}

// markPatchFailed — executeHeal 실패 후 상태 업데이트
func markPatchFailed(id, reason string) {
	patchQueueMu.Lock()
	defer patchQueueMu.Unlock()
	for i, p := range patchProposalQueue {
		if p.ID == id {
			p.Status = "failed"
			if p.Evidence == nil {
				p.Evidence = map[string]any{}
			}
			p.Evidence["fail_reason"] = reason
			patchProposalQueue[i] = p
			break
		}
	}
	go savePatchHistory()
}

// handleAgentAnalyzeNow — POST /api/agent/analyze-now (수동 트리거, 검증/테스트용)
func handleAgentAnalyzeNow(w http.ResponseWriter, r *http.Request) {
	proposals := runAllAgents()
	addToPatchQueue(proposals)
	json200(w, map[string]any{
		"success":   true,
		"generated": len(proposals),
		"proposals": proposals,
	})
}
