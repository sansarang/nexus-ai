import type { Dispatch, SetStateAction, MutableRefObject } from 'react'
import { stopSpeaking, speak } from '../../lib/nexus/tts'
import { callGemini, callOllama, fallbackResponse, trackUsage, getLastPreviewItems, clearLastPreviewItems, isFollowUpQuestion } from '../../lib/nexus/gemini_engine'
import { incrementServerUsage, DAILY_FREE_LIMIT } from '../../lib/nexus/usageTracker'
import { appendHistory } from './ChatBubble'
import type { ChatMessage } from './ChatBubble'
import { detectIntent, extractFolderName, extractVolume, extractBrightness, extractWifiAction, extractPowerAction, extractAppName, extractNoteContent, extractTwoFilePaths, extractVisionQuestion, extractDeepSearchQuery } from '../../lib/nexus/intentDetector'
import type { Intent } from '../../lib/nexus/intentDetector'
import { routeWithLLM } from '../../lib/nexus/llmToolRouter'
import { backendAPI, sendCommand, emailInbox, emailSend, emailSummarize, emailClassify, emailDraftReply, virusTotalCheck, historyStats, historyAnomalies, processKill, appPermissions, windowsUpdates, gpuStats, priceCompare, newsSearch, youtubeSearch, tiktokSearch, naverShoppingSearch, coupangSearch, videoDownload, videoQuickSearch, recallCapture, recallSearch, clipboardHistory, clipboardHistoryClear, meetingStart, meetingStop, meetingList, meetingTranscribe, meetingSummarize, dictationType, dictationPaste, weatherGet, travelTime, personaList, personaSet, personaCurrent, brainSearch, brainStats, brainRebuild, workflowRun, workflowPlan, workflowList, workflowFromText, workflowTemplates, captionStart, captionStop, captionLatest, briefingNow, taskList, taskCancel, multiAgentRun, multiAgentPlan, searchAndPDF, siteSearch, getAuthHeader } from '../../lib/nexus/backendAPI'
import type { PersonaDef } from '../../lib/nexus/backendAPI'
import { learnFromTurn, saveHistory, toStoredTurns, buildMemoryContext } from '../../lib/nexus/memory'
import { safeCall } from '../../lib/nexus/environment'
import type { BackendStatus } from '../../lib/nexus/environment'
import type { NexusEmotion } from '../../types/nexus'
import type { InlineCardData } from './InlineCards'
import type { InlineCardData2 } from './InlineCards2'
import type { InlineCard3Data } from './InlineCards3'
import type { InlineCard4Data } from './InlineCards4'

interface ConversationTurn {
  role: 'user' | 'model'
  parts: Array<{ text: string }>
}

type PreviewItem = { title: string; url: string; isVideo?: boolean; isSocial?: boolean; isMap?: boolean; mapType?: string; service?: string; isImage?: boolean }

function tagPreviewItem(item: { title: string; url: string; isVideo?: boolean; isImage?: boolean; source?: string; type?: string }): PreviewItem {
  const u = item.url.toLowerCase()
  const isVideo = item.isVideo ||
    u.includes('youtube.com') || u.includes('youtu.be') ||
    u.includes('tiktok.com') || u.includes('tv.naver.com') ||
    u.includes('tving.com') || u.includes('wavve.com') ||
    item.source === 'youtube' || item.source === 'video' || item.type === 'video'
  const isSocial = !isVideo && (
    u.includes('instagram.com') || u.includes('x.com/') || u.includes('twitter.com/') ||
    item.source === 'instagram' || item.source === 'x' || item.type === 'social'
  )
  const isImage = item.isImage || u.match(/\.(jpg|jpeg|png|webp|gif)(\?|$)/i) !== null
  return { ...item, isVideo: isVideo || undefined, isSocial: isSocial || undefined, isImage: isImage || undefined }
}

function buildFrontendFallbackURLs(query: string, site: string): PreviewItem[] {
  const enc = encodeURIComponent(query)
  const s = site.toLowerCase()
  const q = query.toLowerCase()
  if (s === 'coupang' || q.includes('쿠팡'))
    return [{ title: `쿠팡에서 "${query}" 검색`, url: `https://www.coupang.com/np/search?q=${enc}` }]
  if (s === 'youtube' || q.includes('유튜브') || q.includes('youtube'))
    return [{ title: `YouTube: ${query}`, url: `https://www.youtube.com/results?search_query=${enc}`, isVideo: true }]
  return []
}

type CharacterEmotion = NexusEmotion

function isEnglishQuery(q: string): boolean {
  if (!q) return false
  const chars = [...q]
  const ascii = chars.filter(c => c.charCodeAt(0) < 128).length
  return ascii / chars.length > 0.6
}

function cleanForHistory(text: string): string {
  return text
    .replace(/\[tavily\]/gi, '')
    .replace(/https?:\/\/\S+/g, '')
    .replace(/•\s*/g, '')
    .replace(/\n{3,}/g, '\n\n')
    .trim()
}

export interface ChatSenderDeps {
  userLang: 'ko' | 'en'
  assistantName: string
  isActive: boolean
  backendStatus: BackendStatus
  subscriptionStatus: string
  clarifyPendingIntent: string | null
  clarifyPendingParams: Record<string, unknown> | null
  clarifyPendingQuestion: string | null
  floatingPreview: PreviewItem[] | null
  ttsVoice: string
  typingRef: MutableRefObject<boolean>
  historyRef: MutableRefObject<ConversationTurn[]>
  isMountedRef: MutableRefObject<boolean>
  setMessages: Dispatch<SetStateAction<ChatMessage[]>>
  setInput: Dispatch<SetStateAction<string>>
  setListening: Dispatch<SetStateAction<boolean>>
  setTyping: Dispatch<SetStateAction<boolean>>
  setTypingSteps: Dispatch<SetStateAction<string[]>>
  setEmotion: Dispatch<SetStateAction<CharacterEmotion>>
  setSpeaking: Dispatch<SetStateAction<boolean>>
  setUserLang: (lang: 'ko' | 'en') => void
  setHistoryVersion: Dispatch<SetStateAction<number>>
  setToastAlerts: Dispatch<SetStateAction<Array<{id: string; title: string; message: string; level: string}>>>
  setIsActive: Dispatch<SetStateAction<boolean>>
  setFloatingPreview: Dispatch<SetStateAction<PreviewItem[] | null>>
  setPreviewType: Dispatch<SetStateAction<string>>
  setClarifyPendingIntent: Dispatch<SetStateAction<string | null>>
  setClarifyPendingParams: Dispatch<SetStateAction<Record<string, unknown> | null>>
  setClarifyPendingQuestion: Dispatch<SetStateAction<string | null>>
  speakText: (text: string, em?: CharacterEmotion) => void
  resetClarify: () => void
  pushModelHistory: (userText: string, modelText: string) => void
  handleVoiceToggle: () => void
  handleBackendIntent: (intent: Intent, msgId: string, originalText?: string) => Promise<{ text: string; card?: InlineCardData; card2?: InlineCardData2; card3?: InlineCard3Data; card4?: InlineCard4Data; card5?: import('./InlineCards5').InlineCard5Data; emotion: CharacterEmotion }>
  renderCommandResult: (action: string, result: unknown, trimmed: string) => Promise<{ card?: InlineCardData; card2?: InlineCardData2; card3?: InlineCard3Data; card4?: InlineCard4Data; card5?: import('./InlineCards5').InlineCard5Data; emotion: CharacterEmotion }>
  showPaywall?: (feature: string, used: number, limit: number) => void
}

