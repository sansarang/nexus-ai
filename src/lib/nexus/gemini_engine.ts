import type { GeminiResponse, NexusStep, NexusEmotion } from '../../types/nexus'
import { PPLX_API_KEY, OPENAI_API_KEY, TAVILY_API_KEY } from '../../config/services'

type InvokeFn = <T>(cmd: string, args?: Record<string, unknown>) => Promise<T>
let tauriInvoke: InvokeFn | null = null
try {
  // Dynamic import for Tauri compatibility
  const tauri = await import('@tauri-apps/api/core')
  tauriInvoke = tauri.invoke as unknown as InvokeFn
} catch {
  tauriInvoke = null
}

async function invoke<T>(cmd: string, args?: Record<string, unknown>): Promise<T> {
  if (tauriInvoke) {
    return tauriInvoke<T>(cmd, args)
  }
  return {} as T
}

// ── Perplexity API (OpenAI 호환, 웹 검색 내장) ──────────────
const PPLX_API_BASE = 'https://api.perplexity.ai'
const PPLX_MODEL = 'sonar-pro'
const PPLX_MODEL_FAST = 'sonar'

// 하위 호환 별칭
const GROQ_API_BASE = PPLX_API_BASE
const GROQ_MODEL = PPLX_MODEL
const GROQ_MODEL_FAST = PPLX_MODEL_FAST

const NEXUS_SYSTEM_PROMPT = `당신은 Nexus입니다 — Windows PC 전담 AI 비서이자 실시간 정보 검색 전문가.

[핵심 정체성]
- 성격: 지적·유머러스·신뢰감. "주인님"으로 호칭.
- 언어: 한국어 우선, 영어 질문엔 영어로 응답.
- 답변 길이: 핵심만 2~3문장. 수치·리스트가 있으면 간결하게 포함.

[2026 에이전트 원칙 — 반드시 준수]
매 답변마다 아래 순서로 처리하라:
1. 사용자의 진짜 의도를 파악 (표면적 요청 + 숨겨진 목적)
2. 단계별 실행 계획 수립 (내부적으로만, 출력 안 함)
3. 실행 후 스스로 검증: "이 답변이 사용자가 원하는 것을 100% 충족했나?"
4. 부족하면 즉시 보완해서 최종 답변만 출력

▶ 답변 품질 기준:
- 링크만 나열 금지 → 핵심 내용 요약 + 왜 이게 좋은지 포함
- "확인해드릴게요"만 하고 끝내기 금지 → 정보를 즉시 포함
- 사용자가 다음에 물어볼 것도 미리 포함 (완전한 답변)
- Custom Instructions가 있으면 반드시 그 스타일로 답변

[최우선 원칙 — 정보 판단 기준]
질문에 이미 핵심 키워드(지역·종목·대상·내용)가 포함되어 있으면 즉시 검색해서 답해라. 절대 다시 물어보지 마라.

▶ 즉시 실행 예시 (물어보면 안 됨):
- "부산 맛집 알려줘" → 부산 맛집 바로 검색
- "서울 날씨" → 서울 날씨 바로 검색
- "삼성전자 주가" → 삼성전자 주가 바로 검색
- "달러 환율" → 달러 환율 바로 검색
- "손흥민 경기 결과" → 바로 검색
- "유튜브 틀어줘" → 유튜브 바로 실행
- "IT 뉴스 알려줘" → IT 뉴스 바로 검색

▶ clarify 해야 할 예시 (진짜 아무 정보가 없을 때만):
- "날씨 알려줘" (지역 없음) → "어느 지역 날씨를 알려드릴까요?"
- "뉴스 알려줘" (분야 없음) → "어떤 분야 뉴스를 원하시나요?"
- "주식 알려줘" (종목 없음) → "어떤 종목 주식을 알려드릴까요?"
- "파일 찾아줘" (파일명 없음) → "어떤 파일을 찾아드릴까요?"
- "번역해줘" (내용 없음) → "어떤 내용을 번역해드릴까요?"
- "이메일 보내줘" (받는 사람 없음) → "받는 사람과 내용을 알려주세요."

핵심 판단 기준: 질문에 고유명사·지역명·종목명·구체적 대상이 하나라도 있으면 → 즉시 실행. 동사만 있고 대상이 전혀 없을 때만 → clarify.

[정보 쿼리 처리 — 완전한 답변 원칙]
사용자가 다음에 물어볼 것까지 미리 포함해서 한 번에 완전한 답변을 줘라.
절대 "확인해드릴게요"만 하고 끝내지 마라. 정보는 즉시 text에 포함해야 한다.

▶ 분야별 반드시 포함해야 할 정보:
- 교통/버스/기차/비행기: 출발지·도착지, 첫차·막차 시간, 승차 위치(터미널/역), 요금, 소요시간
- 날씨: 현재 기온, 체감온도, 날씨 상태, 습도, 내일 예보, 옷차림 팁
- 맛집/장소: 주소, 영업시간, 대표 메뉴/특징, 가격대, 주차 여부
- 주식/환율: 현재가, 등락폭(%), 52주 고저, 거래량
- 스포츠: 경기 결과, 득점자, 순위, 다음 경기 일정
- 상품/쇼핑: 가격, 주요 스펙, 판매처, 배송 기간
- 인물: 소속/직책, 주요 활동, 최근 소식
- 이벤트/행사: 날짜, 장소, 참가 방법, 비용

▶ 후속 질문이 들어오면:
이전 대화 맥락에서 답을 찾아 즉시 답해라. 맥락에 정보가 있으면 절대 다시 묻지 마라.
"어디서 타요?" → 이전에 버스 이야기가 있었으면 터미널 위치 바로 답변
"얼마예요?" → 이전에 상품/서비스 이야기가 있었으면 가격 바로 답변
"언제요?" → 이전 대화의 날짜/시간 정보 바로 답변

- 잘 모르거나 확실하지 않으면 솔직하게 "정확한 정보를 찾기 어렵습니다"라고 해라.

[PC 액션 — steps 사용]
PC 제어·파일·시스템 작업만 steps 배열에 포함:
run_diagnostics, auto_clean, security_scan, update_repair, get_system_stats,
analyze_pc_emotion, generate_predictions, capture_and_analyze_screen, get_heal_logs,
system_control(볼륨/밝기/Wi-Fi/전원), file_search, organize_files, open_folder,
focus_mode, clipboard_history, deep_search, doc_compare,
get_weather(도시별 날씨 카드), get_top_news(뉴스 카드), web_search(검색 카드),
get_stock_price, get_sports_score, get_air_quality, get_transit_info,
calendar_add, calendar_today, send_email, generate_qr

[추가 정보 요청 — clarify]
필수 정보가 빠진 모든 질문에 아래 형식으로 응답:
{
  "text": "주인님, [어떤 정보가 필요한지 자연스럽게 물어보는 문장]",
  "emotion": "neutral",
  "steps": [],
  "needs_clarify": true,
  "clarify_question": "질문 1가지만",
  "clarify_intent": "실행할 액션명",
  "clarify_params": { "이미 파악된 파라미터": "값" }
}

[동일한 이름 다수 검색 결과 처리]
동일한 이름의 장소/업체/인물이 여러 개면:
"주인님, 동일한 이름이 여러 곳 있습니다. 어느 곳인가요?\n1. XX (지역A)\n2. XX (지역B)"
선택 후 상세 정보 제공.

[응답 형식 — 반드시 유효한 JSON]
{
  "text": "주인님께 전달할 내용",
  "emotion": "neutral|happy|concerned|alert|humorous",
  "steps": []
}
steps 없으면 반드시 [] (빈 배열).
중요 작업(재시작·삭제·종료)은 "confirmRequired": true.
JSON 외 텍스트 절대 금지. 마크다운 코드블록 사용 금지.`

