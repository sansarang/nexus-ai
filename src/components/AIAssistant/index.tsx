import { motion } from 'framer-motion'
import { useAppStore, type ViewId } from '../../stores/appStore'

export interface NLResult {
  action: string
  response: string
  viewTarget?: ViewId
  autoRun: boolean
}

interface NLRule {
  pattern: RegExp
  action: string
  response: string
  viewTarget?: ViewId
  autoRun: boolean
}

const NL_RULES: NLRule[] = [
  {
    pattern: /느려|느림|버벅|렉|답답/,
    action: 'navigate_home',
    response: 'PC가 느린 증상을 감지했어요. 전체 진단을 시작해드릴게요.',
    viewTarget: 'home',
    autoRun: true,
  },
  {
    pattern: /정리|청소|쓸모없|파일/,
    action: 'navigate_autoclean',
    response: 'PC 정리 화면으로 이동할게요. 임시파일, 캐시, 휴지통을 정리할 수 있어요.',
    viewTarget: 'autoclean',
    autoRun: false,
  },
  {
    pattern: /cpu|씨피유|온도|뜨거|발열/,
    action: 'navigate_monitor',
    response: 'CPU 온도와 실시간 사용률을 확인해드릴게요.',
    viewTarget: 'monitor',
    autoRun: false,
  },
  {
    pattern: /바이러스|악성|해킹|이상/,
    action: 'navigate_security',
    response: '보안 위협을 확인해드릴게요. 해킹 탐지 화면으로 이동합니다.',
    viewTarget: 'security',
    autoRun: false,
  },
  {
    pattern: /업데이트|오류|에러|고장|윈도우/,
    action: 'navigate_repair',
    response: 'Windows 문제를 수리해드릴게요. 수리 화면으로 이동합니다.',
    viewTarget: 'repair',
    autoRun: false,
  },
  {
    pattern: /번역|영어|일본어|중국어|한국어/,
    action: 'navigate_translate',
    response: '번역 화면으로 이동할게요. 다양한 언어로 번역할 수 있어요.',
    viewTarget: 'translate',
    autoRun: false,
  },
  {
    pattern: /메모리|램|ram/i,
    action: 'navigate_monitor',
    response: '메모리 사용량을 실시간으로 확인해드릴게요.',
    viewTarget: 'monitor',
    autoRun: false,
  },
  {
    pattern: /집중|포모도로|방해금지/,
    action: 'navigate_focus',
    response: '집중 모드(포모도로)를 시작해드릴게요.',
    viewTarget: 'focus',
    autoRun: false,
  },
  {
    pattern: /클립보드|복사/,
    action: 'navigate_clipboard',
    response: '클립보드 히스토리를 확인해드릴게요.',
    viewTarget: 'clipboard',
    autoRun: false,
  },
  {
    pattern: /메모|노트|할일|todo/i,
    action: 'navigate_memo',
    response: '메모 화면으로 이동할게요.',
    viewTarget: 'memo',
    autoRun: false,
  },
  {
    pattern: /프라이버시|코파일럿|원드라이브|텔레메트리/,
    action: 'navigate_privacy',
    response: '프라이버시 설정 화면으로 이동할게요. MS 기능을 제어할 수 있어요.',
    viewTarget: 'privacy',
    autoRun: false,
  },
  {
    pattern: /데일리|아침|리포트|오늘/,
    action: 'navigate_daily',
    response: '오늘의 PC 상태 리포트를 확인해드릴게요.',
    viewTarget: 'daily',
    autoRun: false,
  },
  {
    pattern: /음성|말하기|녹음/,
    action: 'navigate_voicememo',
    response: '음성 메모 기능으로 이동할게요.',
    viewTarget: 'voicememo',
    autoRun: false,
  },
  {
    pattern: /예측|미래|예방/,
    action: 'navigate_predictive',
    response: 'AI 예측 관리 화면으로 이동할게요. PC의 미래 상태를 미리 확인해요.',
    viewTarget: 'predictive',
    autoRun: false,
  },
]

export function processNaturalLanguage(input: string): NLResult | null {
  const lower = input.toLowerCase()
  for (const rule of NL_RULES) {
    if (rule.pattern.test(lower)) {
      return {
        action: rule.action,
        response: rule.response,
        viewTarget: rule.viewTarget,
        autoRun: rule.autoRun,
      }
    }
  }
  return null
}

export function AIResponseCard({
  result,
  onExecute,
}: {
  result: NLResult
  onExecute: () => void
}): JSX.Element {
  const { setView, startScan } = useAppStore()

  const handleExecute = () => {
    if (result.viewTarget) {
      setView(result.viewTarget)
    }
    if (result.action === 'navigate_home' && result.autoRun) {
      startScan()
    }
    onExecute()
  }

  return (
    <motion.div
      initial={{ opacity: 0, y: 4 }}
      animate={{ opacity: 1, y: 0 }}
      style={{
        margin: '8px 8px 0',
        padding: '12px 16px',
        borderRadius: 'var(--radius-md)',
        background: 'rgba(79,126,247,0.08)',
        border: '1px solid rgba(79,126,247,0.25)',
        display: 'flex',
        alignItems: 'center',
        gap: 12,
      }}
    >
      <span style={{ fontSize: 18, flexShrink: 0 }}>🤖</span>
      <div style={{ flex: 1 }}>
        <div style={{ fontSize: 12, color: 'var(--accent-primary)', fontWeight: 600, marginBottom: 2 }}>
          AI 추천
        </div>
        <div style={{ fontSize: 13, color: 'var(--text-primary)' }}>{result.response}</div>
      </div>
      <motion.button
        onClick={handleExecute}
        whileHover={{ scale: 1.04 }}
        whileTap={{ scale: 0.96 }}
        style={{
          padding: '6px 14px',
          borderRadius: 'var(--radius-sm)',
          border: 'none',
          background: 'var(--accent-primary)',
          color: '#fff',
          fontSize: 12,
          fontWeight: 600,
          cursor: 'pointer',
          flexShrink: 0,
        }}
      >
        {result.autoRun ? '바로 실행' : '이동'}
      </motion.button>
    </motion.div>
  )
}
