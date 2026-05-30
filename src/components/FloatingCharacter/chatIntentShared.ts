import type { Intent } from '../../lib/nexus/intentDetector'
import type { NexusEmotion } from '../../types/nexus'
import { BackendError, type BackendErrorCode } from '../../lib/nexus/backendAPI'
import { intentLabel as registryLabel } from '../../lib/nexus/intentRegistry'
import type { InlineCardData } from './InlineCards'

export type CharacterEmotion = NexusEmotion

export function t(ko: string, en: string, lang: 'ko' | 'en'): string {
  return lang === 'en' ? en : ko
}

/* ─────────────────────────────────────────────────────────── */
/* 에러 → 사용자에게 보이는 카드/메시지 변환 헬퍼              */
/* (mock 폴백 금지 — 실패는 무조건 사용자에게 정확히 표시)     */
/* ─────────────────────────────────────────────────────────── */

const HINTS_KO: Partial<Record<BackendErrorCode, string>> = {
  no_backend:      '백엔드(nexus-backend.exe)가 실행 중인지 확인해주세요. 앱을 재시작하면 자동으로 다시 켜집니다.',
  no_api_key:      '설정 → API 키에서 해당 서비스 키를 등록해주세요.',
  windows_only:    'Windows에서만 사용할 수 있는 기능이에요. 정식 빌드(.exe)에서 동작합니다.',
  rate_limited:    '잠시 후 다시 시도하거나 Pro로 업그레이드해주세요.',
  timeout:         '요청이 너무 오래 걸려요. 네트워크/PC 상태를 확인해주세요.',
  not_implemented: '아직 이 기능은 라우터/백엔드에 연결돼 있지 않아요. 곧 지원될 예정입니다.',
  server_error:    '백엔드 내부 오류예요. 잠시 후 다시 시도해주세요.',
  forbidden:       '권한이 필요한 기능이에요. 관리자 권한으로 실행해주세요.',
  bad_request:     '요청 형식이 잘못됐어요. 입력을 다시 확인해주세요.',
}
const HINTS_EN: Partial<Record<BackendErrorCode, string>> = {
  no_backend:      'Make sure nexus-backend.exe is running. Restarting the app should relaunch it.',
  no_api_key:      'Add the missing API key in Settings → API Keys.',
  windows_only:    'This feature is Windows-only. Use the installed .exe build.',
  rate_limited:    'Try again later or upgrade to Pro.',
  timeout:         'Request took too long. Check network/PC.',
  not_implemented: 'This feature is not wired yet — coming soon.',
  server_error:    'Backend internal error. Try again shortly.',
  forbidden:       'Requires elevated permissions.',
  bad_request:     'Invalid request. Check your input.',
}

/**
 * @deprecated chatIntentShared.intentLabel은 호환용. 새 코드는 intentRegistry.intentLabel 사용.
 */
export function intentLabel(intent: Intent | string, lang: 'ko' | 'en'): string {
  return registryLabel(intent, lang)
}

export interface IntentResult {
  text: string
  card?: InlineCardData
  emotion: CharacterEmotion
}

/**
 * 백엔드/실행 실패를 통일된 ErrorCard 응답으로 변환.
 * mock 데이터 폴백 금지의 핵심 헬퍼.
 */
export function errorReturn(intent: Intent | string, err: unknown, lang: 'ko' | 'en' = 'ko'): IntentResult {
  const isBe = err instanceof BackendError
  const code: BackendErrorCode | 'not_implemented' = isBe ? err.code : 'unknown'
  const path = isBe ? err.path : undefined
  const detail = isBe ? err.detail : (err instanceof Error ? err.message : String(err))

  const label = intentLabel(intent, lang)
  const titleKo = `${label} 실행에 실패했어요`
  const titleEn = `${label} failed`
  const title = lang === 'en' ? titleEn : titleKo

  const hint = (lang === 'en' ? HINTS_EN : HINTS_KO)[code as BackendErrorCode]

  const textKo = code === 'not_implemented' || code === 'windows_only'
    ? `${label}는 ${code === 'windows_only' ? 'Windows 정식 빌드' : '곧 지원될'} 기능이에요.`
    : `${label}를 처리하지 못했어요. 아래에서 원인을 확인해주세요.`
  const textEn = code === 'not_implemented' || code === 'windows_only'
    ? `${label} is ${code === 'windows_only' ? 'Windows-only' : 'coming soon'}.`
    : `Couldn't complete "${label}". See details below.`

  const emotion: CharacterEmotion = (code === 'no_backend' || code === 'server_error') ? 'alert' : 'concerned'

  return {
    text: lang === 'en' ? textEn : textKo,
    card: {
      type: 'error',
      intent: String(intent),
      code,
      title,
      detail,
      hint,
      path,
    },
    emotion,
  }
}

