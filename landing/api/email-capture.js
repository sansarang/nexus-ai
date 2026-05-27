// Vercel Serverless Function — 이메일 수집 → Supabase email_waitlist 테이블 저장
// 환경변수: VITE_SUPABASE_URL, VITE_SUPABASE_ANON_KEY (Vercel 대시보드에서 설정)

module.exports = async function handler(req, res) {
  // CORS
  res.setHeader('Access-Control-Allow-Origin', '*')
  res.setHeader('Access-Control-Allow-Methods', 'POST, OPTIONS')
  res.setHeader('Access-Control-Allow-Headers', 'Content-Type')

  if (req.method === 'OPTIONS') return res.status(204).end()
  if (req.method !== 'POST') return res.status(405).json({ error: 'Method not allowed' })

  const { email, source, lang } = req.body || {}

  if (!email || typeof email !== 'string' || !email.includes('@') || !email.includes('.')) {
    return res.status(400).json({ error: 'Invalid email' })
  }

  const supabaseUrl = process.env.VITE_SUPABASE_URL
  const supabaseKey = process.env.VITE_SUPABASE_ANON_KEY

  if (!supabaseUrl || !supabaseKey) {
    // 환경변수 미설정 시 조용히 성공 반환 (사용자 UX 깨지지 않도록)
    console.error('[email-capture] Supabase env vars not set')
    return res.status(200).json({ success: true, warn: 'env not configured' })
  }

  try {
    const response = await fetch(`${supabaseUrl}/rest/v1/email_waitlist`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'apikey': supabaseKey,
        'Authorization': `Bearer ${supabaseKey}`,
        'Prefer': 'return=minimal',
      },
      body: JSON.stringify({
        email: email.toLowerCase().trim(),
        source: source || 'landing',
        lang: lang || 'ko',
      }),
    })

    // 409 = 이미 등록된 이메일 (UNIQUE 제약) → 성공으로 처리
    if (response.ok || response.status === 409) {
      return res.status(200).json({ success: true })
    }

    const err = await response.text()
    console.error('[email-capture] Supabase error:', response.status, err)
    return res.status(500).json({ error: 'Failed to save email' })

  } catch (e) {
    console.error('[email-capture] Network error:', e.message)
    return res.status(500).json({ error: 'Network error' })
  }
}
