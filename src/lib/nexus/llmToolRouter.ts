/**
 * LLM Tool Router — LLM이 직접 인텐트와 인자를 선택
 * 대화 컨텍스트(최근 3~5턴) 주입으로 맥락 기반 라우팅 지원
 */

import { PPLX_API_KEY, OPENAI_API_KEY } from '../../config/services'

export interface ToolCall {
  tool: string
  args: Record<string, string>
}

export interface HistoryTurn {
  role: 'user' | 'assistant'
  content: string
}

// ── 전체 Nexus 도구 목록 ──────────────────────────────────────────
const NEXUS_TOOLS = [
  // PC 상태 & 진단
  { name: 'pc_status',       description: 'CPU, RAM, 디스크, 온도 등 PC 상태 실시간 조회. "PC 어때", "컴퓨터 상태", "CPU 몇%야" 등', params: {} },
  { name: 'security_scan',   description: '해킹·악성코드 탐지, 보안 스캔. "보안 점검", "해킹당한 거 아냐", "악성코드 있어?" 등', params: {} },
  { name: 'full_scan',       description: '전체 PC 종합 진단. "전체 진단", "PC 완전 검사", "다 점검해줘" 등', params: {} },
  { name: 'clean',           description: '임시파일 정리, PC 청소, 디스크 공간 확보. "정리해줘", "청소해줘", "용량 줄여줘" 등', params: {} },
  { name: 'daily_report',    description: '오늘 PC 사용 리포트, 일일 요약. "오늘 리포트", "오늘 PC 어떻게 썼어" 등', params: {} },
  { name: 'repair',          description: 'PC 문제 수리, 오류 수정. "수리해줘", "고쳐줘", "오류 수정" 등', params: {} },
  { name: 'gpu_stats',       description: 'GPU 상태, 그래픽카드 정보. "GPU 어때", "그래픽카드", "VRAM" 등', params: {} },
  { name: 'process_top',     description: 'CPU·메모리 많이 쓰는 프로세스 상위 목록. "뭐가 CPU 잡아먹어", "프로세스 목록", "무거운 프로그램" 등', params: {} },
  { name: 'pc_report',       description: 'PC 건강 리포트 파일 생성. "리포트 만들어줘", "PC 보고서 생성" 등', params: {} },

  // 보안 상세
  { name: 'remote_access',   description: '원격 접속 흔적 탐지. "원격 접속 흔적", "누가 내 PC 접속했어" 등', params: {} },
  { name: 'process_security',description: '수상한 프로세스·열린 포트 점검. "수상한 프로세스", "이상한 포트" 등', params: {} },
  { name: 'startup_items',   description: '시작 프로그램 목록·자동 실행 항목. "시작 프로그램", "자동 실행" 등', params: {} },
  { name: 'defender_status', description: 'Windows Defender 백신 상태. "백신 상태", "Defender", "윈도우 디펜더" 등', params: {} },
  { name: 'account_check',   description: '이상한 윈도우 계정 확인. "계정 점검", "모르는 계정" 등', params: {} },
  { name: 'virus_check',     description: '파일 바이러스 검사 (VirusTotal). "바이러스 검사", "이 파일 위험해?" 등', params: { file: '(선택) 검사할 파일 경로' } },
  { name: 'process_kill',    description: '프로세스 강제 종료. "강제 종료", "이 프로세스 죽여", "킬해줘" 등', params: { name: '종료할 프로세스 이름' } },
  { name: 'app_permissions', description: '앱 권한 감사. "앱 권한 확인", "무슨 권한 써?" 등', params: {} },
  { name: 'windows_updates', description: '윈도우 업데이트 확인·설치. "윈도우 업데이트", "업데이트 확인" 등', params: {} },

  // 시스템 제어
  { name: 'volume_control',  description: '볼륨 조절·음소거. "소리 키워", "볼륨 낮춰", "음소거" 등', params: { action: 'up|down|mute|unmute|set', value: '(선택) 0-100' } },
  { name: 'brightness',      description: '화면 밝기 조절. "밝기 낮춰", "화면 어둡게", "밝게 해줘" 등', params: { action: 'up|down|set', value: '(선택) 0-100' } },
  { name: 'wifi_toggle',     description: 'Wi-Fi 켜기·끄기. "와이파이 꺼줘", "와이파이 켜줘" 등', params: { action: 'on|off' } },
  { name: 'power_action',    description: '잠금·절전·재시작·종료. "잠금", "절전", "재시작", "종료해줘" 등', params: { action: 'lock|sleep|restart|shutdown' } },
  { name: 'launch_app',      description: '앱 실행. "크롬 켜줘", "카톡 열어줘", "메모장 실행해줘" 등', params: { app: '앱 이름' } },

  // 고급 시스템
  { name: 'driver_check',    description: '드라이버 점검·업데이트. "드라이버 확인", "드라이버 문제" 등', params: {} },
  { name: 'network_analysis',description: '네트워크 분석, IP·DNS 확인. "인터넷 왜 느려", "네트워크 점검", "IP 확인" 등', params: {} },
  { name: 'restore_create',  description: '시스템 복구 포인트 생성. "복구 포인트 만들어", "백업 포인트" 등', params: {} },
  { name: 'browser_clean',   description: '브라우저 캐시·기록 정리. "브라우저 정리", "크롬 캐시 삭제" 등', params: {} },
  { name: 'programs_list',   description: '설치된 프로그램 목록. "뭐가 설치돼있어", "프로그램 목록" 등', params: {} },
  { name: 'boot_analysis',   description: '부팅 속도 분석. "부팅이 왜 느려", "부팅 속도" 등', params: {} },
  { name: 'disk_check',      description: '디스크 오류 검사. "디스크 검사", "하드 점검" 등', params: {} },
  { name: 'registry_clean',  description: '레지스트리 정리. "레지스트리 정리", "레지스트리 청소" 등', params: {} },

  // 파일 관리
  { name: 'file_search',     description: 'PC 내 파일 검색. "파일 어디있어", "엑셀 파일 찾아줘", "보고서 PDF" 등', params: { query: '검색 키워드', folder: '(선택) 검색 폴더' } },
  { name: 'file_organize',   description: '바탕화면·다운로드 폴더 자동 정리', params: { target: '(선택) desktop|downloads|all' } },
  { name: 'file_duplicates', description: '중복 파일 찾기·삭제. "중복 파일", "같은 파일 두 개" 등', params: {} },
  { name: 'deep_search',     description: 'PC 파일 내용 심층 검색. "계약서 안에서 해지 조항 찾아줘" 등', params: { query: '검색할 내용 키워드' } },
  { name: 'doc_compare',     description: '두 문서 비교. "이 두 파일 비교해줘", "뭐가 달라?" 등', params: {} },
  { name: 'smart_organize',  description: '다운로드·바탕화면 스마트 자동 정리', params: {} },
  { name: 'open_folder',     description: '특정 폴더 열기. "다운로드 폴더 열어줘", "바탕화면 열어" 등', params: { folder: 'desktop|downloads|documents|pictures|music' } },

  // 캘린더
  { name: 'calendar_today',  description: '오늘 일정 조회. "오늘 일정", "오늘 뭐 있어" 등', params: {} },
  { name: 'calendar_week',   description: '이번 주 일정 조회. "이번 주 일정", "주간 일정" 등', params: {} },
  { name: 'calendar_add',    description: '일정 추가. "일정 추가해줘", "~에 미팅 등록해줘" 등', params: { title: '일정 제목', date: '날짜/시간' } },
  { name: 'calendar_find_slot', description: '미팅 가능한 빈 시간 탐색', params: { duration_min: '(선택) 분 단위', prefer_time: '(선택) morning|afternoon|evening' } },

  // 이메일
  { name: 'email_inbox',     description: '받은 메일 확인. "메일 확인", "받은 메일 있어?" 등', params: {} },
  { name: 'email_send',      description: '메일 전송. "메일 보내줘", "이메일 발송" 등', params: { to: '수신자', subject: '제목', body: '내용' } },
  { name: 'email_summarize', description: '받은 메일 AI 요약. "메일 요약해줘", "받은 메일 정리" 등', params: {} },
  { name: 'email_classify',  description: '이메일 AI 분류·우선순위 정리', params: {} },
  { name: 'email_draft',     description: '이메일 답장 초안 작성', params: { tone: '(선택) formal|casual' } },

  // 검색·미디어
  { name: 'web_search',      description: '웹 검색. 최신 정보, 뉴스, 일반 지식. "검색해줘", "알려줘" 등', params: { query: '검색할 키워드', site: '(선택) 특정 사이트' } },
  { name: 'news_search',     description: '최신 뉴스 검색. "뉴스 알려줘", "오늘 뉴스" 등', params: { query: '뉴스 주제' } },
  { name: 'youtube_search',  description: '유튜브 영상 검색. "유튜브에서 찾아줘", "영상 보여줘" 등', params: { query: '검색할 영상 키워드' } },
  { name: 'price_compare',   description: '쿠팡·네이버쇼핑 가격 비교·최저가. "얼마야", "최저가", "쿠팡에서 찾아줘" 등', params: { query: '상품명' } },
  { name: 'video_download',  description: 'YouTube·TikTok 등 URL로 영상 다운로드. 반드시 URL 포함 시만', params: { url: '영상 URL', quality: '(선택) 720p|480p|best' } },
  { name: 'search_pdf',      description: '웹 검색 후 PDF 보고서 자동 생성', params: { query: '검색 주제' } },

  // 날씨·교통
  { name: 'weather',         description: '날씨 조회. "날씨 어때", "비 와?", "기온" 등', params: { city: '도시명 (기본: 서울)' } },
  { name: 'travel_time',     description: '이동 시간·경로 계산. "거기 얼마나 걸려", "이동 시간" 등', params: { origin: '출발지', destination: '목적지' } },

  // 생산성
  { name: 'focus_mode',      description: '집중 모드 설정·해제. "집중 모드", "방해 금지", "집중할게" 등', params: { duration: '(선택) 분 단위' } },
  { name: 'notes',           description: '메모 추가·조회. "메모해줘", "기록해줘", "메모 뭐 있어" 등', params: { content: '(선택) 메모 내용' } },
  { name: 'translate',       description: '번역. "번역해줘", "영어로", "한국어로 바꿔줘" 등', params: { target_lang: '번역할 언어' } },
  { name: 'doc_summary',     description: '문서 요약. "이 파일 요약해줘", "문서 핵심만 알려줘" 등', params: { file: '(선택) 파일 경로' } },
  { name: 'briefing_now',    description: '모닝 브리핑 (날씨·일정·이메일). "브리핑해줘", "오늘 요약" 등', params: {} },

  // 회의·녹음
  { name: 'meeting_start',   description: '회의 녹음 시작. "회의 녹음 시작", "녹음 시작해줘" 등', params: {} },
  { name: 'meeting_stop',    description: '회의 녹음 종료. "녹음 끝", "녹음 종료" 등', params: {} },
  { name: 'meeting_summary', description: '회의 내용 전사·요약. "회의 요약", "회의 내용 알려줘" 등', params: {} },

  // Vision·화면
  { name: 'vision_screen',   description: '현재 화면 캡처 후 AI 분석. "화면에 뭐라고 써있어", "지금 화면 분석해줘" 등', params: { question: '(선택) 분석 질문' } },
  { name: 'vision_ocr',      description: '클립보드 이미지 텍스트 추출(OCR). "클립보드 이미지 읽어줘", "복사한 이미지 텍스트" 등', params: {} },
  { name: 'caption_start',   description: '실시간 자막 시작. "자막 켜줘", "실시간 자막" 등', params: {} },
  { name: 'caption_stop',    description: '실시간 자막 종료. "자막 꺼줘", "자막 종료" 등', params: {} },
  { name: 'recall_search',   description: '과거 화면 기억 검색. "아까 봤던 거", "예전에 뭐 봤더라" 등', params: { query: '검색 키워드' } },
  { name: 'recall_capture',  description: '현재 화면을 기억에 저장. "이 화면 기억해줘", "화면 저장" 등', params: {} },

  // AI & 자동화
  { name: 'workflow_run',    description: '워크플로 실행. "워크플로 실행", "자동화 실행" 등', params: { name: '(선택) 워크플로 이름' } },
  { name: 'workflow_list',   description: '저장된 워크플로 목록 조회', params: {} },
  { name: 'multi_agent',     description: '복잡한 목표를 여러 AI 에이전트가 병렬 처리', params: { goal: '처리할 목표' } },
  { name: 'journal_today',   description: '오늘 업무 일지 생성. "오늘 일지 써줘", "업무 기록" 등', params: {} },
  { name: 'persona_switch',  description: 'AI 모드 변경. "개발자 모드로", "의료 전문가 모드" 등', params: { persona: '변경할 모드' } },
  { name: 'brain_search',    description: 'Second Brain 기억 검색. "예전에 저장한 거", "두뇌 검색" 등', params: { query: '검색 키워드' } },

  // 폴백
  { name: 'general_answer',  description: '위 도구 없이 AI가 직접 답변. 지식, 조언, 창작, 일반 대화', params: { answer_type: 'knowledge|advice|conversation|creative' } },
]