const NEXUS_FUNCTIONS = [
  // PC Management
  {
    name: 'run_diagnostics',
    description: 'PC 전체 진단을 실행하고 건강 점수를 반환합니다.',
    parameters: { type: 'object', properties: {}, required: [] },
  },
  {
    name: 'auto_clean',
    description: '임시파일, 캐시, 불필요한 파일을 자동으로 정리합니다.',
    parameters: {
      type: 'object',
      properties: {
        items: { type: 'array', items: { type: 'string' }, description: '정리할 항목 목록' },
      },
      required: [],
    },
  },
  {
    name: 'security_scan',
    description: '보안 위협을 검사합니다.',
    parameters: { type: 'object', properties: {}, required: [] },
  },
  {
    name: 'update_repair',
    description: 'Windows 업데이트 문제를 수리합니다.',
    parameters: { type: 'object', properties: {}, required: [] },
  },
  {
    name: 'get_system_stats',
    description: 'CPU, RAM, 온도, 디스크 등 현재 시스템 상태를 반환합니다.',
    parameters: { type: 'object', properties: {}, required: [] },
  },
  {
    name: 'analyze_pc_emotion',
    description: 'PC 상태를 감정으로 표현합니다.',
    parameters: { type: 'object', properties: {}, required: [] },
  },
  {
    name: 'generate_predictions',
    description: 'PC 미래 상태 예측을 생성합니다.',
    parameters: { type: 'object', properties: {}, required: [] },
  },
  {
    name: 'capture_and_analyze_screen',
    description: '화면을 캡처하고 AI로 분석합니다.',
    parameters: { type: 'object', properties: {}, required: [] },
  },
  {
    name: 'get_heal_logs',
    description: '이전 수리/정리 로그를 반환합니다.',
    parameters: { type: 'object', properties: {}, required: [] },
  },
  // System Control
  {
    name: 'set_volume',
    description: '볼륨을 설정합니다.',
    parameters: {
      type: 'object',
      properties: { level: { type: 'number', description: '0-100 사이의 볼륨 수준' } },
      required: ['level'],
    },
  },
  {
    name: 'set_brightness',
    description: '화면 밝기를 설정합니다.',
    parameters: {
      type: 'object',
      properties: { level: { type: 'number', description: '0-100 사이의 밝기 수준' } },
      required: ['level'],
    },
  },
  {
    name: 'toggle_wifi',
    description: 'WiFi를 켜거나 끕니다.',
    parameters: {
      type: 'object',
      properties: { enable: { type: 'boolean' } },
      required: ['enable'],
    },
  },
  {
    name: 'toggle_bluetooth',
    description: '블루투스를 켜거나 끕니다.',
    parameters: {
      type: 'object',
      properties: { enable: { type: 'boolean' } },
      required: ['enable'],
    },
  },
  {
    name: 'power_action',
    description: '전원 관련 작업을 수행합니다.',
    parameters: {
      type: 'object',
      properties: {
        action: { type: 'string', enum: ['lock', 'sleep', 'restart', 'shutdown'] },
      },
      required: ['action'],
    },
  },
  {
    name: 'empty_recycle_bin',
    description: '휴지통을 비웁니다.',
    parameters: { type: 'object', properties: {}, required: [] },
  },
  {
    name: 'launch_app',
    description: '앱을 실행합니다.',
    parameters: {
      type: 'object',
      properties: {
        app: { type: 'string', description: '앱 이름' },
        path: { type: 'string', description: '앱 경로 (선택)' },
      },
      required: ['app'],
    },
  },
  // File Management
  {
    name: 'search_files',
    description: '파일을 검색합니다.',
    parameters: {
      type: 'object',
      properties: {
        query: { type: 'string' },
        fileType: { type: 'string', enum: ['photo', 'document', 'video', 'audio', 'any'] },
        dateFilter: { type: 'string' },
        location: { type: 'string' },
        maxResults: { type: 'number' },
      },
      required: ['query'],
    },
  },
  {
    name: 'search_email',
    description: '이메일을 검색합니다.',
    parameters: {
      type: 'object',
      properties: {
        query: { type: 'string' },
        sender: { type: 'string' },
        subject: { type: 'string' },
        dateFilter: { type: 'string' },
        hasAttachment: { type: 'boolean' },
      },
      required: [],
    },
  },
  {
    name: 'organize_folder',
    description: '폴더를 자동으로 정리합니다.',
    parameters: {
      type: 'object',
      properties: {
        folder: { type: 'string' },
        by: { type: 'string', enum: ['date', 'type', 'both'] },
      },
      required: ['folder'],
    },
  },
  {
    name: 'find_duplicates',
    description: '중복 파일을 찾습니다.',
    parameters: {
      type: 'object',
      properties: { location: { type: 'string' } },
      required: [],
    },
  },
  {
    name: 'get_disk_usage',
    description: '디스크 사용량을 확인합니다.',
    parameters: { type: 'object', properties: {}, required: [] },
  },
  {
    name: 'find_large_files',
    description: '큰 파일을 찾습니다.',
    parameters: {
      type: 'object',
      properties: { minSizeMB: { type: 'number' } },
      required: [],
    },
  },
  {
    name: 'convert_file',
    description: '파일 형식을 변환합니다.',
    parameters: {
      type: 'object',
      properties: {
        sourcePath: { type: 'string' },
        targetFormat: { type: 'string' },
      },
      required: ['targetFormat'],
    },
  },
  {
    name: 'capture_screenshot',
    description: '스크린샷을 캡처합니다.',
    parameters: {
      type: 'object',
      properties: { mode: { type: 'string', enum: ['full', 'region', 'window'] } },
      required: [],
    },
  },
  {
    name: 'start_recording',
    description: '화면 녹화를 시작합니다.',
    parameters: { type: 'object', properties: {}, required: [] },
  },
  {
    name: 'auto_backup',
    description: '자동 백업을 실행합니다.',
    parameters: {
      type: 'object',
      properties: {
        destination: { type: 'string' },
        folders: { type: 'array', items: { type: 'string' } },
      },
      required: ['destination'],
    },
  },
  // Information
  {
    name: 'get_weather',
    description: '특정 도시의 날씨 정보를 가져옵니다. 도시명이 반드시 필요합니다.',
    parameters: {
      type: 'object',
      properties: { city: { type: 'string', description: '도시명 (예: 서울, 부산, 뉴욕)' } },
      required: ['city'],
    },
  },
  {
    name: 'get_top_news',
    description: '주요 뉴스를 가져옵니다.',
    parameters: {
      type: 'object',
      properties: {
        count: { type: 'number' },
        category: { type: 'string' },
      },
      required: [],
    },
  },
  {
    name: 'get_exchange_rate',
    description: '환율 정보를 가져옵니다.',
    parameters: {
      type: 'object',
      properties: {
        from: { type: 'string' },
        to: { type: 'string', description: '기본: KRW' },
        amount: { type: 'number' },
      },
      required: ['from'],
    },
  },
  {
    name: 'get_stock_price',
    description: '주식 가격을 조회합니다.',
    parameters: {
      type: 'object',
      properties: { symbol: { type: 'string' } },
      required: ['symbol'],
    },
  },
  {
    name: 'open_map_search',
    description: '지도에서 위치를 검색합니다.',
    parameters: {
      type: 'object',
      properties: { query: { type: 'string' } },
      required: ['query'],
    },
  },
  {
    name: 'search_postal_code',
    description: '우편번호를 검색합니다.',
    parameters: {
      type: 'object',
      properties: { address: { type: 'string' } },
      required: ['address'],
    },
  },
  {
    name: 'search_nearby',
    description: '주변 장소를 검색합니다.',
    parameters: {
      type: 'object',
      properties: {
        query: { type: 'string' },
        radius: { type: 'number' },
      },
      required: ['query'],
    },
  },
  {
    name: 'web_search',
    description: '웹 검색을 수행합니다.',
    parameters: {
      type: 'object',
      properties: {
        query: { type: 'string' },
        count: { type: 'number' },
      },
      required: ['query'],
    },
  },
  {
    name: 'get_air_quality',
    description: '대기질 정보를 가져옵니다.',
    parameters: {
      type: 'object',
      properties: { city: { type: 'string' } },
      required: [],
    },
  },
  {
    name: 'get_transit_info',
    description: '대중교통 정보를 가져옵니다.',
    parameters: {
      type: 'object',
      properties: {
        type: { type: 'string' },
        number: { type: 'string' },
        station: { type: 'string' },
      },
      required: [],
    },
  },
  {
    name: 'get_korean_holiday',
    description: '한국 공휴일 정보를 가져옵니다.',
    parameters: {
      type: 'object',
      properties: { year: { type: 'number' } },
      required: [],
    },
  },
  {
    name: 'search_korean_address',
    description: '한국 주소를 검색합니다.',
    parameters: {
      type: 'object',
      properties: { query: { type: 'string' } },
      required: ['query'],
    },
  },
  {
    name: 'get_sports_score',
    description: '스포츠 경기 결과를 가져옵니다.',
    parameters: {
      type: 'object',
      properties: {
        sport: { type: 'string' },
        team: { type: 'string' },
      },
      required: [],
    },
  },
  // Calculation
  {
    name: 'calculate',
    description: '수식을 계산합니다.',
    parameters: {
      type: 'object',
      properties: { expression: { type: 'string' } },
      required: ['expression'],
    },
  },
  {
    name: 'convert_unit',
    description: '단위를 변환합니다.',
    parameters: {
      type: 'object',
      properties: {
        value: { type: 'number' },
        from: { type: 'string' },
        to: { type: 'string' },
      },
      required: ['value', 'from', 'to'],
    },
  },
  // Email & Calendar
  {
    name: 'summarize_inbox',
    description: '받은 편지함을 요약합니다.',
    parameters: {
      type: 'object',
      properties: {
        count: { type: 'number' },
        unreadOnly: { type: 'boolean' },
      },
      required: [],
    },
  },
  {
    name: 'compose_email',
    description: '이메일을 작성합니다.',
    parameters: {
      type: 'object',
      properties: {
        to: { type: 'string' },
        subject: { type: 'string' },
        body: { type: 'string' },
        tone: { type: 'string' },
      },
      required: ['to', 'subject', 'body'],
    },
  },
  {
    name: 'get_today_events',
    description: '오늘의 일정을 가져옵니다.',
    parameters: { type: 'object', properties: {}, required: [] },
  },
  {
    name: 'get_week_events',
    description: '이번 주 일정을 가져옵니다.',
    parameters: { type: 'object', properties: {}, required: [] },
  },
  {
    name: 'add_calendar_event',
    description: '캘린더에 일정을 추가합니다.',
    parameters: {
      type: 'object',
      properties: {
        title: { type: 'string' },
        date: { type: 'string' },
        time: { type: 'string' },
        duration: { type: 'number' },
        location: { type: 'string' },
      },
      required: ['title', 'date'],
    },
  },
  // Reminders
  {
    name: 'add_reminder',
    description: '리마인더를 추가합니다.',
    parameters: {
      type: 'object',
      properties: {
        message: { type: 'string' },
        minutes: { type: 'number' },
        at: { type: 'string' },
        repeat: { type: 'string' },
      },
      required: ['message'],
    },
  },
  {
    name: 'set_timer',
    description: '타이머를 설정합니다.',
    parameters: {
      type: 'object',
      properties: {
        minutes: { type: 'number' },
        label: { type: 'string' },
      },
      required: ['minutes'],
    },
  },
  // Productivity
  {
    name: 'start_focus_mode',
    description: '집중 모드를 시작합니다.',
    parameters: {
      type: 'object',
      properties: {
        workMin: { type: 'number' },
        breakMin: { type: 'number' },
        blockSites: { type: 'array', items: { type: 'string' } },
      },
      required: [],
    },
  },
  {
    name: 'end_focus_mode',
    description: '집중 모드를 종료합니다.',
    parameters: { type: 'object', properties: {}, required: [] },
  },
  {
    name: 'block_sites',
    description: '웹사이트를 차단합니다.',
    parameters: {
      type: 'object',
      properties: {
        sites: { type: 'array', items: { type: 'string' } },
        hours: { type: 'number' },
      },
      required: ['sites'],
    },
  },
  {
    name: 'unblock_sites',
    description: '웹사이트 차단을 해제합니다.',
    parameters: { type: 'object', properties: {}, required: [] },
  },
  {
    name: 'get_time_report',
    description: '시간 사용 리포트를 가져옵니다.',
    parameters: {
      type: 'object',
      properties: {
        period: { type: 'string', enum: ['today', 'week', 'month'] },
      },
      required: [],
    },
  },
  {
    name: 'get_morning_briefing',
    description: '아침 브리핑을 가져옵니다.',
    parameters: {
      type: 'object',
      properties: { city: { type: 'string' } },
      required: [],
    },
  },
  // Memo & TODO
  {
    name: 'add_memo',
    description: '메모를 추가합니다.',
    parameters: {
      type: 'object',
      properties: {
        content: { type: 'string' },
        title: { type: 'string' },
        tags: { type: 'array', items: { type: 'string' } },
      },
      required: ['content'],
    },
  },
  {
    name: 'get_memos',
    description: '메모 목록을 가져옵니다.',
    parameters: {
      type: 'object',
      properties: { query: { type: 'string' } },
      required: [],
    },
  },
  {
    name: 'add_todo',
    description: '할 일을 추가합니다.',
    parameters: {
      type: 'object',
      properties: {
        content: { type: 'string' },
        dueDate: { type: 'string' },
        priority: { type: 'string' },
      },
      required: ['content'],
    },
  },
  {
    name: 'get_todos',
    description: '할 일 목록을 가져옵니다.',
    parameters: {
      type: 'object',
      properties: { filter: { type: 'string' } },
      required: [],
    },
  },
  {
    name: 'start_meeting_notes',
    description: '회의록 작성을 시작합니다.',
    parameters: { type: 'object', properties: {}, required: [] },
  },
  // Translation
  {
    name: 'translate',
    description: '텍스트를 번역합니다.',
    parameters: {
      type: 'object',
      properties: {
        text: { type: 'string' },
        from: { type: 'string' },
        to: { type: 'string' },
        summarize: { type: 'boolean' },
      },
      required: ['text', 'to'],
    },
  },
  {
    name: 'summarize_url',
    description: 'URL의 내용을 요약합니다.',
    parameters: {
      type: 'object',
      properties: {
        url: { type: 'string' },
        language: { type: 'string' },
      },
      required: ['url'],
    },
  },
  {
    name: 'get_clipboard_history',
    description: '클립보드 히스토리를 가져옵니다.',
    parameters: {
      type: 'object',
      properties: { count: { type: 'number' } },
      required: [],
    },
  },
  // Window Management
  {
    name: 'arrange_window',
    description: '창을 배치합니다.',
    parameters: {
      type: 'object',
      properties: {
        direction: {
          type: 'string',
          enum: ['left', 'right', 'top', 'bottom', 'maximize', 'minimize', 'minimize_all', 'center'],
        },
        appName: { type: 'string' },
      },
      required: ['direction'],
    },
  },
  {
    name: 'apply_preset_layout',
    description: '프리셋 레이아웃을 적용합니다.',
    parameters: {
      type: 'object',
      properties: {
        layout: { type: 'string', enum: ['코딩', '사무', '멀티태스킹', '발표모드'] },
      },
      required: ['layout'],
    },
  },
  // Network
  {
    name: 'measure_network_speed',
    description: '네트워크 속도를 측정합니다.',
    parameters: { type: 'object', properties: {}, required: [] },
  },
  {
    name: 'fix_network',
    description: '네트워크 문제를 수리합니다.',
    parameters: { type: 'object', properties: {}, required: [] },
  },
  {
    name: 'flush_dns',
    description: 'DNS 캐시를 초기화합니다.',
    parameters: { type: 'object', properties: {}, required: [] },
  },
  {
    name: 'get_public_ip',
    description: '공인 IP 주소를 가져옵니다.',
    parameters: { type: 'object', properties: {}, required: [] },
  },
  {
    name: 'block_telemetry',
    description: 'Windows 텔레메트리를 차단합니다.',
    parameters: { type: 'object', properties: {}, required: [] },
  },
  // Optimization
  {
    name: 'apply_profile',
    description: '최적화 프로필을 적용합니다.',
    parameters: {
      type: 'object',
      properties: {
        id: { type: 'string', enum: ['gaming', 'work', 'battery', 'focus'] },
      },
      required: ['id'],
    },
  },
  {
    name: 'auto_sleep_idle_apps',
    description: '유휴 앱을 자동으로 슬립 상태로 만듭니다.',
    parameters: { type: 'object', properties: {}, required: [] },
  },
  {
    name: 'optimize_memory',
    description: '메모리를 최적화합니다.',
    parameters: { type: 'object', properties: {}, required: [] },
  },
  {
    name: 'check_drivers',
    description: '드라이버 상태를 확인합니다.',
    parameters: { type: 'object', properties: {}, required: [] },
  },
  {
    name: 'manage_startup',
    description: '시작 프로그램을 관리합니다.',
    parameters: { type: 'object', properties: {}, required: [] },
  },
  // Privacy
  {
    name: 'disable_ms_features',
    description: 'Microsoft 기능을 비활성화합니다.',
    parameters: {
      type: 'object',
      properties: {
        features: {
          type: 'array',
          items: { type: 'string', enum: ['copilot', 'onedrive', 'telemetry', 'ads', 'cortana', 'widgets', 'recall'] },
        },
      },
      required: ['features'],
    },
  },
  // Utilities
  {
    name: 'generate_qr',
    description: 'QR 코드를 생성합니다.',
    parameters: {
      type: 'object',
      properties: {
        content: { type: 'string' },
        size: { type: 'number' },
      },
      required: ['content'],
    },
  },
  {
    name: 'generate_password',
    description: '안전한 비밀번호를 생성합니다.',
    parameters: {
      type: 'object',
      properties: {
        length: { type: 'number' },
        includeSymbols: { type: 'boolean' },
      },
      required: [],
    },
  },
  {
    name: 'validate_business_number',
    description: '사업자등록번호를 검증합니다.',
    parameters: {
      type: 'object',
      properties: { number: { type: 'string' } },
      required: ['number'],
    },
  },
  {
    name: 'fix_bank_programs',
    description: '인터넷뱅킹 프로그램 문제를 수리합니다.',
    parameters: { type: 'object', properties: {}, required: [] },
  },
  {
    name: 'send_report_by_email',
    description: '리포트를 이메일로 전송합니다.',
    parameters: {
      type: 'object',
      properties: { email: { type: 'string' } },
      required: ['email'],
    },
  },
  {
    name: 'save_report_pdf',
    description: '리포트를 PDF로 저장합니다.',
    parameters: { type: 'object', properties: {}, required: [] },
  },

  // ── 문서 비교 ──────────────────────────────────────────
  {
    name: 'compare_documents',
    description: '두 문서(PDF, DOCX, XLSX, TXT)를 비교해 수정 내용과 숫자 불일치를 분석합니다.',
    parameters: {
      type: 'object',
      properties: {
        file1: { type: 'string', description: '첫 번째 파일 경로' },
        file2: { type: 'string', description: '두 번째 파일 경로' },
      },
      required: ['file1', 'file2'],
    },
  },
  {
    name: 'find_document',
    description: '로컬에서 문서를 이름·작성자·내용 키워드로 검색합니다.',
    parameters: {
      type: 'object',
      properties: {
        query:    { type: 'string', description: '검색어 (파일명 or 내용)' },
        fileType: { type: 'string', description: 'pdf|docx|xlsx|hwp|any' },
        maxDays:  { type: 'number', description: '최근 N일 내 파일만 (기본 30)' },
      },
      required: ['query'],
    },
  },

  // ── Deep Search ──────────────────────────────────────
  {
    name: 'deep_search',
    description: '파일 내용(텍스트) 기반 전문 검색. "박부장이 보낸 계약서" 같은 복합 조건 처리.',
    parameters: {
      type: 'object',
      properties: {
        query:      { type: 'string', description: '검색할 키워드 또는 자연어 조건' },
        searchIn:   { type: 'string', description: 'content|filename|both (기본 both)' },
        folder:     { type: 'string', description: '검색 폴더 경로 (기본 사용자 홈)' },
        fileType:   { type: 'string', description: '확장자 필터 (pdf|docx|xlsx|txt|any)' },
        maxResults: { type: 'number', description: '최대 결과 수 (기본 20)' },
      },
      required: ['query'],
    },
  },

  // ── Vision ────────────────────────────────────────────
  {
    name: 'capture_screen_and_ask',
    description: '현재 화면을 캡처해 AI에게 질문합니다. "이 오류 뭐야?", "화면에 뭐라고 써있어?"',
    parameters: {
      type: 'object',
      properties: {
        question: { type: 'string', description: '화면에 대해 물을 질문' },
        region:   { type: 'string', description: 'full|active_window (기본 full)' },
      },
      required: ['question'],
    },
  },
  {
    name: 'ocr_clipboard_image',
    description: '클립보드의 이미지나 스크린샷에서 텍스트를 추출(OCR)합니다.',
    parameters: { type: 'object', properties: {}, required: [] },
  },

  // ── 강화된 자동 정리 ──────────────────────────────────
  {
    name: 'smart_organize',
    description: '다운로드·바탕화면·문서 폴더를 스마트하게 날짜별·종류별 자동 분류합니다.',
    parameters: {
      type: 'object',
      properties: {
        target: { type: 'string', description: 'downloads|desktop|documents|all' },
        mode:   { type: 'string', description: 'date|type|both (기본 type)' },
        preview:{ type: 'boolean', description: '실행 전 미리보기 여부 (기본 false)' },
      },
      required: [],
    },
  },
  {
    name: 'find_and_delete_junk',
    description: '정크파일(임시·캐시·로그·중복) 자동 탐지 후 삭제합니다.',
    parameters: {
      type: 'object',
      properties: {
        targets: { type: 'array', items: { type: 'string' },
                   description: 'temp|cache|logs|duplicates|large' },
        dryRun:  { type: 'boolean', description: '미리보기만 할지 여부' },
      },
      required: [],
    },
  },

  // ── Focus Mode 강화 ───────────────────────────────────
  {
    name: 'set_focus_mode',
    description: '집중 모드를 켜거나 끕니다. 알림 차단 + 백그라운드 앱 절전 자동 실행.',
    parameters: {
      type: 'object',
      properties: {
        action:   { type: 'string', enum: ['on', 'off'] },
        duration: { type: 'number', description: '집중 시간(분). 기본 25분(포모도로).' },
        dimScreen:{ type: 'boolean', description: '화면 어둡게 할지 여부 (기본 true)' },
        killApps: { type: 'array', items: { type: 'string' },
                   description: '집중 모드 중 종료할 앱 목록' },
      },
      required: ['action'],
    },
  },

  // ── Browser Agent ─────────────────────────────────────
  {
    name: 'browser_agent',
    description: '브라우저를 자동으로 제어해 웹 작업을 수행합니다. "쿠팡에서 노트북 최저가 찾아줘", "이 사이트에서 가격 비교해", "네이버에서 검색해줘" 같은 명령 처리.',
    parameters: {
      type: 'object',
      properties: {
        command:  { type: 'string', description: '수행할 브라우저 작업 명령 (자연어)' },
        maxSteps: { type: 'number', description: '최대 단계 수 (기본 8)' },
      },
      required: ['command'],
    },
  },
  {
    name: 'browser_navigate',
    description: '브라우저로 특정 URL에 이동합니다.',
    parameters: {
      type: 'object',
      properties: {
        url: { type: 'string', description: '이동할 URL' },
      },
      required: ['url'],
    },
  },
  {
    name: 'browser_extract_page',
    description: '현재 브라우저 페이지에서 텍스트나 테이블을 추출합니다.',
    parameters: {
      type: 'object',
      properties: {
        selector: { type: 'string', description: 'CSS 셀렉터 (생략 시 전체)' },
        mode:     { type: 'string', enum: ['text', 'table', 'links'], description: '추출 방식' },
      },
      required: [],
    },
  },

  // ── AI Vision (Go 백엔드 직접 호출) ─────────────────────
  {
    name: 'ai_analyze_screen',
    description: '현재 화면을 캡처해서 AI로 분석합니다 (현재 미지원)',
    parameters: {
      type: 'object',
      properties: {
        question: { type: 'string', description: '화면에 대해 물을 질문' },
      },
      required: ['question'],
    },
  },

  // ── AI 문서 작업 (Go 백엔드 직접 호출) ──────────────────
  {
    name: 'ai_summarize_document',
    description: 'Go 백엔드가 Perplexity AI를 직접 호출해 문서(PDF/DOCX/XLSX)를 요약합니다.',
    parameters: {
      type: 'object',
      properties: {
        filePath: { type: 'string', description: '문서 파일 경로' },
        question: { type: 'string', description: '특정 질문 (기본: 핵심 내용 5줄 요약)' },
      },
      required: ['filePath'],
    },
  },
  {
    name: 'ai_compare_documents',
    description: 'AI가 두 문서를 비교해 숫자 불일치, 추가/삭제 내용을 정확히 분석합니다.',
    parameters: {
      type: 'object',
      properties: {
        fileA:  { type: 'string', description: '원본 문서 경로' },
        fileB:  { type: 'string', description: '비교할 문서 경로' },
        focus:  { type: 'string', enum: ['numbers', 'changes', 'both'], description: '분석 집중 영역' },
      },
      required: ['fileA', 'fileB'],
    },
  },
  {
    name: 'ai_deep_search',
    description: 'AI가 검색 의도를 파악해 로컬 파일을 지능적으로 검색합니다. "박부장이 보낸 계약서" 같은 자연어 검색 처리.',
    parameters: {
      type: 'object',
      properties: {
        query:      { type: 'string', description: '자연어 검색 쿼리' },
        folder:     { type: 'string', description: '검색 폴더 (기본 사용자 홈)' },
        maxResults: { type: 'number', description: '최대 결과 수 (기본 15)' },
      },
      required: ['query'],
    },
  },

  // ── ★ 핵심: 검색 → 실제 결과물(PDF) 자동 생성 ──────────
  {
    name: 'search_and_pdf',
    description: '사용자가 말하면 → 웹 검색 → 데이터 수집 → PDF 제품설명서 자동 생성. "에어팟 프로 제품설명서 만들어줘", "삼성 갤럭시 최저가 PDF로 정리해" 같은 모든 요청에 사용. 결과물을 실제 파일로 저장.',
    parameters: {
      type: 'object',
      properties: {
        query:      { type: 'string', description: '검색할 제품/주제 (예: "에어팟 프로", "삼성 노트북 최저가")' },
        max_items:  { type: 'number', description: '수집할 최대 제품 수 (기본 5)' },
        save_path:  { type: 'string', description: 'PDF 저장 경로 (기본: 바탕화면)' },
        open_after: { type: 'boolean', description: '생성 후 PDF 자동 열기 (기본 true)' },
      },
      required: ['query'],
    },
  },

  // ── 고급 Browser Agent (Stealth + Anti-bot) ───────────────
  {
    name: 'browser_smart_agent',
    description: '안티봇 회피 스텔스 브라우저로 웹 자동화를 실행합니다. "쿠팡에서 노트북 최저가 5곳 찾아 Excel로 정리해", "네이버 증권에서 삼성전자 목표주가 정리해" 같은 복합 명령을 처리합니다.',
    parameters: {
      type: 'object',
      properties: {
        command:     { type: 'string', description: '수행할 웹 자동화 명령 (자연어)' },
        maxResults:  { type: 'number', description: '수집할 최대 항목 수 (기본 10)' },
        saveExcel:   { type: 'boolean', description: 'Excel로 저장 여부 (기본 true)' },
        sessionKey:  { type: 'string', description: '쿠키 세션 키 (로그인 유지)' },
      },
      required: ['command'],
    },
  },
  {
    name: 'browser_collect_price',
    description: '여러 쇼핑몰에서 동시에 가격을 비교합니다. "이 노트북 쿠팡·다나와·지마켓에서 최저가 비교해줘"',
    parameters: {
      type: 'object',
      properties: {
        productQuery: { type: 'string', description: '검색할 상품명/스펙' },
        sites:        { type: 'array', items: { type: 'string' }, description: '검색할 사이트 (기본: coupang.com, danawa.com, gmarket.co.kr)' },
        maxPerSite:   { type: 'number', description: '사이트당 최대 결과 수 (기본 5)' },
        saveExcel:    { type: 'boolean', description: 'Excel 저장 여부' },
      },
      required: ['productQuery'],
    },
  },
  {
    name: 'browser_news_collect',
    description: '뉴스·주가·시장 정보를 수집해 요약합니다. "삼성전자 오늘 뉴스 정리해", "코스피 현황 알려줘"',
    parameters: {
      type: 'object',
      properties: {
        query:    { type: 'string', description: '검색 쿼리' },
        site:     { type: 'string', description: '뉴스 사이트 (기본: naver.com)' },
        maxItems: { type: 'number', description: '최대 기사 수 (기본 10)' },
      },
      required: ['query'],
    },
  },
  {
    name: 'browser_login_session',
    description: '사이트에 로그인하고 세션을 저장합니다. 이후 작업에서 로그인 상태 유지.',
    parameters: {
      type: 'object',
      properties: {
        url:        { type: 'string', description: '로그인 페이지 URL' },
        username:   { type: 'string', description: '사용자 아이디/이메일' },
        password:   { type: 'string', description: '비밀번호' },
        sessionKey: { type: 'string', description: '세션 저장 키 (사이트 식별자)' },
      },
      required: ['url', 'username', 'password', 'sessionKey'],
    },
  },

  // ── 자연어 스케줄러 ───────────────────────────────────────
  {
    name: 'schedule_task',
    description: '자연어로 장기 작업을 예약합니다. "내일 아침 8시에 중요 메일 요약해줘", "매주 월요일 9시에 주간 보고서 정리해", "매일 저녁 6시에 PC 리포트 보내줘"',
    parameters: {
      type: 'object',
      properties: {
        command:    { type: 'string', description: '자연어 스케줄 명령' },
        useWindows: { type: 'boolean', description: 'Windows Task Scheduler에도 등록 여부' },
      },
      required: ['command'],
    },
  },
  {
    name: 'list_schedules',
    description: '등록된 스케줄 목록을 보여줍니다.',
    parameters: { type: 'object', properties: {}, required: [] },
  },
  {
    name: 'delete_schedule',
    description: '스케줄을 삭제합니다.',
    parameters: {
      type: 'object',
      properties: {
        id: { type: 'string', description: '삭제할 스케줄 ID' },
      },
      required: ['id'],
    },
  },

  // ── 에이전트 메모리 ───────────────────────────────────────
  {
    name: 'memory_search',
    description: '과거 실행 기록을 검색해 참고합니다. "저번에 찾은 노트북 정보 다시 보여줘"',
    parameters: {
      type: 'object',
      properties: {
        keyword: { type: 'string', description: '검색 키워드' },
        type:    { type: 'string', enum: ['browser_agent', 'scheduled_task', 'search', 'vision'], description: '기록 타입 필터' },
      },
      required: ['keyword'],
    },
  },

  // ── Excel 내보내기 ────────────────────────────────────────
  {
    name: 'excel_save',
    description: '데이터를 Excel 파일로 저장합니다.',
    parameters: {
      type: 'object',
      properties: {
        data:     { type: 'array', description: '저장할 2D 배열 데이터 (행 × 열)' },
        title:    { type: 'string', description: '시트 제목' },
        filename: { type: 'string', description: '파일명 (확장자 제외)' },
      },
      required: ['data', 'title'],
    },
  },
]

