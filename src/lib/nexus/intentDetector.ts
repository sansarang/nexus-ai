/**
 * 키워드 기반 인텐트 감지 — LLM 호출 없음, 즉시 처리
 */

export type Intent =
  // ── 기존 ───────────────────────────────────
  | 'pc_status'        // CPU·RAM·온도·디스크 상태
  | 'security_scan'    // 해킹 탐지·보안 스캔
  | 'full_scan'        // 전체 PC 진단
  | 'clean'            // 파일 정리·청소
  | 'daily_report'     // 오늘 리포트
  | 'repair'           // 문제 수리
  | 'open_folder'      // 폴더 열기
  // ── 보안 상세 ──────────────────────────────
  | 'remote_access'    // 원격 접속 탐지
  | 'process_security' // 수상한 프로세스·포트
  | 'hosts_check'      // hosts 파일 변조
  | 'startup_items'    // 시작 프로그램
  | 'defender_status'  // Windows Defender
  | 'account_check'    // 이상 계정
  | 'virus_check'      // VirusTotal 파일 검사
  | 'process_kill'     // 프로세스 강제 종료
  | 'app_permissions'  // 앱 권한 감사
  | 'windows_updates'  // Windows 업데이트 확인
  | 'gpu_stats'        // GPU 상세 모니터링
  // ── 시스템 제어 ────────────────────────────
  | 'volume_control'   // 볼륨 조절·음소거
  | 'brightness'       // 화면 밝기
  | 'wifi_toggle'      // Wi-Fi 켜기/끄기
  | 'power_action'     // 잠금·절전·재시작·종료
  | 'launch_app'       // 앱 실행
  | 'process_top'      // 프로세스 상위 목록
  // ── 고급 기능 ──────────────────────────────
  | 'driver_check'     // 드라이버 점검
  | 'registry_clean'   // 레지스트리 정리
  | 'power_plan'       // 전원 계획
  | 'network_analysis' // 네트워크 분석
  | 'restore_create'   // 시스템 복구 포인트
  | 'disk_check'       // 디스크 검사
  | 'browser_clean'    // 브라우저 데이터 정리
  | 'programs_list'    // 설치 프로그램 목록
  | 'boot_analysis'    // 부팅 속도 분석
  // ── 파일 관리 ──────────────────────────────
  | 'file_search'      // 파일 검색
  | 'file_organize'    // 폴더 자동 정리
  | 'file_duplicates'  // 중복 파일 찾기
  // ── 생산성 ─────────────────────────────────
  | 'focus_mode'       // 집중 모드
  | 'clipboard'        // 클립보드
  | 'notes'            // 메모
  // ── 신규: 문서 비교 ────────────────────────
  | 'doc_compare'      // 두 문서 비교
  | 'doc_find'         // 문서 검색
  | 'deep_search'      // 파일 내용 심층 검색
  // ── 신규: Vision ────────────────────────────
  | 'vision_screen'    // 화면 분석 ("지금 화면에 뭐라고 써있어?")
  | 'vision_ocr'       // 클립보드 이미지 OCR
  // ── 신규: 스마트 정리 ──────────────────────
  | 'smart_organize'   // 다운로드·바탕화면 스마트 정리
  // ── 수익 기능 ──────────────────────────────
  | 'journal_today'    // 오늘 업무 일지
  | 'journal_generate' // 업무 일지 파일 생성
  | 'journal_history'  // 과거 일지 기록
  | 'macro_list'       // 매크로 목록
  | 'macro_create'     // 매크로 생성 (자연어)
  | 'macro_run'        // 매크로 실행
  | 'pc_report'        // PC 건강 리포트
  | 'report_email'     // 리포트 이메일 전송
  | 'doc_summary'      // 문서 요약
  // ── 📅 캘린더 ──────────────────────────────
  | 'calendar_today'   // 오늘 일정
  | 'calendar_week'    // 이번 주 일정
  | 'calendar_add'     // 일정 추가
  // ── 📧 이메일 ──────────────────────────────
  | 'email_inbox'      // 받은 메일 확인
  | 'email_send'       // 메일 전송
  | 'email_summarize'  // 메일 AI 요약
  // ── 🌐 웹 검색·가격 비교 ───────────────────
  | 'price_compare'    // 가격 비교 (쿠팡·네이버)
  | 'news_search'      // 뉴스 검색
  | 'youtube_search'   // 유튜브 영상 검색
  | 'video_search'     // 틱톡/유튜브 영상 검색
  | 'video_download'    // 유튜브/틱톡 영상 다운로드
  | 'video_transcript'  // 영상 URL 내용 요약/전사
  | 'multi_action'     // 멀티 액션 (검색 + 파일 저장)
  // ── ⏰ 스케줄러 ────────────────────────────
  | 'schedule_list'    // 스케줄 목록
  | 'schedule_add'     // 스케줄 추가
  | 'schedule_delete'  // 스케줄 삭제
  // ── 📊 성능 이력 ───────────────────────────
  | 'perf_history'     // 성능 이력 조회
  | 'perf_anomaly'     // 이상 탐지
  // ── 🖥️ Windows Recall ─────────────────────────
  | 'recall_search'    // 과거 화면 기억 검색
  | 'recall_capture'   // 지금 화면 기억 저장
  // ── 🎙️ 회의 어시스턴트 ───────────────────────
  | 'meeting_start'    // 회의 녹음 시작
  | 'meeting_stop'     // 회의 녹음 종료
  | 'meeting_summary'  // 회의 요약
  | 'meeting_list'     // 녹음 목록
  // ── ⌨️ 음성 받아쓰기 ──────────────────────────
  | 'dictation_start'  // 받아쓰기 시작 (현재 앱에 타이핑)
  // ── 🏠 스마트홈 ────────────────────────────────
  // ── 🌤️ 날씨 + 교통 ────────────────────────────
  | 'weather'          // 날씨 조회
  | 'travel_time'      // 교통 시간 조회
  // ── 🌐 번역 ────────────────────────────────────
  | 'translate'        // 클립보드/텍스트 번역
  // ── 📋 클립보드 AI ──────────────────────────────
  | 'clipboard_ai'     // 클립보드 내용 AI 처리 (요약·교정·번역)
  // ── 📝 음성 메모→할일 ───────────────────────────
  | 'voice_todo'       // 음성으로 메모 + 캘린더 동시 추가
  // ── 🎭 AI 멀티 페르소나 ────────────────────────
  | 'persona_list'     // 페르소나 목록
  | 'persona_switch'   // 페르소나 전환
  // ── 🧠 Second Brain ────────────────────────────
  | 'brain_search'     // Second Brain 검색
  | 'brain_stats'      // 인덱스 통계
  // ── ⚡ Auto Workflow ─────────────────────────────
  | 'workflow_run'     // 워크플로 자동 실행
  | 'workflow_plan'    // 워크플로 계획만 조회
  // ── 🎬 Live Caption ─────────────────────────────
  | 'caption_start'    // 실시간 자막 시작
  | 'caption_stop'     // 실시간 자막 종료
  // ── 📧 이메일 심화 ────────────────────────────────────────────
  | 'email_classify'   // 이메일 분류·우선순위
  | 'email_draft'      // 이메일 답장 초안 작성
  // ── 📅 캘린더 심화 ───────────────────────────────────────────
  | 'calendar_find_slot'  // 빈 시간 찾기
  | 'calendar_smart_add'  // 자연어 일정 추가
  // ── ⚡ 워크플로 관리 ──────────────────────────────────────────
  | 'workflow_list'    // 저장된 워크플로 목록
  | 'workflow_create'  // 자연어로 워크플로 생성
  | 'workflow_templates' // 워크플로 템플릿 목록
  // ── 📨 IMAP 이메일 ───────────────────────────────────────────
  | 'imap_inbox'       // IMAP 받은 메일 확인
  | 'imap_send'        // IMAP 메일 전송
  // ── 🤖 멀티 에이전트 ─────────────────────────────────────────
  | 'multi_agent'      // 멀티 에이전트 실행
  // ── 📢 브리핑 ───────────────────────────────────────────────
  | 'briefing_now'     // 모닝 브리핑 실행
  // ── ❌ 작업 취소 ─────────────────────────────────────────────
  | 'task_cancel'      // 실행 중 작업 취소
  // ── 🔍 검색+PDF ──────────────────────────────────────────────
  | 'search_pdf'       // 웹 검색 후 PDF 보고서 생성
  | 'none'             // LLM으로 위임

