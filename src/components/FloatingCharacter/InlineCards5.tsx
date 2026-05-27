/**
 * InlineCards5 — 웹검색 / 뉴스 / 유튜브 결과 전용 카드
 */
import React from 'react'
import { motion } from 'framer-motion'

/* ── 공통 타입 ──────────────────────────────── */
export interface SearchResultItem {
  title: string
  url: string
  snippet?: string
  source?: string
  published?: string
  thumbnail?: string
}

export type InlineCard5Data =
  | { type: 'web_search';  query: string; summary: string; items: SearchResultItem[] }
  | { type: 'news_search'; query: string; summary: string; items: SearchResultItem[] }
  | { type: 'youtube';     query: string; items: SearchResultItem[] }

/* ── 유틸 ──────────────────────────────────── */
function domain(url: string) {
  try { return new URL(url).hostname.replace('www.', '') } catch { return url.slice(0, 28) }
}

function faviconUrl(url: string) {
  try {
    const host = new URL(url).hostname
    return `https://www.google.com/s2/favicons?domain=${host}&sz=16`
  } catch { return '' }
}

function timeAgo(iso?: string): string {
  if (!iso) return ''
  const d = new Date(iso)
  if (isNaN(d.getTime())) return iso.slice(0, 10)
  const diff = Date.now() - d.getTime()
  const mins = Math.floor(diff / 60000)
  if (mins < 60) return `${mins}분 전`
  const hrs = Math.floor(mins / 60)
  if (hrs < 24) return `${hrs}시간 전`
  const days = Math.floor(hrs / 24)
  if (days < 7) return `${days}일 전`
  return `${d.getMonth() + 1}/${d.getDate()}`
}

/* ── 래퍼 카드 ─────────────────────────────── */
function Card5Wrap({ children, accentColor }: { children: React.ReactNode; accentColor: string }) {
  return (
    <motion.div
      initial={{ opacity: 0, y: 8, scale: 0.97 }}
      animate={{ opacity: 1, y: 0, scale: 1 }}
      transition={{ duration: 0.22, ease: 'easeOut' }}
      style={{
        background: 'rgba(8,8,22,0.97)',
        border: `1px solid ${accentColor}44`,
        borderLeft: `3px solid ${accentColor}`,
        borderRadius: 14,
        padding: '10px 12px',
        display: 'flex',
        flexDirection: 'column',
        gap: 0,
        width: '100%',
        boxShadow: `0 6px 28px rgba(0,0,0,0.7)`,
        marginTop: 6,
      }}
    >
      {children}
    </motion.div>
  )
}

/* ── 웹검색 카드 ────────────────────────────── */
function WebSearchCard({ data, accentColor }: { data: Extract<InlineCard5Data, { type: 'web_search' }>; accentColor: string }) {
  const topItems = data.items.slice(0, 3)
  return (
    <Card5Wrap accentColor={accentColor}>
      {/* 헤더 */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginBottom: 8 }}>
        <span style={{ fontSize: 13 }}>🔍</span>
        <span style={{ fontSize: 11, color: accentColor, fontWeight: 800, letterSpacing: '0.04em' }}>
          웹 검색: {data.query}
        </span>
      </div>
      {/* 요약 */}
      {data.summary && (
        <div style={{
          fontSize: 11.5,
          color: 'rgba(255,255,255,0.82)',
          lineHeight: 1.6,
          marginBottom: 8,
          paddingBottom: 8,
          borderBottom: '1px solid rgba(255,255,255,0.07)',
        }}>
          {data.summary.slice(0, 220)}{data.summary.length > 220 ? '…' : ''}
        </div>
      )}
      {/* 출처 상위 3개 */}
      {topItems.length > 0 && (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 5 }}>
          {topItems.map((item, i) => (
            <div key={i}
              onClick={() => window.open(item.url, '_blank')}
              style={{ display: 'flex', alignItems: 'center', gap: 6, cursor: 'pointer', padding: '4px 5px', borderRadius: 7, transition: 'background 0.15s' }}
              onMouseEnter={e => (e.currentTarget.style.background = 'rgba(255,255,255,0.06)')}
              onMouseLeave={e => (e.currentTarget.style.background = 'transparent')}
            >
              <img src={faviconUrl(item.url)} width={14} height={14} style={{ borderRadius: 3, flexShrink: 0 }} onError={e => { (e.target as HTMLImageElement).style.display = 'none' }} />
              <div style={{ flex: 1, minWidth: 0 }}>
                <div style={{ fontSize: 10.5, color: 'rgba(255,255,255,0.88)', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', fontWeight: 600 }}>
                  {item.title}
                </div>
                <div style={{ fontSize: 9, color: 'rgba(255,255,255,0.35)', marginTop: 1 }}>{domain(item.url)}</div>
              </div>
              <span style={{ fontSize: 9, color: accentColor, flexShrink: 0 }}>↗</span>
            </div>
          ))}
        </div>
      )}
    </Card5Wrap>
  )
}