export function buildAgentSteps(intent: Intent, lang: 'ko' | 'en' = 'ko'): string[] {
  if (lang === 'en') {
    switch (intent) {
      case 'pc_status': return ['Fetching PC status...', 'Collecting CPU/Memory/Disk', 'Building status card']
      case 'security_scan': return ['Starting security scan...', 'Checking remote access', 'Scanning suspicious processes', 'Analyzing results']
      case 'full_scan': return ['Starting full diagnostic...', 'Searching system issues', 'Classifying severity', 'Generating report']
      case 'clean': return ['Identifying targets...', 'Checking temp files & cache', 'Running safe cleanup']
      case 'daily_report': return ['Collecting system data...', 'Analyzing usage', 'Building report']
      case 'repair': return ['Diagnosing issues...', 'Applying fixes', 'Verifying results']
      case 'file_search': return ['Starting file search...', 'Collecting results']
      case 'file_organize': return ['Scanning files...', 'Categorizing', 'Moving to folders']
      case 'vision_screen': return ['Capturing screen...', 'Sending to AI', 'Generating answer']
      case 'doc_compare': return ['Opening files...', 'Extracting text', 'Running diff', 'Summarizing differences']
      case 'deep_search': return ['Collecting files...', 'Indexing content', 'Ranking results']
      case 'news_search': return ['Searching news...', 'Collecting articles']
      case 'youtube_search': return ['Searching videos...', 'Collecting results']
      case 'reddit_search': return ['Launching stealth browser...', 'Crawling Reddit...', 'Collecting posts']
      case 'price_compare': return ['Searching...', 'Checking Coupang', 'Checking Naver', 'Comparing prices']
      case 'weather': return ['Fetching weather data...', 'Analyzing forecast']
      case 'calendar_today': return ['Connecting to Outlook...', 'Loading today\'s events']
      case 'email_inbox': return ['Connecting to Outlook...', 'Fetching inbox']
      case 'workflow_run': return ['Generating workflow plan...', 'Executing steps...', 'Compiling results']
      case 'parallel_queries': return ['Splitting queries...', 'Dispatching in parallel', 'Gathering results']
      case 'multi_agent': return ['Preparing agents...', 'Deploying team', 'Running in parallel']
      case 'briefing_now': return ['Checking weather...', 'Fetching calendar...', 'Checking email...', 'Generating briefing']
      case 'open_folder': return ['Identifying folder...', 'Checking path', 'Opening Explorer']
      case 'remote_access': return ['Scanning remote tools...', 'Checking RDP port', 'Matching processes']
      case 'process_security': return ['Collecting processes...', 'Checking risk patterns', 'Scanning ports']
      case 'startup_items': return ['Loading startup items...', 'Analyzing suspicious keywords']
      case 'defender_status': return ['Checking Windows Defender...']
      case 'account_check': return ['Listing local accounts...', 'Analyzing anomalies']
      case 'volume_control': return ['Adjusting volume...']
      case 'brightness': return ['Adjusting brightness...']
      case 'wifi_toggle': return ['Checking Wi-Fi...', 'Changing setting']
      case 'power_action': return ['Executing power command...']
      case 'launch_app': return ['Finding app path...', 'Launching']
      case 'process_top': return ['Collecting processes...', 'Sorting by CPU/Memory']
      case 'driver_check': return ['Querying drivers...', 'Filtering issues']
      case 'network_analysis': return ['Checking network adapters...', 'Looking up DNS/IP', 'Measuring ping']
      case 'programs_list': return ['Listing installed programs...']
      case 'boot_analysis': return ['Analyzing boot event log...', 'Counting startup items']
      case 'file_duplicates': return ['Collecting files...', 'Finding duplicates']
      case 'browser_clean': return ['Finding browser cache...', 'Cleaning data']
      case 'registry_clean': return ['Scanning registry...', 'Removing invalid entries']
      case 'restore_create': return ['Creating restore point...']
      case 'focus_mode': return ['Configuring focus mode...']
      case 'notes': return ['Loading notes...']
      case 'doc_find': return ['Starting file scan...', 'Matching name/content', 'Sorting results']
      case 'vision_ocr': return ['Checking clipboard image...', 'Running Windows OCR', 'Extracting text']
      case 'smart_organize': return ['Collecting files...', 'Classifying types', 'Moving to folders', 'Done']
      case 'journal_today': return ['Collecting recent file history...', 'Analyzing app usage', 'Estimating work hours', 'Generating journal']
      case 'journal_generate': return ['Collecting journal data...', 'Generating format', 'Saving file']
      case 'journal_history': return ['Loading past journals...']
      case 'macro_list': return ['Loading macro list...']
      case 'macro_create': return ['Parsing command...', 'Building actions', 'Registering schedule']
      case 'macro_run': return ['Running macro...', 'Processing actions', 'Verifying completion']
      case 'pc_report': return ['Collecting system state...', 'Running security check', 'Generating report', 'Saving HTML']
      case 'report_email': return ['Generating report...', 'Connecting SMTP', 'Sending email']
      case 'doc_summary': return ['Opening file...', 'Extracting text', 'Analyzing key points', 'Generating summary']
      case 'calendar_week': return ['Connecting to Outlook...', 'Loading this week\'s schedule']
      case 'calendar_add': return ['Creating event...', 'Saving to Outlook']
      case 'email_send': return ['Composing email...', 'Sending via SMTP']
      case 'email_summarize': return ['Fetching inbox...', 'Generating AI summary']
      case 'virus_check': return ['Computing file hash...', 'Querying VirusTotal', 'Analyzing results']
      case 'perf_history': return ['Loading performance history...', 'Analyzing trends']
      case 'perf_anomaly': return ['Analyzing history data...', 'Detecting anomalies']
      case 'multi_action': return ['Starting multi-action...', 'Searching', 'Compiling results', 'Saving file']
      case 'schedule_list': return ['Loading schedule list...']
      case 'schedule_add': return ['Parsing command...', 'Registering schedule']
      case 'schedule_delete': return ['Deleting schedule...']
      case 'process_kill': return ['Finding process...', 'Force terminating']
      case 'app_permissions': return ['Checking registry...', 'Collecting permissions']
      case 'windows_updates': return ['Connecting to Windows Update...', 'Checking update list']
      case 'gpu_stats': return ['Collecting GPU info...', 'Checking nvidia-smi']
      case 'recall_search': return ['Searching screen memory...', 'Sorting matches']
      case 'recall_capture': return ['Capturing screen...', 'Extracting OCR text', 'Saving memory']
      case 'meeting_start': return ['Checking microphone...', 'Starting recording']
      case 'meeting_stop': return ['Stopping recording...', 'Saving file']
      case 'meeting_summary': return ['Checking recording file...', 'Transcribing with Whisper...', 'Generating AI summary']
      case 'meeting_list': return ['Loading meeting list...']
      case 'dictation_start': return ['Analyzing text...', 'Typing into current app']
      case 'travel_time': return ['Looking up coordinates...', 'Calculating route']
      case 'translate': return ['Checking clipboard...', 'Translating...']
      case 'clipboard_ai': return ['Getting clipboard content...', 'Processing with AI']
      case 'clipboard_history': return ['Loading clipboard history...']
      case 'voice_todo': return ['Analyzing content...', 'Saving note', 'Registering to calendar']
      case 'persona_list': return ['Loading persona list...']
      case 'persona_switch': return ['Switching persona...']
      case 'brain_search': return ['🧠 Searching Second Brain...', 'Analyzing related memories']
      case 'brain_stats': return ['Loading index stats...']
      case 'workflow_plan': return ['⚡ Generating workflow plan...']
      case 'caption_start': return ['🎬 Initializing audio capture...', 'Starting live captions']
      case 'caption_stop': return ['Stopping captions...']
      case 'video_download': return ['Checking video URL...', 'Downloading with yt-dlp...', 'Saving file']
      case 'video_transcript': return ['Checking video URL...', 'Extracting subtitles...', 'Generating AI summary']
      case 'email_classify': return ['Fetching inbox...', 'Classifying with AI...', 'Sorting by priority']
      case 'email_draft': return ['Analyzing email...', 'Drafting reply with AI']
      case 'calendar_find_slot': return ['Checking calendar...', 'Finding free slots', 'Listing options']
      case 'calendar_smart_add': return ['Parsing natural language...', 'Creating event', 'Saving to Outlook']
      case 'workflow_list': return ['Loading saved workflows...']
      case 'workflow_create': return ['Parsing natural language...', 'Building workflow', 'Saving']
      case 'workflow_templates': return ['Loading workflow templates...']
      case 'imap_inbox': return ['Connecting to IMAP server...', 'Fetching inbox']
      case 'imap_send': return ['Connecting to IMAP server...', 'Sending email']
      case 'task_cancel': return ['Checking running tasks...', 'Sending cancel signal']
      case 'search_pdf': return ['Searching web...', 'Collecting results', 'Generating PDF report', 'Saving file']
      default: return ['Processing...', 'Analyzing', 'Done']
    }
  }

  switch (intent) {
    case 'pc_status':
      return ['PC 상태 데이터 요청 중...', 'CPU·메모리·디스크 수집', '시각 카드 생성 중']
    case 'security_scan':
      return ['보안 스캔 시작...', '원격 접속 흔적 확인', '수상한 프로세스 검사', '결과 분석 중']
    case 'full_scan':
      return ['전체 진단 시작...', '시스템 이슈 탐색', '심각도 분류', '리포트 생성 중']
    case 'clean':
      return ['정리 대상 파악...', '임시 파일·캐시 확인', '안전 정리 실행 중']
    case 'daily_report':
      return ['일일 데이터 수집...', '통계 분석 중', '예측 모델 실행', '리포트 완성']
    case 'open_folder': return ['폴더 이름 파악 중...', '경로 확인', '탐색기 실행 중']
    case 'remote_access': return ['원격 접속 도구 검색 중...', 'RDP 포트 확인', '프로세스 대조 중']
    case 'process_security': return ['프로세스 목록 수집...', '위험 패턴 대조', '포트 스캔 중']
    case 'startup_items': return ['시작 항목 조회 중...', '수상 키워드 분석']
    case 'defender_status': return ['Windows Defender 상태 확인...']
    case 'account_check': return ['로컬 계정 목록 조회...', '이상 계정 분석']
    case 'volume_control': return ['볼륨 조절 중...']
    case 'brightness': return ['밝기 조절 중...']
    case 'wifi_toggle': return ['Wi-Fi 상태 확인...', '설정 변경 중']
    case 'power_action': return ['전원 명령 실행 중...']
    case 'launch_app': return ['앱 경로 확인 중...', '실행 중']
    case 'process_top': return ['프로세스 목록 수집...', 'CPU·메모리 정렬 중']
    case 'driver_check': return ['드라이버 목록 조회...', '문제 항목 필터링']
    case 'network_analysis': return ['네트워크 어댑터 확인...', 'DNS·IP 조회 중', 'Ping 측정 중']
    case 'programs_list': return ['설치 프로그램 조회 중...']
    case 'boot_analysis': return ['부팅 이벤트 로그 분석...', '시작 항목 집계 중']
    case 'file_search': return ['파일 검색 시작...', '결과 수집 중']
    case 'file_organize': return ['파일 분류 중...', '폴더 이동 중']
    case 'file_duplicates': return ['파일 목록 수집...', '중복 분석 중']
    case 'browser_clean': return ['브라우저 캐시 위치 확인...', '데이터 정리 중']
    case 'registry_clean': return ['레지스트리 항목 스캔...', '무효 항목 정리 중']
    case 'restore_create': return ['복구 포인트 생성 중...']
    case 'focus_mode': return ['집중 모드 설정 중...']
    case 'notes': return ['메모 불러오는 중...']
    case 'doc_compare': return ['파일 열기...', '텍스트 추출 중', 'Diff 알고리즘 실행', '숫자 불일치 검사', '결과 정리 중']
    case 'doc_find': return ['파일 탐색 시작...', '이름·내용 대조 중', '결과 정렬 중']
    case 'deep_search': return ['파일 목록 수집...', '내용 인덱싱 중', '관련도 계산 중', '결과 정렬 중']
    case 'vision_screen': return ['화면 캡처 중...', 'AI에게 분석 요청', '답변 생성 중']
    case 'vision_ocr': return ['클립보드 이미지 확인...', 'Windows OCR 실행 중', '텍스트 추출 중']
    case 'smart_organize': return ['파일 목록 수집...', '파일 유형 분류 중', '폴더 이동 중', '정리 완료']
    case 'journal_today': return ['최근 파일 기록 수집...', '앱 사용 분석 중', '업무 시간 추정', '일지 생성 중']
    case 'journal_generate': return ['일지 데이터 수집...', '포맷 생성 중', '파일 저장 중']
    case 'journal_history': return ['과거 일지 조회 중...']
    case 'macro_list': return ['매크로 목록 조회 중...']
    case 'macro_create': return ['명령 파싱 중...', '액션 구성 중', '스케줄 등록 중']
    case 'macro_run': return ['매크로 실행 중...', '액션 순서 처리 중', '완료 확인']
    case 'pc_report': return ['시스템 상태 수집...', '보안 점검 중', '리포트 생성 중', 'HTML 저장 중']
    case 'report_email': return ['리포트 생성 중...', 'SMTP 연결 중', '이메일 전송 중']
    case 'doc_summary': return ['파일 열기...', '텍스트 추출 중', '핵심 분석 중', '요약 생성 중']
    case 'calendar_today': return ['Outlook 연결 중...', '오늘 일정 불러오는 중']
    case 'calendar_week': return ['Outlook 연결 중...', '이번 주 일정 불러오는 중']
    case 'calendar_add': return ['일정 생성 중...', 'Outlook에 저장 중']
    case 'email_inbox': return ['Outlook 연결 중...', '받은 편지함 확인 중']
    case 'email_send': return ['메일 작성 중...', 'SMTP 전송 중']
    case 'email_summarize': return ['받은 메일 가져오는 중...', 'AI 요약 생성 중']
    case 'virus_check': return ['파일 해시 계산 중...', 'VirusTotal 조회 중', '결과 분석 중']
    case 'perf_history': return ['성능 이력 불러오는 중...', '트렌드 분석 중']
    case 'perf_anomaly': return ['이력 데이터 분석 중...', '이상 패턴 탐지 중']
    case 'price_compare': return ['검색 시작...', '쿠팡 확인 중', '네이버 확인 중', '가격 비교 중']
    case 'multi_action': return ['멀티 액션 시작...', '검색 중', '결과 정리 중', '파일 저장 중']
    case 'news_search': return ['뉴스 검색 중...', '최신 기사 수집 중']
    case 'schedule_list': return ['스케줄 목록 불러오는 중...']
    case 'schedule_add': return ['명령 파싱 중...', '스케줄 등록 중']
    case 'schedule_delete': return ['스케줄 삭제 중...']
    case 'process_kill': return ['프로세스 찾는 중...', '강제 종료 중']
    case 'app_permissions': return ['레지스트리 확인 중...', '권한 목록 수집 중']
    case 'windows_updates': return ['Windows Update 서비스 연결 중...', '업데이트 목록 확인 중']
    case 'gpu_stats': return ['GPU 정보 수집 중...', 'nvidia-smi 확인 중']
    case 'recall_search': return ['화면 기억 데이터 검색 중...', '매칭 결과 정렬 중']
    case 'recall_capture': return ['화면 캡처 중...', 'OCR 텍스트 추출 중', '기억 저장 중']
    case 'meeting_start': return ['마이크 확인 중...', '녹음 시작 중']
    case 'meeting_stop': return ['녹음 종료 중...', '파일 저장 중']
    case 'meeting_summary': return ['녹음 파일 확인 중...', 'Whisper 전사 중...', 'AI 요약 생성 중']
    case 'meeting_list': return ['회의 목록 불러오는 중...']
    case 'dictation_start': return ['텍스트 분석 중...', '현재 앱에 입력 중']
    case 'weather': return ['날씨 데이터 수집 중...', '예보 분석 중']
    case 'travel_time': return ['출발지·목적지 좌표 조회 중...', '경로 계산 중']
    case 'translate': return ['클립보드 내용 확인 중...', '번역 중...']
    case 'clipboard_ai': return ['클립보드 내용 가져오는 중...', 'AI 처리 중']
    case 'clipboard_history': return ['클립보드 히스토리 불러오는 중...']
    case 'voice_todo': return ['내용 분석 중...', '메모 저장 중', '캘린더 등록 중']
    case 'persona_list': return ['페르소나 목록 불러오는 중...']
    case 'persona_switch': return ['페르소나 전환 중...']
    case 'brain_search': return ['🧠 Second Brain 검색 중...', '관련 기억 분석 중']
    case 'brain_stats': return ['인덱스 통계 조회 중...']
    case 'workflow_run': return ['⚡ 워크플로 계획 생성 중...', '단계별 실행 중...', '결과 정리 중...']
    case 'workflow_plan': return ['⚡ 워크플로 계획 생성 중...']
    case 'caption_start': return ['🎬 오디오 캡처 초기화 중...', '실시간 자막 시작']
    case 'caption_stop': return ['자막 종료 중...']
    case 'video_download': return ['영상 URL 확인 중...', 'yt-dlp로 다운로드 중...', '파일 저장 중']
    case 'video_transcript': return ['영상 URL 확인 중...', '자막 추출 중...', 'AI 요약 생성 중...']
    case 'email_classify': return ['받은 메일 가져오는 중...', 'AI 분류 중...', '우선순위 정리 중']
    case 'email_draft': return ['메일 내용 분석 중...', 'AI 답장 초안 작성 중']
    case 'calendar_find_slot': return ['캘린더 확인 중...', '빈 시간 탐색 중', '가능한 슬롯 정리 중']
    case 'calendar_smart_add': return ['자연어 파싱 중...', '일정 생성 중', 'Outlook 저장 중']
    case 'workflow_list': return ['저장된 워크플로 조회 중...']
    case 'workflow_create': return ['자연어 파싱 중...', '워크플로 생성 중', '저장 중']
    case 'workflow_templates': return ['워크플로 템플릿 불러오는 중...']
    case 'imap_inbox': return ['IMAP 서버 연결 중...', '받은 메일 불러오는 중']
    case 'imap_send': return ['IMAP 서버 연결 중...', '메일 전송 중']
    case 'parallel_queries': return ['쿼리 분리 중...', '병렬 실행 중...', '결과 취합 중']
    case 'multi_agent': return ['멀티 에이전트 준비 중...', '에이전트 팀 배치 중', '병렬 실행 중']
    case 'briefing_now': return ['날씨 확인 중...', '일정 수집 중...', '이메일 확인 중...', '브리핑 생성 중']
    case 'task_cancel': return ['실행 중 작업 확인...', '취소 신호 전송 중']
    case 'search_pdf': return ['웹 검색 중...', '결과 수집 중', 'PDF 보고서 생성 중', '파일 저장 중']
    default:
      return ['요청 분석 중...']
  }
}