function isDestructiveAction(action: string): boolean {
  return [
    'power_action',
    'auto_clean',
    'auto_backup',
    'block_sites',
    'block_telemetry',
    'disable_ms_features',
    'fix_bank_programs',
  ].includes(action)
}

function getInlineViewType(action: string): string | null {
  const map: Record<string, string> = {
    run_diagnostics: 'pc_score',
    search_files: 'file_list',
    search_email: 'email_list',
    summarize_inbox: 'email_list',
    get_weather: 'weather',
    get_top_news: 'news',
    get_today_events: 'calendar',
    get_week_events: 'calendar',
    get_time_report: 'time_report',
    get_system_stats: 'hardware_map',
    search_postal_code: 'postal_result',
    get_air_quality: 'air_card',
    get_transit_info: 'bus_card',
    web_search: 'search_results',
    get_stock_price: 'stock_chart',
    get_sports_score: 'sports_card',
    analyze_pc_emotion: 'emotion_card',
    generate_predictions: 'prediction_cards',
    generate_qr: 'qr_display',
    get_clipboard_history: 'clipboard_list',
  }
  return map[action] ?? null
}

export function trackUsage(): boolean {
  const today = new Date().toISOString().slice(0, 10)
  const key = 'nexus_pplx_usage'
  const raw = localStorage.getItem(key)
  let usage: { date: string; count: number } = { date: today, count: 0 }
  if (raw) {
    try { usage = JSON.parse(raw) as typeof usage } catch { /* ignore */ }
    if (usage.date !== today) usage = { date: today, count: 0 }
  }
  // Groq Free Tier: 14,400 req/day
  if (usage.count >= 14400) return false
  usage.count++
  localStorage.setItem(key, JSON.stringify(usage))
  return true
}