// ── 시스템 프롬프트 (컨텍스트 포함) ─────────────────────────────
function buildSystemPrompt(recentHistory?: HistoryTurn[]): string {
  const toolList = NEXUS_TOOLS.map(t =>
    `- ${t.name}: ${t.description}${Object.keys(t.params).length ? ' | params: ' + JSON.stringify(t.params) : ''}`
  ).join('\n')

  const contextSection = recentHistory && recentHistory.length > 0
    ? `\n최근 대화 컨텍스트 (맥락 참고용):\n${
        recentHistory.slice(-4).map(h => `${h.role === 'user' ? '사용자' : 'Nexus'}: ${h.content.slice(0, 120)}`).join('\n')
      }\n`
    : ''

  return `You are a tool selector for NEXUS AI, a Korean Windows PC assistant.
Given the user message (and recent conversation context), pick the SINGLE BEST tool and extract arguments.
${contextSection}
Available tools:
${toolList}

Rules:
- Pick exactly ONE tool, always
- For contextual requests ("그거 다시 해줘", "방금 한 거 취소해줘", "다시 해줘") → infer from recent context what action to repeat or undo
- For ambiguous Korean expressions: "PC 이상해" → pc_status, "뭔가 느려" → pc_status or clean, "돈 아껴야겠는데" → price_compare
- For video_download: ONLY when a URL (http/https) is in the message
- For general_answer: only when truly no tool fits (pure knowledge/conversation)
- Extract the most specific args from the message

Respond ONLY with valid JSON, no explanation, no markdown:
{"tool": "tool_name", "args": {"key": "value"}}`
}

