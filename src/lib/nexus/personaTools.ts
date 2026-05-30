/**
 * 페르소나 → 추천 인텐트(도구) 매핑
 *
 * 사용자가 페르소나 전환 후 "이 모드에서 자주 쓰는 명령" 을 자동 노출.
 * 페르소나가 단순 시스템 프롬프트 변경이 아니라 "직업별 도구함" 으로 기능.
 */

import type { Intent } from './intentDetector'
import { getIntentSpec } from './intentRegistry'

export interface RecommendedTool {
  intent: Intent
  /** UI 라벨 (자연어 명령 예시) */
  label: string
  emoji?: string
}

export const PERSONA_TOOLS: Record<string, RecommendedTool[]> = {
  nexus: [
    { intent: 'pc_status',      label: 'PC 상태 알려줘' },
    { intent: 'security_scan',  label: '보안 점검해줘' },
    { intent: 'clean',          label: '디스크 정리해줘' },
    { intent: 'weather',        label: '오늘 날씨' },
    { intent: 'briefing_now',   label: '모닝 브리핑' },
  ],
  research: [
    { intent: 'deep_search',    label: '심층 조사: 양자컴퓨터' },
    { intent: 'news_search',    label: '최신 뉴스 검색' },
    { intent: 'doc_compare',    label: '두 문서 비교 분석' },
    { intent: 'search_pdf',     label: '리서치 PDF 보고서' },
    { intent: 'brain_search',   label: '과거 메모 검색' },
  ],
  finance: [
    { intent: 'stock_analysis', label: '종목 재무 분석' },
    { intent: 'pc_report',      label: '월간 보고서 생성' },
    { intent: 'doc_summary',    label: '재무제표 요약' },
    { intent: 'price_compare',  label: '가격 비교 분석' },
    { intent: 'report_email',   label: '리포트 메일 발송' },
  ],
  meeting: [
    { intent: 'meeting_start',     label: '회의 녹음 시작' },
    { intent: 'meeting_summary',   label: '회의 요약' },
    { intent: 'calendar_smart_add', label: '일정 추가' },
    { intent: 'email_draft',       label: '회의 후속 메일' },
    { intent: 'doc_summary',       label: '회의록 요약' },
  ],
  creative: [
    { intent: 'video_search',      label: '레퍼런스 영상 찾기' },
    { intent: 'content_script',    label: '콘텐츠 스크립트 생성' },
    { intent: 'translate',         label: '카피 번역' },
    { intent: 'vision_screen',     label: '화면 분석' },
    { intent: 'multi_agent',       label: '아이디어 브레인스토밍' },
  ],
  security: [
    { intent: 'security_scan',     label: '전체 보안 점검' },
    { intent: 'process_security',  label: '수상한 프로세스' },
    { intent: 'remote_access',     label: '원격 접속 탐지' },
    { intent: 'virus_check',       label: 'VirusTotal 파일 검사' },
    { intent: 'defender_status',   label: 'Windows Defender 상태' },
  ],
  legal: [
    { intent: 'contract_review',   label: '계약서 AI 검토' },
    { intent: 'legal_search',      label: '법률·판례 검색' },
    { intent: 'doc_compare',       label: '계약서 변경점 비교' },
    { intent: 'doc_summary',       label: '법률 문서 요약' },
    { intent: 'deep_search',       label: '규정 심층 조사' },
  ],
  // ── 직업군 12종 (Phase 5) ─────────────────────────
  developer: [
    { intent: 'doc_compare',       label: '코드 diff 비교' },
    { intent: 'doc_summary',       label: 'PR/문서 요약' },
    { intent: 'deep_search',       label: '코드베이스 검색' },
    { intent: 'process_top',       label: '리소스 잡아먹는 프로세스' },
    { intent: 'launch_app',        label: 'VS Code 열기' },
  ],
  marketer: [
    { intent: 'news_search',       label: '경쟁사 뉴스' },
    { intent: 'youtube_search',    label: '광고 영상 트렌드' },
    { intent: 'reddit_search',     label: 'Reddit 인사이트' },
    { intent: 'content_script',    label: '광고 카피 생성' },
    { intent: 'search_pdf',        label: '시장 리포트 PDF' },
  ],
  sales: [
    { intent: 'email_draft',       label: '콜드 이메일 작성' },
    { intent: 'email_send',        label: '제안서 메일 발송' },
    { intent: 'calendar_smart_add', label: '미팅 일정 추가' },
    { intent: 'calendar_find_slot', label: '빈 시간 찾기' },
    { intent: 'doc_summary',       label: '제안서 요약' },
  ],
  pm: [
    { intent: 'journal_today',     label: '오늘 업무 일지' },
    { intent: 'workflow_create',   label: 'PRD 워크플로 생성' },
    { intent: 'doc_summary',       label: '문서 요약' },
    { intent: 'meeting_summary',   label: '회의 결정사항 정리' },
    { intent: 'deep_search',       label: '경쟁 제품 조사' },
  ],
  designer: [
    { intent: 'vision_screen',     label: '디자인 화면 분석' },
    { intent: 'vision_ocr',        label: '화면 텍스트 추출' },
    { intent: 'youtube_search',    label: '디자인 트렌드 영상' },
    { intent: 'price_compare',     label: '디자인 툴 가격' },
    { intent: 'file_search',       label: '레퍼런스 파일 찾기' },
  ],
  freelancer: [
    { intent: 'pc_report',         label: '월간 작업 리포트' },
    { intent: 'email_draft',       label: '인보이스 메일' },
    { intent: 'calendar_today',    label: '오늘 일정' },
    { intent: 'doc_compare',       label: '계약서 변경점' },
    { intent: 'schedule_add',      label: '마감일 알림 추가' },
  ],
  smallbiz: [
    { intent: 'price_compare',     label: '식자재 가격 비교' },
    { intent: 'news_search',       label: '정부 지원사업 / 정책 뉴스' },
    { intent: 'reddit_search',     label: '소상공인 커뮤니티 검색' },
    { intent: 'calendar_today',    label: '오늘 예약' },
    { intent: 'report_email',      label: '월 매출 리포트' },
  ],
  corporate: [
    { intent: 'calendar_week',     label: '주간 회의 일정' },
    { intent: 'email_classify',    label: '메일 우선순위 분류' },
    { intent: 'workflow_run',      label: '결재 워크플로' },
    { intent: 'doc_summary',       label: '품의서 요약' },
    { intent: 'pc_report',         label: '월간 KPI 리포트' },
  ],
  medical: [
    { intent: 'medical_search',    label: '임상 가이드라인 검색' },
    { intent: 'deep_search',       label: 'PubMed 논문 심층 조사' },
    { intent: 'doc_summary',       label: '의료 문서 요약' },
    { intent: 'translate',         label: '영문 논문 번역' },
    { intent: 'doc_compare',       label: '두 가이드라인 비교' },
  ],
  creator: [
    { intent: 'video_download',    label: '영상 다운로드' },
    { intent: 'video_transcript',  label: '영상 자막 요약' },
    { intent: 'content_script',    label: '쇼츠 스크립트' },
    { intent: 'youtube_search',    label: '트렌드 영상' },
    { intent: 'vision_screen',     label: '썸네일 분석' },
  ],
  investor: [
    { intent: 'stock_analysis',    label: '종목 정밀 분석' },
    { intent: 'news_search',       label: '실시간 시장 뉴스' },
    { intent: 'perf_history',      label: '포트폴리오 이력' },
    { intent: 'doc_summary',       label: '리포트 핵심 요약' },
    { intent: 'briefing_now',      label: '오전 시장 브리핑' },
  ],
  tutor: [
    { intent: 'deep_search',       label: '교재 심층 조사' },
    { intent: 'doc_summary',       label: '학습 자료 요약' },
    { intent: 'translate',         label: '영어 지문 번역' },
    { intent: 'vision_screen',     label: '문제 풀이 도움' },
    { intent: 'voice_todo',        label: '학습 일정 추가' },
  ],
}

/** 페르소나의 추천 도구 가져오기 (없으면 기본 nexus) */
export function getPersonaTools(personaId: string): RecommendedTool[] {
  return PERSONA_TOOLS[personaId] ?? PERSONA_TOOLS.nexus
}

/** 페르소나의 추천 도구를 Registry 메타와 결합 (emoji 자동) */
export function getEnrichedPersonaTools(personaId: string) {
  return getPersonaTools(personaId).map(t => ({
    ...t,
    emoji: t.emoji ?? getIntentSpec(t.intent).emoji,
  }))
}