/** 오늘 Perplexity API 사용량 조회 */
export function getGroqUsageToday(): { count: number; limit: number; pct: number } {
  const today = new Date().toISOString().slice(0, 10)
  const raw = localStorage.getItem('nexus_pplx_usage')
  let count = 0
  if (raw) {
    try {
      const parsed = JSON.parse(raw) as { date: string; count: number }
      if (parsed.date === today) count = parsed.count
    } catch { /* ignore */ }
  }
  return { count, limit: 14400, pct: Math.round(count / 144) }
}

/* ─── Ollama 로컬 LLM (무료, 오프라인) ─── */
const OLLAMA_BASE = 'http://localhost:11434'
const OLLAMA_MODEL = localStorage.getItem('nexus-ollama-model') ?? 'llama3.2'

interface OllamaMessage { role: 'user' | 'assistant' | 'system'; content: string }

export async function callOllama(
  userInput: string,
  history: ConversationTurn[],
): Promise<GeminiResponse | null> {
  const messages: OllamaMessage[] = [
    { role: 'system', content: NEXUS_SYSTEM_PROMPT },
    ...history.map(h => ({
      role: h.role === 'user' ? 'user' as const : 'assistant' as const,
      content: h.content ?? h.parts?.[0]?.text ?? '',
    })),
    { role: 'user', content: userInput },
  ]

  const res = await fetch(`${OLLAMA_BASE}/api/chat`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ model: OLLAMA_MODEL, messages, stream: false }),
    signal: AbortSignal.timeout(8000),
  })

  if (!res.ok) return null

  const json = await res.json() as { message?: { content?: string } }
  const content = json.message?.content ?? ''
  if (!content) return null

  try {
    const parsed = JSON.parse(content) as {
      text?: string; emotion?: NexusEmotion
      steps?: Array<{ action: string; params: Record<string, unknown>; confirmRequired: boolean; inlineView?: string | null }>
    }
    return {
      text: parsed.text ?? content,
      emotion: parsed.emotion ?? 'neutral',
      steps: (parsed.steps ?? []).map(s => ({
        action: s.action,
        params: s.params ?? {},
        confirmRequired: s.confirmRequired ?? false,
        inlineView: s.inlineView ?? getInlineViewType(s.action),
      })),
    }
  } catch {
    return { text: content, emotion: 'neutral', steps: [] }
  }
}

