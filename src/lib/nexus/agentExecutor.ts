/**
 * Multi-Step Agent Executor — JARVIS 수준 자율 에이전트
 * 복잡한 요청을 여러 단계로 분해 → 순차 실행 → 자기검증 → 최종 답변 합성
 */

const BASE = 'http://127.0.0.1:17891'
const GROQ_BASE = 'https://api.groq.com/openai/v1'
const GROQ_MODEL = 'llama-3.1-8b-instant'

// ── 타입 ────────────────────────────────────────────────────────

type StepType =
  | 'web_search'      // Tavily 웹 검색
  | 'llm_task'        // LLM 글쓰기/요약/분석
  | 'screen_capture'  // 화면 캡처 + Vision AI 분석
  | 'file_move'       // 파일/폴더 이동
  | 'file_metadata'   // 파일 메타데이터 수집
  | 'excel_create'    // 엑셀 파일 생성
  | 'ui_control'      // 데스크톱 UI 자동화 (클릭·타이핑)
  | 'backend_call'    // 직접 백엔드 API 호출
  | 'email_inbox'     // 받은 메일 조회
  | 'email_summarize' // 메일 AI 요약
  | 'email_classify'  // 메일 분류·우선순위
  | 'email_draft'     // 답장 초안 작성
  | 'email_send'      // 메일 전송
  | 'calendar_today'  // 오늘 일정 조회
  | 'calendar_add'    // 일정 추가

interface AgentStep {
  id: number
  type: StepType
  description: string
  query: string
  params?: Record<string, string>
  result?: string
}

interface AgentPlan {
  finalGoal: string
  steps: AgentStep[]
}

// ── LLM 호출 ────────────────────────────────────────────────────

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
      max_tokens: 4000,
    }),
  })
  if (!res.ok) throw new Error(`groq ${res.status}`)
  const data = await res.json()
  return (data.choices?.[0]?.message?.content ?? '').trim()
}

// ── 도구 실행 함수들 ─────────────────────────────────────────────

async function tavilySearch(query: string): Promise<string> {
  const key = localStorage.getItem('nexus-tavily-key') ?? ''
  if (!key) return '(웹 검색 키 없음 — 내부 지식으로 진행)'
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
  } catch { return '(검색 오류)' }
}