export async function sendTextImpl(text: string, d: ChatSenderDeps): Promise<void> {
  const { userLang, assistantName, isActive, backendStatus, subscriptionStatus,
    clarifyPendingIntent, clarifyPendingParams, clarifyPendingQuestion,
    floatingPreview, ttsVoice,
    typingRef, historyRef, isMountedRef,
    setMessages, setInput, setListening, setTyping, setTypingSteps, setEmotion, setSpeaking,
    setUserLang, setHistoryVersion, setToastAlerts, setIsActive, setFloatingPreview,
    setPreviewType, setClarifyPendingIntent, setClarifyPendingParams, setClarifyPendingQuestion,
    speakText, resetClarify, pushModelHistory, handleVoiceToggle,
    handleBackendIntent, renderCommandResult, showPaywall,
  } = d

    const trimmed = text.trim()
    if (!trimmed || typingRef.current) return

    // 새 질문 시작 → 이전 음성 즉시 중지 (말풍선·미리보기는 새 답변 올 때 교체)
    stopSpeaking()
    setSpeaking(false)

    // 비활성화 상태: 이름 호출 시만 재활성화
    if (!isActive) {
      const lower = trimmed.toLowerCase()
      const nameLower = assistantName.toLowerCase()
      if (lower.includes(nameLower) || lower.includes('넥서스') || lower.includes('nexus')) {
        setIsActive(true)
        speakText(`네, 주인님! 다시 돌아왔어요 😊`)
      }
      return
    }

    // 영어 입력 자동 감지 → UI 언어 전환
    const detectedLang: 'ko' | 'en' = isEnglishQuery(trimmed) ? 'en' : 'ko'
    if (detectedLang !== userLang) setUserLang(detectedLang)

    typingRef.current = true

    const msgId = Date.now().toString()
    // 메모리 관리: messages 100개 초과 시 오래된 일반 메시지 제거 (inlineCard 있는 메시지는 보존)
    setMessages(prev => {
      const next = [...prev, { id: msgId, role: 'user' as const, text: trimmed }]
      if (next.length <= 100) return next
      const cardIds = new Set(next.filter(m => m.inlineCard || m.inlineCard2 || m.inlineCard3 || m.inlineCard4).map(m => m.id))
      const plain = next.filter(m => !cardIds.has(m.id))
      const excess = next.length - 80
      const toRemove = new Set(plain.slice(0, excess).map(m => m.id))
      return next.filter(m => !toRemove.has(m.id))
    })
    setInput('')
    setListening(false)
    setTyping(true)
    setEmotion('neutral')
    historyRef.current.push({ role: 'user', parts: [{ text: trimmed }] })
    // 장기 메모리에 저장
    saveHistory(toStoredTurns(historyRef.current as ConversationTurn[]))

    // ── LLM clarify 해소: 원래 질문 + 사용자 답변 합쳐서 재호출 ──
    if (clarifyPendingIntent === 'llm_clarify' && clarifyPendingParams) {
      const originalQuery = (clarifyPendingParams.original_query as string) ?? ''
      const combinedInput = originalQuery
        ? `${originalQuery} (추가 정보: ${trimmed})`
        : trimmed
      resetClarify()
      const apiKey = localStorage.getItem('nexus-pplx-key') ?? ''
      let llmRes
      try { llmRes = await callGemini(apiKey, combinedInput, historyRef.current) }
      catch { /* 폴백 */ }
      if (!llmRes?.text?.trim()) llmRes = fallbackResponse(combinedInput, assistantName)
      setTyping(false)
      typingRef.current = false
      const emoMap: Record<NexusEmotion, CharacterEmotion> = {
        neutral: 'neutral', happy: 'happy', concerned: 'concerned', alert: 'alert', humorous: 'humorous',
      }
      setEmotion(emoMap[llmRes.emotion ?? 'neutral'])
      setMessages(prev => [...prev, { id: `${msgId}-res`, role: 'nexus', text: llmRes!.text }])
      pushModelHistory(trimmed, llmRes.text)
      speakText(llmRes.text)
      appendHistory({ id: msgId, ts: Date.now(), q: trimmed, a: cleanForHistory(llmRes.text) })
      setHistoryVersion(v => v + 1)
      return
    }

    // ── 0.4순위: 길찾기 / 장소 로드뷰 ─────────────────────────
    // 길찾기 감지: "에서 [목적지]" 패턴 (방법/경로 없어도 인식)
    const isDirections = (
      // 한국어: "A에서 B" + 반드시 교통/이동 키워드 포함 (기기·앱 비교 등 false-match 방지)
      /\S{2,}에서\s*\S{2,}(?:까지|가려면|가는\s*방법|가는\s*법|가는\s*길|경로|어떻게\s*가|대중교통|버스|지하철|길찾기)/i.test(trimmed) ||
      /\S{2,}에서\s+\S{2,}/.test(trimmed) && /버스|지하철|기차|ktx|택시|교통|이동|경로|길찾기|가려면|가는법|갈때/.test(trimmed.toLowerCase()) ||
      // 화살표/물결 구분자
      /\S+\s*(?:→|->|~)\s*\S+/.test(trimmed) ||
      // 영어
      /from\s+\S+.{1,30}\s+to\s+\S+/i.test(trimmed) ||
      /(?:directions?|route|how\s+(?:do\s+i\s+)?get)\s+(?:from|to)\s+\S+/i.test(trimmed) ||
      /\S+\s+to\s+\S+\s+(?:by\s+bus|by\s+train|transit|subway|directions?)/i.test(trimmed)
    )
    const isPlaceView = !isDirections && /(?:위치|어디야|어디\s*있어|어디에\s*있|주소|로드뷰|지도에서|지도\s*보여|어디\s*있는지|위치\s*알려|어디야)|(?:where\s+is\s+\S)|(?:street\s+view\s+of)|(?:location\s+of\s+\S)|(?:show\s+me\s+\S+.*(?:map|location))/i.test(trimmed)

    if (isDirections && backendStatus === 'connected') {
      try {
        setMessages(prev => [...prev, {
          id: `think-${msgId}`, role: 'nexus', text: '',
          inlineCard: { type: 'agent_thinking', steps: isEnglishQuery(trimmed) ? ['Analyzing route...', 'Connecting to map services...', 'Searching transit options...'] : ['경로 분석 중...', '지도 앱 연결 중...', '버스 노선 검색 중...'] },
        }])
        const dirCtrl = new AbortController()
        const dirTimer = setTimeout(() => dirCtrl.abort(), 10000)
        const res = await fetch('http://127.0.0.1:17891/api/directions', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json', ...await getAuthHeader() },
          body: JSON.stringify({ query: trimmed }),
          signal: dirCtrl.signal,
        }).then(r => r.json()).catch(() => null).finally(() => clearTimeout(dirTimer))

        setMessages(prev => prev.filter(m => m.id !== `think-${msgId}`))
        setTyping(false); typingRef.current = false

        if (res?.success) {
          // 교통수단별 지도 링크를 미리보기로 표시
          const mapLinks: Array<{ title: string; url: string; type?: string; service?: string; mode?: string; modeKo?: string; modeEmoji?: string }> = res.map_links ?? []
          // Google Maps(directions type) 우선, 카카오 일부, 예매 링크 제외
          const googleLinks = mapLinks.filter(l => l.service === 'google' && l.type === 'directions')
          const kakaoLinks  = mapLinks.filter(l => l.service === 'kakao'  && l.type === 'directions')
          const extraLinks  = mapLinks.filter(l => l.type !== 'directions')
          const previewLinks = [
            ...googleLinks.map(l => ({
              title: l.title ?? '', url: l.url, isMap: true, mapType: 'directions' as const,
              service: 'google', mode: l.mode, modeKo: l.modeKo, modeEmoji: l.modeEmoji,
            })),
            ...kakaoLinks.slice(0, 3).map(l => ({
              title: l.title ?? '', url: l.url, isMap: true, mapType: 'directions' as const,
              service: 'kakao', mode: l.mode, modeKo: l.modeKo, modeEmoji: l.modeEmoji,
            })),
            ...extraLinks.slice(0, 2).map(l => ({
              title: l.title ?? '', url: l.url, isMap: true, mapType: (l.type ?? 'directions') as any,
              service: l.service,
            })),
          ]
          if (previewLinks.length > 0) {
            setFloatingPreview(previewLinks as any)
          } else {
            setFloatingPreview(null)
          }

          const displayText = res.travel_summary || res.summary || (isEnglishQuery(trimmed)
            ? `**${res.from} → ${res.to}** — Check the map app for routes.`
            : `**${res.from} → ${res.to}** 경로를 지도 앱에서 확인하세요.`)

          setEmotion('happy')
          const routeCard2 = previewLinks.length > 0 ? {
            type: 'system_action' as const, icon: '🗺️',
            title: isEnglishQuery(trimmed) ? `${res.from} → ${res.to}` : `${res.from} → ${res.to} 경로`,
            detail: previewLinks.slice(0, 4).map((it: { title: string }) => `• ${it.title}`).join('\n'),
            success: true,
          } : undefined
          setMessages(prev => [...prev, {
            id: `${msgId}-res`, role: 'nexus', text: displayText, inlineCard2: routeCard2,
          }])
          pushModelHistory(trimmed, displayText)
          speakText(displayText)
          appendHistory({ id: msgId, ts: Date.now(), q: trimmed, a: cleanForHistory(displayText) })
          setHistoryVersion(v => v + 1)
          return
        }
      } catch { setMessages(prev => prev.filter(m => m.id !== `think-${msgId}`)); setTyping(false); typingRef.current = false }
    }

    if (isPlaceView && backendStatus === 'connected') {
      try {
        setMessages(prev => [...prev, {
          id: `think-${msgId}`, role: 'nexus', text: '',
          inlineCard: { type: 'agent_thinking', steps: isEnglishQuery(trimmed) ? ['Searching location...', 'Generating street view links...', 'Preparing map info...'] : ['장소 검색 중...', '로드뷰 링크 생성 중...', '지도 정보 준비 중...'] },
        }])
        const res = await fetch('http://127.0.0.1:17891/api/place-view', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ query: trimmed }),
        }).then(r => r.json()).catch(() => null)

        setMessages(prev => prev.filter(m => m.id !== `think-${msgId}`))
        setTyping(false); typingRef.current = false

        if (res?.success) {
          const mapLinks = (res.map_links ?? []).map((l: { title: string; url: string; type?: string; service?: string }) => ({
            title: l.title, url: l.url, isMap: true, mapType: l.type, service: l.service,
          }))
          const placeLinks = (res.place_info ?? []).map((l: { title: string; url: string }) => ({ title: l.title, url: l.url }))
          const allLinks = [...mapLinks, ...placeLinks]
          if (allLinks.length > 0) setFloatingPreview(allLinks as any)

          const displayText = isEnglishQuery(trimmed) ? `**${res.place}** — View the location on the map app. Click Street View to see real-world photos.` : `**${res.place}** 위치를 지도 앱에서 확인하세요. 로드뷰 버튼을 클릭하면 실제 거리 사진을 볼 수 있어요.`
          const placeCard2 = allLinks.length > 0 ? {
            type: 'system_action' as const, icon: '📍',
            title: res.place,
            detail: allLinks.slice(0, 4).map((it: { title: string }) => `• ${it.title}`).join('\n'),
            success: true,
          } : undefined
          setEmotion('happy')
          setMessages(prev => [...prev, { id: `${msgId}-res`, role: 'nexus', text: displayText, inlineCard2: placeCard2 }])
          pushModelHistory(trimmed, displayText)
          speakText(displayText)
          appendHistory({ id: msgId, ts: Date.now(), q: trimmed, a: cleanForHistory(displayText) })
          setHistoryVersion(v => v + 1)
          return
        }
      } catch { setMessages(prev => prev.filter(m => m.id !== `think-${msgId}`)); setTyping(false); typingRef.current = false }
    }

    // ── 0.5순위: 딥서치 ──────────────────────────────────────
    const isDeepSearch = /딥\s*서치|deep\s*search|자세히\s*찾|깊게\s*검색|심층\s*검색/i.test(trimmed)
    if (isDeepSearch && backendStatus === 'connected') {
      const q = trimmed.replace(/딥\s*서치|deep\s*search|자세히\s*찾아줘|깊게\s*검색해줘|심층\s*검색/gi, '').trim() || trimmed
      try {
        setMessages(prev => [...prev, {
          id: `think-${msgId}`, role: 'nexus', text: '',
          inlineCard: { type: 'agent_thinking', steps: ['딥서치 시작...', '여러 소스 병렬 검색 중...', 'AI 통합 요약 중...'] },
        }])
        const res = await backendAPI.llmDeepSearchWeb(q, 10)
        setMessages(prev => prev.filter(m => m.id !== `think-${msgId}`))
        setTyping(false); typingRef.current = false
        if (res.success) {
          const previewItems = (res.items ?? []).filter((it: { url?: string }) => it.url)
          // 문제 #2: 항상 플로팅 패널 표시 보장 (결과 없으면 검색엔진 링크라도)
          if (previewItems.length > 0) {
            setFloatingPreview(previewItems)
          } else {
            const enc = encodeURIComponent(q)
            setFloatingPreview([
              { title: `Google: ${q}`, url: `https://www.google.com/search?q=${enc}` },
              { title: `Naver: ${q}`, url: `https://search.naver.com/search.naver?query=${enc}` },
              { title: `Bing: ${q}`, url: `https://www.bing.com/search?q=${enc}` },
            ])
          }
          const displayText = res.summary || (userLang === 'en' ? 'Deep search complete.' : '딥서치 완료')
          setEmotion('happy')
          // 문제 #5: 딥서치 결과도 인라인 카드로 표시
          setMessages(prev => [...prev, {
            id: `${msgId}-res`, role: 'nexus', text: displayText,
            inlineCard2: previewItems.length > 0 ? {
              type: 'system_action',
              icon: '🔍',
              title: userLang === 'en' ? `Deep Search: ${q}` : `딥서치: ${q}`,
              detail: previewItems.slice(0, 5).map((it: { title: string }) => `• ${it.title}`).join('\n'),
              success: true,
            } : undefined,
          }])
          pushModelHistory(trimmed, displayText)
          speakText(displayText)
          appendHistory({ id: msgId, ts: Date.now(), q: trimmed, a: cleanForHistory(displayText) })
          setHistoryVersion(v => v + 1)
          return
        }
      } catch { setMessages(prev => prev.filter(m => m.id !== `think-${msgId}`)); setTyping(false); typingRef.current = false }
    }



    // ── 0.8순위: 사이트 직접 검색 (LLM 우회 → 항상 링크+미리보기 보장) ──
    const SITE_MAP: Record<string, string> = {
      // 한국 중고차
      '헤이딜러': 'heydealer.com', 'heydealer': 'heydealer.com',
      '엔카': 'encar.com', 'encar': 'encar.com',
      'kb차차차': 'kbchachacha.com', '차차차': 'kbchachacha.com',
      '보배드림': 'bobaedream.co.kr',
      // 한국 중고거래
      '당근': 'daangn.com', '당근마켓': 'daangn.com',
      '번개장터': 'bunjang.co.kr', '번개': 'bunjang.co.kr',
      '중고나라': 'joongna.com',
      // 한국 쇼핑
      '쿠팡': 'coupang.com', 'coupang': 'coupang.com',
      '네이버쇼핑': 'shopping.naver.com',
      '11번가': '11st.co.kr',
      '지마켓': 'gmarket.co.kr',
      '옥션': 'auction.co.kr',
      '무신사': 'musinsa.com',
      '에이블리': 'a-bly.com',
      '지그재그': 'zigzag.kr',
      '오늘의집': 'ohou.se',
      '태무': 'temu.com', '테무': 'temu.com', 'temu': 'temu.com',
      '알리': 'aliexpress.com', '알리익스프레스': 'aliexpress.com', 'aliexpress': 'aliexpress.com',
      '아마존': 'amazon.com', 'amazon': 'amazon.com',
      // 국제 쇼핑
      'ebay': 'ebay.com', 'etsy': 'etsy.com', 'walmart': 'walmart.com',
      'target': 'target.com', 'bestbuy': 'bestbuy.com', 'best buy': 'bestbuy.com',
      // 한국 부동산/숙박
      '직방': 'zigbang.com',
      '다방': 'dabangapp.com',
      '야놀자': 'yanolja.com',
      '여기어때': 'goodchoice.kr',
      // 국제 여행/숙박
      'airbnb': 'airbnb.com', 'booking.com': 'booking.com', 'booking': 'booking.com',
      'expedia': 'expedia.com', 'tripadvisor': 'tripadvisor.com', 'yelp': 'yelp.com',
      // 국제 부동산
      'zillow': 'zillow.com', 'realtor': 'realtor.com',
      // 가격비교
      '다나와': 'danawa.com',
      '에누리': 'enuri.com',
      // 배달
      '배민': 'baemin.com', '배달의민족': 'baemin.com',
      // 기술/개발
      'github': 'github.com', 'stackoverflow': 'stackoverflow.com', 'stack overflow': 'stackoverflow.com',
      // 엔터
      'reddit': 'reddit.com', 'imdb': 'imdb.com',
      // 교육
      'coursera': 'coursera.org', 'udemy': 'udemy.com',
    }
    const msgLower = trimmed.toLowerCase()
    let detectedSite = ''
    // 긴 키워드 먼저 체크 (당근마켓 > 당근)
    const sortedKeys = Object.keys(SITE_MAP).sort((a, b) => b.length - a.length)
    for (const kw of sortedKeys) {
      if (msgLower.includes(kw.toLowerCase())) {
        detectedSite = SITE_MAP[kw]
        break
      }
    }

    if (detectedSite && backendStatus === 'connected' && !clarifyPendingIntent) {
      // 사이트 이름 제거 후 검색어 추출
      let searchQuery = trimmed
      for (const kw of Object.keys(SITE_MAP)) {
        searchQuery = searchQuery.replace(new RegExp(kw, 'gi'), '')
      }
      searchQuery = searchQuery.replace(/에서|찾아줘|검색해줘|보여줘|알려줘|추천해줘/g, '').trim() || trimmed

      const siteLabel = detectedSite.replace('heydealer.com','헤이딜러').replace('encar.com','엔카')
        .replace('kbchachacha.com','KB차차차').replace('daangn.com','당근마켓')
        .replace('bunjang.co.kr','번개장터').replace('joongna.com','중고나라')
        .replace('coupang.com','쿠팡').replace('shopping.naver.com','네이버쇼핑')
        .replace('temu.com','테무').replace('musinsa.com','무신사')
        .replace('danawa.com','다나와').replace('yanolja.com','야놀자')
        .replace('baemin.com','배달의민족').replace('zigbang.com','직방')
        || detectedSite

      try {
        setMessages(prev => [...prev, {
          id: `think-${msgId}`, role: 'nexus', text: '',
          inlineCard: { type: 'agent_thinking', steps: userLang === 'en' ? [`Searching ${siteLabel}...`, 'Collecting live results...', 'Preparing preview...'] : [`${siteLabel} 검색 중...`, '실시간 결과 수집 중...', '미리보기 준비 중...'] },
        }])

        // 1차: /api/site-search (새 백엔드) 시도
        // 2차: /api/llm/deep-search-web (기존 백엔드) 폴백 - 백엔드 재시작 없이 작동
        let previewItems: Array<{ title: string; url: string }> = []
        let displayText = ''

        try {
          const res = await siteSearch(searchQuery, detectedSite, 8)
          if (res.success && res.results.length > 0) {
            previewItems = res.results.map((it: { name: string; link: string }) => ({ title: it.name, url: it.link }))
            displayText = res.summary
          }
        } catch {
          // /api/site-search 없는 구버전 백엔드 → deep-search-web 폴백
        }

        if (previewItems.length === 0) {
          const siteQuery = `site:${detectedSite} ${searchQuery}`
          const dr = await backendAPI.llmDeepSearchWeb(siteQuery, 8)
          if (dr.success && dr.items && dr.items.length > 0) {
            previewItems = dr.items
              .filter((it: { url?: string }) => it.url)
              .map((it: { title: string; url: string }) => ({ title: it.title, url: it.url }))
            displayText = dr.summary || `${siteLabel}에서 "${searchQuery}" 결과예요.`
          }
        }

        // 그래도 없으면 해당 사이트 검색 직접 링크라도 제공
        if (previewItems.length === 0) {
          const enc = encodeURIComponent(searchQuery)
          const fallbackURLs: Record<string, string> = {
            'daangn.com': `https://www.daangn.com/search/${enc}`,
            'bunjang.co.kr': `https://m.bunjang.co.kr/search/products?q=${enc}`,
            'encar.com': `https://www.encar.com/search/car?searchKey=${enc}`,
            'heydealer.com': `https://www.heydealer.com/car/search?keyword=${enc}`,
            'coupang.com': `https://www.coupang.com/np/search?q=${enc}`,
            'shopping.naver.com': `https://search.shopping.naver.com/search/all?query=${enc}`,
            'musinsa.com': `https://www.musinsa.com/search/musinsa/integration?q=${enc}`,
            'danawa.com': `https://search.danawa.com/dsearch.php?query=${enc}`,
            'yanolja.com': `https://www.yanolja.com/keyword/${enc}`,
          }
          const url = fallbackURLs[detectedSite] || `https://www.${detectedSite}/search?q=${enc}`
          previewItems = [{ title: `${siteLabel}에서 "${searchQuery}" 검색하기`, url }]
          displayText = `${siteLabel} 앱이나 사이트에서 직접 확인해보세요.`
        }

        setMessages(prev => prev.filter(m => m.id !== `think-${msgId}`))
        setTyping(false); typingRef.current = false
        setFloatingPreview(previewItems)
        setEmotion('happy')
        if (!displayText) displayText = `${siteLabel}에서 "${searchQuery}" 결과예요.`
        setMessages(prev => [...prev, {
          id: `${msgId}-res`, role: 'nexus', text: displayText,
          inlineCard2: { type: 'system_action', icon: '🔍', title: `${siteLabel}: ${searchQuery}`, detail: previewItems.slice(0,5).map(it => `• ${it.title}`).join('\n'), success: true },
        }])
        pushModelHistory(trimmed, displayText)
        speakText(displayText)
        appendHistory({ id: msgId, ts: Date.now(), q: trimmed, a: cleanForHistory(displayText) })
        setHistoryVersion(v => v + 1)
        return
      } catch {
        setMessages(prev => prev.filter(m => m.id !== `think-${msgId}`))
        setTyping(false)
        typingRef.current = false
      }
    }

    // ── 0.9순위: 키워드 즉시 감지 인텐트 — LLM 라우팅 완전 우회 ──
    // 날씨/PC상태/시스템제어 등 키워드로 확정되는 인텐트는
    // Go 백엔드 LLM 라우팅(최대 60s 타임아웃) 없이 즉시 처리
    {
      const FAST_INTENTS = new Set<Intent>([
        'weather', 'travel_time',
        'pc_status', 'gpu_stats', 'process_top',
        'volume_control', 'brightness', 'wifi_toggle', 'power_action', 'launch_app',
        'focus_mode',
        'calendar_today', 'calendar_week',
        'email_inbox',
        'network_analysis', 'defender_status', 'startup_items', 'windows_updates',
      ])
      const fastIntent = detectIntent(trimmed)
      if (FAST_INTENTS.has(fastIntent) && backendStatus === 'connected') {
        setMessages(prev => [...prev, {
          id: `think-${msgId}`, role: 'nexus', text: '',
          inlineCard: { type: 'agent_thinking', steps: fastIntent === 'weather'
            ? (detectedLang === 'en' ? ['Fetching weather...', 'Getting current conditions...'] : ['🌤️ 날씨 데이터 가져오는 중...', '현재 날씨 확인 중...'])
            : fastIntent === 'pc_status'
              ? (detectedLang === 'en' ? ['Collecting PC stats...', 'CPU / Memory / Disk'] : ['📊 PC 상태 수집 중...', 'CPU / 메모리 / 디스크'])
              : [detectedLang === 'en' ? 'Processing...' : '처리 중...'] },
        }])
        const { text: resText, card, card2, card3, card4, emotion: resEmotion } = await handleBackendIntent(fastIntent, msgId, trimmed)
        setMessages(prev => prev.filter(m => m.id !== `think-${msgId}`))
        setTyping(false)
        typingRef.current = false
        setEmotion(resEmotion)
        setMessages(prev => [...prev, { id: `${msgId}-res`, role: 'nexus', text: resText, inlineCard: card, inlineCard2: card2, inlineCard3: card3, inlineCard4: card4 }])
        pushModelHistory(trimmed, resText)
        if (resText) {
          speakText(resText)
          appendHistory({ id: msgId, ts: Date.now(), q: trimmed, a: cleanForHistory(resText) })
          setHistoryVersion(v => v + 1)
        }
        return
      }
    }

    // ── 1순위: Go 백엔드 /api/command (LLM 자동 라우팅 + 멀티턴) ─
    if (backendStatus === 'connected') {
      try {
        const getThinkSteps = (msg: string): string[] => {
          const m = msg.toLowerCase()
          if (/주가|코스피|나스닥|비트코인|코인|etf|ticker/i.test(m)) return ['📊 주가 데이터 조회 중...', '실시간 시세 가져오는 중...', '결과 정리 중...']
          if (/환율|달러|엔화|유로|위안|환전/i.test(m)) return ['💱 환율 데이터 가져오는 중...', '최신 환율 계산 중...']
          if (/화면|스크린|screenshot|screen/i.test(m)) return ['🖥️ 화면 캡처 중...', 'AI 이미지 분석 중...', '결과 정리 중...']
          if (/클립보드|복사한|방금 복사/i.test(m)) return ['📋 클립보드 읽는 중...', 'AI 처리 중...', '결과 정리 중...']
          if (/파일|폴더|정리|downloads|바탕화면/i.test(m)) return ['📁 파일 스캔 중...', '분류 기준 적용 중...', '정리 완료 준비 중...']
          if (/날씨|기온|미세먼지|비|눈/i.test(m)) return ['🌤️ 날씨 데이터 가져오는 중...', '현재 위치 확인 중...']
          if (/번역|translat/i.test(m)) return ['🌐 번역 중...', '결과 정리 중...']
          if (/검색|찾아|뉴스|알려줘|리서치|조사/i.test(m)) return ['🔍 웹 검색 중...', '실시간 결과 수집 중...', 'AI 요약 중...']
          if (/앱.*켜|열어|실행|launch/i.test(m)) return ['🚀 앱 실행 중...']
          if (/자세히|깊게|deep|분석/i.test(m)) return ['🔬 심층 리서치 시작...', '여러 소스 검색 중...', 'AI 종합 분석 중...']
          return ['🤔 요청 분석 중...', '처리 중...']
        }
        const thinkSteps = getThinkSteps(trimmed)
        setTypingSteps(thinkSteps)
        setMessages(prev => [...prev, {
          id: `think-${msgId}`, role: 'nexus', text: '',
          inlineCard: { type: 'agent_thinking', steps: thinkSteps },
        }])

        // 멀티턴: clarify 컨텍스트 + 최근 대화 이력 포함
        const recentHistory = historyRef.current.map(h => ({
          role: (h.role === 'user' ? 'user' : 'assistant') as 'user' | 'assistant',
          content: h.parts?.[0]?.text ?? '',
        })).filter(h => h.content.length > 0)

        // 페르소나 ID: 백엔드 Tavily 도메인 필터 적용에 사용
        const activePersonaId = localStorage.getItem('nexus_vertical_id') ?? 'general'
        const cmd = await sendCommand(trimmed, {
          lang:            detectedLang,
          pendingIntent:   clarifyPendingIntent   ?? undefined,
          pendingParams:   clarifyPendingParams   ?? undefined,
          pendingQuestion: clarifyPendingQuestion ?? undefined,
          history:         recentHistory,
          userEmail:       localStorage.getItem('nexus-user-email') ?? '',
          context:         activePersonaId !== 'general' ? `persona:${activePersonaId}` : undefined,
        })
        setMessages(prev => prev.filter(m => m.id !== `think-${msgId}`))

        // ── upgrade_required: 페이월 표시 ───────────────────────
        if (cmd.upgrade_required) {
          setTyping(false)
          typingRef.current = false
          setEmotion('concerned')
          setMessages(prev => [...prev, {
            id: `${msgId}-res`, role: 'nexus',
            text: cmd.message,
            emotion: 'sad',
          }])
          showPaywall?.(
            cmd.feature_name ?? cmd.action,
            cmd.used_count ?? 0,
            cmd.limit_count ?? 0,
          )
          return
        }

        // ── API 키 미설정 오류: 설정 화면 유도 ─────────────────
        if (!cmd.success && cmd.message && (
          /api.?key|키.*없|키.*설정|not configured|required/i.test(cmd.message)
        )) {
          setTyping(false); typingRef.current = false
          setEmotion('concerned')
          const isEn = detectedLang === 'en'
          const guideMsg = isEn
            ? `${cmd.message}\n\n👉 Open **Settings (⚙️)** → API Keys to configure your keys.`
            : `${cmd.message}\n\n👉 우측 상단 **설정(⚙️) → API 키**에서 키를 입력해주세요.`
          setMessages(prev => [...prev, { id: `${msgId}-res`, role: 'nexus', text: guideMsg }])
          return
        }

        if (cmd.success) {
          // ── clarify: 추가 질문 필요 ──────────────────────────
          if (cmd.action === 'clarify' && cmd.needs_clarify) {
            const question = cmd.clarify_question || cmd.message || (detectedLang === 'en' ? 'Could you provide more details?' : '조금 더 알려주세요.')
            setClarifyPendingIntent(cmd.pending_intent ?? null)
            setClarifyPendingParams((cmd.pending_params as Record<string, unknown>) ?? null)
            setClarifyPendingQuestion(question)

            setTyping(false)
            typingRef.current = false
            setEmotion('neutral')
            const clarifyOpts: string[] = (cmd as any).clarify_questions ?? []
            setMessages(prev => [...prev, {
              id: `${msgId}-res`, role: 'nexus',
              text: question,
              clarifyOptions: clarifyOpts,
              onClarifySelect: (opt: string) => { sendTextImpl(opt, d) },
            }])
            pushModelHistory(trimmed, question)
            // Clarify 질문 TTS — 자동 마이크 시작은 설정에 따름
            const clarifyAutoMic = localStorage.getItem('nexus-clarify-auto-mic') !== 'false'
            speak(question, userLang, () => setSpeaking(true), () => {
              setSpeaking(false)
              if (clarifyAutoMic && isMountedRef.current) {
                setTimeout(() => {
                  if (isMountedRef.current && !typingRef.current) handleVoiceToggle()
                }, 300)
              }
            })
            return
          }

          // ── 정상 실행: clarify 상태 초기화 ──────────────────
          resetClarify()
          const { card, card2, card3, card4, card5, emotion: cmdEmotion } = await renderCommandResult(cmd.action, cmd.result, trimmed)
          const displayText = cmd.message || ''
          setTyping(false)
          typingRef.current = false
          setEmotion(cmdEmotion)

          let previewSet = false

          // 가격 비교 결과 미리보기
          if (cmd.action === 'price_compare') {
            const r = cmd.result as { results?: Array<{ name?: string; link?: string }> } | undefined
            const priceItems = (r?.results ?? [])
              .filter((it): it is { name: string; link: string } => !!(it.link))
              .map(it => ({ title: it.name ?? it.link, url: it.link }))
            if (priceItems.length > 0) { setFloatingPreview(priceItems); previewSet = true }
          }

          // 영상 검색 결과 미리보기 (video_search)
          if (cmd.action === 'video_search') {
            const resultObj = cmd.result as { items?: Array<{ title?: string; url?: string }> } | undefined
            const videoItems = (resultObj?.items ?? [])
              .filter((it): it is { title: string; url: string } => !!(it.url))
              .map(it => ({ title: it.title ?? it.url, url: it.url, isVideo: true }))
            if (videoItems.length > 0) { setFloatingPreview(videoItems); previewSet = true }
          }

          // 멀티액션 미리보기
          if (cmd.action === 'multi_action') {
            const r = cmd.result as { results?: Array<{ name?: string; link?: string }> } | undefined
            const maItems = (r?.results ?? []).filter((it): it is { name: string; link: string } => !!(it.link))
              .map(it => ({ title: it.name ?? it.link, url: it.link }))
            if (maItems.length > 0) { setFloatingPreview(maItems); previewSet = true }
          }

          // ── 모든 액션 공통: result.items / articles / results → floatingPreview ──
          // web_search, news_search, chat, weather, stock, exchange_rate, trip_plan 등
          if (!previewSet && cmd.result) {
            const anyResult = cmd.result as Record<string, unknown>
            // 다양한 필드명 정규화: items > articles > results
            const rawAny: Array<Record<string, string>> = (
              (anyResult.items as Array<Record<string, string>> | undefined) ??
              (anyResult.articles as Array<Record<string, string>> | undefined) ??
              []
            )
            if (rawAny.length === 0 && Array.isArray(anyResult.results)) {
              // results 배열: { name/title, link/url } 형태 정규화
              const normalized = (anyResult.results as Array<Record<string, string>>)
                .map(r => ({ title: r.name ?? r.title ?? r.link ?? r.url, url: r.link ?? r.url }))
                .filter(it => !!it.url)
              if (normalized.length > 0) {
                setFloatingPreview(normalized.map(it => tagPreviewItem({ title: it.title, url: it.url })) as any)
                previewSet = true
              }
            } else if (rawAny.length > 0) {
              const universalItems = rawAny
                .filter(it => !!(it.url ?? it.link))
                .map(it => tagPreviewItem({ title: it.title ?? it.name ?? it.url ?? it.link, url: it.url ?? it.link }))
              if (universalItems.length > 0) {
                setFloatingPreview(universalItems as any)
                previewSet = true
              }
            }
          }

          // web_search / chat 공통: result.items → tagPreviewItem 적용 (추가 정보 enrichment용)
          if (!previewSet && (cmd.action === 'web_search' || cmd.action === 'chat' || cmd.action === 'weather')) {
            const resultObj = cmd.result as { items?: Array<{ title?: string; url?: string; type?: string; source?: string }>; query?: string; site?: string; preview_type?: string } | undefined
            // preview_type 설정
            if (resultObj?.preview_type) setPreviewType(resultObj.preview_type)
            let rawItems: Array<{ title?: string; url?: string; type?: string; source?: string }> = resultObj?.items ?? []

            // items 없으면 카테고리 인식 fallback
            if (rawItems.length === 0) {
              const searchQuery = resultObj?.query ?? trimmed
              const site = resultObj?.site ?? 'auto'
              rawItems = buildFrontendFallbackURLs(searchQuery, site)
            }

            const previewItems = rawItems
              .filter(it => !!(it.url))
              .map(it => tagPreviewItem({ title: (it.title ?? it.url) as string, url: it.url as string, source: it.source, type: it.type }))

            if (previewItems.length > 0) { setFloatingPreview(previewItems as any); previewSet = true }

            // 백그라운드로 YouTube/Instagram/X 병렬 검색 → preview에 append
            const isTransitQuery = /버스|지하철|기차|ktx|택시|경로|길찾기|에서.*가는|에서.*까지|directions|route|transit/i.test(trimmed)
            // 정보성 쿼리(우편번호·날씨·계산·환율·번역 등)는 SNS 검색 건너뜀
            const isInfoQuery = /우편번호|zip\s*code|postal|날씨|기온|미세먼지|환율|주가|주식|계산|더하기|빼기|곱하기|나누기|몇\s*살|나이|생일|번역|translate|정의|뜻|이란|이란\?|공식|수식|공항|비행|편명|시간표|전화번호|주소\s*알려|몇\s*시|몇\s*층|몇\s*호|층수|면적|넓이|인구|gdp|환산|convert/i.test(trimmed)
            if (previewSet && !isTransitQuery && !isInfoQuery) {
              Promise.allSettled([
                videoQuickSearch(trimmed, 'youtube', 3),
                videoQuickSearch(trimmed, 'tiktok', 2),
                videoQuickSearch(trimmed, 'instagram', 2),
                videoQuickSearch(trimmed, 'x', 2),
              ]).then(results => {
                const newItems: typeof floatingPreview = []
                for (const r of results) {
                  if (r.status === 'fulfilled' && r.value.items?.length) {
                    for (const it of r.value.items) {
                      newItems.push(tagPreviewItem({ title: it.title, url: it.url, type: it.type ?? 'video', source: it.platform }) as any)
                    }
                  }
                }
                if (newItems.length > 0) {
                  setFloatingPreview(prev => prev ? [...prev, ...newItems] : newItems)
                }
              })
            }
          }

          setMessages(prev => [...prev, {
            id: `${msgId}-res`, role: 'nexus', text: displayText,
            inlineCard: card, inlineCard2: card2, inlineCard3: card3, inlineCard4: card4, inlineCard5: card5,
            action: cmd.action, // follow-up 칩 표시를 위한 액션 키
          }])
          pushModelHistory(trimmed, displayText)
          if (displayText) {
            speakText(displayText)
            appendHistory({ id: msgId, ts: Date.now(), q: trimmed, a: cleanForHistory(displayText) })
            setHistoryVersion(v => v + 1)
          }
          return
        }
      } catch {
        setMessages(prev => prev.filter(m => m.id !== `think-${msgId}`))
        resetClarify()
      }
    }

    // ── 2순위: 로컬 detectIntent ──────────────────────────────
    const intent = detectIntent(trimmed)
    if (intent !== 'none') {
      const { text: resText, card, card2, card3, card4, emotion: resEmotion } = await handleBackendIntent(intent, msgId, trimmed)
      setTyping(false)
      typingRef.current = false
      setEmotion(resEmotion)
      setMessages(prev => [...prev, { id: `${msgId}-res`, role: 'nexus', text: resText, inlineCard: card, inlineCard2: card2, inlineCard3: card3, inlineCard4: card4 }])
      pushModelHistory(trimmed, resText)
      if (resText) {
        speakText(resText)
        appendHistory({ id: msgId, ts: Date.now(), q: trimmed, a: cleanForHistory(resText) })
        setHistoryVersion(v => v + 1)
      }
      return
    }

    // ── 3순위: LLM 일반 대화 ─────────────────────────────────
    // apiKey는 env 또는 localStorage 어디서든 가져옴
    const apiKey = localStorage.getItem('nexus-pplx-key') ?? ''
    let response

    try {
      const r = await callOllama(trimmed, historyRef.current)
      if (r) response = r
    } catch { /* Ollama 미실행 */ }

    // ── 사용량 체크: 무료 유저는 서버(Supabase) 연동, 프리미엄은 패스 ──
    const isFreeUser = !subscriptionStatus || subscriptionStatus === 'none' || subscriptionStatus === 'expired'
    if (isFreeUser) {
      const { allowed, used } = await incrementServerUsage()
      if (!allowed) {
        setTyping(false)
        typingRef.current = false
        showPaywall?.('ai_chat', used, DAILY_FREE_LIMIT)
        return
      }
    }

    if (!response) {
      if (apiKey) {
        // API 키 있음 → 직접 호출
        try { response = await callGemini(apiKey, trimmed, historyRef.current) }
        catch (e) { console.warn('[LLM] 직접 호출 실패:', e) }
      }
      if (!response) {
        // API 키 없음 or 실패 → 백엔드 프록시 경유 (로그인 JWT 사용)
        try {
          const auth = await getAuthHeader()
          const messages = [
            { role: 'system', content: detectedLang === 'en'
              ? `You are Nexus AI, a helpful assistant. Answer naturally and helpfully in English.`
              : `당신은 Nexus AI 비서입니다. 사용자를 "주인님"으로 부르며 친절하게 답변하세요.` },
            ...historyRef.current.slice(-12).map((t: { role: string; parts: Array<{ text: string }> }) => ({
              role: t.role === 'user' ? 'user' : 'assistant',
              content: t.parts[0]?.text ?? '',
            })),
            { role: 'user', content: trimmed },
          ]
          const res = await fetch('http://127.0.0.1:17891/api/llm/chat', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json', ...auth },
            body: JSON.stringify({ messages, max_tokens: 1024 }),
          })
          const json = await res.json() as { success: boolean; answer: string }
          if (json.success && json.answer) {
            response = { text: json.answer, emotion: 'neutral', steps: [] }
          }
        } catch (e) { console.warn('[LLM] 백엔드 프록시 실패:', e) }
      }
    }

    if (!response || !response.text?.trim()) response = fallbackResponse(trimmed, assistantName)

    setTyping(false)
    typingRef.current = false

    // ── LLM clarify: 추가 정보 필요 ──────────────────────────
    if (response.needs_clarify && response.clarify_question) {
      const question = response.clarify_question
      setClarifyPendingIntent(response.clarify_intent ?? 'llm_clarify')
      setClarifyPendingParams(response.clarify_params ?? { original_query: trimmed })
      setClarifyPendingQuestion(question)
      setEmotion('neutral')
      setMessages(prev => [...prev, { id: `${msgId}-res`, role: 'nexus', text: response!.text }])
      pushModelHistory(trimmed, response!.text)
      speakText(response.text)
      return
    }

    const emotionMap: Record<NexusEmotion, CharacterEmotion> = {
      neutral: 'neutral', happy: 'happy', concerned: 'concerned',
      alert: 'alert', humorous: 'humorous',
    }
    setEmotion(emotionMap[(response.emotion ?? 'neutral') as NexusEmotion])

    // 미리보기는 오른쪽 플로팅 패널에만 표시
    const previewItems = response.preview_items ?? getLastPreviewItems()
    clearLastPreviewItems()
    if ((response as any).preview_type) setPreviewType((response as any).preview_type)
    if (previewItems.length > 0) setFloatingPreview(previewItems)

    // UI 일관성: 링크가 있으면 채팅 버블에도 inlineCard2 요약 카드 표시
    const llmCard2 = previewItems.length > 0 ? {
      type: 'system_action' as const, icon: '🔍',
      title: userLang === 'en' ? 'Related Links' : '관련 링크',
      detail: previewItems.slice(0, 5).map((it: { title: string }) => `• ${it.title}`).join('\n'),
      success: true,
    } : undefined

    setMessages(prev => [...prev, {
      id: `${msgId}-res`, role: 'nexus', text: response!.text, inlineCard2: llmCard2,
    }])
    pushModelHistory(trimmed, response.text)
    if (response.text) {
      speakText(response.text)
      appendHistory({ id: msgId, ts: Date.now(), q: trimmed, a: cleanForHistory(response.text) })
      setHistoryVersion(v => v + 1)
    }
}