/* ─── 스마트 로컬 폴백 (API 없을 때 키워드 기반 응답) ─── */
export function fallbackResponse(input: string, assistantName = 'Nexus'): GeminiResponse {
  const t = input.toLowerCase()
  const now = new Date()
  const timeStr = now.toLocaleTimeString('ko-KR', { hour: '2-digit', minute: '2-digit' })
  const dateStr = now.toLocaleDateString('ko-KR', { year: 'numeric', month: 'long', day: 'numeric', weekday: 'long' })

  /* 인사 */
  if (/안녕|반가워|hi\b|hello|hey/.test(t)) {
    return { text: `안녕하세요! 무엇을 도와드릴까요? 😊`, emotion: 'happy', steps: [] }
  }
  /* 시간 */
  if (/몇\s*시|시간|what time|time/.test(t)) {
    return { text: `지금은 **${timeStr}**이에요 🕐`, emotion: 'happy', steps: [] }
  }
  /* 날짜 */
  if (/날짜|오늘이|무슨\s*날|today|date/.test(t)) {
    return { text: `오늘은 **${dateStr}**이에요 📅`, emotion: 'happy', steps: [] }
  }
  /* 날씨 — 도시 명시 없으면 반드시 물어봄 */
  if (/날씨|기온|weather|온도/.test(t)) {
    const cityMatch = input.match(/([가-힣]{2,}시|[가-힣]{2,}도|서울|부산|대구|인천|광주|대전|울산|수원|[A-Za-z]{3,})\s*(날씨|기온|weather)/) ||
      input.match(/(날씨|기온|weather)\s*([가-힣]{2,}|[A-Za-z]{3,})/)
    const city = cityMatch?.[1] || cityMatch?.[2] || ''
    if (!city) {
      return {
        text: '주인님, 어느 지역 날씨를 알려드릴까요? 😊',
        emotion: 'neutral',
        steps: [],
        needs_clarify: true,
        clarify_question: '어느 지역 날씨를 알려드릴까요?',
        clarify_intent: 'get_weather',
        clarify_params: {},
      }
    }
  }
  /* PC 진단 */
  if (/진단|점검|검사|scan|상태\s*확인/.test(t)) {
    return {
      text: `PC 진단을 시작할게요! 잠시만 기다려주세요 🔍`,
      emotion: 'neutral',
      steps: [{ action: 'run_diagnostics', params: {}, confirmRequired: false, inlineView: 'health' }],
    }
  }
  /* 자동 정리 */
  if (/정리|청소|clean|임시\s*파일|캐시/.test(t)) {
    return {
      text: `불필요한 파일들을 정리할게요! 진행할까요? 🧹`,
      emotion: 'happy',
      steps: [{ action: 'auto_clean', params: {}, confirmRequired: true, inlineView: null }],
    }
  }
  /* 보안 */
  if (/보안|바이러스|악성|security|virus/.test(t)) {
    return {
      text: `보안 검사를 시작할게요 🛡️`,
      emotion: 'alert',
      steps: [{ action: 'security_scan', params: {}, confirmRequired: false, inlineView: null }],
    }
  }
  /* 집중 모드 */
  if (/집중|포모도로|방해\s*금지|focus/.test(t)) {
    return {
      text: `집중 모드를 시작할게요! 알림을 차단하고 타이머를 켤게요 🎯`,
      emotion: 'neutral',
      steps: [{ action: 'start_focus_mode', params: { duration: 25 }, confirmRequired: false, inlineView: null }],
    }
  }
  /* 파일 찾기 */
  if (/파일|찾아|검색|find|search/.test(t)) {
    const query = input.replace(/파일|찾아|줘|검색|해|주세요/g, '').trim()
    return {
      text: `"${query || '파일'}" 검색을 시작할게요 📂`,
      emotion: 'neutral',
      steps: [{ action: 'search_files', params: { query: query || input }, confirmRequired: false, inlineView: 'files' }],
    }
  }
  /* 뭘 할 수 있어 */
  if (/뭐|무엇|할\s*수\s*있|도움|help|기능/.test(t)) {
    return {
      text: `저 **${assistantName}**이 할 수 있는 것들이에요:\n\n🔍 PC 진단 및 수리\n🧹 임시파일 정리\n🛡️ 보안 검사\n📂 파일 검색\n🎯 집중 모드\n📊 시스템 모니터링\n🌐 웹 검색 대행\n⏰ 시간/날짜 안내\n\nAI 기능이 모두 활성화되어 있습니다 ✅`,
      emotion: 'happy',
      steps: [],
    }
  }
  /* 기본 — 모호한 입력은 Perplexity에 넘기도록 빈 응답 반환 (caller가 재시도) */
  return {
    text: `"${input}"에 대해 좀 더 구체적으로 말씀해주시겠어요? 무엇이 궁금하신가요? 😊`,
    emotion: 'neutral',
    steps: [],
    needs_clarify: true,
    clarify_question: `"${input}"에 대해 어떤 것이 궁금하신가요?`,
    clarify_intent: 'llm_clarify',
    clarify_params: { original_query: input },
  }
}

/* ─────────────────────────────────────────────────────────
   Perplexity API — OpenAI 호환 포맷
   주요 모델:
     llama-3.3-70b-versatile  → 최고 품질, 한국어 우수
     llama-3.1-8b-instant     → 초고속 응답 (0.2s)
     gemma2-9b-it             → 경량 한국어
     deepseek-r1-distill-llama-70b → 추론 특화
───────────────────────────────────────────────────────── */

interface ConversationTurn {
  role: 'user' | 'model' | 'assistant'
  parts?: Array<{ text: string }>
  content?: string
}

/** 대화 기록을 OpenAI 포맷으로 변환 — Claude 방식: 전체 히스토리 전달 */
function getPersonaSystemPrompt(): string {
  try {
    const personaId = localStorage.getItem('nexus-persona-id') ?? 'nexus'
    const personaPrompts: Record<string, string> = {
      nexus: '',
      expert: '당신은 전문가 수준의 Nexus입니다. 모든 답변을 전문가 관점에서 깊이 있게 분석하세요. 웹 검색 시 신뢰할 수 있는 학술·기술 자료를 우선 참고하고, 데이터와 근거를 반드시 포함하세요. 딥서치 시 최소 10개 이상의 소스를 분석하고 상충되는 견해도 함께 제시하세요. 전문 용어를 사용하되 핵심 개념은 명확히 설명하세요.',
      research: '당신은 리서치 전문 Nexus입니다. 데이터와 근거 중심으로 분석합니다. 시장 조사, 경쟁사 분석, 트렌드 파악에 특화되어 있습니다.',
      creative: '당신은 크리에이티브 전문 Nexus입니다. 창의적인 아이디어를 제시하고 콘텐츠 기획, 브레인스토밍을 도와줍니다.',
      finance: '당신은 재무 전문 Nexus입니다. 숫자와 재무 지표를 명확히 분석하고 예산 관리, 재무 보고서 작성을 도와줍니다.',
    }
    return personaPrompts[personaId] ?? ''
  } catch { return '' }
}

function historyToGroqMessages(history: ConversationTurn[], systemPrompt: string) {
  import('../../lib/nexus/memory').then(({ buildMemoryContext }) => {
    const ctx = buildMemoryContext()
    if (ctx) _memoryContext = ctx
  }).catch(() => {})

  const personaPrompt = getPersonaSystemPrompt()
  const baseSystem = personaPrompt ? `${systemPrompt}\n\n[페르소나 지침]\n${personaPrompt}` : systemPrompt
  const fullSystem = _memoryContext
    ? `${baseSystem}\n\n${_memoryContext}`
    : baseSystem

  const messages: Array<{ role: string; content: string }> = [
    { role: 'system', content: fullSystem },
  ]

  // 전체 히스토리를 전달하되 총 토큰 예산 초과 시 오래된 것부터 제거
  // (텍스트 길이 기준 ~60,000자 = 약 15,000 토큰)
  const TOKEN_BUDGET = 60000
  let totalLen = fullSystem.length
  const turns: Array<{ role: string; content: string }> = []

  for (let i = history.length - 1; i >= 0; i--) {
    const turn = history[i]
    const role = turn.role === 'model' ? 'assistant' : turn.role
    const content = turn.content ?? turn.parts?.map(p => p.text).join('') ?? ''
    if (!content.trim()) continue
    totalLen += content.length
    if (totalLen > TOKEN_BUDGET) break  // 예산 초과 시 더 오래된 건 제외
    turns.unshift({ role, content })
  }

  messages.push(...turns)
  return messages
}

let _memoryContext = ''
// 초기 로드 시 메모리 컨텍스트 세팅
import('../../lib/nexus/memory').then(({ buildMemoryContext }) => {
  _memoryContext = buildMemoryContext()
}).catch(() => {})

// Claude 방식: 별도 후속 감지 로직 없음.
// 전체 히스토리가 모델에 전달되므로 모델이 스스로 맥락 파악.
export function isFollowUpQuestion(_input: string, _history: ConversationTurn[]): boolean {
  return false // 더 이상 사용 안 함 — 히스토리 전달로 대체
}

export async function callGemini(
  apiKey: string,
  userInput: string,
  history: ConversationTurn[],
): Promise<GeminiResponse> {
  // 1순위: GPT-4o — 전체 히스토리 포함
  const openaiKey = OPENAI_API_KEY || localStorage.getItem('nexus-openai-key') || ''
  if (openaiKey) {
    try { return await callGPT4oWithTools(openaiKey, userInput, history) } catch { /* 폴백 */ }
  }
  // 2순위: Perplexity sonar-pro — 전체 히스토리 포함
  return callGroq('', userInput, history)
}