async function captureScreen(): Promise<string> {
  try {
    const res = await fetch(`${BASE}/api/vision/screenshot`, { method: 'POST' })
    if (!res.ok) return '(화면 캡처 실패)'
    const data = await res.json()
    const b64 = data.image_base64 ?? data.data ?? ''
    if (!b64) return data.description ?? '(캡처 데이터 없음)'

    // Groq Vision으로 화면 분석
    const key = localStorage.getItem('nexus-groq-key') ?? ''
    if (!key) return '(Groq Vision 키 없음)'
    const visionRes = await fetch(`${GROQ_BASE}/chat/completions`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${key}` },
      body: JSON.stringify({
        model: 'llama-3.2-11b-vision-preview',
        messages: [{
          role: 'user',
          content: [
            { type: 'text', text: '이 화면에 있는 파일, 폴더, 앱, 텍스트를 상세히 설명하라. 파일명과 위치를 정확히 포함할 것.' },
            { type: 'image_url', image_url: { url: `data:image/png;base64,${b64}` } },
          ],
        }],
        max_tokens: 1500,
      }),
    })
    const vd = await visionRes.json()
    return vd.choices?.[0]?.message?.content ?? '(Vision 분석 실패)'
  } catch { return '(화면 캡처 오류)' }
}

async function moveFile(src: string, dst: string): Promise<string> {
  try {
    const res = await fetch(`${BASE}/api/files/move`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ src, dst }),
    })
    const data = await res.json()
    return data.message ?? (data.success ? '이동 완료' : '이동 실패')
  } catch { return '(파일 이동 오류)' }
}

async function getFilesMetadata(path: string, recursive = false): Promise<string> {
  try {
    const res = await fetch(`${BASE}/api/files/metadata`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ path, recursive }),
    })
    const data = await res.json()
    if (!data.files?.length) return '(파일 없음)'
    return JSON.stringify(data.files)
  } catch { return '(메타데이터 수집 오류)' }
}

async function createExcel(rows: string[][], title: string, filename: string): Promise<string> {
  try {
    const res = await fetch(`${BASE}/api/excel/save`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ data: rows, title, filename }),
    })
    const data = await res.json()
    return data.message ?? (data.success ? `엑셀 저장: ${data.path}` : '엑셀 생성 실패')
  } catch { return '(엑셀 생성 오류)' }
}

async function runDesktopAgent(goal: string): Promise<string> {
  try {
    const res = await fetch(`${BASE}/api/desktop/agent/run`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ goal, require_approval: false, max_steps: 15 }),
    })
    const data = await res.json()
    return data.message ?? '데스크톱 에이전트 실행 시작'
  } catch { return '(UI 자동화 오류)' }
}

async function backendCall(path: string, body: Record<string, unknown>): Promise<string> {
  try {
    const res = await fetch(`${BASE}${path}`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    })
    const data = await res.json()
    return data.message ?? JSON.stringify(data).slice(0, 500)
  } catch { return '(백엔드 호출 오류)' }
}

// ── 멀티스텝 감지 ─────────────────────────────────────────────

export function isMultiStepTask(text: string): boolean {
  const triggers = [
    // 검색 + 작성 조합
    '검색해서', '찾아서', '조사해서', '알아봐서', '분석해서',
    '검색하고', '찾고', '검색 후', '조사 후', '알아보고',
    // 이동 + 생성 조합
    '이동해주고', '이동하고', '옮겨서', '옮기고',
    // 생성/작성 요청
    '작성해줘', '써줘', '만들어줘', '만들어 줘', '생성해줘',
    '정리해줘', '정리해서',
    // 문서 유형
    '제품설명서', '사용설명서', '보고서', '기획서', '제안서',
    '엑셀로', '엑셀 파일', '스프레드시트',
    // 복합 동작
    '그리고', '다음에', '후에',
  ]

  // 동사 2개 이상 조합 (이동 + 생성, 검색 + 분석 등)
  const actionVerbs = ['이동', '옮기', '검색', '찾아', '분석', '정리', '작성', '생성', '만들', '써줘']
  let verbCount = 0
  for (const v of actionVerbs) {
    if (text.includes(v)) verbCount++
  }

  // 이메일 복합 액션 (확인+요약, 분류+답장 등 2개 이상)
  const hasEmail = /메일|이메일|email|inbox/i.test(text)
  const emailActions = ['확인', '요약', '분류', '답장', '정리', '분석', '보내', '전송']
  if (hasEmail && emailActions.filter(a => text.includes(a)).length >= 2) return true

  // 캘린더 복합 액션
  const hasCalendar = /일정|캘린더|calendar/i.test(text)
  if (hasCalendar && ['추가', '확인', '조회', '등록'].filter(a => text.includes(a)).length >= 2) return true

  const lower = text.toLowerCase()
  return triggers.some(k => lower.includes(k)) || verbCount >= 2
}

// ── 이메일·캘린더 실행 함수 ──────────────────────────────────────

async function callEmailInbox(): Promise<string> {
  try {
    const res = await fetch(`${BASE}/api/email/inbox?limit=10`)
    if (!res.ok) return '(메일 조회 실패)'
    const data = await res.json()
    if (!data.success) return data.message ?? '(메일 조회 실패)'
    const list = (data.emails ?? []).slice(0, 5)
      .map((e: { subject: string; sender: string; is_read: boolean }) =>
        `${e.is_read ? '📨' : '📩'} ${e.subject} — ${e.sender}`)
      .join('\n')
    return `받은 메일 ${data.total}개 (읽지 않음 ${data.unread}개)\n${list}`
  } catch { return '(메일 조회 오류)' }
}

async function callEmailSummarize(): Promise<string> {
  try {
    const res = await fetch(`${BASE}/api/email/summarize`, {
      method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({}),
    })
    if (!res.ok) return '(메일 요약 실패)'
    const data = await res.json()
    return data.summary || data.message || '(요약 없음)'
  } catch { return '(메일 요약 오류)' }
}

async function callEmailClassify(): Promise<string> {
  try {
    const res = await fetch(`${BASE}/api/email/classify`, {
      method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ limit: 20 }),
    })
    if (!res.ok) return '(메일 분류 실패)'
    const data = await res.json()
    return data.message ?? '(분류 완료)'
  } catch { return '(메일 분류 오류)' }
}

async function callEmailDraft(subject: string, sender: string, bodyCtx: string, tone = 'formal'): Promise<string> {
  try {
    const res = await fetch(`${BASE}/api/email/draft`, {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ subject, sender, body: bodyCtx, tone }),
    })
    if (!res.ok) return '(답장 초안 생성 실패)'
    const data = await res.json()
    return data.draft || data.message || '(초안 없음)'
  } catch { return '(답장 초안 오류)' }
}

async function callEmailSend(to: string, subject: string, body: string): Promise<string> {
  try {
    const res = await fetch(`${BASE}/api/email/send`, {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ to, subject, body }),
    })
    if (!res.ok) return '(메일 전송 실패)'
    const data = await res.json()
    return data.message ?? (data.success ? '메일 전송 완료' : '메일 전송 실패')
  } catch { return '(메일 전송 오류)' }
}

async function callCalendarToday(): Promise<string> {
  try {
    const res = await fetch(`${BASE}/api/calendar/today`)
    if (!res.ok) return '(일정 조회 실패)'
    const data = await res.json()
    return data.message ?? JSON.stringify(data).slice(0, 500)
  } catch { return '(일정 조회 오류)' }
}

async function callCalendarAdd(title: string, date: string): Promise<string> {
  try {
    const res = await fetch(`${BASE}/api/calendar/add`, {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ title, date }),
    })
    if (!res.ok) return '(일정 추가 실패)'
    const data = await res.json()
    return data.message ?? (data.success ? '일정 추가 완료' : '일정 추가 실패')
  } catch { return '(일정 추가 오류)' }
}

// ── 유틸 ─────────────────────────────────────────────────────

function isStepFailed(result: string): boolean {
  return ['오류', '실패', 'error', '키 없음', '찾을 수 없', 'failed', '불가'].some(p =>
    result.toLowerCase().includes(p.toLowerCase())
  )
}

const delay = (ms: number) => new Promise<void>(r => setTimeout(r, ms))

// ── 계획 수립 ─────────────────────────────────────────────────

async function planSteps(userMessage: string): Promise<AgentPlan> {
  const system = `당신은 JARVIS급 AI 에이전트 플래너입니다. 사용자 요청을 실행 단계 JSON으로 반환하세요.

사용 가능한 도구:
- web_search: Tavily 웹 실시간 검색 (query: 검색어)
- screen_capture: 화면 캡처 후 Vision AI 분석 (query: 분석 목적)
- file_move: 파일/폴더 이동 (query: "src경로 → dst경로")
- file_metadata: 폴더 내 파일 메타데이터 수집 (query: 폴더 경로)
- excel_create: 수집된 데이터로 엑셀 생성 (query: 파일명|시트제목)
- ui_control: 화면 UI 직접 제어 — 클릭, 타이핑 (query: 수행할 UI 작업 설명)
- llm_task: LLM으로 글쓰기/요약/분석 (query: 구체적 지시)
- backend_call: 백엔드 API 직접 호출 (query: "경로|JSON본문")
- email_inbox: 받은 메일 조회 (query: 조회 목적)
- email_summarize: 메일 AI 요약 (query: 요약 목적)
- email_classify: 메일 분류·우선순위 정리 (query: 분류 기준)
- email_draft: 답장 초안 작성 (query: "subject|sender|tone")
- email_send: 메일 전송 (query: "수신자|제목|본문")
- calendar_today: 오늘 일정 조회 (query: 조회 목적)
- calendar_add: 일정 추가 (query: "제목|날짜시간")

반환 형식 (JSON만, 코드블록 금지):
{
  "finalGoal": "목표 한 줄",
  "steps": [
    { "id": 1, "type": "screen_capture", "description": "바탕화면 확인", "query": "바탕화면 폴더 목록 파악" },
    { "id": 2, "type": "file_move", "description": "폴더 이동", "query": "바탕화면/폴더명 → 다운로드" },
    { "id": 3, "type": "file_metadata", "description": "파일 목록 수집", "query": "다운로드/폴더명" },
    { "id": 4, "type": "excel_create", "description": "엑셀 생성", "query": "파일목록|파일 수정날짜" }
  ]
}

규칙: 최대 15단계, JSON만, 마크다운 코드블록 금지`

  try {
    const raw = await callGroq(system, userMessage)
    const cleaned = raw.replace(/```json\n?|```\n?/g, '').trim()
    const parsed = JSON.parse(cleaned) as AgentPlan
    // steps 배열 검증
    if (!Array.isArray(parsed.steps)) throw new Error('invalid plan')
    return parsed
  } catch {
    // 폴백 플랜
    return {
      finalGoal: userMessage,
      steps: [
        { id: 1, type: 'web_search', description: '정보 검색', query: userMessage },
        { id: 2, type: 'llm_task', description: '결과 정리', query: `"${userMessage}" 요청을 완수하라` },
      ],
    }
  }
}

// ── 단일 스텝 실행 ────────────────────────────────────────────

async function doStep(
  step: AgentStep,
  context: string[],
  excelRows: string[][],
  onProgress: (msg: string) => void,
): Promise<string> {
  switch (step.type) {
    case 'web_search':
      onProgress(`🔍 "${step.query}" 검색 중...`)
      return tavilySearch(step.query)

    case 'screen_capture':
      onProgress('📸 화면 캡처 및 Vision AI 분석 중...')
      return captureScreen()

    case 'file_move': {
      onProgress(`📁 파일 이동: ${step.query}`)
      const [src, dst] = step.query.split(/→|->/).map(s => s.trim())
      return moveFile(src || '', dst || '')
    }

    case 'file_metadata':
      onProgress(`📋 "${step.query}" 파일 목록 수집 중...`)
      return getFilesMetadata(step.query, true)

    case 'excel_create': {
      onProgress('📊 엑셀 파일 생성 중...')
      const [filename, title] = step.query.split('|').map(s => s.trim())
      const rows = excelRows.length > 0 ? excelRows : await (async () => {
        const rawData = await callGroq(
          '아래 정보에서 표 형식 데이터를 추출하라. 첫 줄 헤더, 각 행은 "|"로 구분.',
          context.slice(-3).join('\n'),
        )
        return rawData.split('\n').filter(Boolean).map(row => row.split('|').map(c => c.trim()))
      })()
      return createExcel(rows, title || '데이터', filename || 'nexus_export')
    }

    case 'ui_control':
      onProgress(`🖱️ UI 자동화: ${step.query}`)
      return runDesktopAgent(step.query)

    case 'backend_call': {
      const [path, bodyStr] = step.query.split('|')
      let body: Record<string, unknown> = {}
      try { body = JSON.parse(bodyStr ?? '{}') } catch { body = {} }
      return backendCall(path.trim(), body)
    }

    case 'email_inbox':
      onProgress('📧 받은 메일 조회 중...')
      return callEmailInbox()

    case 'email_summarize':
      onProgress('📧 메일 AI 요약 중...')
      return callEmailSummarize()

    case 'email_classify':
      onProgress('📧 메일 분류 중...')
      return callEmailClassify()

    case 'email_draft': {
      onProgress('✉️ 답장 초안 작성 중...')
      const [subject, sender, tone] = step.query.split('|').map(s => s.trim())
      return callEmailDraft(subject || '', sender || '', context.slice(-2).join('\n'), tone || 'formal')
    }

    case 'email_send': {
      onProgress('📤 메일 전송 중...')
      const [to, subject, ...bodyParts] = step.query.split('|').map(s => s.trim())
      return callEmailSend(to || '', subject || '', bodyParts.join('\n'))
    }

    case 'calendar_today':
      onProgress('📅 오늘 일정 조회 중...')
      return callCalendarToday()

    case 'calendar_add': {
      onProgress('📅 일정 추가 중...')
      const [title, date] = step.query.split('|').map(s => s.trim())
      return callCalendarAdd(title || step.query, date || '')
    }

    case 'llm_task':
    default: {
      onProgress(`✍️ ${step.description} 중...`)
      const ctx = context.length > 0 ? `\n\n[이전 단계 결과]\n${context.slice(-3).join('\n---\n')}` : ''
      return callGroq(
        'Nexus AI입니다. 전문적이고 체계적으로 작업을 수행하세요.',
        step.query + ctx,
      )
    }
  }
}

// ── 에이전트 실행 ─────────────────────────────────────────────

export async function runAgent(
  userMessage: string,
  onProgress: (msg: string) => void,
): Promise<string> {
  onProgress('🧠 작업 계획 수립 중...')
  const plan = await planSteps(userMessage)

  const context: string[] = []
  let excelRows: string[][] = []

  for (const step of plan.steps) {
    onProgress(`⚙️ [${step.id}/${plan.steps.length}] ${step.description}...`)

    let result = await doStep(step, context, excelRows, onProgress)

    // 실패 감지 → 1.5초 대기 후 1회 재시도 (LLM 호출 없음)
    if (isStepFailed(result) && step.type !== 'llm_task') {
      onProgress(`🔄 [${step.id}/${plan.steps.length}] 재시도 중...`)
      await delay(1500)
      result = await doStep(step, context, excelRows, onProgress)
    }

    // file_metadata 결과로 엑셀 행 구성
    if (step.type === 'file_metadata' && !isStepFailed(result)) {
      try {
        const files = JSON.parse(result) as Array<{name:string;path:string;size_mb:number;modified:string;ext:string}>
        excelRows = [
          ['파일명', '경로', '크기(MB)', '수정날짜', '확장자'],
          ...files.map(f => [f.name, f.path, f.size_mb.toFixed(2), f.modified, f.ext]),
        ]
      } catch { /* 파싱 실패 시 텍스트로 유지 */ }
    }

    step.result = result
    context.push(`[${step.description}]\n${result.slice(0, 800)}`)
  }

  // 최종 합성 (LLM 호출 1회)
  onProgress('🔍 결과 검증 및 최종 답변 정리 중...')
  const synthesis = `사용자 요청: "${userMessage}"

실행 결과:
${context.join('\n\n---\n\n')}

위 결과를 바탕으로:
1. 요청이 완전히 완수됐는지 확인
2. 완수됐으면 결과를 마크다운 형식으로 체계적으로 보고
3. 미완수 부분이 있으면 이유와 대안 제시
파일 경로, 생성된 엑셀 위치 등 구체적 정보를 반드시 포함할 것.`

  return callGroq(
    '당신은 Nexus AI입니다. 실행 결과를 사용자에게 명확하게 보고하세요.',
    synthesis,
  )
}