const PATTERNS: { intent: Intent; patterns: RegExp[] }[] = [
  // ── 날씨 (pc_status보다 먼저 체크 — "온도" 충돌 방지) ──
  // 날씨는 LLM(Groq fallback)으로 보내야 함 → 여기서 'none' 대신 명시적으로 처리
  // → intentDetector에서는 매칭 안 되게 두고 fallbackResponse가 처리

  // ── PC 상태 (날씨 제외한 온도는 여기서) ──
  {
    intent: 'pc_status',
    patterns: [
      /cpu|ram|메모리|디스크|disk|실시간|지금\s*(pc|컴)|pc\s*상태|컴퓨터\s*상태|얼마나/i,
      /how.*pc|pc.*how|system.*status|status.*system|gpu|그래픽/i,
      // "온도" 단독은 날씨와 구분: "CPU 온도" / "PC 온도" 형태만 매칭
      /(?:cpu|pc|컴퓨터)\s*온도|온도.*(?:cpu|pc|컴)|temperature.*(?:cpu|pc)/i,
      /상태.*어때|어때.*상태|pc.*어때/i,
    ],
  },
  // ── 보안 스캔 ──
  {
    intent: 'security_scan',
    patterns: [
      /해킹|보안.*스캔|security.*scan|scan.*security|악성|malware|침입|intrusion/i,
      /바이러스|virus|감염|infected|위협|threat|안전한가|안전해/i,
    ],
  },
  // ── 전체 진단 ──
  {
    intent: 'full_scan',
    patterns: [
      /진단|전체.*검사|검사.*전체|pc.*점검|점검.*pc|풀.*스캔|full.*scan/i,
    ],
  },
  // ── 정리 ──
  {
    intent: 'clean',
    patterns: [
      /청소|임시.*파일|temp.*file|pc.*정리|정리.*pc|캐시.*지워|지워.*캐시/i,
      /느려.*졌|최적화.*해줘|빠르게.*해줘|optimize/i,
    ],
  },
  // ── 데일리 리포트 ──
  {
    intent: 'daily_report',
    patterns: [
      /리포트|report|오늘.*요약|요약.*오늘|데일리|daily|주간|weekly/i,
      /어떻게.*됐어|전반적.*상태/i,
    ],
  },
  // ── 수리 ──
  {
    intent: 'repair',
    patterns: [
      /수리|repair|고쳐|문제.*수정|수정.*문제|윈도우.*오류.*수리|sfc|dism/i,
    ],
  },
  // ── 폴더 열기 ──
  {
    intent: 'open_folder',
    patterns: [
      /폴더\s*(열어|띄워|보여|오픈|열기|show|open)/i,
      /(열어|띄워|보여|오픈|open|show)\s*(줘|줄래|주세요|please)?\s*(폴더|folder|디렉토리)/i,
      /(바탕화면|다운로드|문서|사진|음악|비디오|동영상|downloads?|documents?|pictures?|desktop|music|videos?)\s*(폴더)?\s*(열어|띄워|보여|오픈|열기)/i,
      /(열어|띄워)\s*(줘|줄래)\s*(바탕화면|다운로드|문서|사진|음악|비디오|동영상|desktop)/i,
    ],
  },
  // ── 원격 접속 탐지 ──
  {
    intent: 'remote_access',
    patterns: [
      /원격.*접속|접속.*원격|teamviewer|anydesk|rdp|vnc|remote.*desktop|원격.*탐지/i,
      /누군가.*접속|접속한.*사람|몰래.*접속/i,
    ],
  },
  // ── 수상한 프로세스 ──
  {
    intent: 'process_security',
    patterns: [
      /수상한.*프로세스|프로세스.*보안|이상한.*앱|이상한.*프로그램|수상한.*포트/i,
      /백도어|backdoor|채굴|mining|스파이웨어|spyware/i,
    ],
  },
  // ── hosts 파일 ──
  {
    intent: 'hosts_check',
    patterns: [
      /hosts|호스트.*파일|파일.*변조|dns.*변조|hosts.*변조/i,
    ],
  },
  // ── 시작 프로그램 ──
  {
    intent: 'startup_items',
    patterns: [
      /시작.*프로그램|startup|자동.*실행|부팅.*프로그램|켜질때.*실행/i,
    ],
  },
  // ── Windows Defender ──
  {
    intent: 'defender_status',
    patterns: [
      /디펜더|defender|백신.*상태|바이러스.*백신.*상태|실시간.*보호/i,
    ],
  },
  // ── 계정 확인 ──
  {
    intent: 'account_check',
    patterns: [
      /계정.*확인|숨겨진.*계정|이상한.*계정|계정.*보안|account.*check/i,
    ],
  },
  // ── 볼륨 조절 ──
  {
    intent: 'volume_control',
    patterns: [
      /볼륨|volume|음소거|mute|소리.*높여|소리.*낮춰|소리.*키워|소리.*줄여/i,
      /음량|sound.*level|소리.*off|소리.*on/i,
    ],
  },
  // ── 화면 밝기 ──
  {
    intent: 'brightness',
    patterns: [
      /밝기|brightness|화면.*밝|밝게|어둡게|brighter|dimmer/i,
    ],
  },
  // ── Wi-Fi ──
  {
    intent: 'wifi_toggle',
    patterns: [
      /와이파이|wifi|wi-fi|무선.*인터넷|인터넷.*끄|인터넷.*켜/i,
    ],
  },
  // ── 전원 제어 ──
  {
    intent: 'power_action',
    patterns: [
      /잠금|화면.*잠|lock.*screen|절전|sleep.*mode|재시작|restart|종료|shutdown|끄다|꺼줘/i,
    ],
  },
  // ── 앱 실행 ──
  {
    intent: 'launch_app',
    patterns: [
      /(크롬|chrome|파이어폭스|firefox|엣지|edge|메모장|notepad|계산기|calc|탐색기|explorer|워드|word|엑셀|excel|vscode|vs\s*code|스팀|steam|디스코드|discord|카카오|kakao|작업관리자|taskmgr|제어판|control)\s*(열어|켜줘|실행|띄워|오픈|launch|open|start)/i,
      /(열어|켜줘|실행|띄워)\s*(줘|줄래|주세요)?\s*(크롬|chrome|파이어폭스|firefox|엣지|edge|메모장|notepad|계산기|calc)/i,
    ],
  },
  // ── 프로세스 TOP ──
  {
    intent: 'process_top',
    patterns: [
      /어떤.*앱.*많이|cpu.*많이.*쓰는|메모리.*많이.*쓰는|느린.*이유|process.*top|프로세스.*목록|앱.*목록/i,
      /top.*process|무거운.*앱|ram.*먹는/i,
    ],
  },
  // ── 드라이버 ──
  {
    intent: 'driver_check',
    patterns: [
      /드라이버|driver|장치.*오류|기기.*오류|device.*error/i,
    ],
  },
  // ── 레지스트리 ──
  {
    intent: 'registry_clean',
    patterns: [
      /레지스트리|registry|reg.*clean|레지.*정리/i,
    ],
  },
  // ── 전원 계획 ──
  {
    intent: 'power_plan',
    patterns: [
      /전원.*계획|power.*plan|고성능.*모드|절전.*모드.*설정|균형.*모드/i,
    ],
  },
  // ── 네트워크 분석 ──
  {
    intent: 'network_analysis',
    patterns: [
      /네트워크.*분석|network.*analysis|연결.*기기|연결된.*장치|dns.*확인|내.*ip|ip.*주소/i,
      /ping|지연.*시간|latency/i,
    ],
  },
  // ── 복구 포인트 ──
  {
    intent: 'restore_create',
    patterns: [
      /복구.*포인트|restore.*point|시스템.*복구|백업.*만들|checkpoint/i,
    ],
  },
  // ── 디스크 검사 ──
  {
    intent: 'disk_check',
    patterns: [
      /디스크.*검사|chkdsk|disk.*check|하드.*오류|디스크.*오류/i,
    ],
  },
  // ── 브라우저 정리 ──
  {
    intent: 'browser_clean',
    patterns: [
      /브라우저.*정리|크롬.*정리|엣지.*정리|히스토리.*삭제|쿠키.*삭제|browser.*clean|browsing.*history/i,
    ],
  },
  // ── 설치 프로그램 ──
  {
    intent: 'programs_list',
    patterns: [
      /설치.*프로그램|installed.*program|프로그램.*목록|어떤.*앱.*설치|앱.*목록/i,
    ],
  },
  // ── 부팅 분석 ──
  {
    intent: 'boot_analysis',
    patterns: [
      /부팅.*느려|부팅.*속도|켜지는.*시간|startup.*time|boot.*slow|시작.*오래/i,
    ],
  },
  // ── 파일 검색 ──
  {
    intent: 'file_search',
    patterns: [
      /파일.*찾아|찾아줘.*파일|파일.*어디|어디.*파일|file.*search|search.*file/i,
      /\.pdf|\.docx|\.xlsx|\.hwp\s*(찾아|어디)/i,
    ],
  },
  // ── 폴더 자동 정리 ──
  {
    intent: 'file_organize',
    patterns: [
      /(바탕화면|다운로드|downloads|desktop)\s*(정리|organize|깔끔|청소)/i,
      /폴더\s*(자동\s*)?정리|파일\s*(자동\s*)?정리/i,
    ],
  },
  // ── 중복 파일 ──
  {
    intent: 'file_duplicates',
    patterns: [
      /중복.*파일|duplicate.*file|같은.*파일.*여러|파일.*중복/i,
    ],
  },
  // ── 집중 모드 ──
  {
    intent: 'focus_mode',
    patterns: [
      /집중.*모드|focus.*mode|방해.*금지|do.*not.*disturb|알림.*끄|알림.*차단/i,
    ],
  },
  // ── 클립보드 ──
  {
    intent: 'clipboard',
    patterns: [
      /클립보드|clipboard|복사한.*내용|복사.*내역/i,
    ],
  },
  // ── 메모 ──
  {
    intent: 'notes',
    patterns: [
      /메모|note|할일|todo|기록.*해줘|적어줘|잊지.*마/i,
    ],
  },
  // ── 문서 비교 ──
  {
    intent: 'doc_compare',
    patterns: [
      /비교.*해줘|비교.*해봐|두.*파일|두.*문서|비교.*파일|파일.*비교/i,
      /compare.*doc|doc.*compare|차이.*알려|달라진.*부분|수정.*내용|변경.*사항/i,
      /계약서.*비교|보고서.*비교|엑셀.*비교|pdf.*비교|워드.*비교/i,
    ],
  },
  // ── 문서 찾기 ──
  {
    intent: 'doc_find',
    patterns: [
      /문서.*찾아|파일.*찾아서|계약서.*찾아|보고서.*찾아|파일.*어디/i,
      /(?:보낸|받은|작성한).*파일|문서.*검색|find.*document/i,
    ],
  },
  // ── Deep Search ──
  {
    intent: 'deep_search',
    patterns: [
      /내용.*검색|파일.*안.*내용|텍스트.*검색|전문.*검색|deep.*search/i,
      /파일.*내용.*찾|찾아서.*비교|심층.*검색/i,
    ],
  },
  // ── Vision: 화면 분석 ──
  {
    intent: 'vision_screen',
    patterns: [
      /화면.*뭐라|화면.*읽어|지금.*화면|스크린.*분석|오류창|에러창/i,
      /what.*screen|screen.*say|화면.*캡처.*분석|screenshot.*analyze/i,
      /화면.*봐줘|이.*오류.*왜|이.*에러.*왜|왜.*뜨는/i,
    ],
  },
  // ── Vision: OCR ──
  {
    intent: 'vision_ocr',
    patterns: [
      /클립보드.*텍스트|이미지.*글자|ocr|사진.*텍스트|글자.*추출/i,
      /extract.*text|이미지.*읽어|그림.*내용/i,
    ],
  },
  // ── 스마트 정리 ──
  {
    intent: 'smart_organize',
    patterns: [
      /다운로드.*정리|다운로드.*폴더.*정리|바탕화면.*정리|원클릭.*정리/i,
      /smart.*organize|파일.*자동.*분류|날짜별.*정리|종류별.*정리/i,
      /내.*pc.*정리.*해줘|전체.*정리|폴더.*깔끔/i,
    ],
  },
  // ── 업무 일지 ──
  {
    intent: 'journal_today',
    patterns: [
      /업무.*일지|오늘.*일지|일지.*써줘|work.*journal|오늘.*뭐.*했|뭐.*했는지/i,
      /하루.*정리|오늘.*활동|작업.*기록|오늘.*업무/i,
    ],
  },
  {
    intent: 'journal_generate',
    patterns: [
      /일지.*저장|일지.*파일|일지.*만들어|일지.*출력|일지.*내보내/i,
      /업무.*일지.*다운|일지.*word|일지.*txt/i,
    ],
  },
  {
    intent: 'journal_history',
    patterns: [
      /일지.*기록|지난.*일지|이번.*주.*일지|history.*journal|과거.*활동/i,
    ],
  },
  // ── 자동화 매크로 ──
  {
    intent: 'macro_list',
    patterns: [
      /매크로.*목록|매크로.*뭐|등록.*매크로|자동화.*뭐|어떤.*매크로/i,
      /macro.*list|자동.*실행.*목록/i,
    ],
  },
  {
    intent: 'macro_create',
    patterns: [
      /매크로.*만들어|자동.*실행.*해줘|매일.*시에|아침.*시에.*자동/i,
      /할때마다.*자동|시작할.*때.*자동|매일.*자동|되풀이.*해줘/i,
      /크롬.*자동|정리.*자동.*매일|schedule|스케줄/i,
    ],
  },
  {
    intent: 'macro_run',
    patterns: [
      /매크로.*실행|실행.*매크로|매크로.*돌려|지금.*매크로/i,
    ],
  },
  // ── PC 리포트 ──
  {
    intent: 'pc_report',
    patterns: [
      /pc.*리포트|건강.*리포트|리포트.*만들어|pc.*보고서|상태.*리포트/i,
      /health.*report|pc.*report|pc.*정기.*점검/i,
    ],
  },
  {
    intent: 'report_email',
    patterns: [
      /리포트.*이메일|이메일.*리포트|리포트.*보내줘|pc.*상태.*메일|메일.*보내줘/i,
      /report.*email|email.*report/i,
    ],
  },
  // ── 문서 요약 ──
  {
    intent: 'doc_summary',
    patterns: [
      /요약.*해줘|요약해|summarize|문서.*요약|계약서.*요약|보고서.*요약/i,
      /핵심.*내용|중요.*내용.*뽑아|간단히.*정리|요점.*정리/i,
    ],
  },

  // ── 📅 캘린더: 오늘 일정 ──
  {
    intent: 'calendar_today',
    patterns: [
      /오늘.*일정|오늘.*스케줄|오늘.*약속|일정.*알려|today.*schedule|today.*calendar/i,
      /오늘.*뭐.*있어|오늘.*미팅|오늘.*회의/i,
    ],
  },
  // ── 📅 캘린더: 이번 주 일정 ──
  {
    intent: 'calendar_week',
    patterns: [
      /이번.*주.*일정|주간.*일정|이번.*주.*스케줄|weekly.*schedule|이번주.*뭐/i,
      /다음.*일정|앞으로.*일정|앞으로.*스케줄/i,
    ],
  },
  // ── 📅 캘린더: 일정 추가 ──
  {
    intent: 'calendar_add',
    patterns: [
      /일정.*추가|일정.*등록|일정.*넣어|약속.*추가|미팅.*추가|add.*event|add.*schedule/i,
      /(\d+월\d+일|\d+일|\d+시).*일정.*등록|일정.*(\d+월|\d+일|\d+시)/i,
    ],
  },

  // ── 📧 이메일: 받은 메일 ──
  {
    intent: 'email_inbox',
    patterns: [
      /이메일.*확인|메일.*확인|받은.*메일|받은.*이메일|inbox|mail.*check/i,
      /오늘.*온.*메일|새.*메일|읽지.*않은.*메일|안.*읽은.*메일/i,
      /이메일.*보여|메일.*보여|메일.*목록/i,
    ],
  },
  // ── 📧 이메일: 전송 ──
  {
    intent: 'email_send',
    patterns: [
      /메일.*보내|이메일.*보내|email.*send|send.*email|메일.*전송/i,
      /~에게.*메일|에게.*보내줘|에게.*이메일/i,
    ],
  },
  // ── 📧 이메일: AI 요약 ──
  {
    intent: 'email_summarize',
    patterns: [
      /메일.*요약|이메일.*요약|받은.*메일.*요약|중요한.*메일|email.*summary/i,
      /메일.*정리해|이메일.*정리|오늘.*온.*중요한/i,
    ],
  },

  // ── 🌐 가격 비교 ──
  {
    intent: 'price_compare',
    patterns: [
      /가격.*비교|최저가|쿠팡.*가격|네이버.*가격|price.*compare|싸게.*파는/i,
      /얼마야|얼마에.*팔아|어디가.*싸|할인.*제품|가격.*검색/i,
      /에어팟|갤럭시|아이폰|노트북.*가격|모니터.*가격/i,
    ],
  },
  // ── 🌐 뉴스 검색 ──
  {
    intent: 'news_search',
    patterns: [
      /뉴스.*검색|최신.*뉴스|오늘.*뉴스|news.*search|latest.*news/i,
      /뭐.*화제|핫.*이슈|트렌딩|요즘.*뭐가.*화제|요즘.*뉴스/i,
    ],
  },
  // ── 🎬 유튜브 검색 ──
  {
    intent: 'youtube_search',
    patterns: [
      /유튜브.*찾아|유튜브.*검색|youtube.*찾아|youtube.*검색/i,
      /유튜브에서.*보여|유튜브.*영상.*찾아|영상.*찾아줘/i,
      /유튜브.*어떻게|유튜브.*방법|유튜브.*강의|유튜브.*레시피/i,
      /틱톡.*찾아|틱톡.*검색|tiktok.*찾아|tiktok.*검색/i,
    ],
  },
  // ── ⬇️ 영상 다운로드 ──
  {
    intent: 'video_download',
    patterns: [
      /유튜브.*다운|youtube.*다운|영상.*다운로드|video.*download/i,
      /틱톡.*다운|tiktok.*다운|영상.*저장|동영상.*다운/i,
      /다운.*유튜브|다운.*틱톡|저장.*영상|download.*video/i,
    ],
  },

  // ── 🎬 영상 URL 요약/전사 (URL + 요약 의도) ──
  {
    intent: 'video_transcript',
    patterns: [
      /https?:\/\/(www\.)?(youtube\.com|youtu\.be|tiktok\.com|twitter\.com|x\.com|instagram\.com).*요약/i,
      /https?:\/\/(www\.)?(youtube\.com|youtu\.be|tiktok\.com|twitter\.com|x\.com|instagram\.com).*내용/i,
      /https?:\/\/(www\.)?(youtube\.com|youtu\.be|tiktok\.com|twitter\.com|x\.com|instagram\.com).*분석/i,
      /https?:\/\/(www\.)?(youtube\.com|youtu\.be|tiktok\.com|twitter\.com|x\.com|instagram\.com).*전사/i,
      /요약.*https?:\/\/(www\.)?(youtube\.com|youtu\.be|tiktok\.com)/i,
      /내용.*https?:\/\/(www\.)?(youtube\.com|youtu\.be|tiktok\.com)/i,
      /https?:\/\/(www\.)?(youtube\.com|youtu\.be).*summarize/i,
      /summarize.*https?:\/\/(www\.)?(youtube\.com|youtu\.be)/i,
    ],
  },

  // ── ⏰ 스케줄러: 목록 ──
  {
    intent: 'schedule_list',
    patterns: [
      /스케줄.*목록|자동화.*목록|예약된.*작업|scheduler.*list|등록된.*스케줄/i,
      /어떤.*자동.*실행|자동.*작업.*뭐/i,
    ],
  },
  // ── ⏰ 스케줄러: 추가 ──
  {
    intent: 'schedule_add',
    patterns: [
      /매일.*시에.*자동|매일.*시작|아침마다.*자동|schedule.*add|자동.*예약/i,
      /매일.*오전|매일.*오후|주마다.*자동|정기적으로.*실행/i,
      /(\d+시|\d+시간마다).*자동|자동.*(\d+시|\d+분마다)/i,
    ],
  },
  // ── ⏰ 스케줄러: 삭제 ──
  {
    intent: 'schedule_delete',
    patterns: [
      /스케줄.*삭제|자동화.*삭제|예약.*취소|schedule.*delete|자동.*취소/i,
    ],
  },

  // ── 📊 성능 이력 ──
  {
    intent: 'perf_history',
    patterns: [
      /성능.*이력|성능.*기록|pc.*추세|cpu.*추세|메모리.*추세|performance.*history/i,
      /지난.*\d+일|최근.*\d+일|이번.*주.*pc|pc.*변화|성능.*변화/i,
      /언제부터.*느려|언제부터.*높아|평균.*사용률/i,
    ],
  },
  // ── 📊 이상 탐지 ──
  {
    intent: 'perf_anomaly',
    patterns: [
      /이상.*탐지|이상한.*패턴|비정상.*성능|anomaly|갑자기.*높아진|급등/i,
      /언제부터.*이상|무슨.*일|갑자기.*왜.*느려/i,
    ],
  },

  // ── 🦠 VirusTotal 검사 ──
  {
    intent: 'virus_check',
    patterns: [
      /바이러스.*확인|virustotal|파일.*안전|파일.*검사|악성.*확인/i,
      /이.*파일.*안전해|안전한.*파일인지|파일.*바이러스/i,
    ],
  },
  // ── 🔫 프로세스 강제 종료 ──
  {
    intent: 'process_kill',
    patterns: [
      /프로세스.*종료|강제.*종료|앱.*죽여|앱.*강제|process.*kill|kill.*process/i,
      /(\w+).*종료해줘|응답.*없는.*앱|응답.*없는.*프로그램/i,
    ],
  },
  // ── 🔑 앱 권한 감사 ──
  {
    intent: 'app_permissions',
    patterns: [
      /앱.*권한|프로그램.*권한|권한.*감사|permission.*check|어떤.*권한/i,
      /카메라.*접근|마이크.*접근|권한.*쓰고|권한.*사용/i,
    ],
  },
  // ── 🔄 Windows 업데이트 ──
  {
    intent: 'windows_updates',
    patterns: [
      /윈도우.*업데이트|windows.*update|업데이트.*확인|업데이트.*몇.*개|업데이트.*대기/i,
      /최신.*버전|업데이트.*해야|패치.*있어/i,
    ],
  },
  // ── 🎮 GPU 모니터링 ──
  {
    intent: 'gpu_stats',
    patterns: [
      /gpu.*상태|그래픽카드.*상태|gpu.*온도|gpu.*사용률|graphics.*card/i,
      /그래픽.*얼마|vram|gpu.*몇.*퍼센트/i,
    ],
  },

  // ── 🖥️ Windows Recall ──
  {
    intent: 'recall_search',
    patterns: [
      /기억.*찾아|화면.*기억|recall.*search|언제.*봤던|어제.*봤던|전에.*봤던/i,
      /화면.*검색|스크린.*검색|과거.*화면|기억.*검색/i,
    ],
  },
  {
    intent: 'recall_capture',
    patterns: [
      /화면.*기억해|지금.*화면.*저장|화면.*기록해|screen.*remember/i,
    ],
  },

  // ── 🎙️ 회의 어시스턴트 ──
  {
    intent: 'meeting_start',
    patterns: [
      /회의.*시작|녹음.*시작|미팅.*녹음|meeting.*start|record.*meeting|회의.*녹음/i,
    ],
  },
  {
    intent: 'meeting_stop',
    patterns: [
      /회의.*끝|녹음.*끝|녹음.*중지|녹음.*멈춰|meeting.*stop|stop.*recording/i,
    ],
  },
  {
    intent: 'meeting_summary',
    patterns: [
      /회의.*요약|미팅.*요약|녹음.*요약|회의.*정리|meeting.*summary/i,
      /회의.*내용|무슨.*얘기|뭐.*논의|액션.*아이템/i,
    ],
  },
  {
    intent: 'meeting_list',
    patterns: [
      /회의.*목록|녹음.*목록|지난.*회의|meeting.*list/i,
    ],
  },

  // ── ⌨️ 음성 받아쓰기 ──
  {
    intent: 'dictation_start',
    patterns: [
      /받아쓰기|dictation|타이핑.*해줘|입력.*해줘|써줘.*지금|적어줘.*지금/i,
      /대신.*타이핑|대신.*입력|대신.*써줘|자동.*입력/i,
    ],
  },

  {
    intent: 'persona_list',
    patterns: [/페르소나|모드.*목록|ai.*모드/i],
  },

  // ── 🌤️ 날씨 ──
  {
    intent: 'weather',
    patterns: [
      /날씨|weather|기온|온도.*오늘|오늘.*온도|비.*올까|맑아|흐려|우산/i,
      /내일.*날씨|이번.*주.*날씨|주말.*날씨|강수확률/i,
    ],
  },
  // ── 🚗 교통 시간 ──
  {
    intent: 'travel_time',
    patterns: [
      /얼마나.*걸려|교통.*시간|몇.*분.*걸려|가는.*시간|도착.*시간|출발.*몇.*시/i,
      /에서.*까지.*시간|travel.*time|driving.*time|길.*얼마/i,
    ],
  },

  // ── 🌐 번역 ──
  {
    intent: 'translate',
    patterns: [
      /번역.*해줘|번역해|translate|영어로.*바꿔|한국어로.*바꿔|일본어로|중국어로/i,
      /영문.*번역|한영.*번역|클립보드.*번역|이거.*영어로/i,
    ],
  },

  // ── 📋 클립보드 AI ──
  {
    intent: 'clipboard_ai',
    patterns: [
      /클립보드.*요약|클립보드.*교정|클립보드.*다듬어|클립보드.*ai|클립보드.*정리/i,
      /복사한.*내용.*요약|복사한.*거.*번역|클립보드.*처리/i,
    ],
  },

  // ── 📝 음성 메모→할일 ──
  {
    intent: 'voice_todo',
    patterns: [
      /할일.*추가|todo.*추가|할일.*등록|기억해줘.*날짜|해야.*한다.*기억/i,
      /(\d+일|\d+월|\d+시).*까지.*해야|마감.*기억|데드라인.*메모/i,
      /메모.*하고.*일정|일정.*하고.*메모|동시에.*추가/i,
    ],
  },
  // ── 🎭 AI 멀티 페르소나 ──────────────────────────────────────
  {
    intent: 'persona_list',
    patterns: [
      /페르소나.*목록|ai.*모드.*뭐|어떤.*모드|persona.*list/i,
      /넥서스.*종류|ai.*팀.*보여|전문가.*목록/i,
    ],
  },
  {
    intent: 'persona_switch',
    patterns: [
      /페르소나.*바꿔|모드.*전환|전문가.*모드|persona.*switch/i,
      /리서치.*모드|연구.*모드|재무.*모드|회의.*모드|크리에이티브.*모드|보안.*모드/i,
      /research.*nexus|finance.*nexus|meeting.*nexus|creative.*nexus|security.*nexus|legal.*nexus/i,
      /연구.*전문|재무.*전문|회의.*전문|창의.*전문|보안.*전문|법무.*전문/i,
      /법무.*모드|법률.*모드|계약.*모드/i,
    ],
  },
  // ── 🧠 Second Brain ──────────────────────────────────────────
  {
    intent: 'brain_search',
    patterns: [
      /second.*brain|세컨드.*브레인|기억.*검색|장기.*기억.*찾아/i,
      /과거.*(?:메모|내용|파일|회의|이메일).*찾아|예전에.*내가|작년에.*내가/i,
      /내가.*했던.*(?:프로젝트|작업|문서|메모).*찾아줘/i,
      /기억.*뒤져|brain.*search|지식.*그래프|knowledge.*graph/i,
    ],
  },
  {
    intent: 'brain_stats',
    patterns: [
      /brain.*통계|인덱스.*통계|second.*brain.*현황|몇.*개.*기억/i,
    ],
  },
  // ── ⚡ Auto Workflow ──────────────────────────────────────────
  {
    intent: 'workflow_run',
    patterns: [
      /자동.*해줘|한.*번에.*다|워크플로.*실행|workflow.*run/i,
      /(?:검색|수집|정리|요약|전송).*(?:하고|한.*다음|그리고).*(?:검색|수집|정리|요약|전송)/i,
      /만들어서.*보내줘|요약하고.*이메일|찾아서.*정리.*보고/i,
      /보고서.*만들어서.*전송|리포트.*생성.*발송/i,
      /이번\s*주.*보고서.*(?:만들|생성|작성).*(?:메일|이메일|전송)/i,
    ],
  },
  {
    intent: 'workflow_plan',
    patterns: [
      /워크플로.*계획|어떻게.*자동화|단계.*알려줘.*자동/i,
      /workflow.*plan|자동화.*방법|순서.*알려줘/i,
    ],
  },
  // ── 🎬 Live Caption ──────────────────────────────────────────
  // ── 📧 이메일 분류 ──
  {
    intent: 'email_classify',
    patterns: [
      /이메일.*분류|메일.*분류|메일.*우선순위|중요한.*메일.*구분|email.*classify/i,
      /메일.*카테고리|메일.*정리.*ai|이메일.*ai.*분석/i,
    ],
  },
  // ── 📧 이메일 답장 초안 ──
  {
    intent: 'email_draft',
    patterns: [
      /답장.*초안|메일.*초안|이메일.*답변.*써줘|reply.*draft|draft.*reply/i,
      /답장.*써줘|이메일.*대신.*써|메일.*작성.*해줘/i,
    ],
  },
  // ── 📅 빈 시간 찾기 ──
  {
    intent: 'calendar_find_slot',
    patterns: [
      /빈.*시간.*찾아|일정.*빈.*슬롯|언제.*가능|스케줄.*빈.*시간|find.*slot/i,
      /회의.*잡을.*시간|미팅.*가능한.*시간|약속.*가능.*언제/i,
    ],
  },
  // ── 📅 자연어 일정 추가 ──
  {
    intent: 'calendar_smart_add',
    patterns: [
      /다음.*주.*미팅|내일.*점심.*일정|오후.*회의.*추가|스마트.*일정/i,
      /(".*")\s*(일정|미팅|약속).*추가|일정.*자연어.*추가/i,
    ],
  },
  // ── ⚡ 워크플로 목록 ──
  {
    intent: 'workflow_list',
    patterns: [
      /워크플로.*목록|자동화.*목록.*저장|저장된.*워크플로|workflow.*list/i,
      /만들어둔.*자동화|등록.*워크플로|자동화.*뭐.*있어/i,
    ],
  },
  // ── ⚡ 워크플로 생성 ──
  {
    intent: 'workflow_create',
    patterns: [
      /워크플로.*만들어|자동화.*새로.*만들|workflow.*create|새.*자동화.*생성/i,
      /자연어.*워크플로|텍스트로.*자동화.*만들/i,
    ],
  },
  // ── ⚡ 워크플로 템플릿 ──
  {
    intent: 'workflow_templates',
    patterns: [
      /워크플로.*템플릿|자동화.*템플릿|workflow.*template|자동화.*예시/i,
    ],
  },
  // ── 📨 IMAP 받은 메일 ──
  {
    intent: 'imap_inbox',
    patterns: [
      /imap.*메일|개인.*메일.*서버|gmail.*직접|계정.*메일.*확인/i,
      /외부.*메일.*확인|서드파티.*메일|직접.*메일.*서버/i,
    ],
  },
  // ── 📨 IMAP 메일 전송 ──
  {
    intent: 'imap_send',
    patterns: [
      /imap.*보내|메일.*서버.*전송|gmail.*직접.*보내/i,
    ],
  },
  // ── 🤖 멀티 에이전트 ──
  {
    intent: 'multi_agent',
    patterns: [
      /멀티.*에이전트|여러.*ai.*동시|multi.*agent|에이전트.*팀/i,
      /병렬.*처리.*ai|동시에.*여러.*작업.*ai/i,
    ],
  },
  // ── 📢 브리핑 ──
  {
    intent: 'briefing_now',
    patterns: [
      /브리핑.*해줘|모닝.*브리핑|briefing.*now|오늘.*브리핑|아침.*브리핑/i,
      /오늘.*요약.*시작|하루.*시작.*보고|daily.*briefing/i,
    ],
  },
  // ── ❌ 작업 취소 ──
  {
    intent: 'task_cancel',
    patterns: [
      /작업.*취소|실행.*중.*멈춰|task.*cancel|진행.*중.*중지/i,
      /멈춰줘|중단해줘|취소해줘.*작업/i,
    ],
  },
  // ── 🔍 검색+PDF 보고서 ──
  {
    intent: 'search_pdf',
    patterns: [
      /검색.*pdf|pdf.*보고서.*만들어|search.*pdf|웹.*검색.*pdf/i,
      /조사.*보고서.*저장|검색.*결과.*파일로|pdf.*리포트.*생성/i,
    ],
  },
  {
    intent: 'caption_start',
    patterns: [
      /자막.*시작|실시간.*자막|live.*caption|캡션.*켜줘/i,
      /번역.*자막|자막.*번역|실시간.*번역.*자막/i,
      /유튜브.*자막|zoom.*자막|영상.*자막.*켜/i,
    ],
  },
  {
    intent: 'caption_stop',
    patterns: [
      /자막.*끄|자막.*종료|caption.*stop|캡션.*꺼줘/i,
      /자막.*멈춰|번역.*자막.*꺼/i,
    ],
  },
]

