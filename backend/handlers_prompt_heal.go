// handlers_prompt_heal.go — 자가치유 AI: 프롬프트 파일 기반 관리 + LLM 자동 개선
//
// 흐름:
//   텔레메트리 이상 감지 → Agent가 PatchProposal 생성
//   → approvePatch() 호출 → executeHeal() → LLM에게 "이 프롬프트 고쳐줘"
//   → ~/.nexus/prompts/{name}.txt 저장 → 다음 요청부터 즉시 반영
//
// 핵심: Go 소스코드 수정 없이 AI 행동(프롬프트)만 런타임에 교체

package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ── 파일 경로 ──────────────────────────────────────────────

func promptsDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".nexus", "prompts")
}

func promptFilePath(name string) string {
	return filepath.Join(promptsDir(), name+".txt")
}

// ── 읽기 / 쓰기 ───────────────────────────────────────────

// loadHealedPrompt: 힐링된 버전이 있으면 그걸, 없으면 기본값(fallback) 반환
func loadHealedPrompt(name, fallback string) string {
	data, err := os.ReadFile(promptFilePath(name))
	if err != nil {
		return fallback
	}
	s := strings.TrimSpace(string(data))
	if s == "" {
		return fallback
	}
	return s
}

// saveHealedPrompt: 개선된 프롬프트를 파일에 저장
func saveHealedPrompt(name, content string) error {
	dir := promptsDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(promptFilePath(name), []byte(content), 0644)
}

// ── LLM으로 프롬프트 개선 ─────────────────────────────────

// healPromptWithLLM: 현재 프롬프트 + 문제 근거 → LLM → 개선된 프롬프트
func healPromptWithLLM(proposal PatchProposal, currentPrompt string) (string, error) {
	metaLines := []string{}
	for k, v := range proposal.Evidence {
		metaLines = append(metaLines, fmt.Sprintf("  - %s: %v", k, v))
	}
	evidenceStr := strings.Join(metaLines, "\n")

	systemMsg := `You are a prompt engineer AI. Your job is to improve a system prompt based on observed failure evidence.
Rules:
1. Return ONLY the improved prompt text. No explanation, no markdown, no code blocks.
2. Keep the original structure and language (Korean system prompts stay Korean).
3. Make targeted fixes based on the evidence — do not rewrite everything.
4. Preserve all existing action/JSON format instructions.`

	userMsg := fmt.Sprintf(`Current prompt (name: %s):
---
%s
---

Problem detected by agent "%s":
Title: %s
Description: %s
Evidence:
%s

Fix the prompt to address the problem above. Return only the improved prompt text.`,
		proposal.TargetFile,
		currentPrompt,
		proposal.AgentName,
		proposal.Title,
		proposal.Description,
		evidenceStr,
	)

	msgs := []groqMsg{
		{Role: "system", Content: systemMsg},
		{Role: "user", Content: userMsg},
	}

	// 기존 LLM 인프라 재사용 (Groq → Ollama 폴백)
	result, provider, err := callGroqWithFallback(msgs, 2048, false)
	if err != nil {
		return "", fmt.Errorf("healPromptWithLLM LLM error: %w", err)
	}
	if result == "" {
		return "", fmt.Errorf("healPromptWithLLM: empty response from %s", provider)
	}

	// 코드블록 래퍼 제거 (LLM이 실수로 붙이는 경우)
	result = strings.TrimSpace(result)
	result = strings.TrimPrefix(result, "```")
	result = strings.TrimSuffix(result, "```")
	result = strings.TrimSpace(result)

	log.Printf("[HealPrompt] '%s' improved via %s (%d chars → %d chars)",
		proposal.TargetFile, provider, len(currentPrompt), len(result))

	return result, nil
}

// ── 프롬프트 이름 → 현재 내용 매핑 ──────────────────────────

// promptNameFromProposal: 패치 제안에서 어떤 프롬프트를 고칠지 결정
func promptNameFromProposal(p PatchProposal) string {
	target := strings.ToLower(p.TargetFile + " " + p.AgentName)

	switch {
	case strings.Contains(target, "intent") ||
		strings.Contains(target, "haiku") ||
		strings.Contains(target, "classify") ||
		p.AgentName == "KoreanNLP":
		return "intent"

	case strings.Contains(target, "system") ||
		strings.Contains(target, "nexus") ||
		strings.Contains(target, "command") ||
		p.AgentName == "LLMPrompt" ||
		p.AgentName == "GoBackend":
		return "system"

	case strings.Contains(target, "persona") ||
		p.AgentName == "CardDesigner":
		return "persona_base"
	}
	// 기본: 시스템 프롬프트
	return "system"
}

// currentPromptContent: 프롬프트 이름 → 현재 내용 (힐링본 우선)
func currentPromptContent(name string) string {
	switch name {
	case "intent":
		return loadHealedPrompt("intent", haikuIntentPromptDefault)
	case "system":
		// getSystemPrompt()는 플랫폼별 파일에 정의 (Windows: 전체 프롬프트, stub: "")
		current := getSystemPrompt()
		if current == "" {
			return ""
		}
		return current
	case "persona_base":
		return loadHealedPrompt("persona_base", "")
	default:
		return loadHealedPrompt(name, "")
	}
}

// ── 실행 진입점 ───────────────────────────────────────────

// executeHeal: 승인된 패치를 실제로 적용 (LLM 호출 + 파일 저장)
// phase_d_orchestrator.go의 approvePatch()에서 goroutine으로 호출
func executeHeal(p PatchProposal) {
	name := promptNameFromProposal(p)
	current := currentPromptContent(name)

	if current == "" {
		log.Printf("[HealPrompt] skip '%s': no base prompt found", name)
		return
	}

	improved, err := healPromptWithLLM(p, current)
	if err != nil {
		log.Printf("[HealPrompt] ERROR for proposal %s: %v", p.ID, err)
		markPatchFailed(p.ID, err.Error())
		return
	}

	if err := saveHealedPrompt(name, improved); err != nil {
		log.Printf("[HealPrompt] save error: %v", err)
		markPatchFailed(p.ID, err.Error())
		return
	}

	markPatchApplied(p.ID, name)
	log.Printf("[HealPrompt] ✅ prompt '%s' healed and saved (proposal %s)", name, p.ID)

	// 힐링 이벤트를 텔레메트리에 기록
	recordTelemetry(TelemetryEntry{
		Timestamp: time.Now().Unix(),
		Action:    "self_heal",
		CardType:  "patch_applied",
		Params: map[string]any{
			"prompt":   name,
			"agent":    p.AgentName,
			"proposal": p.ID,
		},
		Success: true,
	})
}
