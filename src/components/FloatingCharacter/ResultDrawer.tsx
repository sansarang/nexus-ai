/**
 * ResultDrawer — 검색 결과 전체보기 드로어
 * FloatingPreview "전체보기" 버튼 클릭 시 오른쪽에서 슬라이드 인
 */
import React, { useState, useMemo } from 'react'
import { motion, AnimatePresence } from 'framer-motion'

export interface DrawerItem {
  title: string
  url: string
  isVideo?: boolean
  isSocial?: boolean
  isMap?: boolean
  isImage?: boolean
  source?: string
}

interface ResultDrawerProps {
  open: boolean
  onClose: () => void
  items: DrawerItem[]
  primaryColor: string
  userLang?: 'ko' | 'en'
  title?: string
}

/* ── 유틸 ── */
function domain(url: string) {
  try { return new URL(url).hostname.replace('www.', '') } catch { return '' }
}
function faviconUrl(url: string) {
  try { return `https://www.google.com/s2/favicons?domain=${new URL(url).hostname}&sz=16` } catch { return '' }
}

type FilterKey = 'all' | 'video' | 'news' | 'shopping' | 'blog' | 'social'
type SortKey = 'default' | 'domain'

export function ResultDrawer({ open, onClose, items, primaryColor, userLang = 'ko', title }: ResultDrawerProps) {
  const [filter, setFilter] = useState<FilterKey>('all')
  const [sort, setSort] = useState<SortKey>('default')
  const [copied, setCopied] = useState<string | null>(null)

  const ko = userLang === 'ko'

  /* ── 필터 탭 계산 ── */
  const hasVideo    = items.some(x => x.isVideo || /youtube\.|youtu\.be|tiktok\./i.test(x.url))
  const hasNews     = items.some(x => /news\.|chosun\.|yna\.|yonhap\.|hankyung\.|khan\.|donga\.|joongang\.|kbs\.|mbc\.|sbs\./i.test(x.url))
  const hasShopping = items.some(x => /coupang\.|shopping\.naver\.|11st\.|gmarket\.|temu\.|aliexpress\.|amazon\./i.test(x.url))
  const hasBlog     = items.some(x => /blog\.naver\.|tistory\.|velog\.|brunch\.|medium\./i.test(x.url))
  const hasSocial   = items.some(x => x.isSocial || /instagram\.|x\.com|twitter\./i.test(x.url))

  const tabs: { key: FilterKey; label: string }[] = [
    { key: 'all',      label: ko ? `전체 (${items.length})` : `All (${items.length})` },
    ...(hasVideo    ? [{ key: 'video'    as FilterKey, label: ko ? '🎬 영상'  : '🎬 Video'   }] : []),
    ...(hasNews     ? [{ key: 'news'     as FilterKey, label: ko ? '📰 뉴스'  : '📰 News'    }] : []),
    ...(hasShopping ? [{ key: 'shopping' as FilterKey, label: ko ? '🛒 쇼핑'  : '🛒 Shopping'}] : []),
    ...(hasBlog     ? [{ key: 'blog'     as FilterKey, label: ko ? '📝 블로그': '📝 Blog'    }] : []),
    ...(hasSocial   ? [{ key: 'social'   as FilterKey, label: ko ? '💬 SNS'   : '💬 Social'  }] : []),
  ]

  /* ── 필터 + 정렬 적용 ── */
  const filtered = useMemo(() => {
    let list = items
    if (filter === 'video')    list = list.filter(x => x.isVideo || /youtube\.|youtu\.be|tiktok\./i.test(x.url))
    if (filter === 'news')     list = list.filter(x => /news\.|chosun\.|yna\.|yonhap\.|hankyung\.|khan\.|donga\.|joongang\.|kbs\.|mbc\.|sbs\./i.test(x.url))
    if (filter === 'shopping') list = list.filter(x => /coupang\.|shopping\.naver\.|11st\.|gmarket\.|temu\.|aliexpress\.|amazon\./i.test(x.url))
    if (filter === 'blog')     list = list.filter(x => /blog\.naver\.|tistory\.|velog\.|brunch\.|medium\./i.test(x.url))
    if (filter === 'social')   list = list.filter(x => x.isSocial || /instagram\.|x\.com|twitter\./i.test(x.url))
    if (sort === 'domain')     list = [...list].sort((a, b) => domain(a.url).localeCompare(domain(b.url)))
    return list
  }, [items, filter, sort])

  async function copyUrl(url: string) {
    try {
      await navigator.clipboard.writeText(url)
      setCopied(url)
      setTimeout(() => setCopied(null), 1500)
    } catch { /* 복사 실패 */ }
  }

  return (
    <AnimatePresence>
      {open && (
        <>
          {/* 백드롭 */}
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            onClick={onClose}
            style={{
              position: 'fixed', inset: 0,
              background: 'rgba(0,0,0,0.45)',
              zIndex: 10100,
            }}
          />
          {/* 드로어 패널 */}
          <motion.div
            initial={{ x: 420, opacity: 0 }}
            animate={{ x: 0, opacity: 1 }}
            exit={{ x: 420, opacity: 0 }}
            transition={{ type: 'spring', stiffness: 340, damping: 32 }}
            style={{
              position: 'fixed',
              top: 0, right: 0, bottom: 0,
              width: 400,
              background: 'rgba(6,6,18,0.99)',
              borderLeft: `2px solid ${primaryColor}55`,
              boxShadow: `-8px 0 48px rgba(0,0,0,0.8)`,
              zIndex: 10101,
              display: 'flex',
              flexDirection: 'column',
              backdropFilter: 'blur(24px)',
            }}
          >
            {/* 헤더 */}
            <div style={{
              display: 'flex', alignItems: 'center', justifyContent: 'space-between',
              padding: '16px 18px 12px',
              borderBottom: `1px solid ${primaryColor}33`,
              flexShrink: 0,
            }}>
              <div>
                <div style={{ fontSize: 13, color: primaryColor, fontWeight: 800, letterSpacing: '0.04em' }}>
                  {title ?? (ko ? '🔍 전체 검색 결과' : '🔍 All Results')}
                </div>
                <div style={{ fontSize: 10, color: 'rgba(255,255,255,0.38)', marginTop: 2 }}>
                  {ko ? `${filtered.length}개 결과` : `${filtered.length} results`}
                </div>
              </div>
              <button
                onClick={onClose}
                style={{
                  background: 'rgba(255,255,255,0.07)', border: '1px solid rgba(255,255,255,0.12)',
                  borderRadius: 8, color: 'rgba(255,255,255,0.7)', fontSize: 14,
                  width: 30, height: 30, cursor: 'pointer', display: 'flex', alignItems: 'center', justifyContent: 'center',
                }}
              >✕</button>
            </div>

            {/* 필터 탭 */}
            <div style={{
              display: 'flex', gap: 5, padding: '10px 18px 8px',
              overflowX: 'auto', flexShrink: 0,
              borderBottom: `1px solid rgba(255,255,255,0.06)`,
            }}>
              {tabs.map(tab => (
                <button
                  key={tab.key}
                  onClick={() => setFilter(tab.key)}
                  style={{
                    padding: '4px 11px', borderRadius: 8, fontSize: 10, fontWeight: 700,
                    whiteSpace: 'nowrap', cursor: 'pointer', transition: 'all 0.15s',
                    background: filter === tab.key ? primaryColor : 'rgba(255,255,255,0.07)',
                    color: filter === tab.key ? '#fff' : 'rgba(255,255,255,0.5)',
                    border: filter === tab.key ? 'none' : '1px solid rgba(255,255,255,0.11)',
                    flexShrink: 0,
                  }}
                >{tab.label}</button>
              ))}
              {/* 정렬 */}
              <button
                onClick={() => setSort(s => s === 'default' ? 'domain' : 'default')}
                style={{
                  marginLeft: 'auto', padding: '4px 10px', borderRadius: 8, fontSize: 10,
                  fontWeight: 700, cursor: 'pointer', flexShrink: 0,
                  background: sort === 'domain' ? `${primaryColor}44` : 'rgba(255,255,255,0.05)',
                  color: 'rgba(255,255,255,0.55)',
                  border: '1px solid rgba(255,255,255,0.1)',
                }}
              >{ko ? (sort === 'domain' ? '🔤 도메인순' : '📋 기본순') : (sort === 'domain' ? '🔤 A-Z' : '📋 Default')}</button>
            </div>

            {/* 결과 목록 */}
            <div style={{ flex: 1, overflowY: 'auto', padding: '8px 14px 16px' }}>
              {filtered.length === 0 ? (
                <div style={{ textAlign: 'center', color: 'rgba(255,255,255,0.3)', fontSize: 12, marginTop: 40 }}>
                  {ko ? '해당 카테고리 결과 없음' : 'No results in this category'}
                </div>
              ) : (
                filtered.map((item, i) => {
                  const isYt = /youtube\.|youtu\.be/.test(item.url)
                  const isTikTok = item.url.includes('tiktok.com')
                  const typeBadge = isYt ? { label: 'YT', color: '#e53e3e' }
                    : isTikTok ? { label: 'TikTok', color: '#010101' }
                    : item.isVideo ? { label: ko ? '영상' : 'Video', color: '#e53e3e' }
                    : item.isSocial ? { label: 'SNS', color: '#7c3aed' }
                    : null
                  return (
                    <div
                      key={i}
                      style={{
                        display: 'flex', alignItems: 'flex-start', gap: 10,
                        padding: '9px 8px', borderRadius: 10,
                        borderBottom: i < filtered.length - 1 ? '1px solid rgba(255,255,255,0.05)' : 'none',
                        cursor: 'pointer', transition: 'background 0.15s',
                      }}
                      onMouseEnter={e => (e.currentTarget.style.background = 'rgba(255,255,255,0.04)')}
                      onMouseLeave={e => (e.currentTarget.style.background = 'transparent')}
                      onClick={() => window.open(item.url, '_blank')}
                    >
                      {/* favicon */}
                      <div style={{ width: 22, height: 22, borderRadius: 5, background: `${primaryColor}18`, display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0, overflow: 'hidden', marginTop: 1 }}>
                        <img
                          src={faviconUrl(item.url)} width={15} height={15} style={{ borderRadius: 2 }}
                          onError={e => { (e.target as HTMLImageElement).style.display = 'none' }}
                        />
                      </div>
                      {/* 내용 */}
                      <div style={{ flex: 1, minWidth: 0 }}>
                        <div style={{ display: 'flex', alignItems: 'center', gap: 5, marginBottom: 3 }}>
                          {typeBadge && (
                            <span style={{ fontSize: 8.5, fontWeight: 700, color: '#fff', background: typeBadge.color, borderRadius: 3, padding: '1px 5px', flexShrink: 0 }}>
                              {typeBadge.label}
                            </span>
                          )}
                          <div style={{ fontSize: 12, color: 'rgba(255,255,255,0.92)', fontWeight: 600, lineHeight: 1.4, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                            {item.title}
                          </div>
                        </div>
                        <div style={{ fontSize: 10, color: 'rgba(255,255,255,0.32)', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                          {item.url.replace(/^https?:\/\//, '').slice(0, 52)}
                        </div>
                      </div>
                      {/* 복사 버튼 */}
                      <button
                        title={ko ? 'URL 복사' : 'Copy URL'}
                        onClick={async (e) => {
                          e.stopPropagation()
                          copyUrl(item.url)
                        }}
                        style={{
                          background: copied === item.url ? `${primaryColor}33` : 'rgba(255,255,255,0.06)',
                          border: '1px solid rgba(255,255,255,0.1)',
                          borderRadius: 6, color: copied === item.url ? primaryColor : 'rgba(255,255,255,0.4)',
                          fontSize: 11, width: 26, height: 26, cursor: 'pointer',
                          display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0,
                          transition: 'all 0.15s',
                        }}
                      >{copied === item.url ? '✓' : '⎘'}</button>
                    </div>
                  )
                })
              )}
            </div>

            {/* 하단: 전체 복사 */}
            <div style={{
              padding: '10px 18px 14px',
              borderTop: `1px solid ${primaryColor}22`,
              flexShrink: 0,
            }}>
              <button
                onClick={() => {
                  const text = filtered.map((x, i) => `${i + 1}. ${x.title}\n   ${x.url}`).join('\n\n')
                  navigator.clipboard.writeText(text).catch(() => {})
                }}
                style={{
                  width: '100%', padding: '8px 0',
                  background: `${primaryColor}18`, border: `1px solid ${primaryColor}44`,
                  borderRadius: 10, color: primaryColor, fontSize: 11, fontWeight: 700,
                  cursor: 'pointer', transition: 'background 0.15s',
                }}
                onMouseEnter={e => { (e.currentTarget as HTMLButtonElement).style.background = `${primaryColor}30` }}
                onMouseLeave={e => { (e.currentTarget as HTMLButtonElement).style.background = `${primaryColor}18` }}
              >
                {ko ? '📋 전체 URL 복사' : '📋 Copy all URLs'}
              </button>
            </div>
          </motion.div>
        </>
      )}
    </AnimatePresence>
  )
}
