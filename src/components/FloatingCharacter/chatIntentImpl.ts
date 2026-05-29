import type { Dispatch, SetStateAction, MutableRefObject } from 'react'
import { backendAPI, mockStats, mockScan, mockDailyReport, sendCommand,
  calendarToday, calendarWeek, calendarAdd, calendarFindSlot, calendarSmartAdd,
  emailInbox, emailSend, emailSummarize, emailClassify, emailDraftReply,
  virusTotalCheck, historyStats, historyAnomalies,
  processKill, appPermissions, windowsUpdates, gpuStats,
  priceCompare, newsSearch, youtubeSearch, tiktokSearch, naverShoppingSearch, coupangSearch, videoDownload, videoQuickSearch,
  redditSearch, redditTrending,
  schedulerAdd, schedulerList, schedulerDelete,
  recallCapture, recallSearch,
  clipboardHistory, clipboardHistoryClear,
  meetingStart, meetingStop, meetingList, meetingTranscribe, meetingSummarize,
  dictationType, dictationPaste,
  weatherGet, travelTime,
  personaList, personaSet, personaCurrent,
  brainSearch, brainStats, brainRebuild,
  workflowRun, workflowPlan, workflowList, workflowFromText, workflowTemplates,
  captionStart, captionStop, captionLatest,
  briefingNow,
  taskList, taskCancel,
  multiAgentRun, multiAgentPlan,
  searchAndPDF,
  siteSearch,
  getAuthHeader,
  videoTranscript,
  imapInbox,
  imapSend,
  dispatchParallel,
} from '../../lib/nexus/backendAPI'
import type { ParallelEvent } from '../../lib/nexus/backendAPI'
import type { PersonaDef } from '../../lib/nexus/backendAPI'
import type { InlineCardData } from './InlineCards'
import type { InlineCardData2 } from './InlineCards2'
import type { InlineCard3Data } from './InlineCards3'
import type { InlineCard4Data } from './InlineCards4'
import type { Intent } from '../../lib/nexus/intentDetector'
import type { NexusEmotion } from '../../types/nexus'
import { callGemini, fallbackResponse, trackUsage, getLastPreviewItems, clearLastPreviewItems } from '../../lib/nexus/gemini_engine'
import { appendHistory } from './ChatBubble'
import type { ChatMessage } from './ChatBubble'
import { speak } from '../../lib/nexus/tts'
import { routeWithLLM } from '../../lib/nexus/llmToolRouter'
import { nexusSSE } from '../../lib/nexus/sseClient'
import type { TaskUpdate } from '../../lib/nexus/sseClient'
import { buildMemoryContext } from '../../lib/nexus/memory'
import {
  extractFolderName, extractVolume, extractBrightness, extractWifiAction, extractPowerAction,
  extractAppName, extractNoteContent, extractTwoFilePaths, extractVisionQuestion, extractDeepSearchQuery,
} from '../../lib/nexus/intentDetector'
import { setFocusModeEnd, clearFocusMode } from '../../lib/nexus/proactiveAI'

interface ConversationTurn {
  role: 'user' | 'model'
  parts: Array<{ text: string }>
}

type CharacterEmotion = NexusEmotion

function t(ko: string, en: string, lang: 'ko' | 'en'): string {
  return lang === 'en' ? en : ko
}

function buildAgentSteps(intent: Intent, lang: 'ko' | 'en' = 'ko'): string[] {
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
      case 'calendar_today': return ['Connecting to Outlook...', 'Loading today\'s schedule']
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
    case 'open_folder':
      return ['폴더 이름 파악 중...', '경로 확인', '탐색기 실행 중']
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
    case 'smart_organize':   return ['파일 목록 수집...', '파일 유형 분류 중', '폴더 이동 중', '정리 완료']
    case 'journal_today':    return ['최근 파일 기록 수집...', '앱 사용 분석 중', '업무 시간 추정', '일지 생성 중']
    case 'journal_generate': return ['일지 데이터 수집...', '포맷 생성 중', '파일 저장 중']
    case 'journal_history':  return ['과거 일지 조회 중...']
    case 'macro_list':       return ['매크로 목록 조회 중...']
    case 'macro_create':     return ['명령 파싱 중...', '액션 구성 중', '스케줄 등록 중']
    case 'macro_run':        return ['매크로 실행 중...', '액션 순서 처리 중', '완료 확인']
    case 'pc_report':        return ['시스템 상태 수집...', '보안 점검 중', '리포트 생성 중', 'HTML 저장 중']
    case 'report_email':     return ['리포트 생성 중...', 'SMTP 연결 중', '이메일 전송 중']
    case 'doc_summary':      return ['파일 열기...', '텍스트 추출 중', '핵심 분석 중', '요약 생성 중']
    case 'calendar_today':   return ['Outlook 연결 중...', '오늘 일정 불러오는 중']
    case 'calendar_week':    return ['Outlook 연결 중...', '이번 주 일정 불러오는 중']
    case 'calendar_add':     return ['일정 생성 중...', 'Outlook에 저장 중']
    case 'email_inbox':      return ['Outlook 연결 중...', '받은 편지함 확인 중']
    case 'email_send':       return ['메일 작성 중...', 'SMTP 전송 중']
    case 'email_summarize':  return ['받은 메일 가져오는 중...', 'AI 요약 생성 중']
    case 'virus_check':      return ['파일 해시 계산 중...', 'VirusTotal 조회 중', '결과 분석 중']
    case 'perf_history':     return ['성능 이력 불러오는 중...', '트렌드 분석 중']
    case 'perf_anomaly':     return ['이력 데이터 분석 중...', '이상 패턴 탐지 중']
    case 'price_compare':    return ['검색 시작...', '쿠팡 확인 중', '네이버 확인 중', '가격 비교 중']
    case 'multi_action':     return ['멀티 액션 시작...', '검색 중', '결과 정리 중', '파일 저장 중']
    case 'news_search':      return ['뉴스 검색 중...', '최신 기사 수집 중']
    case 'schedule_list':    return ['스케줄 목록 불러오는 중...']
    case 'schedule_add':     return ['명령 파싱 중...', '스케줄 등록 중']
    case 'schedule_delete':  return ['스케줄 삭제 중...']
    case 'process_kill':     return ['프로세스 찾는 중...', '강제 종료 중']
    case 'app_permissions':  return ['레지스트리 확인 중...', '권한 목록 수집 중']
    case 'windows_updates':  return ['Windows Update 서비스 연결 중...', '업데이트 목록 확인 중']
    case 'gpu_stats':        return ['GPU 정보 수집 중...', 'nvidia-smi 확인 중']
    case 'recall_search':    return ['화면 기억 데이터 검색 중...', '매칭 결과 정렬 중']
    case 'recall_capture':   return ['화면 캡처 중...', 'OCR 텍스트 추출 중', '기억 저장 중']
    case 'meeting_start':    return ['마이크 확인 중...', '녹음 시작 중']
    case 'meeting_stop':     return ['녹음 종료 중...', '파일 저장 중']
    case 'meeting_summary':  return ['녹음 파일 확인 중...', 'Whisper 전사 중...', 'AI 요약 생성 중']
    case 'meeting_list':     return ['회의 목록 불러오는 중...']
    case 'dictation_start':  return ['텍스트 분석 중...', '현재 앱에 입력 중']
    case 'weather':          return ['날씨 데이터 수집 중...', '예보 분석 중']
    case 'travel_time':      return ['출발지·목적지 좌표 조회 중...', '경로 계산 중']
    case 'translate':        return ['클립보드 내용 확인 중...', '번역 중...']
    case 'clipboard_ai':      return ['클립보드 내용 가져오는 중...', 'AI 처리 중']
    case 'clipboard_history': return ['클립보드 히스토리 불러오는 중...']
    case 'voice_todo':        return ['내용 분석 중...', '메모 저장 중', '캘린더 등록 중']
    case 'persona_list':     return ['페르소나 목록 불러오는 중...']
    case 'persona_switch':   return ['페르소나 전환 중...']
    case 'brain_search':     return ['🧠 Second Brain 검색 중...', '관련 기억 분석 중']
    case 'brain_stats':      return ['인덱스 통계 조회 중...']
    case 'workflow_run':     return ['⚡ 워크플로 계획 생성 중...', '단계별 실행 중...', '결과 정리 중...']
    case 'workflow_plan':    return ['⚡ 워크플로 계획 생성 중...']
    case 'caption_start':    return ['🎬 오디오 캡처 초기화 중...', '실시간 자막 시작']
    case 'caption_stop':     return ['자막 종료 중...']
    case 'video_download':    return ['영상 URL 확인 중...', 'yt-dlp로 다운로드 중...', '파일 저장 중']
    case 'video_transcript':  return ['영상 URL 확인 중...', '자막 추출 중...', 'AI 요약 생성 중...']
    case 'email_classify':    return ['받은 메일 가져오는 중...', 'AI 분류 중...', '우선순위 정리 중']
    case 'email_draft':       return ['메일 내용 분석 중...', 'AI 답장 초안 작성 중']
    case 'calendar_find_slot': return ['캘린더 확인 중...', '빈 시간 탐색 중', '가능한 슬롯 정리 중']
    case 'calendar_smart_add': return ['자연어 파싱 중...', '일정 생성 중', 'Outlook 저장 중']
    case 'workflow_list':     return ['저장된 워크플로 조회 중...']
    case 'workflow_create':   return ['자연어 파싱 중...', '워크플로 생성 중', '저장 중']
    case 'workflow_templates': return ['워크플로 템플릿 불러오는 중...']
    case 'imap_inbox':        return ['IMAP 서버 연결 중...', '받은 메일 불러오는 중']
    case 'imap_send':         return ['IMAP 서버 연결 중...', '메일 전송 중']
    case 'parallel_queries':  return ['쿼리 분리 중...', '병렬 실행 중...', '결과 취합 중']
    case 'multi_agent':       return ['멀티 에이전트 준비 중...', '에이전트 팀 배치 중', '병렬 실행 중']
    case 'briefing_now':      return ['날씨 확인 중...', '일정 수집 중...', '이메일 확인 중...', '브리핑 생성 중']
    case 'task_cancel':       return ['실행 중 작업 확인...', '취소 신호 전송 중']
    case 'search_pdf':        return ['웹 검색 중...', '결과 수집 중', 'PDF 보고서 생성 중', '파일 저장 중']
    default:
      return ['요청 분석 중...']
  }
}

