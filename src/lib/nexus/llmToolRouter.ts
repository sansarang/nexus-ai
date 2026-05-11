/**
 * LLM Tool Router — 사용자 메시지를 LLM에게 보내 어떤 도구를 쓸지 결정
 * 키워드 매칭 대신 LLM이 직접 Tool Calling으로 intent 선택
 */

import { PPLX_API_KEY, OPENAI_API_KEY } from '../../config/services'

export interface ToolCall {
  tool: string
  args: Record<string, string>
}

// NEXUS가 사용할 수 있는 도구 목록 (LLM에게 알려줌)
const NEXUS_TOOLS = [
  {
    name: 'youtube_search',
    description: '유튜브에서 영상 검색. 요리법, 튜토리얼, 음악, 강의 등 영상 콘텐츠 찾기',
    params: { query: '검색할 키워드' },
  },
  {
    name: 'web_search',
    description: '웹에서 정보 검색. 최신 뉴스, 날씨, 일반 지식, 방법론 등',
    params: { query: '검색할 키워드', site: '(선택) 특정 사이트 (예: naver.com)' },
  },
  {
    name: 'price_compare',
    description: '쿠팡, 네이버쇼핑 등에서 상품 가격 비교 및 최저가 검색',
    params: { query: '상품명' },
  },
  {
    name: 'news_search',
    description: '최신 뉴스 검색',
    params: { query: '뉴스 주제' },
  },
  {
    name: 'pc_status',
    description: 'CPU, RAM, 디스크, 온도 등 PC 상태 조회',
    params: {},
  },
  {
    name: 'launch_app',
    description: '앱 또는 프로그램 실행',
    params: { app: '실행할 앱 이름' },
  },
  {
    name: 'weather',
    description: '날씨 조회',
    params: { city: '도시명 (기본: 현재 위치)' },
  },
  {
    name: 'email_classify',
    description: '받은 이메일을 AI로 분류·우선순위 정리',
    params: {},
  },
  {
    name: 'email_draft',
    description: '수신된 이메일에 대한 답장 초안 자동 작성',
    params: { tone: '(선택) formal | casual' },
  },
  {
    name: 'calendar_find_slot',
    description: '캘린더에서 미팅 가능한 빈 시간 탐색',
    params: { duration_min: '(선택) 분 단위 길이', prefer_time: '(선택) morning | afternoon | evening' },
  },
  {
    name: 'workflow_list',
    description: '저장된 자동화 워크플로 목록 조회',
    params: {},
  },
  {
    name: 'briefing_now',
    description: '모닝 브리핑 실행 (날씨·일정·이메일 요약)',
    params: {},
  },
  {
    name: 'search_pdf',
    description: '웹 검색 후 PDF 보고서 자동 생성',
    params: { query: '검색 주제' },
  },
  {
    name: 'multi_agent',
    description: '복잡한 목표를 여러 AI 에이전트가 병렬로 처리',
    params: { goal: '처리할 목표' },
  },
  {
    name: 'video_download',
    description: 'YouTube, TikTok 등 영상 URL로 동영상 다운로드',
    params: { url: '영상 URL', quality: '(선택) 720p, 480p, best' },
  },
  {
    name: 'general_answer',
    description: '위 도구 없이 AI가 직접 답변. 지식 질문, 조언, 대화 등',
    params: { answer_type: 'knowledge | advice | conversation' },
  },
]

const TOOL_ROUTER_SYSTEM_PROMPT = `You are a tool selector for NEXUS AI assistant.
Given a user message in Korean or English, select the BEST tool and extract arguments.

Available tools:
${NEXUS_TOOLS.map(t => `- ${t.name}: ${t.description}. params: ${JSON.stringify(t.params)}`).join('\n')}

Rules:
- Always pick exactly ONE tool
- Extract the most relevant query/args from the user message
- For youtube: extract the actual search topic (e.g. "유튜브에서 김치찌개 끓이는 법" → query: "김치찌개 끓이는 법")
- For web_search: extract the core question
- For general_answer: use when no other tool fits

Respond ONLY with valid JSON, no explanation:
{"tool": "tool_name", "args": {"key": "value"}}`