/* ─── 인텐트 감지 ─── */
export function detectIntent(text: string): Intent {
  for (const { intent, patterns } of PATTERNS) {
    for (const pattern of patterns) {
      if (pattern.test(text)) return intent
    }
  }
  return 'none'
}

/* ─── 폴더 이름 추출 ─── */
const FOLDER_NAMES = [
  '바탕화면', '다운로드', '문서', '사진', '음악', '비디오', '동영상',
  'desktop', 'downloads', 'download', 'documents', 'document',
  'pictures', 'picture', 'photos', 'music', 'videos', 'video',
  'appdata', 'temp', 'windows', 'nexus',
]

export function extractFolderName(text: string): string {
  const lower = text.toLowerCase()
  const pathMatch = text.match(/[A-Za-z]:\\[^\s]*/)?.[0]
  if (pathMatch) return pathMatch
  for (const name of FOLDER_NAMES) {
    if (lower.includes(name)) return name
  }
  const m = text.match(/["']?([^"'\s]+)["']?\s*폴더/i)
  if (m) return m[1]
  const m2 = text.match(/폴더\s+(?:열어|띄워|보여|오픈)\s*["']?([^"'\s]+)["']?/i)
  if (m2) return m2[1]
  return ''
}

/* ─── 볼륨 추출 (0~100) ─── */
export function extractVolume(text: string): { action: string; value: number } {
  if (/음소거|mute/.test(text)) return { action: 'mute', value: 0 }
  if (/음소거.*해제|unmute/.test(text)) return { action: 'unmute', value: 0 }
  const numMatch = text.match(/(\d+)\s*%?/)
  if (numMatch) return { action: 'set', value: parseInt(numMatch[1]) }
  if (/높여|키워|올려|up/.test(text)) return { action: 'set', value: 80 }
  if (/낮춰|줄여|내려|down/.test(text)) return { action: 'set', value: 30 }
  return { action: 'get', value: 0 }
}