export function intentResponseText(intent: Intent, lang: 'ko' | 'en', assistantName: string): string {
  if (lang === 'en') {
    switch (intent) {
      case 'pc_status': return `Here's your real-time PC status, ${assistantName} is watching over it!`
      case 'security_scan': return `Security scan complete. Here are the results:`
      case 'full_scan': return `Full PC diagnostic done! Here's what I found:`
      case 'clean': return `Cleanup complete! I freed up some disk space for you.`
      case 'daily_report': return `Here's today's PC report summary:`
      case 'repair': return `Repair operation finished!`
      case 'file_search': return `Here are the file search results:`
      case 'deep_search': return `Deep search complete! Here are the top results:`
      case 'doc_find': return `Document search complete! Found the following:`
      case 'doc_compare': return `Document comparison complete! Here are the differences:`
      case 'doc_summary': return `Here's the document summary:`
      case 'news_search': return `Here are the latest news results:`
      case 'youtube_search': return `Here are the video results:`
      case 'reddit_search': return `Here are the Reddit posts:`
      case 'price_compare': return `Here are the price comparison results:`
      case 'email_inbox': return `Here's your inbox:`
      case 'email_summarize': return `Here's your email summary:`
      case 'email_classify': return `Email classification complete:`
      case 'email_draft': return `Here's the draft reply:`
      case 'calendar_today': return `Here's today's schedule:`
      case 'calendar_week': return `Here's this week's schedule:`
      case 'calendar_add': return `Event added to your calendar!`
      case 'calendar_find_slot': return `Here are available time slots:`
      case 'calendar_smart_add': return `Schedule created!`
      case 'weather': return `Here's the weather forecast:`
      case 'travel_time': return `Here's your travel time estimate:`
      case 'notes': return `Here are your notes:`
      case 'journal_today': return `Today's work journal is ready:`
      case 'journal_generate': return `Journal generated and saved!`
      case 'journal_history': return `Here's your journal history:`
      case 'macro_list': return `Here are your macros:`
      case 'macro_create': return `Macro created!`
      case 'macro_run': return `Macro executed!`
      case 'schedule_list': return `Here are your scheduled tasks:`
      case 'schedule_add': return `Task scheduled!`
      case 'schedule_delete': return `Schedule deleted!`
      case 'recall_capture': return `Screen memory saved!`
      case 'recall_search': return `Here are the matching screen memories:`
      case 'meeting_start': return `Recording started!`
      case 'meeting_stop': return `Recording stopped and saved!`
      case 'meeting_summary': return `Here's the meeting summary:`
      case 'meeting_list': return `Here are your recorded meetings:`
      case 'workflow_run': return `Workflow complete! Here are the results:`
      case 'workflow_plan': return `Here's the workflow plan:`
      case 'workflow_list': return `Here are your saved workflows:`
      case 'workflow_create': return `Workflow created!`
      case 'workflow_templates': return `Here are available workflow templates:`
      case 'multi_agent': return `Multi-agent task complete!`
      case 'briefing_now': return `Here's your morning briefing:`
      case 'remote_access': return `Remote access scan complete:`
      case 'process_security': return `Process security scan complete:`
      case 'startup_items': return `Here are your startup items:`
      case 'defender_status': return `Windows Defender status:`
      case 'account_check': return `Account security check complete:`
      case 'driver_check': return `Driver check complete:`
      case 'network_analysis': return `Network analysis complete:`
      case 'programs_list': return `Here are your installed programs:`
      case 'boot_analysis': return `Boot analysis complete:`
      case 'process_top': return `Here are the top resource-consuming processes:`
      case 'process_kill': return `Process terminated!`
      case 'gpu_stats': return `Here are your GPU stats:`
      case 'windows_updates': return `Here are the available Windows updates:`
      case 'virus_check': return `VirusTotal scan complete:`
      case 'perf_history': return `Here's your performance history:`
      case 'perf_anomaly': return `Performance anomaly analysis complete:`
      case 'focus_mode': return `Focus mode configured!`
      case 'volume_control': return `Volume adjusted!`
      case 'brightness': return `Brightness adjusted!`
      case 'wifi_toggle': return `Wi-Fi setting changed!`
      case 'launch_app': return `App launched!`
      case 'open_folder': return `Folder opened!`
      case 'power_action': return `Power command executed!`
      case 'restore_create': return `Restore point created!`
      case 'browser_clean': return `Browser data cleaned!`
      case 'registry_clean': return `Registry cleaned!`
      case 'pc_report': return `PC health report generated!`
      case 'persona_list': return `Here are your available personas:`
      case 'persona_switch': return `Persona switched!`
      case 'brain_search': return `Here are the matching memories from your Second Brain:`
      case 'brain_stats': return `Here are your Second Brain stats:`
      case 'caption_start': return `Live captions started!`
      case 'caption_stop': return `Captions stopped!`
      case 'video_download': return `Video download complete!`
      case 'video_transcript': return `Here's the video transcript summary:`
      case 'translate': return `Translation complete:`
      case 'clipboard_ai': return `Here's the AI result:`
      case 'dictation_start': return `Text has been typed!`
      case 'voice_todo': return `Note saved and event registered!`
      case 'task_cancel': return `Task cancellation requested!`
      case 'search_pdf': return `PDF report generated!`
      case 'imap_inbox': return `Here's your IMAP inbox:`
      case 'imap_send': return `Email sent via IMAP!`
      case 'app_permissions': return `Here are your app permissions:`
      case 'file_organize': return `Files organized!`
      case 'file_duplicates': return `Here are the duplicate files found:`
      case 'smart_organize': return `Smart organization complete!`
      case 'vision_screen': return `Here's the screen analysis result:`
      case 'vision_ocr': return `Here's the extracted text:`
      case 'multi_action': return `Multi-action complete! Here are the results:`
      default: return ''
    }
  }

  switch (intent) {
    case 'pc_status': return `실시간 PC 상태를 가져왔어요! 📊`
    case 'security_scan': return `보안 스캔 완료! 결과를 확인해보세요 🔒`
    case 'full_scan': return `전체 진단 완료! 발견된 항목을 정리했어요 🔍`
    case 'clean': return `정리 완료! 디스크 공간을 확보했어요 🧹`
    case 'daily_report': return `오늘의 PC 리포트예요 📊`
    case 'repair': return `수리 작업을 완료했어요 🔧`
    case 'file_search': return `파일 검색 결과예요:`
    case 'deep_search': return `딥서치 완료! 상위 결과를 정리했어요:`
    case 'doc_find': return `문서 검색 완료! 찾은 파일이에요:`
    case 'doc_compare': return `문서 비교 완료! 차이점을 정리했어요:`
    case 'doc_summary': return `문서 요약 결과예요:`
    case 'news_search': return `최신 뉴스 결과예요:`
    case 'youtube_search': return `영상 검색 결과예요:`
    case 'reddit_search': return `Reddit 게시물이에요:`
    case 'price_compare': return `가격 비교 결과예요:`
    case 'email_inbox': return `받은 편지함이에요:`
    case 'email_summarize': return `이메일 요약이에요:`
    case 'email_classify': return `이메일 분류 완료:`
    case 'email_draft': return `답장 초안이에요:`
    case 'calendar_today': return `오늘 일정이에요:`
    case 'calendar_week': return `이번 주 일정이에요:`
    case 'calendar_add': return `일정이 추가됐어요!`
    case 'calendar_find_slot': return `가능한 시간 슬롯이에요:`
    case 'calendar_smart_add': return `일정을 만들었어요!`
    case 'weather': return `날씨 예보예요:`
    case 'travel_time': return `소요 시간 결과예요:`
    case 'notes': return `메모 목록이에요:`
    case 'journal_today': return `오늘 업무 일지예요:`
    case 'journal_generate': return `일지를 생성하고 저장했어요!`
    case 'journal_history': return `과거 일지 목록이에요:`
    case 'macro_list': return `매크로 목록이에요:`
    case 'macro_create': return `매크로가 생성됐어요!`
    case 'macro_run': return `매크로를 실행했어요!`
    case 'schedule_list': return `예약된 작업 목록이에요:`
    case 'schedule_add': return `작업이 예약됐어요!`
    case 'schedule_delete': return `스케줄이 삭제됐어요!`
    case 'recall_capture': return `화면 기억을 저장했어요!`
    case 'recall_search': return `매칭된 화면 기억이에요:`
    case 'meeting_start': return `녹음이 시작됐어요!`
    case 'meeting_stop': return `녹음을 저장했어요!`
    case 'meeting_summary': return `회의 요약이에요:`
    case 'meeting_list': return `녹음된 회의 목록이에요:`
    case 'workflow_run': return `워크플로 완료! 결과를 정리했어요:`
    case 'workflow_plan': return `워크플로 계획이에요:`
    case 'workflow_list': return `저장된 워크플로 목록이에요:`
    case 'workflow_create': return `워크플로가 생성됐어요!`
    case 'workflow_templates': return `워크플로 템플릿이에요:`
    case 'multi_agent': return `멀티 에이전트 작업 완료!`
    case 'briefing_now': return `모닝 브리핑이에요:`
    case 'remote_access': return `원격 접속 스캔 완료:`
    case 'process_security': return `프로세스 보안 스캔 완료:`
    case 'startup_items': return `시작 프로그램 목록이에요:`
    case 'defender_status': return `Windows Defender 상태:`
    case 'account_check': return `계정 보안 점검 완료:`
    case 'driver_check': return `드라이버 점검 완료:`
    case 'network_analysis': return `네트워크 분석 완료:`
    case 'programs_list': return `설치된 프로그램 목록이에요:`
    case 'boot_analysis': return `부팅 분석 완료:`
    case 'process_top': return `리소스 상위 프로세스예요:`
    case 'process_kill': return `프로세스를 종료했어요!`
    case 'gpu_stats': return `GPU 정보예요:`
    case 'windows_updates': return `Windows 업데이트 목록이에요:`
    case 'virus_check': return `VirusTotal 스캔 완료:`
    case 'perf_history': return `성능 이력이에요:`
    case 'perf_anomaly': return `성능 이상 분석 완료:`
    case 'focus_mode': return `집중 모드가 설정됐어요!`
    case 'volume_control': return `볼륨을 조절했어요!`
    case 'brightness': return `밝기를 조절했어요!`
    case 'wifi_toggle': return `Wi-Fi 설정을 변경했어요!`
    case 'launch_app': return `앱을 실행했어요!`
    case 'open_folder': return `폴더를 열었어요!`
    case 'power_action': return `전원 명령을 실행했어요!`
    case 'restore_create': return `복구 포인트를 만들었어요!`
    case 'browser_clean': return `브라우저 데이터를 정리했어요!`
    case 'registry_clean': return `레지스트리를 정리했어요!`
    case 'pc_report': return `PC 건강 리포트가 생성됐어요!`
    case 'persona_list': return `페르소나 목록이에요:`
    case 'persona_switch': return `페르소나를 전환했어요!`
    case 'brain_search': return `Second Brain에서 찾은 기억이에요:`
    case 'brain_stats': return `Second Brain 통계예요:`
    case 'caption_start': return `실시간 자막을 시작했어요!`
    case 'caption_stop': return `자막을 종료했어요!`
    case 'video_download': return `영상 다운로드 완료!`
    case 'video_transcript': return `영상 자막 요약이에요:`
    case 'translate': return `번역 결과예요:`
    case 'clipboard_ai': return `AI 처리 결과예요:`
    case 'dictation_start': return `텍스트를 입력했어요!`
    case 'voice_todo': return `메모 저장 및 일정 등록 완료!`
    case 'task_cancel': return `작업 취소를 요청했어요!`
    case 'search_pdf': return `PDF 리포트가 생성됐어요!`
    case 'imap_inbox': return `IMAP 받은 편지함이에요:`
    case 'imap_send': return `IMAP으로 메일을 보냈어요!`
    case 'app_permissions': return `앱 권한 목록이에요:`
    case 'file_organize': return `파일을 정리했어요!`
    case 'file_duplicates': return `중복 파일 결과예요:`
    case 'smart_organize': return `스마트 정리 완료!`
    case 'vision_screen': return `화면 분석 결과예요:`
    case 'vision_ocr': return `추출된 텍스트예요:`
    case 'multi_action': return `멀티 액션 완료! 결과를 정리했어요:`
    default: return ''
  }
}
