/**
 * dynamicLLM — LLM 이 자유 질문에 Block[] 로 응답할 수 있게 가르치는 유틸.
 *
 * 핵심 아이디어:
 *  1) 사용자가 단순 인사 ("안녕") → text 응답
 *  2) 사용자가 분석/비교/구조화 질문 ("매출 분석") → Block[] 응답
 *
 * LLM 은 다음 중 하나를 JSON 으로 반환:
 *   { "format": "text", "content": "...", "emotion": "happy" }
 *   { "format": "blocks", "text": "한 줄 요약", "blocks": [...], "emotion": "happy" }
 */

import type { Block } from '../../components/FloatingCharacter/cards/DynamicBlocks'

export type DynamicLLMResult =
  | { format: 'text';   text: string; emotion?: 'neutral'|'happy'|'concerned'|'alert'|'humorous' }
  | { format: 'blocks'; text: string; blocks: Block[]; emotion?: 'neutral'|'happy'|'concerned'|'alert'|'humorous' }

/* ─────────────────────────────────────────────────────────── */
/* Block 스키마 LLM 시스템 프롬프트                            */
/* ─────────────────────────────────────────────────────────── */

export function buildBlockSchemaPrompt(lang: 'ko' | 'en' = 'ko'): string {
  if (lang === 'en') {
    return `
You can respond in TWO formats. Choose based on question type:

## FORMAT 1: Simple text (for casual chat, greetings, short answers)
{"format":"text","content":"Your response here","emotion":"happy"}

## FORMAT 2: Rich UI blocks (for analysis, comparisons, structured data, lists, tables, charts)
{"format":"blocks","text":"One-line summary","blocks":[<block>,<block>,...],"emotion":"happy"}

## Available block types:
- {"type":"text","content":"...","tone":"normal|highlight|muted"}
- {"type":"heading","level":1|2|3,"text":"Title","icon":"📊"}
- {"type":"list","items":["...","..."],"ordered":false}
- {"type":"keyvalue","pairs":[{"label":"Revenue","value":"$1.2M","trend":"up","emphasis":true}]}
- {"type":"table","headers":["Col1","Col2"],"rows":[["a","b"],["c","d"]],"caption":"..."}
- {"type":"chart","kind":"bar|line|pie","title":"...","unit":"%","data":[{"label":"Jan","value":120,"color":"#22c55e"}]}
- {"type":"callout","level":"info|tip|warning|critical|success","text":"💡 Insight here"}
- {"type":"action","label":"Save to Excel","command":"save this to excel","icon":"📊","variant":"primary|default|danger"}
- {"type":"divider"}

## CRITICAL RULES:
1. Output ONLY valid JSON — no markdown fences, no commentary outside JSON.
2. Use "blocks" format when answer has: numbers/KPIs, comparisons, multiple options, steps, tables, charts, or recommended actions.
3. Use "text" format when answer is a single sentence/paragraph of natural conversation.
4. ALWAYS include "text" field as 1-line summary (used for text-to-speech and history).
5. "action" blocks: "command" should be a natural Korean/English phrase user can re-send.
`.trim()
  }
  return `
다음 두 가지 형식 중 하나로 응답할 수 있어요. 질문 유형에 따라 선택하세요:

## 형식 1: 단순 텍스트 (일상 대화·인사·짧은 답변)
{"format":"text","content":"답변 내용","emotion":"happy"}

## 형식 2: 동적 UI 블록 (분석·비교·구조화된 데이터·표·차트·추천)
{"format":"blocks","text":"한 줄 요약","blocks":[<block>,<block>,...],"emotion":"happy"}

## 사용 가능한 블록 타입:
- {"type":"text","content":"...","tone":"normal|highlight|muted"}
- {"type":"heading","level":1|2|3,"text":"제목","icon":"📊"}
- {"type":"list","items":["...","..."],"ordered":false}
- {"type":"keyvalue","pairs":[{"label":"매출","value":"₩4.2억","trend":"up","emphasis":true}]}
   · trend: "up"(▲) / "down"(▼) / "flat"(—)
- {"type":"table","headers":["열1","열2"],"rows":[["a","b"],["c","d"]],"caption":"부가설명"}
- {"type":"chart","kind":"bar|line|pie","title":"제목","unit":"%","data":[{"label":"1월","value":120,"color":"#22c55e"}]}
- {"type":"callout","level":"info|tip|warning|critical|success","text":"💡 핵심 인사이트"}
- {"type":"action","label":"엑셀로 저장","command":"이 결과 엑셀로 저장해줘","icon":"📊","variant":"primary|default|danger"}
   · command: 사용자가 클릭하면 자동 재전송되는 자연어 명령
- {"type":"divider"}

## 핵심 규칙:
1. **반드시 유효한 JSON만 출력** — 마크다운 펜스(\`\`\`) 금지, JSON 외 코멘트 금지
2. **"blocks" 형식 사용**: 답변에 숫자/지표, 비교, 여러 옵션, 단계, 표, 차트, 추천 액션이 있을 때
3. **"text" 형식 사용**: 한 문장/문단의 자연스러운 대화일 때
4. **항상 "text" 필드 포함**: 한 줄 요약 (TTS 음성 + 히스토리용)
5. **"action" 블록의 command**: 사용자가 클릭하면 그대로 재전송되는 한국어 명령
6. **emotion**: "neutral" / "happy" / "concerned" / "alert" / "humorous" 중 하나
`.trim()
}