/* ─── 밝기 추출 ─── */
export function extractBrightness(text: string): { action: string; value: number } {
  const numMatch = text.match(/(\d+)\s*%?/)
  if (numMatch) return { action: 'set', value: parseInt(numMatch[1]) }
  if (/높여|밝게|up|brighter/.test(text)) return { action: 'set', value: 80 }
  if (/낮춰|어둡게|down|dimmer/.test(text)) return { action: 'set', value: 30 }
  return { action: 'get', value: 0 }
}

/* ─── Wi-Fi 액션 추출 ─── */
export function extractWifiAction(text: string): 'on' | 'off' | 'status' {
  if (/끄|off|비활성/.test(text)) return 'off'
  if (/켜|on|활성/.test(text)) return 'on'
  return 'status'
}

/* ─── 전원 액션 추출 ─── */
export function extractPowerAction(text: string): string {
  if (/잠금|lock/.test(text)) return 'lock'
  if (/절전|sleep/.test(text)) return 'sleep'
  if (/재시작|restart|다시.*시작/.test(text)) return 'restart'
  if (/종료|shutdown|끄다|꺼줘/.test(text)) return 'shutdown'
  return 'lock'
}

/* ─── 앱 이름 추출 ─── */
export function extractAppName(text: string): string {
  const known = [
    '크롬', 'chrome', '파이어폭스', 'firefox', '엣지', 'edge',
    '메모장', 'notepad', '계산기', 'calc', '탐색기', 'explorer',
    '워드패드', 'wordpad', '페인트', 'paint', 'cmd', 'powershell',
    '작업관리자', 'taskmgr', '제어판', 'control', '설정', 'settings',
    '스팀', 'steam', '디스코드', 'discord', '카카오', 'kakaotalk',
    'vscode', 'vs code',
  ]
  const lower = text.toLowerCase()
  for (const app of known) {
    if (lower.includes(app)) return app
  }
  // 패턴: "X 켜줘" / "X 열어줘"
  const m = text.match(/(.+?)\s*(?:켜줘|열어줘|실행|오픈|launch|start|open)/i)
  if (m) return m[1].trim()
  return ''
}