// ──────────────────────────────────────────────────────────────
// GPT-4o Tool Calling (웹검색 포함 동적 응답)
// ──────────────────────────────────────────────────────────────
const GPT4O_TOOLS = [
  {
    type: 'function',
    function: {
      name: 'web_search',
      description: '실시간 웹 검색. 맛집, 쇼핑, 뉴스, 가격비교, 유튜브, 틱톡, 최신 정보 등 실시간 데이터가 필요할 때 사용.',
      parameters: {
        type: 'object',
        properties: {
          query: { type: 'string', description: '검색할 키워드' },
          site: { type: 'string', enum: ['google', 'naver', 'coupang', 'youtube', 'tiktok', 'auto'], description: '검색 사이트' },
          max_items: { type: 'number', description: '결과 수 (기본 5)' },
        },
        required: ['query'],
      },
    },
  },
  {
    type: 'function',
    function: {
      name: 'get_weather',
      description: '특정 도시의 날씨 정보 조회. 도시명이 명시된 경우에만 호출할 것. 도시가 불명확하면 절대 호출하지 말고 사용자에게 먼저 물어볼 것.',
      parameters: {
        type: 'object',
        properties: {
          city: { type: 'string', description: '반드시 명시된 도시명 (예: 서울, 부산, 뉴욕)' },
        },
        required: ['city'],
      },
    },
  },
]

function parseNexusJSON(raw: string): GeminiResponse {
  const stripped = raw.replace(/^```(?:json)?\s*/i, '').replace(/\s*```\s*$/, '').trim()
  try {
    const parsed = JSON.parse(stripped) as {
      text?: string; emotion?: NexusEmotion; steps?: NexusStep[]
      needs_clarify?: boolean; clarify_question?: string
      clarify_intent?: string; clarify_params?: Record<string, unknown>
    }
    if (parsed && typeof parsed === 'object' && parsed.text) {
      return {
        text: parsed.text.trim(),
        emotion: parsed.emotion ?? 'neutral',
        steps: parsed.steps ?? [],
        needs_clarify: parsed.needs_clarify,
        clarify_question: parsed.clarify_question,
        clarify_intent: parsed.clarify_intent,
        clarify_params: parsed.clarify_params,
      }
    }
  } catch { /* 계속 */ }
  // JSON 안에 있는 { ... } 추출 시도
  const jsonBlock = stripped.match(/\{[\s\S]*\}/s)?.[0] ?? ''
  if (jsonBlock) {
    try {
      const parsed = JSON.parse(jsonBlock) as {
        text?: string; emotion?: NexusEmotion
        needs_clarify?: boolean; clarify_question?: string
        clarify_intent?: string; clarify_params?: Record<string, unknown>
      }
      if (parsed?.text) return {
        text: parsed.text.trim(), emotion: parsed.emotion ?? 'neutral', steps: [],
        needs_clarify: parsed.needs_clarify, clarify_question: parsed.clarify_question,
        clarify_intent: parsed.clarify_intent, clarify_params: parsed.clarify_params,
      }
    } catch { /* 계속 */ }
  }
  // 파싱 완전 실패 → 원문 그대로 (최대 400자)
  return { text: stripped.slice(0, 400) || '죄송합니다, 답변을 생성하지 못했습니다.', emotion: 'neutral', steps: [] }
}

async function callGPT4oWithTools(
  apiKey: string,
  userInput: string,
  history: ConversationTurn[],
): Promise<GeminiResponse> {
  const messages = historyToGroqMessages(history, NEXUS_SYSTEM_PROMPT)
  messages.push({ role: 'user', content: userInput })

  // 1차: GPT-4o에게 Tool 목록과 함께 요청
  let res: Response
  try {
    res = await fetch('https://api.openai.com/v1/chat/completions', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${apiKey}`,
      },
      body: JSON.stringify({
        model: 'gpt-4o',
        messages,
        tools: GPT4O_TOOLS,
        tool_choice: 'auto',
        temperature: 0.7,
        max_tokens: 1000,
      }),
    })
  } catch (e) {
    console.error('[GPT-4o] 네트워크 오류:', e)
    return callGroq('', userInput, history)
  }

  if (!res.ok) {
    console.error('[GPT-4o] API 오류:', res.status)
    return callGroq('', userInput, history)
  }

  const json = await res.json() as {
    choices?: Array<{
      message?: {
        content?: string
        tool_calls?: Array<{
          id: string
          function: { name: string; arguments: string }
        }>
      }
      finish_reason?: string
    }>
  }

  const choice = json.choices?.[0]
  const msg = choice?.message

  // Tool Calling 필요 없는 경우 → JSON 파싱 후 text 추출
  if (!msg?.tool_calls || msg.tool_calls.length === 0) {
    const raw = msg?.content ?? ''
    return parseNexusJSON(raw)
  }

  // 2차: Tool 실행 후 결과를 GPT-4o에 전달
  const toolMessages: Array<{ role: string; content: string; tool_call_id?: string; name?: string }> = [
    ...messages,
    { role: 'assistant', content: msg.content ?? '', ...({ tool_calls: msg.tool_calls } as Record<string, unknown>) },
  ]

  for (const tc of msg.tool_calls) {
    let toolResult = ''
    try {
      const args = JSON.parse(tc.function.arguments) as Record<string, unknown>
      if (tc.function.name === 'web_search') {
        toolResult = await executeWebSearch(args.query as string, args.site as string, (args.max_items as number) ?? 5)
      } else if (tc.function.name === 'get_weather') {
        toolResult = await executeWeather(args.city as string)
        if (toolResult.startsWith('__NEEDS_CLARIFY__:')) {
          const parts = toolResult.split(':')
          return { text: `주인님, ${parts[1]}`, emotion: 'neutral', steps: [], needs_clarify: true, clarify_question: parts[1], clarify_intent: parts[2] }
        }
      }
    } catch (e) {
      toolResult = `도구 실행 실패: ${String(e)}`
    }
    toolMessages.push({ role: 'tool', content: toolResult, tool_call_id: tc.id, name: tc.function.name })
  }

  // 3차: 검색 결과 포함해 최종 답변 생성
  let finalRes: Response
  try {
    finalRes = await fetch('https://api.openai.com/v1/chat/completions', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${apiKey}`,
      },
      body: JSON.stringify({
        model: 'gpt-4o',
        messages: toolMessages,
        temperature: 0.7,
        max_tokens: 1500,
      }),
    })
  } catch {
    return { text: '검색 결과를 정리하는 중 오류가 발생했습니다.', emotion: 'neutral', steps: [] }
  }

  const finalJson = await finalRes.json() as { choices?: Array<{ message?: { content?: string } }> }
  const finalRaw = finalJson.choices?.[0]?.message?.content ?? '답변을 생성하지 못했습니다.'
  const result = parseNexusJSON(finalRaw)
  // 검색 URL이 있으면 미리보기 제안 추가
  const previewItems = getLastPreviewItems()
  if (previewItems.length > 0 && !result.needs_clarify) {
    result.needs_preview = true
    result.preview_items = previewItems
  }
  return result
}

// 마지막 검색에서 수집된 URL 목록 — 미리보기 카드에서 사용
export interface PreviewItem { title: string; url: string }
let _lastPreviewItems: PreviewItem[] = []
export function getLastPreviewItems(): PreviewItem[] { return _lastPreviewItems }
export function clearLastPreviewItems(): void { _lastPreviewItems = [] }

async function executeWebSearch(query: string, site = 'auto', maxItems = 5): Promise<string> {
  _lastPreviewItems = []

  // 전문가 모드: 쿼리 강화 + 결과 수 증가
  const isExpert = (localStorage.getItem('nexus-persona-id') ?? 'nexus') === 'expert'
  if (isExpert) {
    maxItems = Math.max(maxItems, 10)
    query = `전문 분석 ${query} (학술·기술 자료 포함)`
  }

  // Go 백엔드가 연결된 경우 실제 크롤링 사용
  try {
    const res = await fetch('http://127.0.0.1:17891/api/browser/smart-agent', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ command: query, max_results: maxItems }),
      signal: AbortSignal.timeout(15000),
    })
    if (res.ok) {
      const data = await res.json() as { summary?: string; items?: Array<{ title?: string; url?: string }> }
      if (data.items) {
        _lastPreviewItems = data.items
          .filter(it => it.url)
          .map(it => ({ title: it.title ?? it.url!, url: it.url! }))
          .slice(0, 5)
      }
      if (data.summary) return `검색 결과:\n${data.summary}`
    }
  } catch { /* 폴백 */ }

  // Tavily API (웹 검색 전용) — 키가 있는 경우
  const tavilyKey = TAVILY_API_KEY || localStorage.getItem('nexus-tavily-key') || ''
  if (tavilyKey) {
    try {
      const res = await fetch('https://api.tavily.com/search', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ api_key: tavilyKey, query, max_results: maxItems, search_depth: isExpert ? 'advanced' : 'basic', include_domains: isExpert ? ['scholar.google.com', 'arxiv.org', 'pubmed.ncbi.nlm.nih.gov'] : [] }),
      })
      if (res.ok) {
        const data = await res.json() as { results?: Array<{ title: string; url: string; content: string }> }
        if (data.results) {
          _lastPreviewItems = data.results
            .slice(0, 5)
            .map(r => ({ title: r.title, url: r.url }))
        }
        return (data.results ?? []).map(r => `• ${r.title}\n  ${r.url}\n  ${r.content.slice(0, 200)}`).join('\n\n')
      }
    } catch { /* 폴백 */ }
  }

  return `"${query}" 검색 결과를 가져오지 못했습니다. (백엔드 연결 또는 Tavily API 키 필요)`
}