/**
 * LLM에게 어떤 Tool을 써야 할지 물어보고 결과 반환
 */
export async function routeWithLLM(userMessage: string): Promise<ToolCall> {
  // 1순위: Perplexity (sonar-fast, 빠름)
  const pplxKey = PPLX_API_KEY || localStorage.getItem('nexus-pplx-key') || ''
  if (pplxKey) {
    try {
      const res = await fetch('https://api.perplexity.ai/chat/completions', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${pplxKey}` },
        body: JSON.stringify({
          model: 'sonar',
          messages: [
            { role: 'system', content: TOOL_ROUTER_SYSTEM_PROMPT },
            { role: 'user', content: userMessage },
          ],
          max_tokens: 150,
          temperature: 0.1,
        }),
        signal: AbortSignal.timeout(6000),
      })
      if (res.ok) {
        const data = await res.json() as { choices?: Array<{ message?: { content?: string } }> }
        const content = data.choices?.[0]?.message?.content?.trim() || ''
        const parsed = parseToolCall(content)
        if (parsed) return parsed
      }
    } catch { /* 폴백 */ }
  }

  // 2순위: OpenAI
  const openaiKey = OPENAI_API_KEY || localStorage.getItem('nexus-openai-key') || ''
  if (openaiKey) {
    try {
      const res = await fetch('https://api.openai.com/v1/chat/completions', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${openaiKey}` },
        body: JSON.stringify({
          model: 'gpt-4o-mini',
          messages: [
            { role: 'system', content: TOOL_ROUTER_SYSTEM_PROMPT },
            { role: 'user', content: userMessage },
          ],
          max_tokens: 150,
          temperature: 0.1,
          response_format: { type: 'json_object' },
        }),
        signal: AbortSignal.timeout(8000),
      })
      if (res.ok) {
        const data = await res.json() as { choices?: Array<{ message?: { content?: string } }> }
        const content = data.choices?.[0]?.message?.content?.trim() || ''
        const parsed = parseToolCall(content)
        if (parsed) return parsed
      }
    } catch { /* 폴백 */ }
  }

  // 폴백: 간단한 키워드 기반 판단
  return fallbackRoute(userMessage)
}

function parseToolCall(content: string): ToolCall | null {
  try {
    const jsonMatch = content.match(/\{[\s\S]*\}/)
    if (!jsonMatch) return null
    const parsed = JSON.parse(jsonMatch[0]) as { tool?: string; args?: Record<string, string> }
    if (parsed.tool && NEXUS_TOOLS.some(t => t.name === parsed.tool)) {
      return { tool: parsed.tool, args: parsed.args || {} }
    }
  } catch { /* 파싱 실패 */ }
  return null
}

function fallbackRoute(text: string): ToolCall {
  if (/다운로드|download.*http|http.*다운/i.test(text) && /http/i.test(text)) {
    const urlMatch = text.match(/https?:\/\/[^\s]+/)
    return { tool: 'video_download', args: { url: urlMatch?.[0] ?? '' } }
  }
  if (/유튜브|youtube/i.test(text)) {
    const query = text.replace(/유튜브에서|유튜브|youtube|찾아줘|검색해줘|보여줘/gi, '').trim()
    return { tool: 'youtube_search', args: { query } }
  }
  if (/쿠팡|네이버쇼핑|최저가|가격|얼마/i.test(text)) {
    return { tool: 'price_compare', args: { query: text } }
  }
  if (/뉴스|오늘.*뉴스|최신.*뉴스/i.test(text)) {
    return { tool: 'news_search', args: { query: text } }
  }
  if (/날씨/i.test(text)) {
    return { tool: 'weather', args: { city: text.replace(/날씨/g, '').trim() || '서울' } }
  }
  return { tool: 'general_answer', args: { answer_type: 'knowledge' } }
}