/* ─── 메모 내용 추출 ─── */
export function extractNoteContent(text: string): string {
  const m = text.match(/(?:메모|기록|적어줘?|note)\s*[:：]?\s*(.+)/i)
  return m ? m[1].trim() : text.trim()
}

/* ─── 문서 비교: 파일 경로 2개 추출 ─── */
export function extractTwoFilePaths(text: string): [string, string] {
  // 1) Windows 경로 패턴: C:\...\file.ext
  const winPaths = text.match(/[A-Za-z]:\\[^\s,'"]+\.[a-zA-Z0-9]+/g) ?? []
  if (winPaths.length >= 2) return [winPaths[0]!, winPaths[1]!]

  // 2) 따옴표로 감싼 경로
  const quoted = text.match(/["']([^"']+\.[a-zA-Z0-9]+)["']/g) ?? []
  if (quoted.length >= 2) {
    return [
      (quoted[0] ?? '').replace(/['"]/g, ''),
      (quoted[1] ?? '').replace(/['"]/g, ''),
    ]
  }

  // 3) 파일명 패턴 (확장자 있는 단어)
  const fileNames = text.match(/[\w가-힣\-_()]+\.(pdf|docx?|xlsx?|txt|hwp)/gi) ?? []
  if (fileNames.length >= 2) return [fileNames[0]!, fileNames[1]!]

  return ['', '']
}

/* ─── Vision 질문 추출 ─── */
export function extractVisionQuestion(text: string): string {
  // "화면에 뭐라고 써있어?" → "화면에 뭐라고 써있어?"
  const m = text.match(/(?:화면|스크린|screen)[^?？]*[?？]?/)
  if (m) return m[0]
  return text
}

/* ─── Deep Search 쿼리 추출 ─── */
export function extractDeepSearchQuery(text: string): string {
  const m = text.match(/(?:검색|찾아|search)\s*[:：]?\s*(.+)/i)
  if (m) return m[1].trim()
  // 불필요한 단어 제거
  return text
    .replace(/파일.*내용.*검색|심층.*검색|deep.*search|내용.*검색/gi, '')
    .trim()
}

/** Proactive 알림 조건 */
export function shouldProactiveAlert(cpu: number, mem: number, temp: number, disk: number): string | null {
  if (temp > 85) return `⚠️ CPU 온도가 ${temp.toFixed(0)}°C로 매우 높아요! 과열이 우려됩니다.`
  if (cpu > 90) return `⚠️ CPU 사용률이 ${cpu.toFixed(0)}%입니다. PC가 매우 바쁜 상태예요.`
  if (mem > 90) return `⚠️ 메모리 사용량이 ${mem.toFixed(0)}%입니다. 프로그램을 종료해보세요.`
  if (disk > 95) return `⚠️ 디스크 공간이 거의 꽉 찼어요 (${disk.toFixed(0)}%). 정리가 필요합니다.`
  if (cpu > 75) return `💡 CPU 사용률이 ${cpu.toFixed(0)}%로 높네요. 백그라운드 앱을 확인해볼까요?`
  if (mem > 80) return `💡 메모리가 ${mem.toFixed(0)}% 사용 중이에요. 불필요한 탭을 닫으면 좋을 것 같아요.`
  return null
}