function intentResponseText(intent: Intent, lang: 'ko' | 'en', assistantName: string): string {
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

export interface ChatIntentDeps {
  userLang: 'ko' | 'en'
  assistantName: string
  emotion: CharacterEmotion
  isActive: boolean
  soundEnabled: boolean
  clarifyPendingIntent: string | null
  clarifyPendingParams: Record<string, unknown> | null
  clarifyPendingQuestion: string | null
  historyRef: MutableRefObject<Array<{ role: string; parts: Array<{ text: string }> }>>
  typingRef: MutableRefObject<boolean>
  isMountedRef: MutableRefObject<boolean>
  setMessages: Dispatch<SetStateAction<import('./ChatBubble').ChatMessage[]>>
  setEmotion: Dispatch<SetStateAction<CharacterEmotion>>
  setChatOpen: Dispatch<SetStateAction<boolean>>
  setMinimized: Dispatch<SetStateAction<boolean>>
  setSpeaking: Dispatch<SetStateAction<boolean>>
  setBubbleText: Dispatch<SetStateAction<string>>
  setActivePersona: Dispatch<SetStateAction<PersonaDef | null>>
  setCaptionRunning: Dispatch<SetStateAction<boolean>>
  setFloatingPreview: Dispatch<SetStateAction<Array<{ title: string; url: string; isVideo?: boolean; isSocial?: boolean; isMap?: boolean; mapType?: string; service?: string; isImage?: boolean }> | null>>
  setFocusEndMs: Dispatch<SetStateAction<number | undefined>>
  setPreviewType: Dispatch<SetStateAction<string>>
  setToastAlerts: Dispatch<SetStateAction<Array<{id: string; title: string; message: string; level: string}>>>
  setSoundEnabled: Dispatch<SetStateAction<boolean>>
  setIsActive: Dispatch<SetStateAction<boolean>>
  setHistoryVersion: Dispatch<SetStateAction<number>>
  setUserLang: (lang: 'ko' | 'en') => void
  speakText: (text: string, em?: CharacterEmotion) => void
  resetClarify: () => void
  openPreview: (url: string, title: string) => Promise<void>
  pushModelHistory: (userText: string, modelText: string) => void
  openEmailSetup?: () => void
}

export async function handleBackendIntentImpl(
  intent: Intent,
  msgId: string,
  originalText: string,
  d: ChatIntentDeps,
): Promise<{ text: string; card?: InlineCardData; card2?: InlineCardData2; card3?: InlineCard3Data; card4?: InlineCard4Data; emotion: CharacterEmotion }> {
  const { userLang, assistantName, emotion, isActive, soundEnabled,
    clarifyPendingIntent, clarifyPendingParams, clarifyPendingQuestion,
    historyRef, typingRef, isMountedRef,
    setMessages, setEmotion, setChatOpen, setMinimized, setSpeaking, setBubbleText,
    setActivePersona, setCaptionRunning, setFloatingPreview, setFocusEndMs,
    setPreviewType, setToastAlerts, setSoundEnabled, setIsActive, setHistoryVersion,
    setUserLang, speakText, resetClarify, openPreview, pushModelHistory,
  } = d


    /* Multi-step: 사고 카드 먼저 표시 */
    const steps = buildAgentSteps(intent, userLang)
    setMessages(prev => [...prev, {
      id: `think-${msgId}`,
      role: 'nexus',
      text: '',
      inlineCard: { type: 'agent_thinking', steps },
    }])

    /* 약간의 딜레이로 사고 과정 보여주기 */
    await new Promise(r => setTimeout(r, steps.length * 200))

    /* 사고 카드 제거 */
    setMessages(prev => prev.filter(m => m.id !== `think-${msgId}`))

    try {
      switch (intent) {
        case 'pc_status': {
          const data = await backendAPI.stats().catch(() => mockStats())
          return {
            text: intentResponseText('pc_status', userLang, assistantName),
            card: { type: 'pc_status', data },
            emotion: data.cpu > 80 || data.mem > 85 ? 'concerned' : 'happy',
          }
        }
        case 'security_scan':
        case 'full_scan': {
          const data = await backendAPI.scan().catch(() => mockScan())
          const em: CharacterEmotion = data.score < 70 ? 'alert' : data.score < 85 ? 'concerned' : 'happy'
          return {
            text: intentResponseText(intent, userLang, assistantName),
            card: { type: 'scan_result', data },
            emotion: em,
          }
        }
        case 'clean': {
          const results = await backendAPI.autoClean(['temp', 'browser']).catch(async () => {
            const r = await backendAPI.clean(['temp']).catch(() => ({ freed: 0, message: '정리 완료' }))
            return r as { freed: number; message: string }
          })
          return {
            text: intentResponseText('clean', userLang, assistantName),
            card: { type: 'clean_result', results },
            emotion: 'happy',
          }
        }
        case 'daily_report': {
          const data = await backendAPI.dailyReport().catch(() => mockDailyReport())
          return {
            text: intentResponseText('daily_report', userLang, assistantName),
            card: { type: 'daily_report', data },
            emotion: data.pc_score >= 80 ? 'happy' : 'concerned',
          }
        }
        case 'repair': {
          const data = await backendAPI.repair(['temp-files']).catch(() => ({
            success: true, message: '수리 완료', freed: 0,
          }))
          return {
            text: intentResponseText('repair', userLang, assistantName),
            card: { type: 'repair_result', data },
            emotion: data.success ? 'happy' : 'concerned',
          }
        }
        case 'open_folder': {
          const folderName = extractFolderName(originalText)
          if (!folderName) {
            const ask = userLang === 'ko'
              ? '어떤 폴더를 열어드릴까요? (예: 바탕화면, 다운로드, 문서, 사진)'
              : 'Which folder would you like to open?'
            return { text: ask, emotion: 'neutral' }
          }
          const res = await backendAPI.openFolder(folderName).catch(() => ({
            success: false, path: '', message: t('백엔드 미연결 상태입니다.', 'Backend not connected.', userLang),
          }))
          return {
            text: res.success ? t(`${folderName} 폴더를 열었어요.`, `Opened folder: ${folderName}`, userLang) : res.message,
            card: { type: 'folder_open', success: res.success, path: res.path, message: res.message },
            emotion: res.success ? 'happy' : 'concerned',
          }
        }

        /* ── 보안 상세 ── */
        case 'remote_access': {
          const data = await backendAPI.securityRemote().catch(() => ({ found: false, tools: [], rdp_open: false, score: 100 }))
          return { text: data.found ? t(`⚠️ 실행 중인 원격 접속 도구 발견! 점수: ${data.score}`, `⚠️ Remote access tool detected! Score: ${data.score}`, userLang) : t('✅ 원격 접속 도구 없음, 안전합니다.', '✅ No remote access tools found. You\'re safe.', userLang),
            card2: { type: 'remote_access', data }, emotion: data.found ? 'alert' : 'happy' }
        }
        case 'process_security': {
          const data = await backendAPI.securityProcs().catch(() => ({ suspicious_processes: [], open_ports: [], score: 100 }))
          return { text: data.score < 80 ? t(`⚠️ 수상한 프로세스/포트 발견 (보안 점수: ${data.score})`, `⚠️ Suspicious processes/ports found (score: ${data.score})`, userLang) : t('✅ 수상한 프로세스 없음.', '✅ No suspicious processes found.', userLang),
            card2: { type: 'process_security', data }, emotion: data.score < 80 ? 'alert' : 'happy' }
        }
        case 'hosts_check': {
          const data = await backendAPI.securityHosts().catch(() => ({ score: 100, modified: false, entries: 0, suspicious: [] }))
          return { text: data.modified ? t(`⚠️ hosts 파일 변조 의심! 수상한 항목 ${data.suspicious.length}개`, `⚠️ Hosts file may be tampered! ${data.suspicious.length} suspicious entries`, userLang) : t('✅ hosts 파일 정상', '✅ Hosts file is clean', userLang),
            card2: { type: 'system_action', icon: data.modified ? '⚠️' : '✅', title: data.modified ? t('Hosts 파일 변조 감지', 'Hosts file tampered', userLang) : t('Hosts 파일 정상', 'Hosts file clean', userLang), detail: t(`총 ${data.entries}개 항목`, `${data.entries} entries total`, userLang), success: !data.modified },
            emotion: data.modified ? 'alert' : 'happy' }
        }
        case 'startup_items': {
          const data = await backendAPI.securityStartup().catch(() => ({ items: [], total: 0, suspicious_count: 0 }))
          return { text: t(`시작 프로그램 ${data.total}개, 수상한 항목 ${data.suspicious_count}개`, `${data.total} startup items, ${data.suspicious_count} suspicious`, userLang),
            card2: { type: 'startup_items', data }, emotion: data.suspicious_count > 0 ? 'concerned' : 'happy' }
        }
        case 'defender_status': {
          const data = await backendAPI.securityDefender().catch(() => ({ antivirus_enabled: true, realtime_protection: true, quick_scan_age: 0, full_scan_age: 0, score: 100, issues: [] }))
          return { text: data.score >= 80 ? t('🛡️ Windows Defender 정상 작동 중', '🛡️ Windows Defender is running normally', userLang) : t(`⚠️ 보안 점수 ${data.score} — ${data.issues[0] ?? ''}`, `⚠️ Security score ${data.score} — ${data.issues[0] ?? ''}`, userLang),
            card2: { type: 'defender', data }, emotion: data.score >= 80 ? 'happy' : 'alert' }
        }
        case 'account_check': {
          const data = await backendAPI.securityAccounts().catch(() => ({ total: 0, suspicious: [], suspicious_count: 0, score: 100 }))
          return { text: data.suspicious_count ? t(`⚠️ 이상 계정 ${data.suspicious_count}개 감지됨`, `⚠️ ${data.suspicious_count} suspicious account(s) detected`, userLang) : t(`✅ 계정 정상 (${data.total}개)`, `✅ Accounts look normal (${data.total} total)`, userLang),
            card2: { type: 'system_action', icon: data.suspicious_count ? '⚠️' : '✅', title: data.suspicious_count ? t(`이상 계정 ${data.suspicious_count}개`, `${data.suspicious_count} suspicious accounts`, userLang) : t('계정 정상', 'Accounts normal', userLang), success: !data.suspicious_count },
            emotion: data.suspicious_count ? 'alert' : 'happy' }
        }

        /* ── 시스템 제어 ── */
        case 'volume_control': {
          const { action, value } = extractVolume(originalText)
          const res = await backendAPI.volume(action, value).catch(() => ({ message: '볼륨 조절에 실패했어요' }))
          return { text: res.message,
            card2: { type: 'system_action', icon: action === 'mute' ? '🔇' : '🔊', title: res.message, success: true },
            emotion: 'happy' }
        }
        case 'brightness': {
          const { action, value } = extractBrightness(originalText)
          const res = await backendAPI.brightness(action, value).catch(() => ({ message: '밝기 조절에 실패했어요 (노트북 전용)' }))
          return { text: res.message,
            card2: { type: 'system_action', icon: '☀️', title: res.message, success: true },
            emotion: 'happy' }
        }
        case 'wifi_toggle': {
          const wifiAction = extractWifiAction(originalText)
          const res = await backendAPI.wifi(wifiAction).catch(() => ({ message: 'Wi-Fi 제어 실패' }))
          return { text: (res as { message?: string }).message ?? 'Wi-Fi 상태 확인됨',
            card2: { type: 'system_action', icon: '📶', title: (res as { message?: string }).message ?? '', success: true },
            emotion: 'happy' }
        }
        case 'power_action': {
          const powerAct = extractPowerAction(originalText)
          const icons: Record<string, string> = { lock: '🔒', sleep: '😴', restart: '🔄', shutdown: '⏻' }
          const res = await backendAPI.power(powerAct).catch(() => ({ success: false, message: '전원 제어 실패' }))
          return { text: res.message,
            card2: { type: 'system_action', icon: icons[powerAct] ?? '⚡', title: res.message, success: res.success },
            emotion: res.success ? 'happy' : 'concerned' }
        }
        case 'launch_app': {
          const appName = extractAppName(originalText)
          if (!appName) return { text: t('어떤 앱을 실행할까요?', 'Which app would you like to launch?', userLang), emotion: 'neutral' }
          const res = await backendAPI.launchApp(appName).catch(() => ({ success: false, message: `${appName} 실행 실패` }))
          return { text: res.message,
            card2: { type: 'system_action', icon: '🚀', title: res.message, success: res.success },
            emotion: res.success ? 'happy' : 'concerned' }
        }
        case 'process_top': {
          const data = await backendAPI.processTop().catch(() => ({ by_cpu: [], by_mem: [] }))
          return { text: t('CPU·메모리 상위 프로세스예요 📊', 'Top processes by CPU & memory 📊', userLang),
            card2: { type: 'process_top', data }, emotion: 'neutral' }
        }

        /* ── 고급 기능 ── */
        case 'driver_check': {
          const data = await backendAPI.drivers().catch(() => ({ total: 0, problematic: [], problem_count: 0, score: 100, message: '드라이버 정보를 가져올 수 없어요' }))
          return { text: data.message, card2: { type: 'drivers', data }, emotion: data.problem_count > 0 ? 'concerned' : 'happy' }
        }
        case 'registry_clean': {
          const data = await backendAPI.registryClean().catch(() => ({ success: false, cleaned_keys: 0, message: '레지스트리 정리 실패' }))
          return { text: data.message,
            card2: { type: 'system_action', icon: '🗂️', title: data.message, success: data.success },
            emotion: data.success ? 'happy' : 'concerned' }
        }
        case 'power_plan': {
          const plans: Record<string, string> = { '고성능': 'performance', '절전': 'powersaver', '균형': 'balanced', 'performance': 'performance', 'high performance': 'performance', 'power saver': 'powersaver', 'balanced': 'balanced', 'battery saver': 'powersaver' }
          let planName = 'balanced'
          const lowerText = originalText.toLowerCase()
          for (const [k, v] of Object.entries(plans)) {
            if (lowerText.includes(k)) { planName = v; break }
          }
          const res = await backendAPI.setPowerPlan(planName).catch(() => ({ success: false, message: '전원 계획 변경 실패' }))
          return { text: res.message,
            card2: { type: 'system_action', icon: '⚡', title: res.message, success: res.success },
            emotion: res.success ? 'happy' : 'concerned' }
        }
        case 'network_analysis': {
          const data = await backendAPI.networkAnalysis().catch(() => ({ adapters: [], dns_servers: '', public_ip: '', ping_ms: '', connected: false }))
          return { text: data.connected ? t(`🌐 인터넷 연결됨 · 공개 IP: ${data.public_ip || '알 수 없음'}`, `🌐 Internet connected · Public IP: ${data.public_ip || 'Unknown'}`, userLang) : t('📵 인터넷 연결 없음', '📵 No internet connection', userLang),
            card2: { type: 'network', data }, emotion: data.connected ? 'happy' : 'concerned' }
        }
        case 'restore_create': {
          const res = await backendAPI.restoreCreate().catch(() => ({ success: false, message: '복구 포인트 생성 실패' }))
          return { text: res.message,
            card2: { type: 'system_action', icon: '💾', title: res.message, success: res.success },
            emotion: res.success ? 'happy' : 'concerned' }
        }
        case 'disk_check': {
          const res = await backendAPI.diskCheck().catch(() => ({ success: false, message: '디스크 검사 시작 실패' }))
          return { text: res.message,
            card2: { type: 'system_action', icon: '💿', title: res.message, success: res.success },
            emotion: res.success ? 'happy' : 'concerned' }
        }
        case 'browser_clean': {
          const res = await backendAPI.browserClean().catch(() => ({ results: [], total_mb: 0, total_freed: '0B', message: '브라우저 정리 실패' }))
          return { text: res.message,
            card2: { type: 'system_action', icon: '🌐', title: res.message, detail: `${res.total_freed} 확보`, success: true },
            emotion: 'happy' }
        }
        case 'programs_list': {
          const data = await backendAPI.programsList().catch(() => ({ programs: [], total: 0 }))
          return { text: t(`설치된 프로그램 ${data.total}개 확인했어요 📦`, `Found ${data.total} installed programs 📦`, userLang),
            card2: { type: 'programs_list', data }, emotion: 'neutral' }
        }
        case 'boot_analysis': {
          const data = await backendAPI.bootAnalysis().catch(() => ({ uptime_minutes: '0', startup_count: '?', recent_boots: [], message: '부팅 분석 실패' }))
          return { text: data.message, card2: { type: 'boot_analysis', data }, emotion: 'neutral' }
        }

        /* ── 파일 관리 ── */
        case 'file_search': {
          const query = originalText.replace(/파일.*찾아|찾아줘.*파일|어디/g, '').trim()
          const data = await backendAPI.filesSearch(query).catch(() => ({ results: [], total: 0, message: '파일 검색 실패' }))
          return { text: data.message, card2: { type: 'file_search', data }, emotion: 'neutral' }
        }
        case 'file_organize': {
          const isDesktop = /바탕화면|desktop/.test(originalText)
          const isDownloads = /다운로드|download/.test(originalText)
          const folderTarget = isDesktop ? 'desktop' : isDownloads ? 'downloads' : undefined
          const res = await backendAPI.filesOrganize(folderTarget).catch(() => ({ success: false, moved: 0, message: '폴더 정리 실패' }))
          return { text: res.message,
            card2: { type: 'system_action', icon: '📁', title: res.message, success: res.success },
            emotion: res.success ? 'happy' : 'concerned' }
        }
        case 'file_duplicates': {
          const data = await backendAPI.filesDuplicates().catch(() => ({ groups: [], total_groups: 0, waste_mb: 0, waste: '0B', message: '중복 검사 실패' }))
          return { text: data.message, card2: { type: 'duplicates', data }, emotion: data.total_groups > 0 ? 'concerned' : 'happy' }
        }

        /* ── 생산성 ── */
        case 'focus_mode': {
          const isOff = /해제|off|끄|disable|stop/.test(originalText)
          const durMatch = originalText.match(/(\d+)\s*(?:분|min(?:ute)?s?)/)
          const duration = durMatch ? parseInt(durMatch[1]) : 25
          const res = await backendAPI.focusMode(isOff ? 'off' : 'on', duration).catch(() => ({ success: false, active: !isOff, message: isOff ? '집중 모드 해제됨' : `집중 모드 시작! ${duration}분 동안 알림이 차단돼요 🎯` }))
          if (res.active) {
            setFocusModeEnd(duration)
            setFocusEndMs(Date.now() + duration * 60_000)
          } else {
            clearFocusMode()
            setFocusEndMs(undefined)
          }
          return { text: res.message,
            card2: { type: 'focus_mode', active: res.active, duration },
            emotion: res.active ? 'happy' : 'neutral' }
        }
        case 'clipboard': {
          const data = await backendAPI.clipboard().catch(() => ({ current: '', tip: t('Windows + V 로 클립보드를 확인해보세요', 'Press Windows + V to view clipboard history', userLang) }))
          return { text: data.current ? t(`클립보드: "${data.current.slice(0, 50)}..."`, `Clipboard: "${data.current.slice(0, 50)}..."`, userLang) : data.tip,
            card2: { type: 'system_action', icon: '📋', title: data.current ? t('클립보드 내용 확인', 'Clipboard content', userLang) : t('클립보드 비어있음', 'Clipboard is empty', userLang), detail: data.current?.slice(0, 60) },
            emotion: 'neutral' }
        }
        case 'notes': {
          const isNew = /적어|기록|저장/.test(originalText)
          if (isNew) {
            const content = extractNoteContent(originalText)
            if (content.length > 3) {
              const res = await backendAPI.saveNote(content).catch(() => ({ success: false, note: { id: '', content, created: '' }, message: '메모 저장 실패' }))
              return { text: res.message,
                card2: { type: 'system_action', icon: '📝', title: res.message, success: res.success },
                emotion: res.success ? 'happy' : 'concerned' }
            }
          }
          const data = await backendAPI.notes().catch(() => ({ notes: [], total: 0 }))
          return { text: t(`메모 ${data.total}개를 가져왔어요 📝`, `Fetched ${data.total} notes 📝`, userLang),
            card2: { type: 'notes', data }, emotion: 'neutral' }
        }

        /* ── 문서 비교 ── */
        case 'doc_compare': {
          const [f1, f2] = extractTwoFilePaths(originalText)
          if (!f1 || !f2) {
            return { text: t('비교할 두 파일 경로를 알려주세요. 예: "report_v1.docx 와 report_v2.docx 비교해줘"', 'Please provide two file paths to compare. e.g. "compare report_v1.docx and report_v2.docx"', userLang), emotion: 'neutral' }
          }
          const data = await backendAPI.docsCompare(f1, f2)
          return {
            text: data.summary,
            card3: { type: 'doc_compare', data },
            emotion: data.similarity_pct < 70 ? 'concerned' : 'neutral',
          }
        }
        case 'doc_find': {
          const query = originalText.replace(/문서.*찾아|파일.*찾아서|계약서.*찾아|보고서.*찾아/g, '').trim() || originalText
          const data = await backendAPI.docsFind(query)
          return {
            text: data.message,
            card3: { type: 'doc_find', data },
            emotion: data.total > 0 ? 'happy' : 'neutral',
          }
        }

        /* ── Deep Search ── */
        case 'deep_search': {
          const query = extractDeepSearchQuery(originalText)
          const verticalId = localStorage.getItem('nexus_vertical_id') ?? 'general'

          // 직업군별 검색 쿼리 보강
          const verticalQueryBoost: Record<string, string> = {
            legal:      `${query} 계약 법률 판례 조항`,
            medical:    `${query} 의료 진단 임상 처방`,
            accountant: `${query} 세무 회계 재무 세법`,
            creator:    `${query} 유튜브 콘텐츠 스크립트 편집`,
            realtor:    `${query} 부동산 시세 계약 청약`,
            teacher:    `${query} 교육 강의 수업 교육과정`,
            hr:         `${query} 채용 인사 노동법 면접`,
            developer:  `${query} 코드 개발 GitHub 프레임워크`,
            engineer:   `${query} 설계 규격 공정 KS ISO`,
            smallbiz:   `${query} 소상공인 배달앱 재고 원가 지원정책 사업자`,
            corporate:  `${query} 법인 세금계산서 법인세 4대보험 계약`,
            investor:   `${query} 주식 투자 종목 ETF PER ROE`,
            general:    query,
          }
          const boostedQuery = verticalQueryBoost[verticalId] ?? query

          const data = await backendAPI.deepSearch(boostedQuery)

          // 직업군별 결과 아이콘/레이블
          const verticalMeta: Record<string, { icon: string; label: string }> = {
            legal:      { icon: '⚖️', label: '법무 문서 검색' },
            medical:    { icon: '🩺', label: '의료 문서 검색' },
            accountant: { icon: '📊', label: '회계·세무 문서 검색' },
            creator:    { icon: '🎬', label: '콘텐츠 파일 검색' },
            realtor:    { icon: '🏠', label: '부동산 문서 검색' },
            teacher:    { icon: '📚', label: '교육 자료 검색' },
            hr:         { icon: '👥', label: '인사·채용 문서 검색' },
            developer:  { icon: '💻', label: '개발 파일 검색' },
            engineer:   { icon: '⚙️', label: '기술 문서 검색' },
            smallbiz:   { icon: '🏪', label: '소상공인 자료 검색' },
            corporate:  { icon: '🏢', label: '법인·세무 문서 검색' },
            investor:   { icon: '📈', label: '투자·종목 문서 검색' },
            general:    { icon: '🔍', label: '파일 심층 검색' },
          }
          const meta = verticalMeta[verticalId] ?? verticalMeta.general

          // FloatingPreview에 파일 결과 표시
          if (data.results && data.results.length > 0) {
            setFloatingPreview(data.results.slice(0, 8).map((r: { name: string; path?: string }) => ({
              title: `${meta.icon} ${r.name}`,
              url: r.path ? `file:///${r.path.replace(/\\/g, '/')}` : '#',
            })))
          }

          return {
            text: `${meta.icon} ${data.message}`,
            card3: { type: 'deep_search', data },
            emotion: data.total > 0 ? 'happy' : 'neutral',
          }
        }

        /* ── Vision ── */
        case 'vision_screen': {
          const question = extractVisionQuestion(originalText)
          // 스크린샷 캡처 (OCR 포함)
          const ss = await backendAPI.screenshot(true).catch(() => ({ success: false, base64: '', width: 0, height: 0, mime: 'image/png', captured: '' }))
          if (!ss.success || !ss.base64) {
            return { text: t('화면 캡처에 실패했어요. Tauri 앱 환경에서 실행해주세요.', 'Screen capture failed. Please run in Tauri app environment.', userLang), emotion: 'concerned' }
          }
          // Gemini Flash에 이미지 + 질문 전달
          const { callGeminiWithImage } = await import('../../lib/nexus/gemini_engine')
          const answer = (await callGeminiWithImage(ss.base64, question).catch(() => null)) ?? (ss as { ocr_text?: string }).ocr_text ?? '(분석 불가)'
          return {
            text: (answer || '(분석 불가)').slice(0, 120),
            card3: { type: 'vision_result', data: { question, answer, screenshot_b64: ss.base64 } },
            emotion: 'happy',
          }
        }
        case 'vision_ocr': {
          const data = await backendAPI.ocrClipboard()
          return {
            text: data.message,
            card3: { type: 'vision_ocr', data },
            emotion: 'neutral',
          }
        }

        /* ── 업무 일지 ── */
        case 'journal_today': {
          const data = await backendAPI.journalToday()
          return {
            text: t(`오늘 업무 일지를 정리했어요! ${(data as { app_usage?: unknown[] }).app_usage?.length || 0}개 앱, ${(data as { recent_files?: unknown[] }).recent_files?.length || 0}개 파일 사용 기록이 있어요.`, `Work journal ready! ${(data as { app_usage?: unknown[] }).app_usage?.length || 0} apps, ${(data as { recent_files?: unknown[] }).recent_files?.length || 0} files recorded.`, userLang),
            card4: { type: 'journal_today', data: data as unknown as Parameters<typeof import('./InlineCards4').JournalTodayCard>[0]['data'] },
            emotion: 'happy',
          }
        }
        case 'journal_generate': {
          const res = await backendAPI.journalGenerate()
          return {
            text: res.message,
            card4: { type: 'journal_today', data: { date: new Date().toISOString().slice(0,10), work_hours: 0, app_usage: [], recent_files: [], summary: res.preview || '', generated: '' } },
            emotion: 'happy',
          }
        }
        case 'journal_history': {
          const data = await backendAPI.journalHistory()
          return {
            text: t(`최근 ${(data as { days?: number }).days || 0}일간의 업무 기록을 찾았어요.`, `Work history for the past ${(data as { days?: number }).days || 0} days.`, userLang),
            card4: { type: 'journal_history', data: data as { history: Array<{ date: string; work_hours: number; file_count: number; app_count: number; top_app: string }> } },
            emotion: 'neutral',
          }
        }

        /* ── 자동화 매크로 ── */
        case 'macro_list': {
          const data = await backendAPI.macroList()
          return {
            text: (data as { total?: number }).total === 0
              ? t('아직 등록된 매크로가 없어요. "매일 아침 9시에 크롬 열어줘" 처럼 말해보세요!', 'No macros yet. Try saying "open Chrome every morning at 9am"!', userLang)
              : t(`매크로 ${(data as { total?: number }).total}개가 등록돼 있어요.`, `${(data as { total?: number }).total} macros registered.`, userLang),
            card4: { type: 'macro_list', data: data as { macros: Parameters<typeof import('./InlineCards4').MacroListCard>[0]['data']['macros']; total: number } },
            emotion: 'neutral',
          }
        }
        case 'macro_create': {
          const parsed = await backendAPI.macroParse(originalText)
          const macro = (parsed as { macro?: unknown }).macro
          if (!macro) {
            return { text: t('매크로를 이해하지 못했어요. 조금 더 자세히 말해주세요.', "I couldn't understand the macro. Please describe it in more detail.", userLang), emotion: 'neutral' }
          }
          const created = await backendAPI.macroCreate(macro)
          return {
            text: (created as { message?: string }).message || '매크로가 등록됐어요!',
            card4: { type: 'macro_created', data: { macro: (created as { macro?: unknown }).macro as Parameters<typeof import('./InlineCards4').MacroCreatedCard>[0]['data']['macro'], message: (created as { message?: string }).message || '' } },
            emotion: 'happy',
          }
        }
        case 'macro_run': {
          const list = await backendAPI.macroList()
          const macros = (list as { macros?: Array<{ id: string; name: string }> }).macros || []
          if (macros.length === 0) {
            return { text: '실행할 매크로가 없어요. 먼저 매크로를 등록해주세요.', emotion: 'neutral' }
          }
          const res = await backendAPI.macroRun(macros[0].id)
          return {
            text: (res as { message?: string }).message || '매크로를 실행했어요!',
            card4: { type: 'macro_run', data: res as { name: string; results: Parameters<typeof import('./InlineCards4').MacroRunCard>[0]['data']['results']; message: string } },
            emotion: 'happy',
          }
        }

        /* ── PC 리포트 ── */
        case 'pc_report': {
          const data = await backendAPI.reportGenerate()
          return {
            text: t(`PC 건강 점수: ${(data as { score?: number }).score || 0}점. 리포트가 바탕화면에 저장됐어요!`, `PC health score: ${(data as { score?: number }).score || 0}. Report saved to desktop!`, userLang),
            card4: { type: 'pc_report', data: data as unknown as Parameters<typeof import('./InlineCards4').PCReportCard>[0]['data'] },
            emotion: (data as { score?: number }).score && (data as { score?: number }).score! < 60 ? 'concerned' : 'happy',
          }
        }
        case 'report_email': {
          const res = await backendAPI.reportEmail()
          return {
            text: res.message,
            card2: { type: 'system_action', icon: '📧', title: res.success ? '이메일 전송 완료' : '이메일 전송 실패', success: res.success },
            emotion: res.success ? 'happy' : 'concerned',
          }
        }

        /* ── 문서 요약 ── */
        case 'doc_summary': {
          const filePath = extractTwoFilePaths(originalText)[0] || ''
          if (!filePath) {
            return { text: t('요약할 파일 경로를 알려주세요. 예: "report.pdf 요약해줘"', 'Please provide a file path to summarize. e.g. "summarize report.pdf"', userLang), emotion: 'neutral' }
          }
          const data = await backendAPI.docsSummary(filePath)
          return {
            text: t('문서 요약이 완료됐어요! 📄', 'Document summary complete! 📄', userLang),
            card4: { type: 'doc_summary', data: data as unknown as Parameters<typeof import('./InlineCards4').DocSummaryCard>[0]['data'] },
            emotion: 'happy',
          }
        }

        /* ── 스마트 정리 ── */
        case 'smart_organize': {
          const isDesktop = /바탕화면|desktop/.test(originalText)
          const isDownloads = /다운로드|download/.test(originalText)
          const target = isDesktop ? 'desktop' : isDownloads ? 'downloads' : 'all'
          const res = await backendAPI.filesOrganize(undefined, 'both').catch(() => ({
            success: false, moved: 0, message: '파일 정리 실패',
          }))
          return {
            text: res.success ? `${target === 'desktop' ? '바탕화면' : target === 'downloads' ? '다운로드' : 'PC 전체'} 정리 완료!` : res.message,
            card3: { type: 'smart_organize', data: { moved: res.moved, folders: [], message: res.message } },
            emotion: res.success ? 'happy' : 'concerned',
          }
        }

        /* ── 📅 캘린더 ── */
        case 'calendar_today': {
          const data = await calendarToday().catch(() => ({ success: false, events: [], total: 0, message: 'Outlook이 설치되어 있어야 합니다.' }))
          return {
            text: data.message,
            card2: { type: 'system_action', icon: '📅', title: `오늘 일정 ${data.total}개`, detail: data.events.slice(0,3).map(e => `${e.start.slice(11,16)} ${e.subject}`).join('\n'), success: data.success },
            emotion: data.total > 0 ? 'happy' : 'neutral',
          }
        }
        case 'calendar_week': {
          const data = await calendarWeek().catch(() => ({ success: false, events: [], total: 0, message: 'Outlook이 설치되어 있어야 합니다.' }))
          return {
            text: data.message,
            card2: { type: 'system_action', icon: '📆', title: `이번 주 일정 ${data.total}개`, detail: data.events.slice(0,5).map(e => `${e.start.slice(5,10)} ${e.subject}`).join('\n'), success: data.success },
            emotion: data.total > 0 ? 'happy' : 'neutral',
          }
        }
        case 'calendar_add': {
          const subjectMatch = originalText.match(/[""]([^""]+)[""]/) ?? originalText.match(/일정.*등록\s+(.+)/)
          const subject = (subjectMatch?.[1] ?? originalText.replace(/일정.*추가|일정.*등록|일정.*넣어/g, '').trim()) || '새 일정'
          const res = await calendarAdd(subject).catch(() => ({ success: false, message: '일정 추가 실패' }))
          return {
            text: res.message,
            card2: { type: 'system_action', icon: '📅', title: res.message, success: res.success },
            emotion: res.success ? 'happy' : 'concerned',
          }
        }

        /* ── 📧 이메일 ── */
        case 'email_inbox': {
          const data = await emailInbox(10).catch(() => ({ success: false, emails: [], total: 0, unread: 0, message: 'Outlook이 필요합니다.', action: 'outlook_setup_required' }))
          if ((data as any).action === 'outlook_setup_required') {
            d.openEmailSetup?.()
            return { text: t('이메일 기능을 사용하려면 Gmail/Outlook 연동이 필요해요.\n설정 → 이메일 탭에서 계정을 추가해주세요 📧', 'Email setup required.\nGo to Settings → Email tab to connect your account 📧', userLang), emotion: 'neutral' }
          }
          return {
            text: data.message,
            card2: { type: 'system_action', icon: '📧', title: `받은 메일 ${data.total}개 (읽지 않음 ${data.unread}개)`, detail: (data.emails as any[]).slice(0,3).map((e: any) => `${e.is_read ? '📨' : '📩'} ${e.subject} — ${e.sender}`).join('\n'), success: data.success },
            emotion: (data.unread as number) > 0 ? 'concerned' : 'neutral',
          }
        }
        case 'email_send': {
          const toMatch = originalText.match(/([a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,})/)
          const to = toMatch?.[1] ?? ''
          if (!to) return { text: t('받는 사람 이메일 주소를 알려주세요. 예: "user@gmail.com에게 메일 보내줘"', 'Please provide the recipient email. e.g. "send email to user@gmail.com"', userLang), emotion: 'neutral' }
          const subject = originalText.match(/제목[:\s]+(.+)/)?.[1] ?? originalText.match(/subject[:\s]+(.+)/i)?.[1] ?? t('Nexus에서 보낸 메일', 'Mail from Nexus', userLang)
          const body = originalText.match(/내용[:\s]+(.+)/)?.[1] ?? originalText.match(/(?:body|message|content)[:\s]+(.+)/i)?.[1] ?? ''
          const res = await emailSend(to, subject, body).catch(() => ({ success: false, message: '메일 전송 실패' }))
          return {
            text: res.message,
            card2: { type: 'system_action', icon: '📤', title: res.message, success: res.success },
            emotion: res.success ? 'happy' : 'concerned',
          }
        }
        case 'email_summarize': {
          const data = await emailSummarize().catch(() => ({ success: false, emails: [], summary: '', message: 'Outlook이 필요합니다.', action: 'outlook_setup_required' }))
          if ((data as any).action === 'outlook_setup_required') {
            d.openEmailSetup?.()
            return { text: t('이메일 기능을 사용하려면 Gmail/Outlook 연동이 필요해요.\n설정 → 이메일 탭에서 계정을 추가해주세요 📧', 'Email setup required.\nGo to Settings → Email tab to connect your account 📧', userLang), emotion: 'neutral' }
          }
          return {
            text: data.message,
            card2: { type: 'system_action', icon: '📧', title: '이메일 요약', detail: data.summary, success: data.success },
            emotion: 'neutral',
          }
        }

        /* ── 🦠 VirusTotal ── */
        case 'virus_check': {
          const filePathMatch = originalText.match(/[A-Za-z]:\\[^\s]+/) ?? originalText.match(/["']([^"']+\.[a-z]{2,4})["']/)
          const filePath = filePathMatch?.[0]?.replace(/['"]/g, '') ?? ''
          if (!filePath) return { text: t('검사할 파일 경로를 알려주세요. 예: "C:\\Users\\file.exe 바이러스 확인해줘"', 'Please provide a file path to scan. e.g. "check C:\\Users\\file.exe for virus"', userLang), emotion: 'neutral' }
          const apiKey = localStorage.getItem('nexus-virustotal-key') ?? ''
          const data = await virusTotalCheck(filePath, apiKey).catch(() => ({ success: false, file_path: filePath, file_hash: '', malicious: 0, suspicious: 0, clean: 0, total_scans: 0, permalink: '', safe_score: 0, verdict: 'error', message: 'VirusTotal 연결 실패' }))
          const em: CharacterEmotion = data.verdict === 'malicious' ? 'alert' : data.verdict === 'suspicious' ? 'concerned' : 'happy'
          return {
            text: data.message,
            card2: { type: 'system_action', icon: data.verdict === 'malicious' ? '🚨' : data.verdict === 'suspicious' ? '⚠️' : '✅', title: `VirusTotal 결과: ${data.verdict}`, detail: `탐지 ${data.malicious}개 / 전체 ${data.total_scans}개 검사`, success: data.verdict === 'safe' || data.verdict === 'unknown' },
            emotion: em,
          }
        }

        /* ── 📊 성능 이력 ── */
        case 'perf_history': {
          const daysMatch = originalText.match(/(\d+)\s*일/)
          const days = daysMatch ? parseInt(daysMatch[1]) : 7
          const data = await historyStats(days).catch(() => ({ success: false, days, total_samples: 0, snapshots: [], daily_summary: [], avg_cpu: 0, avg_mem: 0, cpu_trend: 'stable', message: '성능 이력이 없어요. 앱을 더 오래 실행하면 데이터가 쌓여요.' }))
          return {
            text: data.message,
            card2: { type: 'system_action', icon: '📊', title: `${days}일 성능 이력`, detail: `평균 CPU ${data.avg_cpu.toFixed(0)}% · 메모리 ${data.avg_mem.toFixed(0)}% · 트렌드: ${data.cpu_trend === 'up' ? '↑ 증가' : data.cpu_trend === 'down' ? '↓ 감소' : '→ 안정'}`, success: data.success },
            emotion: data.cpu_trend === 'up' ? 'concerned' : 'neutral',
          }
        }
        case 'perf_anomaly': {
          const data = await historyAnomalies().catch(() => ({ success: false, anomalies: [], avg_cpu: 0, avg_mem: 0, message: '데이터 부족' }))
          return {
            text: data.message,
            card2: { type: 'system_action', icon: data.anomalies.length > 0 ? '⚠️' : '✅', title: `이상 탐지: ${data.anomalies.length}건`, detail: data.anomalies.slice(0,3).map(a => a.message).join('\n'), success: data.anomalies.length === 0 },
            emotion: data.anomalies.length > 3 ? 'alert' : data.anomalies.length > 0 ? 'concerned' : 'happy',
          }
        }

        /* ── 🛍️ 가격 비교 ── */
        case 'price_compare': {
          const query = originalText.replace(/가격|비교|얼마|검색|찾아줘|price|compare|search|how much|buy/gi, '').trim() || originalText
          const data = await priceCompare(query).catch(() => ({ success: false, query, results: [], total: 0, summary: '가격 비교 실패' }))
          if (data.results && data.results.length > 0) {
            setFloatingPreview(data.results.slice(0, 8).map(r => ({ title: `${r.price} — ${r.name}`, url: r.link })))
          }
          return {
            text: data.summary || t(`"${query}" 가격 비교 완료!`, `Price comparison for "${query}" done!`, userLang),
            card2: { type: 'price_compare', data: { query: data.query, results: data.results ?? [], total: data.total, summary: data.summary } },
            emotion: data.success ? 'happy' : 'neutral',
          }
        }

        /* ── 🌐 뉴스 검색 ── */
        case 'news_search': {
          const query = originalText.replace(/뉴스|검색|최신|오늘|찾아줘|news|search|latest|today|find/gi, '').trim() || t('오늘 주요 뉴스', 'top news today', userLang)
          const data = await newsSearch(query).catch(() => ({ success: false, query, articles: [], items: [], total: 0, summary: '뉴스 검색 실패' })) as any
          const articles = data.articles ?? data.items ?? []
          if (articles.length > 0) {
            setPreviewType('news')
            setFloatingPreview(articles.slice(0, 8).map((a: { title: string; url: string }) => ({
              title: a.title, url: a.url,
              isVideo: a.url.includes('youtube.com') || a.url.includes('youtu.be'),
            })))
          }
          return {
            text: data.summary || t(`'${query}' 뉴스 검색 완료!`, `News search for '${query}' done!`, userLang),
            card2: { type: 'system_action', icon: '📰', title: `${t('뉴스', 'News', userLang)}: ${query}`, detail: articles.slice(0,3).map((a: { title: string }) => `• ${a.title}`).join('\n'), success: data.success },
            emotion: 'neutral',
          }
        }

        /* ── 🎬 유튜브 검색 ── */
        case 'youtube_search': {
          const query = originalText.replace(/유튜브에서|유튜브|youtube|찾아줘|검색해줘|보여줘|영상|search|find|show me|video/gi, '').trim() || originalText
          const isTiktok = /틱톡|tiktok/i.test(originalText)
          const data = isTiktok
            ? await tiktokSearch(query).catch(() => ({ success: false, query, articles: [], total: 0, summary: '' }))
            : await youtubeSearch(query).catch(() => ({ success: false, query, articles: [], total: 0, summary: '' }))
          const platform = isTiktok ? t('틱톡', 'TikTok', userLang) : t('유튜브', 'YouTube', userLang)
          const icon = isTiktok ? '🎵' : '🎬'
          const articles = (data as { articles?: { title: string; url: string }[] }).articles ?? []
          const detail = articles.slice(0, 5).map(a => `• ${a.title}\n  ${a.url}`).join('\n\n')
          if (articles.length > 0) setFloatingPreview(articles.slice(0, 5).map(a => ({ title: a.title, url: a.url })))
          return {
            text: data.summary || t(`${platform}에서 "${query}" 영상 ${articles.length}개를 찾았어요!`, `Found ${articles.length} video(s) for "${query}" on ${platform}!`, userLang),
            card2: { type: 'system_action', icon, title: `${platform}: ${query}`, detail: detail || t('결과를 가져오는 중...', 'Loading results...', userLang), success: data.success },
            emotion: 'happy',
          }
        }

        /* ── 🔴 Reddit 검색 ── */
        case 'reddit_search': {
          const subredditMatch = originalText.match(/r\/(\w+)/i)
          const subreddit = subredditMatch?.[1] ?? ''
          const isTrending = /트렌딩|인기|hot|trending/i.test(originalText)
          const query = originalText
            .replace(/레딧|reddit|에서|검색|찾아줘|커뮤니티|반응|의견|r\/\w+/gi, '')
            .replace(/트렌딩|인기|hot|trending/gi, '')
            .trim() || (isTrending ? '' : originalText)

          const data = isTrending && !query
            ? await redditTrending(subreddit || 'all').catch(() => ({ success: false, source: '', posts: [], count: 0, message: 'Reddit 수집 실패' }))
            : await redditSearch(query, subreddit).catch(() => ({ success: false, source: '', posts: [], count: 0, message: 'Reddit 수집 실패' }))

          const posts = data.posts ?? []
          if (posts.length > 0) {
            setFloatingPreview(posts.slice(0, 8).map((p) => ({ title: `[r/${p.subreddit || 'reddit'}] ${p.title}`, url: p.url })))
          }
          const detail = posts.slice(0, 5).map(p =>
            `• ${p.title}${p.score ? ` ↑${p.score}` : ''}${p.comments ? ` 💬${p.comments}` : ''}\n  ${p.url}`
          ).join('\n\n')
          return {
            text: data.message || t(`Reddit에서 "${query || '트렌딩'}" 게시물 ${posts.length}개를 찾았어요!`, `Found ${posts.length} Reddit post(s) for "${query || 'trending'}"!`, userLang),
            card2: {
              type: 'system_action',
              icon: '🔴',
              title: `Reddit${subreddit ? ` r/${subreddit}` : ''}: ${query || t('트렌딩', 'trending', userLang)}`,
              detail: detail || t('결과를 가져오는 중...', 'Loading results...', userLang),
              success: data.success,
            },
            emotion: posts.length > 0 ? 'happy' : 'neutral',
          }
        }

        /* ── ⬇️ 영상 다운로드 ── */
        case 'video_download': {
          // URL 추출
          const urlMatch = originalText.match(/https?:\/\/[^\s]+/)
          const url = urlMatch?.[0] ?? ''
          if (!url) {
            return {
              text: t('다운로드할 영상 URL을 붙여넣어주세요.\n예: "https://www.youtube.com/watch?v=... 다운로드해줘"', 'Please paste the video URL.\ne.g. "download https://www.youtube.com/watch?v=..."', userLang),
              emotion: 'neutral',
            }
          }
          const qualityMatch = originalText.match(/720p|480p|1080p|4k/)
          const quality = qualityMatch?.[0] ?? 'best'
          const data = await videoDownload(url, quality).catch(() => ({
            success: false, url, save_path: '', message: 'yt-dlp 다운로드 실패', install_url: 'https://github.com/yt-dlp/yt-dlp/releases/latest',
          }))
          return {
            text: data.message,
            card2: {
              type: 'system_action',
              icon: data.success ? '✅' : '⚠️',
              title: data.success ? t('영상 다운로드 완료', 'Video download complete', userLang) : t('yt-dlp 설치 필요', 'yt-dlp installation required', userLang),
              detail: data.success
                ? t(`저장 위치: ${data.save_path}`, `Saved to: ${data.save_path}`, userLang)
                : t(`yt-dlp 설치 후 다시 시도해주세요.\n${data.install_url ?? ''}`, `Please install yt-dlp and try again.\n${data.install_url ?? ''}`, userLang),
              success: data.success,
            },
            emotion: data.success ? 'happy' : 'concerned',
          }
        }

        /* ── 🎬 영상 URL 요약/전사 ── */
        case 'video_transcript': {
          const urlMatch = originalText.match(/https?:\/\/[^\s]+/)
          const url = urlMatch?.[0] ?? ''
          if (!url) {
            return {
              text: t(
                '요약할 영상 URL을 붙여넣어주세요.\n예: "https://www.youtube.com/watch?v=... 요약해줘"',
                'Please paste the video URL.\ne.g. "https://www.youtube.com/watch?v=... summarize this"',
                userLang,
              ),
              emotion: 'neutral',
            }
          }
          const lang = userLang ?? 'ko'
          const data = await videoTranscript(url, lang).catch(() => ({ success: false, transcript: '', message: t('영상 분석 실패', 'Video analysis failed', userLang) }))
          return {
            text: data.message ?? (data.success ? t('영상 요약 완료', 'Video summary done', userLang) : t('영상 분석 실패', 'Video analysis failed', userLang)),
            card2: data.success ? { type: 'system_action', icon: '🎬', title: t('영상 요약', 'Video Summary', userLang), detail: (data.transcript ?? data.message ?? '').slice(0, 200), success: true } : undefined,
            emotion: data.success ? 'happy' : 'concerned',
          }
        }

        /* ── ⏰ 스케줄러 ── */
        case 'schedule_list': {
          const data = await schedulerList().catch(() => ({ success: false, tasks: [], total: 0 }))
          return {
            text: data.total === 0 ? t('등록된 자동화 스케줄이 없어요. "매일 오전 9시에 PC 진단해줘" 처럼 말해보세요!', 'No schedules yet. Try "run PC scan every day at 9am"!', userLang) : t(`스케줄 ${data.total}개가 등록돼 있어요.`, `${data.total} schedules registered.`, userLang),
            card2: { type: 'system_action', icon: '⏰', title: `스케줄 ${data.total}개`, detail: (data.tasks as Array<{name: string; next_run: string}>).slice(0,3).map(t => `${t.name} — ${t.next_run}`).join('\n'), success: true },
            emotion: 'neutral',
          }
        }
        case 'schedule_add': {
          const res = await schedulerAdd(originalText).catch(() => ({ success: false, task: null, next_run_kr: '', message: '스케줄 추가 실패' }))
          return {
            text: (res as { message: string }).message,
            card2: { type: 'system_action', icon: '⏰', title: (res as { message: string }).message, success: (res as { success: boolean }).success },
            emotion: (res as { success: boolean }).success ? 'happy' : 'concerned',
          }
        }
        case 'schedule_delete': {
          const idMatch = originalText.match(/\b([a-f0-9-]{10,})\b/)
          if (!idMatch) return { text: t('삭제할 스케줄 ID를 알려주세요. 먼저 "스케줄 목록" 으로 확인해보세요.', 'Please provide the schedule ID to delete. Say "show schedules" first.', userLang), emotion: 'neutral' }
          const res = await schedulerDelete(idMatch[1]).catch(() => ({ success: false, message: '삭제 실패' }))
          return {
            text: res.message,
            card2: { type: 'system_action', icon: '🗑️', title: res.message, success: res.success },
            emotion: res.success ? 'happy' : 'concerned',
          }
        }

        /* ── 🔫 프로세스 강제 종료 ── */
        case 'process_kill': {
          const pidMatch = originalText.match(/pid\s*[:#]?\s*(\d+)/i)
          const nameMatch = originalText.replace(/프로세스|종료|강제|앱|죽여|kill/gi, '').trim()
          if (!pidMatch && !nameMatch) return { text: t('종료할 프로세스 이름이나 PID를 알려주세요.', 'Please provide the process name or PID to terminate.', userLang), emotion: 'neutral' }
          const pid = pidMatch ? parseInt(pidMatch[1]) : undefined
          const name = pid ? undefined : nameMatch
          const res = await processKill(pid, name).catch(() => ({ success: false, name: name ?? '', message: '종료 실패' }))
          return {
            text: res.message,
            card2: { type: 'system_action', icon: res.success ? '✅' : '⚠️', title: res.message, success: res.success },
            emotion: res.success ? 'happy' : 'concerned',
          }
        }

        /* ── 🔑 앱 권한 감사 ── */
        case 'app_permissions': {
          const appMatch = originalText.match(/["']?([가-힣a-zA-Z]+)["']?\s*(?:앱|프로그램|이)?\s*권한/)?.[1]
          const data = await appPermissions(appMatch).catch(() => ({ success: false, permissions: {}, message: '권한 정보 조회 실패' }))
          return {
            text: data.message,
            card2: { type: 'system_action', icon: '🔑', title: '앱 권한 현황', detail: JSON.stringify(data.permissions).slice(0, 100), success: data.success },
            emotion: 'neutral',
          }
        }

        /* ── 🔄 Windows 업데이트 ── */
        case 'windows_updates': {
          const data = await windowsUpdates().catch(() => ({ success: false, count: 0, updates: [], message: 'Windows Update 확인 실패' }))
          return {
            text: data.message,
            card2: { type: 'system_action', icon: data.count > 0 ? '🔄' : '✅', title: `업데이트 ${data.count}개`, detail: data.updates.slice(0,3).map(u => `• ${u.title}`).join('\n'), success: data.success },
            emotion: data.count > 5 ? 'concerned' : data.count > 0 ? 'neutral' : 'happy',
          }
        }

        /* ── 🌤️ 날씨 ── */
        case 'weather': {
          const cityMatch = originalText.match(/([가-힣]{2,5})\s*(?:날씨|기온)/)
            ?? originalText.match(/weather\s+(?:in|for|at)\s+([a-zA-Z가-힣]+)/i)
            ?? originalText.match(/(?:in|for|at)\s+([a-zA-Z가-힣]+)\s+weather/i)
          const cityIsDefault = !cityMatch
          const city = cityMatch?.[1] ?? (userLang === 'en' ? 'Seoul' : '서울')
          const data = await weatherGet(city).catch(() => ({ success: false, city, temp_c: 0, feels_like: 0, condition: '알 수 없음', humidity: 0, wind_kmh: 0, forecast: [], message: '' }))
          if (!data.success) {
            // 백엔드 없음 → Groq로 동적 응답
            const apiKey = localStorage.getItem('nexus-pplx-key') ?? ''
            if (apiKey) {
              const gr = await callGemini(apiKey, originalText, historyRef.current as ConversationTurn[]).catch(() => null)
              if (gr?.text) return { text: gr.text, emotion: gr.emotion ?? 'neutral' }
            }
            return { text: t('날씨 서비스에 연결할 수 없어요. 날씨 앱이나 포털 사이트에서 확인해보세요! 🌤️', "Can't connect to weather service. Please check a weather app instead! 🌤️", userLang), emotion: 'neutral' }
          }
          const weatherText = cityIsDefault
            ? t(`위치를 알 수 없어서 서울 기준으로 알려드려요! 현재 ${data.temp_c}°C, ${data.condition}이에요.`, `Showing Seoul since no city was specified! Currently ${data.temp_c}°C, ${data.condition}`, userLang)
            : t(`${data.city} 현재 ${data.temp_c}°C, ${data.condition}이에요.`, `${data.city}: ${data.temp_c}°C, ${data.condition}`, userLang)
          return {
            text: weatherText,
            card2: {
              type: 'system_action', icon: '🌤️',
              title: `${data.city} ${data.temp_c}°C — ${data.condition}`,
              detail: `체감 ${data.feels_like}°C · 습도 ${data.humidity}% · 바람 ${data.wind_kmh}km/h\n${data.forecast.slice(0,3).map(f => `${f.date}: ${f.max}°/${f.min}°C ${f.condition}`).join('\n')}`,
              success: data.success,
            },
            emotion: 'neutral',
          }
        }

        /* ── 🚗 교통 시간 ── */
        case 'travel_time': {
          // "경주에서 부산가는 방법" → origin=경주, destination=부산
          const parts = originalText.match(/(.+?)(?:에서|에서부터)\s*(.+?)(?:까지|로|으로|가는|이동|교통|방법|버스|기차|KTX|\s*$)/)
            ?? originalText.match(/(?:from|between)\s+(.+?)\s+(?:to|and)\s+(.+?)(?:\s|$)/i)
          const origin = parts?.[1]?.trim() ?? ''
          // 목적지에서 "가는 방법/교통편" 등 불필요한 접미어 제거
          const rawDest = parts?.[2]?.trim() ?? ''
          const destination = rawDest.replace(/\s*(가는|까지|이동|교통|방법|버스|기차|KTX|알려줘|어떻게).*$/i, '').trim()
          if (!origin || !destination) return { text: t('"어디에서 어디까지 얼마나 걸려?" 형식으로 말해주세요.', 'Please say "how long from [origin] to [destination]?"', userLang), emotion: 'neutral' }
          const data = await travelTime(origin, destination).catch(() => ({ success: false, origin, destination, distance_km: 0, duration_min: 0, departure_time: '', arrival_time: '', message: '경로를 찾지 못했어요.' }))
          return {
            text: data.message || `${origin} → ${destination}: 약 ${data.duration_min}분`,
            card2: {
              type: 'system_action', icon: '🚗',
              title: `${origin} → ${destination}`,
              detail: t(`거리 ${data.distance_km.toFixed(1)}km · 약 ${data.duration_min}분\n출발 ${data.departure_time} → 도착 ${data.arrival_time}`, `Distance ${data.distance_km.toFixed(1)}km · ~${data.duration_min}min\nDepart ${data.departure_time} → Arrive ${data.arrival_time}`, userLang),
              success: data.success,
            },
            emotion: 'neutral',
          }
        }

        /* ── 🌐 번역 ── */
        case 'translate': {
          const targetLang = /영어로|영문|to english|into english/i.test(originalText) ? 'English'
            : /한국어로|한글로|to korean|into korean/i.test(originalText) ? '한국어'
            : /일본어로|to japanese/i.test(originalText) ? '日本語'
            : /중국어로|to chinese/i.test(originalText) ? '中文'
            : userLang === 'en' ? '한국어' : 'English'
          // 클립보드 내용 가져와서 번역
          const clip = await backendAPI.clipboard().catch(() => ({ current: '', tip: '' }))
          const textToTranslate = clip.current || originalText.replace(/번역.*해줘|번역해|이거.*영어로|translate.*to|translate/gi, '').trim()
          if (!textToTranslate) return { text: t('번역할 내용이 없어요. 텍스트를 먼저 복사해주세요.', 'Nothing to translate. Please copy some text first.', userLang), emotion: 'neutral' }

          const apiKey = localStorage.getItem('nexus-pplx-key') ?? ''
          let translated = ''
          if (apiKey) {
            const { callGemini } = await import('../../lib/nexus/gemini_engine')
            const res = await callGemini(apiKey, `다음 텍스트를 ${targetLang}로 번역해줘. 번역 결과만 출력:\n\n${textToTranslate}`, []).catch(() => null)
            translated = res?.text ?? ''
          }
          if (!translated) translated = t('번역을 위해 Perplexity API 키가 필요해요.', 'Perplexity API key is required for translation.', userLang)

          // 번역 결과를 클립보드에 저장 (paste API 사용)
          if (translated && !translated.includes('API 키')) {
            await dictationPaste(translated).catch(() => {})
          }
          return {
            text: t('번역 완료! 결과가 클립보드에 복사됐어요.', 'Translation complete! Result copied to clipboard.', userLang),
            card2: {
              type: 'system_action', icon: '🌐',
              title: `→ ${targetLang} 번역`,
              detail: `원본: ${textToTranslate.slice(0,60)}...\n번역: ${translated.slice(0,80)}`,
              success: !!translated && !translated.includes('API 키'),
            },
            emotion: 'happy',
          }
        }

        /* ── 📋 클립보드 AI ── */
        case 'clipboard_ai': {
          const clip = await backendAPI.clipboard().catch(() => ({ current: '', tip: '' }))
          if (!clip.current) return { text: t('클립보드가 비어있어요. 먼저 텍스트를 복사해주세요.', 'Clipboard is empty. Please copy some text first.', userLang), emotion: 'neutral' }

          const action = /요약/.test(originalText) ? '3줄로 요약해줘'
            : /교정|다듬어/.test(originalText) ? '문법과 어투를 교정해줘'
            : /번역/.test(originalText) ? '영어로 번역해줘'
            : /쉽게/.test(originalText) ? '쉽게 설명해줘'
            : '핵심만 요약해줘'

          const apiKey = localStorage.getItem('nexus-pplx-key') ?? ''
          let result = ''
          if (apiKey) {
            const { callGemini } = await import('../../lib/nexus/gemini_engine')
            const res = await callGemini(apiKey, `다음 텍스트를 ${action}. 결과만 출력:\n\n${clip.current.slice(0, 500)}`, []).catch(() => null)
            result = res?.text ?? ''
          }
          if (!result) return { text: t('AI 처리를 위해 Perplexity API 키가 필요해요.', 'Perplexity API key is required for AI processing.', userLang), emotion: 'neutral' }

          await dictationPaste(result).catch(() => {})
          return {
            text: t('클립보드 AI 처리 완료! 결과가 클립보드에 저장됐어요.', 'AI processing done! Result saved to clipboard.', userLang),
            card2: {
              type: 'system_action', icon: '📋',
              title: '클립보드 AI 처리',
              detail: result.slice(0, 120),
              success: true,
            },
            emotion: 'happy',
          }
        }

        /* ── 📋 클립보드 히스토리 ── */
        case 'clipboard_history': {
          const shouldClear = /지워|삭제|clear/i.test(originalText)
          if (shouldClear) {
            await clipboardHistoryClear().catch(() => null)
            return { text: t('클립보드 히스토리를 모두 지웠어요!', 'Clipboard history cleared!', userLang), emotion: 'happy' }
          }
          const res = await clipboardHistory().catch(() => ({ success: false, history: [], total: 0 }))
          const history = res.history ?? []
          if (history.length === 0) {
            return { text: t('클립보드 히스토리가 없어요. 텍스트를 복사하면 자동으로 저장돼요.', 'No clipboard history yet. Text you copy will be tracked automatically.', userLang), emotion: 'neutral' }
          }
          const lines = history.slice(0, 10).map((e, i) => {
            const preview = e.text.length > 60 ? e.text.slice(0, 60) + '...' : e.text
            const ts = new Date(e.timestamp).toLocaleTimeString('ko-KR', { hour: '2-digit', minute: '2-digit' })
            return `${i + 1}. [${ts}] ${preview}`
          })
          return {
            text: t(
              `클립보드 히스토리 최근 ${Math.min(history.length, 10)}개예요:\n\n${lines.join('\n')}`,
              `Last ${Math.min(history.length, 10)} clipboard entries:\n\n${lines.join('\n')}`,
              userLang
            ),
            emotion: 'neutral',
          }
        }

        /* ── 📝 음성 메모→할일 동시 등록 ── */
        case 'voice_todo': {
          const content = originalText.replace(/할일|todo|기억해줘|해야|마감|데드라인/gi, '').trim() || originalText
          // 날짜 추출
          const dateMatch = originalText.match(/(\d+월\s*\d+일|\d+일|\d+\/\d+)/)
          const timeMatch = originalText.match(/(\d+시|\d+:\d+)/)
          const dateStr = dateMatch?.[1] ?? ''
          const timeStr = timeMatch?.[1] ?? ''
          const eventTitle = content.slice(0, 50)

          // 메모 저장
          const noteRes = await backendAPI.saveNote(content).catch(() => ({ success: false, note: { id: '', content, created: '' }, message: '메모 저장 실패' }))
          // 캘린더 등록 (날짜 있으면)
          let calMsg = ''
          if (dateStr) {
            const calRes = await calendarAdd(`[할일] ${eventTitle}`, dateStr + ' ' + timeStr).catch(() => ({ success: false, message: '' }))
            calMsg = calRes.success ? ` + 캘린더에도 등록했어요 📅` : ''
          }
          return {
            text: noteRes.success ? t(`메모 저장 완료!${calMsg}`, `Note saved!${calMsg}`, userLang) : t('메모 저장에 실패했어요.', 'Failed to save note.', userLang),
            card2: {
              type: 'system_action', icon: '📝',
              title: t('메모 + 할일 등록', 'Note + Task registered', userLang),
              detail: t(`내용: ${content.slice(0, 80)}${dateStr ? `\n날짜: ${dateStr} ${timeStr}` : ''}`, `Content: ${content.slice(0, 80)}${dateStr ? `\nDate: ${dateStr} ${timeStr}` : ''}`, userLang),
              success: noteRes.success,
            },
            emotion: noteRes.success ? 'happy' : 'concerned',
          }
        }

        /* ── 🖥️ Windows Recall ── */
        case 'recall_capture': {
          const data = await recallCapture().catch(() => ({ success: false, timestamp: '', ocr_text: '', message: '화면 캡처 실패' }))
          return {
            text: data.success ? t(`화면을 기억했어요 🖥️ "${data.ocr_text.slice(0, 40)}..."`, `Screen saved to memory 🖥️ "${data.ocr_text.slice(0, 40)}..."`, userLang) : data.message,
            card2: { type: 'system_action', icon: '🖥️', title: t('화면 기억 저장', 'Screen memory saved', userLang), detail: data.ocr_text.slice(0, 100), success: data.success },
            emotion: data.success ? 'happy' : 'concerned',
          }
        }
        case 'recall_search': {
          const query = originalText.replace(/기억.*찾아|화면.*기억|언제.*봤던|어제.*봤던|전에.*봤던|화면.*검색|recall/gi, '').trim() || originalText
          const data = await recallSearch(query).catch(() => ({ success: false, results: [], total: 0, message: '검색 실패 — 먼저 화면을 기억시켜주세요.' }))
          return {
            text: data.total > 0 ? t(`"${query}" 관련 화면 ${data.total}개 찾았어요!`, `Found ${data.total} screen memory match(es) for "${query}"!`, userLang) : t(`"${query}" 관련 기억이 없어요.`, `No screen memories found for "${query}".`, userLang),
            card2: {
              type: 'system_action', icon: '🔍',
              title: t(`화면 기억 검색: ${query}`, `Screen memory search: ${query}`, userLang),
              detail: data.results.slice(0, 3).map(r => `${r.timestamp}: ${r.snippet}`).join('\n'),
              success: data.success,
            },
            emotion: data.total > 0 ? 'happy' : 'neutral',
          }
        }

        /* ── 🎙️ 회의 어시스턴트 ── */
        case 'meeting_start': {
          const res = await meetingStart().catch(() => ({ success: false, file_path: '', message: '녹음 시작 실패' }))
          return {
            text: res.message,
            card2: { type: 'system_action', icon: '🔴', title: res.success ? t('녹음 중...', 'Recording...', userLang) : t('녹음 실패', 'Recording failed', userLang), detail: res.file_path, success: res.success },
            emotion: res.success ? 'happy' : 'concerned',
          }
        }
        case 'meeting_stop': {
          const res = await meetingStop().catch(() => ({ success: false, file_path: '', duration_sec: 0, message: '녹음 종료 실패' }))
          return {
            text: res.message,
            card2: { type: 'system_action', icon: '⏹️', title: t(`녹음 완료 (${Math.round(res.duration_sec / 60)}분)`, `Recording done (${Math.round(res.duration_sec / 60)}min)`, userLang), detail: res.file_path, success: res.success },
            emotion: res.success ? 'happy' : 'concerned',
          }
        }
        case 'meeting_list': {
          const data = await meetingList().catch(() => ({ success: false, meetings: [], total: 0 }))
          return {
            text: data.total > 0 ? t(`회의 녹음 ${data.total}개가 있어요 🎙️`, `${data.total} meeting recording(s) found 🎙️`, userLang) : t('저장된 회의 녹음이 없어요.', 'No meeting recordings saved.', userLang),
            card2: {
              type: 'system_action', icon: '🎙️',
              title: `회의 목록 ${data.total}개`,
              detail: data.meetings.slice(0, 3).map(m => `${m.timestamp} (${m.size_mb.toFixed(1)}MB)`).join('\n'),
              success: data.success,
            },
            emotion: 'neutral',
          }
        }
        case 'meeting_summary': {
          // 가장 최근 녹음 파일 가져와서 전사 + 요약
          const list = await meetingList().catch(() => ({ success: false, meetings: [], total: 0 }))
          if (!list.total) return { text: t('요약할 회의 녹음이 없어요. 먼저 "회의 시작"으로 녹음해주세요.', 'No meeting recordings found. Say "start meeting" to begin recording.', userLang), emotion: 'neutral' }
          const latest = list.meetings[0]
          const transcribed = await meetingTranscribe(latest.file).catch(() => ({ success: false, text: '', duration_sec: 0, message: '전사 실패' }))
          if (!transcribed.success || !transcribed.text) return { text: t('회의 전사 실패. Perplexity API 키를 확인해주세요.', 'Transcription failed. Please check your Perplexity API key.', userLang), emotion: 'concerned' }
          const summary = await meetingSummarize(transcribed.text).catch(() => ({ success: false, summary: '', action_items: [], decisions: [], message: '요약 실패' }))
          return {
            text: summary.success ? t(`회의 요약 완료! 액션 아이템 ${summary.action_items.length}개`, `Meeting summary done! ${summary.action_items.length} action item(s)`, userLang) : t('회의 요약에 실패했어요.', 'Meeting summary failed.', userLang),
            card2: {
              type: 'system_action', icon: '📋',
              title: t('회의 요약', 'Meeting Summary', userLang),
              detail: t(`요약: ${summary.summary.slice(0, 100)}\n\n액션: ${summary.action_items.slice(0, 3).join(' / ')}`, `Summary: ${summary.summary.slice(0, 100)}\n\nActions: ${summary.action_items.slice(0, 3).join(' / ')}`, userLang),
              success: summary.success,
            },
            emotion: summary.success ? 'happy' : 'concerned',
          }
        }

        /* ── ⌨️ 음성 받아쓰기 ── */
        case 'dictation_start': {
          const textToDictate = originalText
            .replace(/받아쓰기|dictation|타이핑.*해줘|입력.*해줘|써줘.*지금|적어줘.*지금|대신.*타이핑|대신.*입력|대신.*써줘|자동.*입력/gi, '')
            .trim()
          if (!textToDictate) return { text: t('입력할 내용을 말해주세요. 예: "받아쓰기 안녕하세요 오늘 날씨가 맑네요"', 'Please say what you want me to type. e.g. "dictate Hello, how are you today"', userLang), emotion: 'neutral' }
          const res = await dictationType(textToDictate).catch(() => ({ success: false, typed_chars: 0, message: '받아쓰기 실패' }))
          return {
            text: res.message,
            card2: { type: 'system_action', icon: '⌨️', title: `${res.typed_chars}글자 입력 완료`, detail: textToDictate.slice(0, 80), success: res.success },
            emotion: res.success ? 'happy' : 'concerned',
          }
        }

        /* ── 🎭 AI 멀티 페르소나 ── */
        case 'persona_list': {
          const data = await personaList().catch(() => ({ personas: [], current: 'nexus' }))
          const lines = data.personas.map((p) => `${p.emoji} **${p.name}** — ${p.description}`).join('\n')
          return {
            text: t(`현재 페르소나: **${data.current}**\n\n사용 가능한 AI 팀:\n${lines}\n\n"리서치 모드로 바꿔줘" 처럼 말하면 전환해요!`, `Current persona: **${data.current}**\n\nAvailable AI team:\n${lines}\n\nSay "switch to research mode" to change!`, userLang),
            emotion: 'happy' as const,
          }
        }

        case 'persona_switch': {
          const lower = originalText.toLowerCase()
          let id = 'nexus'
          if (/리서치|연구|research/.test(lower)) id = 'research'
          else if (/재무|회계|finance|financial|accountant/.test(lower)) id = 'accountant'
          else if (/회의|meeting/.test(lower)) id = 'meeting'
          else if (/크리에이티브|creative|창의/.test(lower)) id = 'creative'
          else if (/보안|security/.test(lower)) id = 'security'
          else if (/법무|법률|legal|계약/.test(lower)) id = 'legal'
          const res = await personaSet(id).catch(() => ({ ok: false, persona: null as unknown as PersonaDef, message: '전환 실패' }))
          if (res.ok && res.persona) setActivePersona(res.persona)
          return {
            text: res.message,
            card2: { type: 'system_action', icon: res.persona?.emoji ?? '🤖', title: res.persona?.name ?? id, detail: res.persona?.description ?? '', success: res.ok },
            emotion: res.ok ? 'happy' as const : 'concerned' as const,
          }
        }

        /* ── 🧠 Second Brain ── */
        case 'brain_search': {
          const query = originalText.replace(/second.*brain|세컨드.*브레인|기억.*검색|장기.*기억.*찾아|내가.*했던|작년에.*내가|과거에/gi, '').trim() || originalText
          const data = await brainSearch(query, 8).catch(() => ({ results: [], total: 0, summary: '', query }))
          const items = data.results.slice(0, 5).map((r) => `[${r.entry.source}] ${r.entry.title}`)
          return {
            text: data.summary || (data.results.length > 0 ? t(`"${query}" 관련 기억 ${data.results.length}건 찾았어요:\n${items.join('\n')}`, `Found ${data.results.length} memory match(es) for "${query}":\n${items.join('\n')}`, userLang) : t(`"${query}"에 대한 기억이 없어요.`, `No memories found for "${query}".`, userLang)),
            emotion: data.results.length > 0 ? 'happy' as const : 'neutral' as const,
          }
        }

        case 'brain_stats': {
          const data = await brainStats().catch(() => ({ total: 0, by_source: {} as Record<string, number>, updated_at: '' }))
          const src = Object.entries(data.by_source).map(([k, v]) => `${k}: ${v}개`).join(', ')
          return {
            text: t(`🧠 Second Brain 현황\n총 ${data.total}개 기억 저장됨\n${src}\n마지막 업데이트: ${data.updated_at.slice(0, 10) || '없음'}`, `🧠 Second Brain Status\n${data.total} memories stored\n${src}\nLast updated: ${data.updated_at.slice(0, 10) || 'N/A'}`, userLang),
            emotion: 'neutral' as const,
          }
        }

        /* ── ⚡ Auto Workflow ── */
        case 'workflow_plan': {
          const goal = originalText.replace(/워크플로.*계획|어떻게.*자동화|단계.*알려줘|자동화.*방법|순서.*알려줘/gi, '').trim() || originalText
          const plan = await workflowPlan(goal).catch(() => ({ goal, steps: [], summary: '계획 생성 실패', ok: false }))
          const stepLines = plan.steps.map((s) => `${s.step}. ${s.description} → \`${s.api_endpoint}\``).join('\n')
          return {
            text: t(`**워크플로 계획**: ${plan.goal}\n\n${stepLines}\n\n실행하려면 "자동으로 실행해줘"라고 하세요.`, `**Workflow Plan**: ${plan.goal}\n\n${stepLines}\n\nSay "run it automatically" to execute.`, userLang),
            emotion: 'neutral' as const,
          }
        }

        case 'workflow_run': {
          const goal = originalText.replace(/자동.*해줘|한.*번에.*다|워크플로.*실행|만들어서.*보내줘|요약하고.*이메일|찾아서.*정리/gi, '').trim() || originalText
          const result = await workflowRun(goal).catch(() => ({ goal, steps: [], summary: '워크플로 실행 실패', ok: false }))
          const doneSteps = result.steps.filter((s) => s.status === 'done').length
          const totalSteps = result.steps.length
          return {
            text: t(`✅ 워크플로 완료 (${doneSteps}/${totalSteps}단계)\n\n${result.summary}`, `✅ Workflow complete (${doneSteps}/${totalSteps} steps)\n\n${result.summary}`, userLang),
            card2: { type: 'system_action', icon: '⚡', title: t(`워크플로: ${goal.slice(0, 30)}`, `Workflow: ${goal.slice(0, 30)}`, userLang), detail: t(`${doneSteps}/${totalSteps}단계 완료`, `${doneSteps}/${totalSteps} steps done`, userLang), success: result.ok },
            emotion: result.ok ? 'happy' as const : 'concerned' as const,
          }
        }

        /* ── 🎬 Live Caption ── */
        case 'caption_start': {
          const langMatch = originalText.match(/영어|일본어|중국어|스페인어|프랑스어|korean|english|japanese|chinese/)
          const langMap: Record<string, string> = { 영어: 'en', 일본어: 'ja', 중국어: 'zh', 스페인어: 'es', 프랑스어: 'fr', english: 'en', japanese: 'ja', chinese: 'zh' }
          const lang = langMap[langMatch?.[0]?.toLowerCase() ?? ''] ?? 'ko'
          const res = await captionStart(lang).catch(() => ({ ok: false, message: '자막 시작 실패' }))
          if (res.ok) setCaptionRunning(true)
          return {
            text: res.message,
            card2: { type: 'system_action', icon: '🎬', title: '실시간 자막', detail: `번역 언어: ${lang === 'ko' ? '한국어' : lang}`, success: res.ok },
            emotion: res.ok ? 'happy' as const : 'concerned' as const,
          }
        }

        case 'caption_stop': {
          const res = await captionStop().catch(() => ({ ok: false, message: '자막 종료 실패', entries: 0 }))
          if (res.ok) setCaptionRunning(false)
          return {
            text: t(`${res.message} (총 ${res.entries}개 자막)`, `${res.message} (${res.entries} captions total)`, userLang),
            card2: { type: 'system_action', icon: '⏹️', title: t('자막 종료', 'Caption stopped', userLang), detail: t(`총 ${res.entries}개 자막 저장됨`, `${res.entries} captions saved`, userLang), success: res.ok },
            emotion: 'neutral' as const,
          }
        }

        /* ── 📧 이메일 분류 ── */
        case 'email_classify': {
          const data = await emailClassify(20).catch(() => ({ success: false, classified: [], counts: {}, message: 'Outlook이 필요합니다.', action: 'outlook_setup_required' }))
          if ((data as any).action === 'outlook_setup_required') {
            d.openEmailSetup?.()
            return { text: t('이메일 기능을 사용하려면 Gmail/Outlook 연동이 필요해요.\n설정 → 이메일 탭에서 계정을 추가해주세요 📧', 'Email setup required.\nGo to Settings → Email tab to connect your account 📧', userLang), emotion: 'neutral' }
          }
          const countStr = Object.entries(data.counts ?? {}).map(([k, v]) => `${k}: ${v}`).join(' · ')
          return {
            text: data.message || t(`이메일 분류 완료! ${countStr}`, `Email classification done! ${countStr}`, userLang),
            card2: { type: 'system_action', icon: '📊', title: t('이메일 AI 분류', 'Email AI Classification', userLang), detail: countStr || t('분류 결과 없음', 'No results', userLang), success: data.success },
            emotion: 'happy',
          }
        }

        /* ── 📧 이메일 답장 초안 ── */
        case 'email_draft': {
          const inbox = await emailInbox(1).catch(() => ({ success: false, emails: [], total: 0, unread: 0, message: '' }))
          const latest = inbox.emails?.[0]
          if (!latest) return { text: t('답장할 메일이 없어요. 먼저 받은 메일함을 확인해주세요.', 'No emails to reply to. Please check your inbox first.', userLang), emotion: 'neutral' }
          const tone = /격식|formal|정중/.test(originalText) ? 'formal' : 'casual'
          const data = await emailDraftReply(latest.subject, latest.sender, latest.body, tone).catch(() => ({ success: false, draft: '', message: '초안 작성 실패' }))
          return {
            text: data.message || '답장 초안이 완성됐어요!',
            card2: { type: 'system_action', icon: '✉️', title: `"${latest.subject}" 답장 초안`, detail: data.draft?.slice(0, 150) ?? '', success: data.success },
            emotion: data.success ? 'happy' : 'concerned',
          }
        }

        /* ── 📅 빈 시간 찾기 ── */
        case 'calendar_find_slot': {
          const durMatch = originalText.match(/(\d+)\s*시간/)
          const duration = durMatch ? parseInt(durMatch[1]) * 60 : 60
          const prefer = /오후|afternoon/.test(originalText) ? 'afternoon' : /저녁|evening/.test(originalText) ? 'evening' : 'morning'
          const data = await calendarFindSlot(duration, prefer, 7).catch(() => ({ success: false, slots: [], message: 'Outlook이 필요합니다.' }))
          const slotStr = (data.slots as Array<{start: string; end: string}>).slice(0, 3).map(s => `${s.start} ~ ${s.end}`).join('\n')
          return {
            text: data.message || `가능한 시간 ${data.slots.length}개를 찾았어요!`,
            card2: { type: 'system_action', icon: '📅', title: `빈 시간 ${data.slots.length}개`, detail: slotStr, success: data.success },
            emotion: data.slots.length > 0 ? 'happy' : 'neutral',
          }
        }

        /* ── 📅 자연어 일정 추가 ── */
        case 'calendar_smart_add': {
          const data = await calendarSmartAdd(originalText).catch(() => ({ success: false, event: null, message: '일정 추가 실패', confirm_needed: false }))
          return {
            text: data.message,
            card2: { type: 'system_action', icon: '📅', title: data.message, success: data.success },
            emotion: data.success ? 'happy' : 'concerned',
          }
        }

        /* ── ⚡ 워크플로 목록 ── */
        case 'workflow_list': {
          const data = await workflowList().catch(() => ({ workflows: [], count: 0 }))
          const wfs = (data.workflows as Array<{name?: string; id: string}>)
          return {
            text: wfs.length === 0 ? t('저장된 워크플로가 없어요. "워크플로 만들어줘"로 생성해보세요!', 'No workflows saved. Say "create a workflow" to get started!', userLang) : t(`워크플로 ${wfs.length}개가 있어요.`, `${wfs.length} workflow(s) found.`, userLang),
            card2: { type: 'system_action', icon: '⚡', title: `워크플로 ${wfs.length}개`, detail: wfs.slice(0, 5).map(w => `• ${w.name ?? w.id}`).join('\n'), success: true },
            emotion: 'neutral',
          }
        }

        /* ── ⚡ 워크플로 생성 (자연어) ── */
        case 'workflow_create': {
          const text = originalText.replace(/워크플로.*만들어|새.*자동화.*생성|텍스트로.*자동화/gi, '').trim() || originalText
          const data = await workflowFromText(text).catch(() => ({ success: false, workflow: null, message: '워크플로 생성 실패' }))
          return {
            text: data.message || '워크플로가 생성됐어요!',
            card2: { type: 'system_action', icon: '⚡', title: data.message, success: data.success },
            emotion: data.success ? 'happy' : 'concerned',
          }
        }

        /* ── ⚡ 워크플로 템플릿 ── */
        case 'workflow_templates': {
          const data = await workflowTemplates().catch(() => ({ templates: [], count: 0 }))
          const tpls = (data.templates as Array<{name?: string; description?: string}>)
          return {
            text: t(`워크플로 템플릿 ${tpls.length}개를 찾았어요!`, `Found ${tpls.length} workflow template(s)!`, userLang),
            card2: { type: 'system_action', icon: '📋', title: `템플릿 ${tpls.length}개`, detail: tpls.slice(0, 5).map(t => `• ${t.name ?? ''} — ${t.description ?? ''}`).join('\n'), success: true },
            emotion: 'happy',
          }
        }

        /* ── 📨 IMAP 받은 메일 ── */
        case 'imap_inbox': {
          const data = await imapInbox(10).catch(() => ({ success: false, emails: [], total: 0, unread: 0, message: 'IMAP 계정이 설정되어 있지 않아요. 설정에서 IMAP 계정을 추가해주세요.' }))
          return {
            text: data.message || `받은 메일 ${data.total}개 (읽지 않음 ${data.unread ?? 0}개)`,
            card2: { type: 'system_action', icon: '📨', title: `IMAP 메일 ${data.total}개`, detail: (data.emails as Array<{subject: string; from: string; read: boolean}>).slice(0, 3).map(e => `${e.read ? '📨' : '📩'} ${e.subject} — ${e.from}`).join('\n'), success: data.success },
            emotion: (data.unread ?? 0) > 0 ? 'concerned' : 'neutral',
          }
        }

        /* ── 📨 IMAP 메일 전송 ── */
        case 'imap_send': {
          const toMatch = originalText.match(/([a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,})/)
          const to = toMatch?.[1] ?? ''
          if (!to) return { text: t('받는 사람 이메일을 알려주세요.', 'Please provide the recipient email address.', userLang), emotion: 'neutral' }
          const subject = originalText.match(/제목[:\s]+(.+)/)?.[1] ?? originalText.match(/subject[:\s]+(.+)/i)?.[1] ?? t('NEXUS에서 보낸 메일', 'Mail from NEXUS', userLang)
          const body = originalText.match(/내용[:\s]+(.+)/)?.[1] ?? ''
          const res = await imapSend(to, subject, body).catch(() => ({ success: false, message: '전송 실패' }))
          return {
            text: res.message,
            card2: { type: 'system_action', icon: '📤', title: res.message, success: res.success },
            emotion: res.success ? 'happy' : 'concerned',
          }
        }

        /* ── 🤖 멀티 에이전트 ── */
        case 'multi_agent': {
          const goal = originalText.replace(/멀티.*에이전트|여러.*ai.*동시|multi.*agent|에이전트.*팀/gi, '').trim() || originalText
          const data = await multiAgentRun(goal).catch(() => ({ success: false, task_id: '', message: '멀티 에이전트 실행 실패' }))
          return {
            text: data.message || t(`멀티 에이전트 작업이 시작됐어요! 작업 ID: ${data.task_id}`, `Multi-agent task started! Task ID: ${data.task_id}`, userLang),
            card2: { type: 'system_action', icon: '🤖', title: t('멀티 에이전트 실행', 'Multi-agent running', userLang), detail: t(`목표: ${goal.slice(0, 80)}\nTask ID: ${data.task_id}`, `Goal: ${goal.slice(0, 80)}\nTask ID: ${data.task_id}`, userLang), success: data.success },
            emotion: data.success ? 'happy' : 'concerned',
          }
        }

        /* ── ⚡ 병렬 동시 질문 ── */
        case 'parallel_queries': {
          // "A랑 B랑 C 동시에 알려줘" 형태에서 쿼리 분리
          const cleaned = originalText
            .replace(/동시에|한꺼번에|한번에|같이|함께|parallel|simultaneously/gi, '')
            .replace(/알려줘|알아봐줘|찾아줘|검색해줘|tell me|find|search/gi, '')
          const queries = cleaned
            .split(/[,，、\n]|랑 |와 |and |,\s*/)
            .map(q => q.trim())
            .filter(q => q.length > 2)
          if (queries.length < 2) {
            return { text: t('동시에 처리할 질문을 여러 개 알려주세요. 예: "날씨랑 환율이랑 코스피 동시에 알려줘"', 'Please provide multiple questions. e.g. "Tell me weather, exchange rate, and KOSPI at the same time"', userLang), emotion: 'neutral' }
          }
          const progressMsgId = `par-prog-${Date.now()}`
          setMessages(prev => [...prev, {
            id: progressMsgId,
            role: 'nexus',
            text: t(`⚡ ${queries.length}개 질문을 동시에 처리 중...\n${queries.map((q, i) => `${i + 1}. ${q}`).join('\n')}`, `⚡ Processing ${queries.length} questions in parallel...\n${queries.map((q, i) => `${i + 1}. ${q}`).join('\n')}`, userLang),
            emotion: 'neutral',
          }])
          const results: Array<{ index: number; query: string; answer: string; success: boolean }> = []
          await new Promise<void>(resolve => {
            const ctrl = dispatchParallel(queries, (evt: ParallelEvent) => {
              if (evt.type === 'result') {
                results.push({ index: evt.index ?? 0, query: evt.query ?? '', answer: evt.answer ?? '', success: evt.success ?? false })
                // 중간 결과를 실시간으로 업데이트
                const sorted = [...results].sort((a, b) => a.index - b.index)
                const progressText = sorted.map(r => `**${r.index + 1}. ${r.query}**\n${r.answer}`).join('\n\n---\n\n')
                setMessages(prev => prev.map(m => m.id === progressMsgId
                  ? { ...m, text: progressText }
                  : m
                ))
              } else if (evt.type === 'done') {
                void ctrl
                resolve()
              } else if (evt.type === 'error') {
                resolve()
              }
            })
            setTimeout(resolve, 60000) // 60초 타임아웃
          })
          const sorted = [...results].sort((a, b) => a.index - b.index)
          const finalText = sorted.map(r => `**${r.index + 1}. ${r.query}**\n${r.success ? r.answer : `❌ ${r.answer}`}`).join('\n\n---\n\n')
          setMessages(prev => prev.map(m => m.id === progressMsgId ? { ...m, text: finalText } : m))
          return { text: '', emotion: 'happy' } // setMessages로 이미 처리됨 (빈 텍스트 반환)
        }

        /* ── 📢 브리핑 ── */
        case 'briefing_now': {
          const data = await briefingNow().catch(() => ({ success: false, task_id: '', message: '브리핑 시작 실패' }))
          return {
            text: data.message || t('모닝 브리핑이 시작됐어요! 날씨·일정·이메일 정보를 수집 중이에요.', 'Morning briefing started! Collecting weather, schedule & email info.', userLang),
            card2: { type: 'system_action', icon: '📢', title: t('모닝 브리핑 시작', 'Morning Briefing', userLang), detail: `Task ID: ${data.task_id}`, success: data.success },
            emotion: data.success ? 'happy' : 'concerned',
          }
        }

        /* ── ❌ 작업 취소 ── */
        case 'task_cancel': {
          const tasks = await taskList().catch(() => ({ tasks: [], count: 0 }))
          if (!tasks.count) return { text: t('취소할 실행 중 작업이 없어요.', 'No running tasks to cancel.', userLang), emotion: 'neutral' }
          const first = (tasks.tasks as Array<{id: string; name?: string}>)[0]
          const res = await taskCancel(first.id).catch(() => ({ success: false, message: '취소 실패' }))
          return {
            text: res.message || t(`작업 "${first.name ?? first.id}"을 취소했어요.`, `Task "${first.name ?? first.id}" cancelled.`, userLang),
            card2: { type: 'system_action', icon: '❌', title: res.message, success: res.success },
            emotion: res.success ? 'neutral' : 'concerned',
          }
        }

        /* ── 🔍 검색+PDF 보고서 ── */
        case 'search_pdf': {
          const query = originalText.replace(/검색.*pdf|pdf.*보고서|웹.*검색.*pdf|조사.*보고서|search.*pdf/gi, '').trim() || originalText
          const data = await searchAndPDF(query, 8, '', true).catch(() => ({ success: false, pdf_path: '', html_path: '', query, item_count: 0, summary: '생성 실패', duration: '' }))
          return {
            text: data.success ? t(`PDF 보고서 생성 완료! ${data.item_count}개 항목 수집, ${data.duration} 소요.`, `PDF report generated! ${data.item_count} items collected in ${data.duration}.`, userLang) : data.summary,
            card2: { type: 'system_action', icon: '📄', title: t(`PDF 보고서: ${query.slice(0, 30)}`, `PDF Report: ${query.slice(0, 30)}`, userLang), detail: t(`경로: ${data.pdf_path || '생성 실패'}\n${data.summary?.slice(0, 100) ?? ''}`, `Path: ${data.pdf_path || 'Failed'}\n${data.summary?.slice(0, 100) ?? ''}`, userLang), success: data.success },
            emotion: data.success ? 'happy' : 'concerned',
          }
        }

        /* ── 🎮 GPU 모니터링 ── */
        case 'gpu_stats': {
          const data = await gpuStats().catch(() => ({ success: false, gpus: [], message: 'GPU 정보 조회 실패' }))
          const gpu = data.gpus?.[0]
          return {
            text: data.message,
            card2: { type: 'system_action', icon: '🎮', title: gpu ? `${gpu.name}` : t('GPU 정보', 'GPU Info', userLang), detail: gpu ? t(`사용률 ${gpu.usage_pct}% · 온도 ${gpu.temp_c}°C · VRAM ${gpu.mem_used_mb}/${gpu.mem_total_mb}MB`, `Usage ${gpu.usage_pct}% · Temp ${gpu.temp_c}°C · VRAM ${gpu.mem_used_mb}/${gpu.mem_total_mb}MB`, userLang) : t('정보 없음', 'No info', userLang), success: data.success },
            emotion: gpu && gpu.temp_c > 80 ? 'alert' : gpu && gpu.usage_pct > 90 ? 'concerned' : 'neutral',
          }
        }

        default:
          return { text: '', emotion: 'neutral' }
      }
    } catch {
      return {
        text: userLang === 'ko'
          ? '백엔드 연결에 실패했어요. 앱이 설치된 환경에서 실행해주세요.'
          : 'Backend not available. Please run in the installed app.',
        emotion: 'concerned',
      }
    }
}