/* ─────────────────────────────────────────────────────────── */
/* 응답 파싱 — JSON 형식 검증 + 폴백                          */
/* ─────────────────────────────────────────────────────────── */

export function parseDynamicResponse(raw: string): DynamicLLMResult | null {
  if (!raw || !raw.trim()) return null

  // 1) 코드 블록 안 JSON 추출
  let candidate = raw.trim()
  const fenceMatch = candidate.match(/```(?:json)?\s*([\s\S]+?)\s*```/)
  if (fenceMatch) candidate = fenceMatch[1].trim()

  // 2) JSON 시작 위치 찾기 (모델이 앞에 잡담 붙이는 경우 대비)
  const jsonStart = candidate.indexOf('{')
  const jsonEnd = candidate.lastIndexOf('}')
  if (jsonStart === -1 || jsonEnd === -1 || jsonEnd <= jsonStart) return null
  candidate = candidate.slice(jsonStart, jsonEnd + 1)

  let parsed: unknown
  try { parsed = JSON.parse(candidate) } catch { return null }

  if (!parsed || typeof parsed !== 'object') return null
  const obj = parsed as Record<string, unknown>

  // 3) format 검증
  const format = obj.format
  if (format === 'text') {
    const content = typeof obj.content === 'string' ? obj.content : ''
    if (!content) return null
    return {
      format: 'text',
      text: content,
      emotion: validEmotion(obj.emotion),
    }
  }
  if (format === 'blocks') {
    const blocksRaw = obj.blocks
    if (!Array.isArray(blocksRaw)) return null
    const blocks = sanitizeBlocks(blocksRaw)
    if (blocks.length === 0) return null
    const text = typeof obj.text === 'string' && obj.text.trim()
      ? obj.text
      : extractFirstText(blocks) || '응답 준비 완료'
    return {
      format: 'blocks',
      text,
      blocks,
      emotion: validEmotion(obj.emotion),
    }
  }
  return null
}

function validEmotion(e: unknown): DynamicLLMResult['emotion'] {
  const allowed = ['neutral', 'happy', 'concerned', 'alert', 'humorous'] as const
  if (typeof e === 'string' && (allowed as readonly string[]).includes(e)) {
    return e as DynamicLLMResult['emotion']
  }
  return 'neutral'
}

