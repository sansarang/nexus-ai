import { motion, AnimatePresence } from 'framer-motion'
import { openCheckout, PADDLE_PRICES } from '../../lib/paddle'
import { useAppStore } from '../../stores/appStore'

const FEATURE_LABELS_KO: Record<string, string> = {
  stock_analysis:    '주식 분석',
  medical_search:    '의료 정보 검색',
  contract_review:   '계약서 검토',
  legal_search:      '법률 검색',
  content_script:    '콘텐츠 스크립트',
  workflow_run:      '워크플로우 실행',
  ai_request:        'AI 요청',
  vision_analyze:    '화면 분석',
  screen_translate:  '화면 번역',
  email_summary:     '이메일 요약',
  content_recommend: '콘텐츠 추천',
  weekly_report:     '주간 리포트',
}

const FEATURE_LABELS_EN: Record<string, string> = {
  stock_analysis:    'Stock Analysis',
  medical_search:    'Medical Search',
  contract_review:   'Contract Review',
  legal_search:      'Legal Search',
  content_script:    'Content Script',
  workflow_run:      'Workflow Run',
  ai_request:        'AI Request',
  vision_analyze:    'Screen Analysis',
  screen_translate:  'Screen Translate',
  email_summary:     'Email Summary',
  content_recommend: 'Content Recommend',
  weekly_report:     'Weekly Report',
}

interface Props {
  feature: string
  used: number
  limit: number
  onClose: () => void
  onUpgrade?: () => void
}

export function PaywallModal({ feature, used, limit, onClose, onUpgrade }: Props) {
  const { userEmail, userLang } = useAppStore()
  const isEn = userLang === 'en'
  const labelMap = isEn ? FEATURE_LABELS_EN : FEATURE_LABELS_KO
  const label = labelMap[feature] ?? feature

  const handlePro = async () => {
    onUpgrade?.()
    await openCheckout(PADDLE_PRICES.pro_monthly, userEmail || undefined)
    onClose()
  }

  const handleTeam = async () => {
    onUpgrade?.()
    await openCheckout(PADDLE_PRICES.team_5, userEmail || undefined)
    onClose()
  }

  return (
    <AnimatePresence>
      {/* Backdrop */}
      <motion.div
        key="paywall-backdrop"
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        exit={{ opacity: 0 }}
        onClick={onClose}
        style={{
          position: 'fixed',
          inset: 0,
          background: 'rgba(0,0,0,0.6)',
          zIndex: 1000,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
        }}
      >
        <motion.div
          key="paywall-modal"
          initial={{ opacity: 0, scale: 0.92, y: 20 }}
          animate={{ opacity: 1, scale: 1, y: 0 }}
          exit={{ opacity: 0, scale: 0.92, y: 20 }}
          transition={{ type: 'spring', stiffness: 300, damping: 24 }}
          onClick={e => e.stopPropagation()}
          style={{
            background: 'var(--bg-surface, #1e1e2e)',
            border: '1px solid var(--border-subtle, #313244)',
            borderRadius: 16,
            padding: '28px 24px 20px',
            width: 320,
            display: 'flex',
            flexDirection: 'column',
            gap: 16,
            boxShadow: '0 24px 64px rgba(0,0,0,0.5)',
          }}
        >
          {/* Header */}
          <div style={{ textAlign: 'center' }}>
            <div style={{ fontSize: 32, marginBottom: 8 }}>🔒</div>
            <h2 style={{ margin: 0, fontSize: 18, fontWeight: 700, color: 'var(--text-primary, #cdd6f4)' }}>
              {isEn ? 'Pro Feature' : 'Pro 기능입니다'}
            </h2>
          </div>

          {/* Usage info */}
          <div
            style={{
              background: 'var(--bg-base, #181825)',
              borderRadius: 10,
              padding: '12px 16px',
              textAlign: 'center',
              color: 'var(--text-secondary, #a6adc8)',
              fontSize: 14,
              lineHeight: 1.6,
            }}
          >
            {limit > 0 ? (
              isEn ? (
                <>
                  You've used{' '}
                  <span style={{ color: '#f38ba8', fontWeight: 700 }}>{used}/{limit}</span>{' '}
                  <strong style={{ color: 'var(--text-primary, #cdd6f4)' }}>{label}</strong> requests today.
                  <br />
                  Upgrade to Pro for <strong>unlimited</strong> access.
                </>
              ) : (
                <>
                  <strong style={{ color: 'var(--text-primary, #cdd6f4)' }}>{label}</strong>은(는){' '}
                  오늘{' '}
                  <span style={{ color: '#f38ba8', fontWeight: 700 }}>
                    {used}/{limit}회
                  </span>{' '}
                  사용했습니다.
                  <br />
                  Pro로 업그레이드하면 <strong>무제한</strong>으로 사용할 수 있어요.
                </>
              )
            ) : (
              isEn ? (
                <>
                  You've used all your <strong style={{ color: 'var(--text-primary, #cdd6f4)' }}>{label}</strong> requests today.
                  <br />
                  Upgrade to Pro for <strong>2,000 requests/day</strong>.
                </>
              ) : (
                <>
                  오늘 <strong style={{ color: 'var(--text-primary, #cdd6f4)' }}>{label}</strong> 한도를 모두 소진했습니다.
                  <br />
                  Pro로 업그레이드하면 하루 <strong>2,000회</strong> 사용할 수 있어요.
                </>
              )
            )}
          </div>

          {/* CTA buttons */}
          <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
            <motion.button
              whileHover={{ scale: 1.02 }}
              whileTap={{ scale: 0.98 }}
              onClick={handlePro}
              style={{
                background: 'linear-gradient(135deg, #cba6f7, #89b4fa)',
                border: 'none',
                borderRadius: 10,
                padding: '11px 0',
                color: '#11111b',
                fontWeight: 700,
                fontSize: 15,
                cursor: 'pointer',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                gap: 6,
              }}
            >
              {isEn
                ? <> ✨ Upgrade to Pro &nbsp;<span style={{ opacity: 0.75, fontWeight: 400 }}>$19/mo</span></>
                : <> ✨ Pro로 업그레이드 &nbsp;<span style={{ opacity: 0.75, fontWeight: 400 }}>₩14,900/월</span></>}
            </motion.button>

            <motion.button
              whileHover={{ scale: 1.02 }}
              whileTap={{ scale: 0.98 }}
              onClick={handleTeam}
              style={{
                background: 'var(--bg-elevated, #313244)',
                border: '1px solid var(--border-subtle, #45475a)',
                borderRadius: 10,
                padding: '11px 0',
                color: 'var(--text-primary, #cdd6f4)',
                fontWeight: 600,
                fontSize: 14,
                cursor: 'pointer',
              }}
            >
              {isEn
                ? <> Team Plan &nbsp;<span style={{ opacity: 0.6, fontWeight: 400 }}>$49/mo</span></>
                : <> Team 플랜 &nbsp;<span style={{ opacity: 0.6, fontWeight: 400 }}>$49/월</span></>}
            </motion.button>

            <button
              onClick={onClose}
              style={{
                background: 'transparent',
                border: 'none',
                color: 'var(--text-muted, #6c7086)',
                fontSize: 13,
                cursor: 'pointer',
                padding: '6px 0',
              }}
            >
              {isEn ? 'Maybe later' : '나중에'}
            </button>
          </div>
        </motion.div>
      </motion.div>
    </AnimatePresence>
  )
}