// ── LLM 라우팅 (컨텍스트 지원) ──────────────────────────────────
export async function routeWithLLM(
  userMessage: string,
  recentHistory?: HistoryTurn[],
): Promise<ToolCall> {
  const systemPrompt = buildSystemPrompt(recentHistory)
  const messages = [
    { role: 'system', content: systemPrompt },
    { role: 'user', content: userMessage },
  ]

  // 1순위: Perplexity sonar (빠름, 저비용)
  const pplxKey = PPLX_API_KEY || localStorage.getItem('nexus-pplx-key') || ''
  if (pplxKey) {
    try {
      const res = await fetch('https://api.perplexity.ai/chat/completions', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${pplxKey}` },
        body: JSON.stringify({ model: 'sonar', messages, max_tokens: 200, temperature: 0.1 }),
        signal: AbortSignal.timeout(6000),
      })
      if (res.ok) {
        const data = await res.json() as { choices?: Array<{ message?: { content?: string } }> }
        const parsed = parseToolCall(data.choices?.[0]?.message?.content?.trim() ?? '')
        if (parsed) return parsed
      }
    } catch { /* 폴백 */ }
  }

  // 2순위: OpenAI gpt-4o-mini
  const openaiKey = OPENAI_API_KEY || localStorage.getItem('nexus-openai-key') || ''
  if (openaiKey) {
    try {
      const res = await fetch('https://api.openai.com/v1/chat/completions', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${openaiKey}` },
        body: JSON.stringify({ model: 'gpt-4o-mini', messages, max_tokens: 200, temperature: 0.1, response_format: { type: 'json_object' } }),
        signal: AbortSignal.timeout(8000),
      })
      if (res.ok) {
        const data = await res.json() as { choices?: Array<{ message?: { content?: string } }> }
        const parsed = parseToolCall(data.choices?.[0]?.message?.content?.trim() ?? '')
        if (parsed) return parsed
      }
    } catch { /* 폴백 */ }
  }

  // 3순위: Groq llama (무료, 매우 빠름)
  const groqKey = localStorage.getItem('nexus-groq-key') || ''
  if (groqKey) {
    try {
      const res = await fetch('https://api.groq.com/openai/v1/chat/completions', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${groqKey}` },
        body: JSON.stringify({ model: 'llama-3.1-8b-instant', messages, max_tokens: 200, temperature: 0.1, response_format: { type: 'json_object' } }),
        signal: AbortSignal.timeout(5000),
      })
      if (res.ok) {
        const data = await res.json() as { choices?: Array<{ message?: { content?: string } }> }
        const parsed = parseToolCall(data.choices?.[0]?.message?.content?.trim() ?? '')
        if (parsed) return parsed
      }
    } catch { /* 폴백 */ }
  }

  return fallbackRoute(userMessage, recentHistory)
}

// ── 멀티툴 라우팅 ────────────────────────────────────────────────

function buildSystemPromptMulti(recentHistory?: HistoryTurn[]): string {
  const toolList = NEXUS_TOOLS.map(t =>
    `- ${t.name}: ${t.description.split('. ')[0]}`
  ).join('\n')
  const ctx = recentHistory?.length
    ? `\n최근 대화:\n${recentHistory.slice(-3).map(h => `${h.role}: ${h.content.slice(0, 100)}`).join('\n')}\n`
    : ''
  return `You are a multi-tool selector for NEXUS AI Korean Windows PC assistant.
Select ALL tools needed to FULLY complete the user's request, in execution order.
${ctx}
Available tools:
${toolList}

Rules:
- Compound requests ("확인하고 요약해줘", "분류하고 답장 써줘") → return multiple tools in order
- Simple requests → return exactly 1 tool
- Use "general_answer" only when no other tool fits

Respond ONLY with valid JSON:
{"tools": [{"tool": "tool_name", "args": {}}, ...]}`
}

function parseToolCallArray(content: string): ToolCall[] {
  try {
    const jsonMatch = content.match(/\{[\s\S]*\}/)
    if (!jsonMatch) return []
    const parsed = JSON.parse(jsonMatch[0]) as { tools?: Array<{ tool?: string; args?: Record<string, string> }> }
    if (!Array.isArray(parsed.tools)) return []
    return parsed.tools
      .filter(t => t.tool && NEXUS_TOOLS.some(n => n.name === t.tool))
      .map(t => ({ tool: t.tool!, args: t.args ?? {} }))
  } catch { return [] }
}

export async function routeWithLLMMulti(
  userMessage: string,
  recentHistory?: HistoryTurn[],
): Promise<ToolCall[]> {
  const systemPrompt = buildSystemPromptMulti(recentHistory)
  const messages = [
    { role: 'system', content: systemPrompt },
    { role: 'user', content: userMessage },
  ]

  // 1순위: Claude Haiku (한국어 복합 의도 분류 정확도 최고)
  const claudeKey = localStorage.getItem('nexus-claude-key') || ''
  if (claudeKey) {
    try {
      const res = await fetch('https://api.anthropic.com/v1/messages', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', 'x-api-key': claudeKey, 'anthropic-version': '2023-06-01' },
        body: JSON.stringify({ model: 'claude-haiku-4-5-20251001', max_tokens: 400, system: systemPrompt, messages: [{ role: 'user', content: userMessage }] }),
        signal: AbortSignal.timeout(10000),
      })
      if (res.ok) {
        const data = await res.json() as { content?: Array<{ text?: string }> }
        const parsed = parseToolCallArray(data.content?.[0]?.text?.trim() ?? '')
        if (parsed.length > 0) return parsed
      }
    } catch { /* fallback */ }
  }

  const openaiKey = OPENAI_API_KEY || localStorage.getItem('nexus-openai-key') || ''
  if (openaiKey) {
    try {
      const res = await fetch('https://api.openai.com/v1/chat/completions', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${openaiKey}` },
        body: JSON.stringify({ model: 'gpt-4o-mini', messages, max_tokens: 400, temperature: 0.1, response_format: { type: 'json_object' } }),
        signal: AbortSignal.timeout(8000),
      })
      if (res.ok) {
        const data = await res.json() as { choices?: Array<{ message?: { content?: string } }> }
        const parsed = parseToolCallArray(data.choices?.[0]?.message?.content?.trim() ?? '')
        if (parsed.length > 0) return parsed
      }
    } catch { /* fallback */ }
  }

  const groqKey = localStorage.getItem('nexus-groq-key') || ''
  if (groqKey) {
    try {
      const res = await fetch('https://api.groq.com/openai/v1/chat/completions', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${groqKey}` },
        body: JSON.stringify({ model: 'llama-3.1-8b-instant', messages, max_tokens: 400, temperature: 0.1, response_format: { type: 'json_object' } }),
        signal: AbortSignal.timeout(5000),
      })
      if (res.ok) {
        const data = await res.json() as { choices?: Array<{ message?: { content?: string } }> }
        const parsed = parseToolCallArray(data.choices?.[0]?.message?.content?.trim() ?? '')
        if (parsed.length > 0) return parsed
      }
    } catch { /* fallback */ }
  }

  const single = await routeWithLLM(userMessage, recentHistory)
  return [single]
}

// ── JSON 파싱 ─────────────────────────────────────────────────────
function parseToolCall(content: string): ToolCall | null {
  try {
    const jsonMatch = content.match(/\{[\s\S]*\}/)
    if (!jsonMatch) return null
    const parsed = JSON.parse(jsonMatch[0]) as { tool?: string; args?: Record<string, string> }
    if (parsed.tool && NEXUS_TOOLS.some(t => t.name === parsed.tool)) {
      return { tool: parsed.tool, args: parsed.args ?? {} }
    }
  } catch { /* 파싱 실패 */ }
  return null
}

// ── 키워드 기반 폴백 (LLM 실패 시) ─────────────────────────────
function fallbackRoute(text: string, history?: HistoryTurn[]): ToolCall {
  const t = text.toLowerCase()

  // 맥락 의존 요청 ("그거 다시", "방금 한 거")
  if (/그거.*다시|방금.*다시|다시.*해줘|또.*해줘|재실행/i.test(text) && history?.length) {
    const lastUserMsg = [...(history ?? [])].reverse().find(h => h.role === 'user')?.content ?? ''
    return fallbackRoute(lastUserMsg)
  }

  // URL 다운로드
  if (/https?:\/\/[^\s]+/.test(text) && /다운|download|저장/i.test(t)) {
    return { tool: 'video_download', args: { url: text.match(/https?:\/\/[^\s]+/)?.[0] ?? '' } }
  }

  const map: Array<[RegExp, string, Record<string, string>]> = [
    // PC 상태
    [/pc.*상태|cpu|ram|메모리|온도|디스크|컴퓨터.*어때|pc.*어때|느려/i, 'pc_status', {}],
    [/보안.*스캔|해킹|악성코드|바이러스.*스캔/i, 'security_scan', {}],
    [/전체.*진단|종합.*점검|다.*점검/i, 'full_scan', {}],
    [/정리해|청소해|용량.*줄여|임시파일/i, 'clean', {}],
    [/gpu|그래픽카드|vram/i, 'gpu_stats', {}],
    [/부팅.*느려|부팅.*속도/i, 'boot_analysis', {}],
    [/드라이버/i, 'driver_check', {}],
    [/네트워크.*점검|인터넷.*느려|ip.*확인|dns/i, 'network_analysis', {}],
    [/윈도우.*업데이트|windows.*update/i, 'windows_updates', {}],
    [/백신|defender|디펜더/i, 'defender_status', {}],
    [/원격.*접속.*흔적|누가.*내.*pc/i, 'remote_access', {}],
    [/시작.*프로그램|자동.*실행/i, 'startup_items', {}],
    [/중복.*파일/i, 'file_duplicates', {}],
    [/브라우저.*정리|크롬.*캐시/i, 'browser_clean', {}],
    // 시스템 제어
    [/볼륨|소리.*키워|소리.*낮춰|음소거/i, 'volume_control', { action: /키워|크게/.test(t) ? 'up' : /낮춰|줄여/.test(t) ? 'down' : /음소거/.test(t) ? 'mute' : 'set' }],
    [/밝기|화면.*밝게|화면.*어둡게/i, 'brightness', { action: /밝게|키워/.test(t) ? 'up' : 'down' }],
    [/와이파이.*꺼|wifi.*off/i, 'wifi_toggle', { action: 'off' }],
    [/와이파이.*켜|wifi.*on/i, 'wifi_toggle', { action: 'on' }],
    [/재시작/i, 'power_action', { action: 'restart' }],
    [/종료해|shut.*down/i, 'power_action', { action: 'shutdown' }],
    [/잠금|lock.*pc/i, 'power_action', { action: 'lock' }],
    [/절전|sleep.*mode/i, 'power_action', { action: 'sleep' }],
    // 파일
    [/파일.*찾아|어디.*있어|파일.*어디/i, 'file_search', { query: text }],
    [/내용.*안에서|심층.*검색|파일.*내용.*검색/i, 'deep_search', { query: text }],
    [/바탕화면|다운로드.*폴더|문서.*폴더/i, 'open_folder', { folder: /바탕화면|desktop/.test(t) ? 'desktop' : /다운로드|download/.test(t) ? 'downloads' : 'documents' }],
    // 캘린더
    [/오늘.*일정|일정.*오늘/i, 'calendar_today', {}],
    [/이번.*주.*일정|주간.*일정/i, 'calendar_week', {}],
    [/일정.*추가|일정.*등록/i, 'calendar_add', { title: text }],
    // 이메일
    [/메일.*확인|받은.*메일/i, 'email_inbox', {}],
    [/메일.*요약/i, 'email_summarize', {}],
    // 화면/Vision
    [/화면.*뭐|화면.*분석|스크린.*분석|지금.*화면/i, 'vision_screen', { question: text }],
    [/클립보드.*이미지|복사한.*이미지.*텍스트/i, 'vision_ocr', {}],
    // 회의
    [/녹음.*시작|회의.*녹음/i, 'meeting_start', {}],
    [/녹음.*종료|녹음.*끝/i, 'meeting_stop', {}],
    [/회의.*요약|회의.*내용/i, 'meeting_summary', {}],
    // 미디어·검색
    [/유튜브|youtube/i, 'youtube_search', { query: text.replace(/유튜브에서?|youtube|찾아줘|검색/gi, '').trim() }],
    [/쿠팡|네이버쇼핑|최저가|가격.*얼마/i, 'price_compare', { query: text }],
    [/뉴스.*알려|오늘.*뉴스|최신.*뉴스/i, 'news_search', { query: text }],
    [/날씨/i, 'weather', { city: text.replace(/날씨/g, '').trim() || '서울' }],
    [/이동.*시간|얼마나.*걸려|교통/i, 'travel_time', { origin: '', destination: text }],
    [/번역해|영어로|한국어로|일본어로/i, 'translate', { target_lang: /영어/.test(t) ? 'en' : /일본/.test(t) ? 'ja' : 'ko' }],
    // 생산성
    [/집중.*모드|방해.*금지/i, 'focus_mode', {}],
    [/메모해|기록해줘|적어줘/i, 'notes', { content: text }],
    [/브리핑|오늘.*요약.*날씨/i, 'briefing_now', {}],
    [/자막.*켜|실시간.*자막/i, 'caption_start', {}],
    [/자막.*꺼/i, 'caption_stop', {}],
    [/화면.*기억|이.*화면.*저장/i, 'recall_capture', {}],
    [/아까.*봤던|예전에.*뭐.*봤/i, 'recall_search', { query: text }],
    [/워크플로.*실행|자동화.*실행/i, 'workflow_run', {}],
    [/워크플로.*목록/i, 'workflow_list', {}],
  ]

  for (const [pattern, tool, args] of map) {
    if (pattern.test(text)) return { tool, args }
  }

  return { tool: 'general_answer', args: { answer_type: 'knowledge' } }
}