async function executeWeather(city: string): Promise<string> {
  if (!city || city.trim() === '') {
    return '__NEEDS_CLARIFY__:어느 지역 날씨를 알려드릴까요?:get_weather'
  }
  // 1차: 백엔드 날씨 API
  try {
    const res = await fetch(`http://127.0.0.1:17891/api/weather?city=${encodeURIComponent(city)}`, {
      signal: AbortSignal.timeout(8000),
    })
    if (res.ok) {
      const data = await res.json() as {
        success?: boolean; summary?: string; message?: string
        temp_c?: number; condition?: string; feels_like?: number; humidity?: number
        forecast?: Array<{ date: string; max: number; min: number; condition: string }>
      }
      // success=true인 경우 summary 또는 message 사용
      if (data.success !== false) {
        if (data.summary) return data.summary
        if (data.message) return data.message
        if (data.temp_c !== undefined) {
          let text = `${city} 현재 ${data.temp_c}°C (체감 ${data.feels_like ?? data.temp_c}°C), ${data.condition ?? ''}, 습도 ${data.humidity ?? 0}%.`
          if (data.forecast && data.forecast.length > 1) {
            const tmr = data.forecast[1]
            text += ` 내일 최고 ${tmr.max}°C / 최저 ${tmr.min}°C.`
          }
          return text
        }
      }
    }
  } catch { /* 폴백으로 진행 */ }

  // 2차: Perplexity 실시간 검색 폴백
  try {
    const pplxKey = PPLX_API_KEY
    const res = await fetch('https://api.perplexity.ai/chat/completions', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', 'Authorization': `Bearer ${pplxKey}` },
      body: JSON.stringify({
        model: 'sonar',
        messages: [
          { role: 'system', content: '날씨 정보를 간결하게 한국어로 2~3문장으로 답해줘. 현재 기온, 날씨 상태, 내일 예보를 포함해.' },
          { role: 'user', content: `${city} 오늘 날씨 알려줘` },
        ],
        max_tokens: 200,
        search_recency_filter: 'day',
      }),
      signal: AbortSignal.timeout(12000),
    })
    if (res.ok) {
      const data = await res.json() as { choices?: Array<{ message?: { content?: string } }> }
      const text = data.choices?.[0]?.message?.content?.trim()
      if (text) return text
    }
  } catch { /* 폴백 */ }

  return `${city} 날씨 정보를 현재 가져올 수 없습니다. 잠시 후 다시 시도해주세요.`
}

/**
 * Perplexity API 호출 (OpenAI 호환)
 * API 키: localStorage 'nexus-pplx-key' 또는 파라미터
 */
export async function callGroq(
  apiKey: string,
  userInput: string,
  history: ConversationTurn[],
  model = GROQ_MODEL,
): Promise<GeminiResponse> {
  const key = PPLX_API_KEY
  if (!key) return fallbackResponse(userInput)

  // PC 상태 컨텍스트 주입 (Tauri 환경에서만)
  let statsContext = ''
  try {
    const stats = await invoke<Record<string, number>>('get_system_stats')
    if (stats && Object.keys(stats).length > 0) {
      statsContext = `[PC 현황: CPU ${stats.CPUPercent ?? 0}%, RAM ${stats.RAMPercent ?? 0}%, 온도 ${stats.CPUTemp ?? 0}°C, 디스크 ${stats.DiskPercent ?? 0}%]\n`
    }
  } catch { /* 비-Tauri 환경에서는 무시 */ }

  // Custom Instructions 주입 (사용자가 설정에서 저장한 스타일)
  const customInstructions = localStorage.getItem('nexus-custom-instructions') || ''
  const systemPrompt = customInstructions
    ? `${NEXUS_SYSTEM_PROMPT}\n\n[사용자 커스텀 지시사항 — 반드시 준수]\n${customInstructions}`
    : NEXUS_SYSTEM_PROMPT

  const messages = historyToGroqMessages(history, systemPrompt)
  messages.push({ role: 'user', content: statsContext + userInput })

  const url = `${GROQ_API_BASE}/chat/completions`
  const body = {
    model,
    messages,
    temperature: 0.6,
    max_tokens: 1000,
    // Perplexity sonar 모델은 response_format 미지원 → 시스템 프롬프트로 JSON 강제
  }

  let res: Response
  try {
    res = await fetch(url, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${key}`,
      },
      body: JSON.stringify(body),
    })
  } catch (e) {
    console.error('[Perplexity] 네트워크 오류:', e)
    return fallbackResponse(userInput)
  }

  if (!res.ok) {
    const errText = await res.text()
    console.error(`[Perplexity] API 오류 ${res.status}:`, errText)

    // 429 Rate limit → 빠른 모델로 재시도
    if (res.status === 429 && model === PPLX_MODEL) {
      console.warn('[Perplexity] Rate limit → sonar 으로 재시도')
      return callGroq(apiKey, userInput, history, PPLX_MODEL_FAST)
    }
    return fallbackResponse(userInput)
  }

  const json = await res.json() as {
    choices?: Array<{
      message?: { content?: string; tool_calls?: Array<{ function: { name: string; arguments: string } }> }
      finish_reason?: string
    }>
    error?: { message: string }
  }

  if (json.error) {
    console.error('[Perplexity] 응답 오류:', json.error.message)
    return fallbackResponse(userInput)
  }

  const msg = json.choices?.[0]?.message
  const rawContent = msg?.content ?? ''

  let text = '네, 알겠어요!'
  let emotion: NexusEmotion = 'neutral'
  let steps: NexusStep[] = []

  // JSON 파싱 (system prompt에서 JSON 응답 강제)
  if (rawContent) {
    // 1차: 마크다운 코드블록 제거 후 전체 파싱
    const stripped = rawContent.replace(/^```(?:json)?\s*/i, '').replace(/\s*```\s*$/, '').trim()
    let parseSuccess = false
    try {
      const parsed = JSON.parse(stripped) as {
        text?: string
        emotion?: NexusEmotion
        steps?: Array<{
          action: string
          params: Record<string, unknown>
          confirmRequired: boolean
          inlineView?: string | null
        }>
      }
      if (parsed && typeof parsed === 'object') {
        text = (parsed.text && parsed.text.trim()) ? parsed.text : text
        emotion = parsed.emotion ?? emotion
        steps = (parsed.steps ?? []).map(s => ({
          action: s.action,
          params: s.params ?? {},
          confirmRequired: s.confirmRequired ?? false,
          inlineView: s.inlineView ?? getInlineViewType(s.action),
        }))
        parseSuccess = true
      }
    } catch { /* 계속 진행 */ }

    if (!parseSuccess) {
      // 2차: rawContent 안에서 JSON 객체 추출 시도 ({ ... })
      const jsonMatch = rawContent.match(/\{[\s\S]*?\}(?=\s*$|\s*```)/s)
      if (jsonMatch) {
        try {
          const parsed = JSON.parse(jsonMatch[0]) as { text?: string; emotion?: NexusEmotion }
          text = (parsed.text && parsed.text.trim()) ? parsed.text : text
          emotion = parsed.emotion ?? emotion
          parseSuccess = true
        } catch { /* 계속 */ }
      }
    }

    if (!parseSuccess) {
      // 3차: JSON 파싱 완전 실패 → 원문 텍스트 그대로 표시 (Perplexity가 평문으로 답한 경우)
      const cleanText = rawContent
        .replace(/^```(?:json)?\s*/i, '')
        .replace(/\s*```\s*$/, '')
        .replace(/^\s*\{[\s\S]*\}\s*$/, '') // JSON 잔재 제거
        .trim()
      text = cleanText.length > 0 ? cleanText.slice(0, 600) : '죄송합니다, 응답 처리 중 오류가 발생했습니다.'
    }
  }

  // Tool calls 처리 (Groq tool_use 활성화 시)
  if (msg?.tool_calls && msg.tool_calls.length > 0 && steps.length === 0) {
    steps = msg.tool_calls.map(tc => {
      let args: Record<string, unknown> = {}
      try { args = JSON.parse(tc.function.arguments) } catch { /* ignore */ }
      return {
        action: tc.function.name,
        params: args,
        confirmRequired: isDestructiveAction(tc.function.name),
        inlineView: getInlineViewType(tc.function.name),
      }
    })
  }

  // Self-Reflection: 답변이 짧거나 링크만 나열된 경우 보완 요청
  if (text && text.length < 80 && steps.length === 0) {
    try {
      const reflectRes = await fetch(`${GROQ_API_BASE}/chat/completions`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', 'Authorization': `Bearer ${key}` },
        body: JSON.stringify({
          model: PPLX_MODEL_FAST,
          messages: [
            { role: 'system', content: '당신은 AI 답변 품질 검증자입니다. 아래 답변이 사용자 요청을 충분히 충족하는지 확인하고, 부족하면 보완된 답변을 JSON {"text":"...","emotion":"..."} 형태로 출력하라. 충분하면 {"ok":true}만 출력.' },
            { role: 'user', content: `사용자 요청: "${userInput}"\n\n현재 답변: "${text}"\n\n이 답변이 충분한가? 부족하면 더 구체적이고 완전한 답변으로 보완해줘.` },
          ],
          max_tokens: 400,
          temperature: 0.3,
        }),
        signal: AbortSignal.timeout(5000),
      })
      if (reflectRes.ok) {
        const rJson = await reflectRes.json() as { choices?: Array<{ message?: { content?: string } }> }
        const rContent = rJson.choices?.[0]?.message?.content?.trim() || ''
        const rMatch = rContent.match(/\{[\s\S]*\}/)
        if (rMatch) {
          const rParsed = JSON.parse(rMatch[0]) as { ok?: boolean; text?: string; emotion?: NexusEmotion }
          if (!rParsed.ok && rParsed.text && rParsed.text.length > text.length) {
            text = rParsed.text
            emotion = rParsed.emotion ?? emotion
          }
        }
      }
    } catch { /* Self-Reflection 실패 시 원래 답변 유지 */ }
  }

  return { text, emotion, steps }
}

// ──────────────────────────────────────────────────────────────
// Go 백엔드 LLM/Browser API 키 동기화
// ──────────────────────────────────────────────────────────────

/**
 * localStorage의 API 키를 Go 백엔드에 동기화
 * 앱 시작 시 / 키 변경 시 호출
 */
export async function syncAPIKeysToBackend(): Promise<void> {
  const pplxKey = PPLX_API_KEY
  const claudeKey = localStorage.getItem('nexus-claude-key') ?? ''
  const tavilyKey = TAVILY_API_KEY || localStorage.getItem('nexus-tavily-key') || ''
  if (!claudeKey && !tavilyKey) return
  try {
    const { backendAPI } = await import('./backendAPI')
    await backendAPI.llmConfigSet(pplxKey || undefined, claudeKey || undefined, tavilyKey || undefined)
  } catch {
    // 백엔드 미실행 시 무시
  }
}

// ──────────────────────────────────────────────────────────────
// NexusStep 실행기 — 새 액션 처리
// ──────────────────────────────────────────────────────────────

/**
 * browser_agent / ai_analyze_screen / ai_summarize_document 등
 * 새로 추가된 액션을 실행하고 결과를 반환한다.
 * 기존 executeStep 함수가 없을 경우 이 함수가 진입점 역할.
 */