/* ── 뉴스 카드 ──────────────────────────────── */
function NewsCard({ data, accentColor }: { data: Extract<InlineCard5Data, { type: 'news_search' }>; accentColor: string }) {
  const topNews = data.items.slice(0, 4)
  return (
    <Card5Wrap accentColor={'#f59e0b'}>
      {/* 헤더 */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginBottom: 8 }}>
        <span style={{ fontSize: 13 }}>📰</span>
        <span style={{ fontSize: 11, color: '#f59e0b', fontWeight: 800 }}>뉴스: {data.query}</span>
      </div>
      {/* 요약 */}
      {data.summary && (
        <div style={{
          fontSize: 11.5, color: 'rgba(255,255,255,0.82)', lineHeight: 1.6,
          marginBottom: 8, paddingBottom: 8, borderBottom: '1px solid rgba(255,255,255,0.07)',
        }}>
          {data.summary.slice(0, 180)}{data.summary.length > 180 ? '…' : ''}
        </div>
      )}
      {/* 뉴스 목록 */}
      <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
        {topNews.map((item, i) => (
          <div key={i}
            onClick={() => window.open(item.url, '_blank')}
            style={{ display: 'flex', alignItems: 'flex-start', gap: 7, cursor: 'pointer', padding: '5px 5px', borderRadius: 8, transition: 'background 0.15s' }}
            onMouseEnter={e => (e.currentTarget.style.background = 'rgba(245,158,11,0.08)')}
            onMouseLeave={e => (e.currentTarget.style.background = 'transparent')}
          >
            {/* 순서 번호 */}
            <div style={{
              width: 17, height: 17, borderRadius: 4, background: '#f59e0b22',
              display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0, marginTop: 1,
            }}>
              <span style={{ fontSize: 8.5, color: '#f59e0b', fontWeight: 800 }}>{i + 1}</span>
            </div>
            <div style={{ flex: 1, minWidth: 0 }}>
              <div style={{ fontSize: 10.5, color: 'rgba(255,255,255,0.9)', lineHeight: 1.4, fontWeight: 600, marginBottom: 2 }}>
                {item.title}
              </div>
              <div style={{ display: 'flex', gap: 6, alignItems: 'center' }}>
                <img src={faviconUrl(item.url)} width={11} height={11} style={{ borderRadius: 2 }} onError={e => { (e.target as HTMLImageElement).style.display = 'none' }} />
                <span style={{ fontSize: 8.5, color: 'rgba(255,255,255,0.35)' }}>{domain(item.url)}</span>
                {item.published && (
                  <span style={{ fontSize: 8.5, color: 'rgba(245,158,11,0.7)' }}>{timeAgo(item.published)}</span>
                )}
              </div>
            </div>
          </div>
        ))}
      </div>
    </Card5Wrap>
  )
}