/** Block 배열의 각 항목이 유효한 type 인지 검증 + 정제 */
function sanitizeBlocks(arr: unknown[]): Block[] {
  const valid = new Set([
    'text', 'heading', 'list', 'keyvalue', 'table', 'chart',
    'image', 'file', 'action', 'steps', 'callout', 'divider',
  ])
  const out: Block[] = []
  for (const item of arr) {
    if (!item || typeof item !== 'object') continue
    const b = item as Record<string, unknown>
    if (typeof b.type !== 'string' || !valid.has(b.type)) continue
    // 기본 형태만 검증 — 세부 필드는 렌더러가 안전하게 다룸
    out.push(b as unknown as Block)
  }
  // 너무 많으면 자르기 (UI 폭주 방지)
  return out.slice(0, 20)
}

function extractFirstText(blocks: Block[]): string {
  for (const b of blocks) {
    if (b.type === 'heading') return b.text
    if (b.type === 'text') return b.content
    if (b.type === 'callout') return b.text
  }
  return ''
}

/* ─────────────────────────────────────────────────────────── */
/* LLM 호출 + 동적 응답 파싱 (백엔드 프록시 경유)              */
/* ─────────────────────────────────────────────────────────── */

interface HistoryTurn {
  role: 'user' | 'model'
  parts: Array<{ text: string }>
}

export interface CallDynamicLLMOptions {
  userMessage: string
  history?: HistoryTurn[]
  lang?: 'ko' | 'en'
  /** 페르소나 시스템 프롬프트 (있으면 Block 스키마 앞에 prepend) */
  personaPrompt?: string
  /** 어시스턴트 이름 (호칭) */
  assistantName?: string
  /** Authorization 헤더 (선택) */
  authHeader?: Record<string, string>
  /** LLM 백엔드 endpoint (기본: 로컬 17891 프록시) */
  endpoint?: string
}

/**
 * LLM 에게 Block 스키마를 가르치고 동적 응답 시도.
 * 실패하거나 잘못된 JSON 이면 null 반환 → 호출자는 기존 텍스트 폴백 사용.
 */
export async function callDynamicLLM(opts: CallDynamicLLMOptions): Promise<DynamicLLMResult | null> {
  const lang = opts.lang ?? 'ko'
  const endpoint = opts.endpoint ?? 'http://127.0.0.1:17891/api/llm/chat'

  const systemParts: string[] = []
  if (opts.personaPrompt) systemParts.push(opts.personaPrompt)
  systemParts.push(buildBlockSchemaPrompt(lang))

  const messages = [
    { role: 'system', content: systemParts.join('\n\n') },
    ...(opts.history ?? []).slice(-8).map(t => ({
      role: t.role === 'user' ? 'user' : 'assistant',
      content: t.parts[0]?.text ?? '',
    })),
    { role: 'user', content: opts.userMessage },
  ]

  try {
    const res = await fetch(endpoint, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', ...(opts.authHeader ?? {}) },
      body: JSON.stringify({
        messages,
        max_tokens: 1500,
        json_mode: true, // 백엔드가 지원하면 JSON-only 강제
      }),
      signal: AbortSignal.timeout(20000),
    })
    if (!res.ok) {
      console.warn('[dynamicLLM] HTTP', res.status)
      return null
    }
    const json = await res.json() as { success: boolean; answer?: string }
    if (!json.success || !json.answer) return null

    const parsed = parseDynamicResponse(json.answer)
    if (!parsed) {
      console.warn('[dynamicLLM] Failed to parse JSON, falling back to text. Raw:', json.answer.slice(0, 200))
      // JSON 파싱 실패해도 원본 텍스트를 text 응답으로 폴백
      return { format: 'text', text: json.answer, emotion: 'neutral' }
    }
    return parsed
  } catch (e) {
    console.warn('[dynamicLLM] call failed:', e)
    return null
  }
}