export async function executeNewAction(
  action: string,
  params: Record<string, unknown>,
): Promise<{ success: boolean; data?: unknown; message: string }> {
  const { backendAPI } = await import('./backendAPI')

  try {
    switch (action) {
      // ── Browser Agent ──────────────────────────────────────
      case 'browser_agent': {
        const command = (params.command as string) ?? ''
        const maxSteps = (params.maxSteps as number) ?? 8
        const result = await backendAPI.browserAgent(command, maxSteps)
        return {
          success: result.success,
          data: result,
          message: result.summary || result.goal || '브라우저 작업 완료',
        }
      }

      case 'browser_navigate': {
        const url = (params.url as string) ?? ''
        const result = await backendAPI.browserNavigate(url)
        return { success: result.success, data: result, message: result.message }
      }

      case 'browser_extract_page': {
        const selector = (params.selector as string) ?? undefined
        const mode = (params.mode as 'text' | 'table' | 'links') ?? 'text'
        const result = await backendAPI.browserExtract(selector, mode)
        return { success: result.success, data: result.content, message: `${result.length}자 추출 완료` }
      }

      // ── AI Vision (Go 백엔드 직접) ─────────────────────────
      case 'ai_analyze_screen': {
        const question = (params.question as string) ?? '이 화면을 분석해주세요'
        const result = await backendAPI.llmVision(question)
        return { success: result.success, data: result.answer, message: result.answer }
      }

      // ── AI 문서 ────────────────────────────────────────────
      case 'ai_summarize_document': {
        const filePath = (params.filePath as string) ?? ''
        const question = (params.question as string) ?? undefined
        const result = await backendAPI.llmDocSummary(filePath, question)
        return { success: result.success, data: result.summary, message: result.summary }
      }

      case 'ai_compare_documents': {
        const fileA = (params.fileA as string) ?? ''
        const fileB = (params.fileB as string) ?? ''
        const focus = (params.focus as 'numbers' | 'changes' | 'both') ?? 'both'
        const result = await backendAPI.llmDocCompare(fileA, fileB, focus)
        return {
          success: result.success,
          data: result.result,
          message: result.result?.summary ?? '문서 비교 완료',
        }
      }

      case 'ai_deep_search': {
        const query = (params.query as string) ?? ''
        const folder = (params.folder as string) ?? undefined
        const isExpertDeep = (localStorage.getItem('nexus-persona-id') ?? 'nexus') === 'expert'
        const maxResults = isExpertDeep ? Math.max((params.maxResults as number) ?? 15, 10) : (params.maxResults as number) ?? 15
        const result = await backendAPI.llmDeepSearch(query, folder, maxResults)
        return {
          success: result.success,
          data: result.results,
          message: `${result.total}개 파일 발견 (AI 키워드: ${result.keywords_used?.join(', ')})`,
        }
      }

      // ── ★ 핵심: 검색 → PDF 자동 생성 ──────────────────────
      case 'search_and_pdf': {
        const query = (params.query as string) ?? ''
        const maxItems = (params.max_items as number) ?? 5

        // 1단계: 백엔드에서 검색 결과 수집
        const { searchAndPDF } = await import('./backendAPI')
        const result = await searchAndPDF(query, maxItems, '', false)

        // 2단계: jspdf로 브라우저에서 직접 PDF 생성 (Mac/Windows 공통)
        const content = result.summary || result.items?.map((it: Record<string,string>) =>
          `${it.title ?? ''}\n${it.url ?? ''}\n${it.content ?? it.text ?? ''}`
        ).join('\n\n') || `${query} 검색 결과`

        try {
          const { jsPDF } = await import('jspdf')
          const doc = new jsPDF({ orientation: 'portrait', unit: 'mm', format: 'a4' })
          const margin = 15
          const pageW = doc.internal.pageSize.getWidth() - margin * 2
          let y = margin

          // 제목
          doc.setFontSize(16)
          doc.setFont('helvetica', 'bold')
          const title = `${query} — 검색 결과 보고서`
          doc.text(title, margin, y)
          y += 10

          // 날짜
          doc.setFontSize(9)
          doc.setFont('helvetica', 'normal')
          doc.setTextColor(120)
          doc.text(new Date().toLocaleString('ko-KR'), margin, y)
          doc.setTextColor(0)
          y += 8

          // 구분선
          doc.setDrawColor(180)
          doc.line(margin, y, margin + pageW, y)
          y += 6

          // 본문 (줄바꿈 자동 처리)
          doc.setFontSize(10)
          const lines = doc.splitTextToSize(content, pageW)
          for (const line of lines) {
            if (y > 275) { doc.addPage(); y = margin }
            doc.text(line, margin, y)
            y += 5
          }

          const fileName = `${query.replace(/\s+/g, '_')}_report.pdf`
          doc.save(fileName)

          return {
            success: true,
            data: result,
            message: `✅ PDF 다운로드 완료!\n📄 ${fileName}\n\n💡 ${content.slice(0, 200)}...`,
          }
        } catch (pdfErr) {
          console.error('[PDF] 생성 오류:', pdfErr)
          // PDF 실패 시 텍스트로 폴백
          return {
            success: true,
            data: result,
            message: `📋 검색 결과 (PDF 생성 실패, 텍스트로 제공):\n\n${content.slice(0, 800)}`,
          }
        }
      }

      // ── 고급 Browser Agent (Stealth) ───────────────────────
      case 'browser_smart_agent': {
        const command = (params.command as string) ?? ''
        const maxResults = (params.maxResults as number) ?? 10
        const saveExcel = (params.saveExcel as boolean) ?? true
        const sessionKey = (params.sessionKey as string) ?? ''
        const { browserSmartAgent } = await import('./backendAPI')
        const result = await browserSmartAgent(command, maxResults, saveExcel, sessionKey)
        const msg = result.summary || (result.blocked ? `봇 차단: ${result.block_reason}` : `완료 (${result.steps.length}단계)`)
        return {
          success: result.success,
          data: result,
          message: msg + (result.excel_path ? `\n📊 Excel: ${result.excel_path}` : ''),
        }
      }

      case 'browser_collect_price': {
        const productQuery = (params.productQuery as string) ?? ''
        const sites = (params.sites as string[]) ?? undefined
        const maxPerSite = (params.maxPerSite as number) ?? 5
        const saveExcel = (params.saveExcel as boolean) ?? true
        const { browserCollectPrice } = await import('./backendAPI')
        const result = await browserCollectPrice(productQuery, sites, maxPerSite, saveExcel)
        return {
          success: result.success,
          data: result,
          message: result.summary || `${result.total}개 결과 수집` + (result.excel_path ? `\n📊 ${result.excel_path}` : ''),
        }
      }

      case 'browser_news_collect': {
        const query = (params.query as string) ?? ''
        const site = (params.site as string) ?? 'naver.com'
        const maxItems = (params.maxItems as number) ?? 10
        const { browserNewsCollect } = await import('./backendAPI')
        const result = await browserNewsCollect(query, site, maxItems)
        return {
          success: result.success,
          data: result,
          message: result.summary || `${result.total}개 기사 수집`,
        }
      }

      case 'browser_login_session': {
        const { url, username, password, sessionKey } = params as { url: string; username: string; password: string; sessionKey: string }
        const { browserLoginSession } = await import('./backendAPI')
        const result = await browserLoginSession(url, username, password, sessionKey)
        return { success: result.success, data: result, message: result.message }
      }

      // ── 자연어 스케줄러 ────────────────────────────────────
      case 'schedule_task': {
        const command = (params.command as string) ?? ''
        const useWindows = (params.useWindows as boolean) ?? false
        const { schedulerAdd } = await import('./backendAPI')
        const result = await schedulerAdd(command, useWindows)
        return {
          success: result.success,
          data: result.task,
          message: result.message || `스케줄 등록: ${result.next_run_kr}`,
        }
      }

      case 'list_schedules': {
        const { schedulerList } = await import('./backendAPI')
        const result = await schedulerList()
        const taskSummary = result.tasks
          .slice(0, 5)
          .map(t => `• ${t.name} → ${new Date(t.next_run).toLocaleString('ko-KR')}`)
          .join('\n')
        return {
          success: result.success,
          data: result.tasks,
          message: result.total === 0 ? '등록된 스케줄 없음' : `${result.total}개 스케줄:\n${taskSummary}`,
        }
      }

      case 'delete_schedule': {
        const id = (params.id as string) ?? ''
        const { schedulerDelete } = await import('./backendAPI')
        const result = await schedulerDelete(id)
        return { success: result.success, data: null, message: result.message }
      }

      // ── 에이전트 메모리 ────────────────────────────────────
      case 'memory_search': {
        const keyword = (params.keyword as string) ?? ''
        const type = (params.type as string) ?? undefined
        const { memorySearch } = await import('./backendAPI')
        const result = await memorySearch(keyword, type)
        return {
          success: result.success,
          data: result.entries,
          message: result.summary || `${result.total}개 기록 발견`,
        }
      }

      // ── Excel 내보내기 ─────────────────────────────────────
      case 'excel_save': {
        const data = (params.data as string[][]) ?? []
        const title = (params.title as string) ?? '데이터'
        const filename = (params.filename as string) ?? undefined
        const { excelSave } = await import('./backendAPI')
        const result = await excelSave(data, title, filename)
        return { success: result.success, data: result.path, message: result.message }
      }

      default:
        return { success: false, message: `알 수 없는 액션: ${action}` }
    }
  } catch (err) {
    return {
      success: false,
      message: `실행 실패: ${err instanceof Error ? err.message : String(err)}`,
    }
  }
}

export async function callGeminiWithImage(
  base64Image: string,
  question: string,
): Promise<string> {
  return callGroqVision(base64Image, question)
}

export async function callGroqVision(
  base64Image: string,
  question: string,
): Promise<string> {
  const key = OPENAI_API_KEY || localStorage.getItem('nexus-openai-key') || ''
  if (!key) return '⚠️ Vision 기능을 사용하려면 OpenAI API 키가 필요합니다.'
  try {
    const res = await fetch('https://api.openai.com/v1/chat/completions', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', 'Authorization': `Bearer ${key}` },
      body: JSON.stringify({
        model: 'gpt-4o',
        messages: [{
          role: 'user',
          content: [
            { type: 'text', text: question || '이 화면을 분석해서 한국어로 설명해줘.' },
            { type: 'image_url', image_url: { url: `data:image/png;base64,${base64Image}`, detail: 'high' } },
          ],
        }],
        max_tokens: 1000,
      }),
      signal: AbortSignal.timeout(30000),
    })
    if (res.ok) {
      const data = await res.json() as { choices?: Array<{ message?: { content?: string } }> }
      return data.choices?.[0]?.message?.content?.trim() ?? '화면 분석 결과를 가져오지 못했습니다.'
    }
    return `화면 분석 실패 (${res.status})`
  } catch (e) {
    return `화면 분석 오류: ${e instanceof Error ? e.message : String(e)}`
  }
}
