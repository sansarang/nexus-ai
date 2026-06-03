# Gmail OAuth 통합 — Google 검증 + Supabase 설정 가이드

> 사장님이 직접 콘솔에서 설정해야 하는 단계 (코드는 이미 적용됨)

## ✅ 코드 적용 현황 (확인)
- ✅ `src/lib/supabase.ts` — signInWithGoogle scope에 gmail.readonly + gmail.send 추가
- ✅ `backend/handlers_gmail.go` — Gmail API v1 호출 (inbox/search/send)
- ✅ `backend/main.go` — /api/gmail/* 엔드포인트 3개 등록

## ⚠️ 사장님 직접 설정 필요 (콘솔 작업)

---

## Step 1. Google Cloud Console — OAuth Consent Screen

### 1-1. 접속
```
https://console.cloud.google.com/apis/credentials/consent
```
- 프로젝트: NEXUS AI (또는 기존 OAuth 프로젝트)

### 1-2. App information
- App name: `NEXUS AI`
- User support email: `사장님 Gmail`
- App logo: `https://nexusai.ai.kr/og-image.png` (또는 정사각 512x512)
- App domain
  - Application home page: `https://nexusai.ai.kr/`
  - Application privacy policy link: `https://nexusai.ai.kr/privacy`
  - Application terms of service link: `https://nexusai.ai.kr/terms`
- Authorized domains: `nexusai.ai.kr`, `supabase.co`
- Developer contact: `사장님 이메일`
- **SAVE AND CONTINUE**

### 1-3. Scopes 추가 (핵심)
- "ADD OR REMOVE SCOPES" 클릭
- 검색창에 `gmail` 입력
- 다음 **2개 체크**:
  ```
  ☑ .../auth/gmail.readonly   — Read all resources and their metadata
  ☑ .../auth/gmail.send        — Send email on your behalf
  ```
- `userinfo.email` `userinfo.profile` `openid` 도 체크 (기본)
- **UPDATE → SAVE AND CONTINUE**

### 1-4. Test users (검증 전 임시)
- 사장님 + 베타 테스터 이메일 입력 (최대 100명)
- 검증 전엔 이 100명만 사용 가능
- **SAVE AND CONTINUE**

### 1-5. Summary → BACK TO DASHBOARD

---

## Step 2. Google Cloud Console — Gmail API 활성화

### 2-1. API Library 접속
```
https://console.cloud.google.com/apis/library/gmail.googleapis.com
```
- **ENABLE** 클릭

### 2-2. 확인
```
https://console.cloud.google.com/apis/dashboard
```
- "Gmail API" 가 활성 목록에 있는지 확인

---

## Step 3. Supabase Console — Google Provider scope 확장

### 3-1. 접속
```
https://supabase.com/dashboard/project/lkkitwetksqkqzfctyne/auth/providers
```

### 3-2. Google provider 클릭
- **Enabled** ON 확인
- **Client ID** / **Client Secret** — Google Cloud Console 의 OAuth 2.0 Client 와 동일해야 함

### 3-3. Authorized scopes (가장 중요)
- Scopes 입력란에 다음 **공백 구분** 추가:
  ```
  https://www.googleapis.com/auth/userinfo.email https://www.googleapis.com/auth/userinfo.profile https://www.googleapis.com/auth/gmail.readonly https://www.googleapis.com/auth/gmail.send openid
  ```
- **Save**

### 3-4. Redirect URLs 확인
- `nexus://auth/callback` 이 Redirect URLs 목록에 있어야 함
- 없으면 추가 → Save

---

## Step 4. Verification 제출 (실 사용자 ↑ 시 필수)

### 4-1. Test users 100명 한도 도달 전 제출
- Gmail scope는 **민감(sensitive) + 제한(restricted)** scope
- 검증 없으면 100명까지만 사용 가능

### 4-2. 제출 절차
```
https://console.cloud.google.com/apis/credentials/consent
→ "PUBLISH APP" 버튼 (Status: Testing → In production)
→ "PREPARE FOR VERIFICATION" 클릭
```

### 4-3. 제출 자료 (Google 요구)
1. **YouTube 데모 영상** (필수)
   - NEXUS AI 가입 → Gmail 권한 동의 → 받은편지함 표시 → 메일 전송 시연
   - 권장 길이: 2~3분
   - 화면 녹화 + 음성 설명
2. **사용 이유 설명**
   - "Why is the requested scope necessary?"
   - 예시 답변:
     ```
     NEXUS AI is a Windows AI assistant that helps users manage their daily email workflow.
     Users can ask "show my inbox" or "send email to John" in natural language.
     gmail.readonly is required to display recent emails as visual cards.
     gmail.send is required to compose and send emails by user command.
     All data is processed locally on the user's machine; no email content is stored on our servers.
     ```
3. **Privacy Policy 명시 조항**
   - Gmail 데이터를 어떻게 처리하는지 명시 필요
   - `landing/privacy.html` 에 다음 섹션 추가 필요:
     ```
     ## Gmail Data Handling
     - Read access: Only fetches metadata (From/Subject/Date) and snippets for display.
     - Send access: Only used when user explicitly commands "send email to ...".
     - No email content is stored on our servers.
     - All processing happens locally on the user's Windows machine.
     - Access tokens are stored in OS-level secure storage (Windows Credential Manager).
     - User can revoke access anytime at https://myaccount.google.com/permissions
     ```

### 4-4. 검증 기간 + 비용
- **소요**: 보통 4~6주 (Google 응답 + 수정 반복)
- **비용**: 무료 (단, 보안 평가 필요 시 $15K~$75K 별도 — 대부분 면제)

---

## Step 5. 검증 완료 후 — Production

### 5-1. Status 확인
```
https://console.cloud.google.com/apis/credentials/consent
→ Publishing status: In production ✅
```

### 5-2. 사용자 한도
- 검증 전: 100명 (test users)
- 검증 후: **무제한**

### 5-3. NEXUS 사용자 흐름
1. Google 로그인 → 동의 화면 (앱 검증됨 표시)
2. "이 앱은 Google에 의해 확인됨" 배지
3. Gmail 권한 자동 부여

---

## ✅ 체크리스트 (사장님 확인)

- [ ] Step 1: OAuth Consent Screen 완성 (App info + Scopes + Test users)
- [ ] Step 2: Gmail API 활성화
- [ ] Step 3: Supabase Google provider scopes 업데이트
- [ ] Step 4-1: 베타 테스터 (사장님 + 친구 5명) 로 작동 확인
- [ ] Step 4-2: 검증 제출 (YouTube 데모 + 답변서)
- [ ] Step 5: 검증 완료 후 Production 전환

---

## 검증 전 임시 사용 (테스트 100명)

지금 당장 (검증 완료 전):
1. 사장님 Gmail 을 Test users 에 추가
2. 사장님 NEXUS 가입 → Gmail 권한 동의
3. "받은 메일 보여줘" 명령으로 작동 확인
4. 베타 테스터 100명까지 동일 사용 가능
5. 검증 완료 후 무제한 전환

---

## 코드 작동 확인 (백엔드 정상)

```bash
# 백엔드 실행 후 (사용자 로그인된 상태)
curl http://127.0.0.1:17891/api/gmail/inbox \
  -H "X-Provider-Token: $GOOGLE_ACCESS_TOKEN"

# 응답 예시
{
  "success": true,
  "messages": [
    { "id":"...", "from":"...", "subject":"...", "snippet":"..." }
  ],
  "count": 10,
  "provider": "gmail"
}
```

---

## 문의 / 트러블슈팅

| 증상 | 원인 | 해결 |
|---|---|---|
| 401 "Google access token required" | provider_token 누락 | Supabase scope 미설정 또는 로그인 재시도 |
| 403 "insufficient permissions" | scope 동의 안 됨 | Google 계정 → 권한 → NEXUS 재인증 |
| 100명 한도 도달 | 검증 미완료 | Step 4 검증 제출 |
| 동의 화면 "확인되지 않음" 경고 | 검증 미완료 | 검증 완료 후 자동 해소 |