/* ── 유튜브 카드 ─────────────────────────────── */
function YoutubeCard({ data, accentColor }: { data: Extract<InlineCard5Data, { type: 'youtube' }>; accentColor: string }) {
  const topVids = data.items.slice(0, 4)
  return (
    <Card5Wrap accentColor={'#e53e3e'}>
      {/* 헤더 */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginBottom: 8 }}>
        <span style={{ fontSize: 13 }}>🎬</span>
        <span style={{ fontSize: 11, color: '#e53e3e', fontWeight: 800 }}>유튜브: {data.query}</span>
      </div>
      {/* 영상 목록 */}
      <div style={{ display: 'flex', flexDirection: 'column', gap: 5 }}>
        {topVids.map((item, i) => {
          // youtu.be/VIDEO_ID 또는 youtube.com/watch?v=VIDEO_ID에서 썸네일
          let thumbUrl = ''
          const ytMatch = item.url.match(/(?:v=|youtu\.be\/)([A-Za-z0-9_-]{11})/)
          if (ytMatch) {
            thumbUrl = `https://img.youtube.com/vi/${ytMatch[1]}/mqdefault.jpg`
          }
          return (
            <div key={i}
              onClick={() => window.open(item.url, '_blank')}
              style={{ display: 'flex', gap: 8, cursor: 'pointer', borderRadius: 8, padding: '4px 4px', transition: 'background 0.15s' }}
              onMouseEnter={e => (e.currentTarget.style.background = 'rgba(229,62,62,0.08)')}
              onMouseLeave={e => (e.currentTarget.style.background = 'transparent')}
            >
              {/* 썸네일 or 번호 */}
              {thumbUrl ? (
                <div style={{ position: 'relative', width: 64, height: 40, borderRadius: 6, overflow: 'hidden', flexShrink: 0 }}>
                  <img src={thumbUrl} width={64} height={40} style={{ objectFit: 'cover', width: '100%', height: '100%' }}
                    onError={e => { (e.target as HTMLImageElement).parentElement!.style.display = 'none' }} />
                  <div style={{ position: 'absolute', inset: 0, display: 'flex', alignItems: 'center', justifyContent: 'center', background: 'rgba(0,0,0,0.25)' }}>
                    <span style={{ fontSize: 14, color: '#fff' }}>▶</span>
                  </div>
                </div>
              ) : (
                <div style={{ width: 64, height: 40, borderRadius: 6, background: '#e53e3e22', display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0 }}>
                  <span style={{ fontSize: 18, color: '#e53e3e' }}>▶</span>
                </div>
              )}
              <div style={{ flex: 1, minWidth: 0 }}>
                <div style={{ fontSize: 10.5, color: 'rgba(255,255,255,0.9)', lineHeight: 1.4, fontWeight: 600, overflow: 'hidden', display: '-webkit-box', WebkitLineClamp: 2, WebkitBoxOrient: 'vertical' }}>
                  {item.title}
                </div>
                {item.source && (
                  <div style={{ fontSize: 9, color: 'rgba(255,255,255,0.4)', marginTop: 3 }}>{item.source}</div>
                )}
              </div>
            </div>
          )
        })}
      </div>
      {topVids.length > 0 && (
        <div style={{ marginTop: 6, fontSize: 9.5, color: 'rgba(255,255,255,0.3)', textAlign: 'right' }}>
          오른쪽 패널에서 전체 목록 보기 →
        </div>
      )}
    </Card5Wrap>
  )
}

/* ── 메인 렌더러 ─────────────────────────────── */
export function InlineCard5Renderer({ card, accentColor }: { card: InlineCard5Data; accentColor: string }) {
  switch (card.type) {
    case 'web_search':  return <WebSearchCard  data={card} accentColor={accentColor} />
    case 'news_search': return <NewsCard        data={card} accentColor={accentColor} />
    case 'youtube':     return <YoutubeCard     data={card} accentColor={accentColor} />
    default:            return null
  }
}
