/**
 * Multi-Step Agent Executor
 * 복잡한 요청을 여러 단계로 분해해서 순차 실행 후 최종 답변 생성
 */

const GROQ_BASE = 'https://api.groq.com/openai/v1'
const GROQ_MODEL = 'llama-3.1-8b-instant'

interface AgentStep {
  id: number
  type: 'web_search' | 'llm_task'
  description: string
  query: string
  result?: string
}

interface AgentPlan {
  finalGoal: string
  steps: AgentStep[]
}

async function callGroq(system: string, user: string): Promise<string> {
  const key = localStorage.getItem('nexus-groq-key') ?? ''
  if (!key) throw new Error('no groq key')

  const res = await fetch(`${GROQ_BASE}/chat/completions`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${key}` },
    body: JSON.stringify({
      model: GROQ_MODEL,
      messages: [{ role: 'system', content: system }, { role: 'user', content: user }],
      temperature: 0.3,
      max_tokens: 3000,
    }),
  })
  if (!res.ok) throw new Error(`groq ${res.status}`)
  const data = await res.json()
  return (data.choices?.[0]?.message?.content ?? '').trim()
}

async function tavilySearch(query: string): Promise<string> {
  const key = localStorage.getItem('nexus-tavily-key') ?? ''
  if (!key) return '(웹 검색 키 없음 — 내부 지식으로 대체)'

  try {
    const res = await fetch('https://api.tavily.com/search', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ api_key: key, query, max_results: 5, search_depth: 'advanced' }),
    })
    if (!res.ok) return '(검색 실패)'
    const data = await res.json()
    return (data.results ?? [])
      .slice(0, 5)
      .map((r: { title: string; content: string }, i: number) => `[${i + 1}] ${r.title}\n${r.content}`)
      .join('\n\n') || '(검색 결과 없음)'
  } catch {
    return '(검색 오류)'
  }
}

// 멀티스텝 필요 여부 판단
export function isMultiStepTask(text: string): boolean {
  const triggers = [
    '검색해서', '찾아서', '조사해서', '알아봐서', '분석해서',
    '정리해서', '작성해줘', '써줘', '만들어줘', '만들어 줘',
    '검색하고', '찾고', '검색 후', '조사 후',
    '제품설명서', '사용설명서', '보고서', '기획서', '제안서',
    '문서로', '요약 후', '정리 후',
    'search and write', 'find and create', 'research and',
  ]
  const lower = text.toLowerCase()
  return triggers.some(k => lower.includes(k))
}

// LLM으로 실행 계획 수립
async function planSteps(userMessage: string): Promise<AgentPlan> {
  const system = `당신은 작업 플래너입니다. 사용자 요청을 분석해서 실행 단계를 JSON으로만 반환하세요.

도구:
- web_search: 웹 실시간 검색 (Tavily)
- llm_task: 글쓰기 / 요약 / 분석 / 정리 (이전 단계 결과 활용)

반환 형식 (JSON만, 코드블록 금지):
{
  "finalGoal": "목표 한 줄",
  "steps": [
    { "id": 1, "type": "web_search", "description": "단계 설명", "query": "검색어" },
    { "id": 2, "type": "llm_task",   "description": "단계 설명", "query": "수행 지시" }
  ]
}

규칙: 최대 4단계, JSON만, 마크다운 코드블록 금지`

  try {
    const raw = await callGroq(system, userMessage)
    const cleaned = raw.replace(/```json\n?|```\n?/g, '').trim()
    return JSON.parse(cleaned) as AgentPlan
  } catch {
    return {
      finalGoal: userMessage,
      steps: [
        { id: 1, type: 'web_search', description: '정보 검색', query: userMessage },
        { id: 2, type: 'llm_task', description: '문서 작성', query: `다음 검색 결과를 바탕으로 "${userMessage}" 요청을 완수해라` },
      ],
    }
  }
}

// 에이전트 실행
export async function runAgent(
  userMessage: string,
  onProgress: (msg: string) => void,
): Promise<string> {
  // 1. 계획 수립
  onProgress('🧠 작업 계획 수립 중...')
  const plan = await planSteps(userMessage)

  const context: string[] = []

  // 2. 단계별 실행
  for (const step of plan.steps) {
    if (step.type === 'web_search') {
      onProgress(`🔍 "${step.query}" 검색 중...`)
      step.result = await tavilySearch(step.query)
    } else {
      onProgress(`✍️ ${step.description} 중...`)
      const ctx = context.length > 0 ? `\n\n[수집 정보]\n${context.join('\n---\n')}` : ''
      step.result = await callGroq(
        'Nexus AI입니다. 전문적이고 체계적으로 답변하세요.',
        step.query + ctx,
      )
    }
    context.push(`[${step.description}]\n${step.result}`)
  }

  // 3. 최종 답변 합성
  onProgress('📝 최종 답변 정리 중...')
  const synthesis = `사용자 요청: "${userMessage}"

수집 및 분석 결과:
${context.join('\n\n---\n\n')}

위 정보를 바탕으로 사용자 요청을 완전히 충족하는 답변을 마크다운 형식으로 작성하라. 제목, 소제목, 목록을 적절히 사용해 체계적으로 작성할 것.`

  return callGroq(
    '당신은 Nexus AI입니다. 수집된 정보를 바탕으로 사용자 요청에 완벽하게 답변하세요.',
    synthesis,
  )
}
