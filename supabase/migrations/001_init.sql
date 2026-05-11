-- ═══════════════════════════════════════════════════════════
--  Nexus 구독 DB 스키마
--  Supabase SQL Editor에서 실행하세요
-- ═══════════════════════════════════════════════════════════

-- 구독 테이블
create table if not exists public.subscriptions (
  id                      uuid primary key default gen_random_uuid(),
  user_id                 uuid not null references auth.users(id) on delete cascade,
  paddle_subscription_id  text unique,
  paddle_customer_id      text,
  status                  text not null default 'trial'
                            check (status in ('active', 'trial', 'expired', 'none')),
  trial_ends_at           timestamptz,
  current_period_end      timestamptz,
  created_at              timestamptz not null default now(),
  updated_at              timestamptz not null default now(),
  unique (user_id)
);

-- RLS 활성화
alter table public.subscriptions enable row level security;

-- 본인 구독만 조회 가능
create policy "users_read_own_subscription"
  on public.subscriptions for select
  using (auth.uid() = user_id);

-- Service Role만 INSERT/UPDATE 가능 (백엔드 웹훅 전용)
create policy "service_role_write_subscription"
  on public.subscriptions for all
  using (auth.role() = 'service_role');

-- updated_at 자동 갱신 트리거
create or replace function public.handle_updated_at()
returns trigger language plpgsql as $$
begin
  new.updated_at = now();
  return new;
end;
$$;

create trigger subscriptions_updated_at
  before update on public.subscriptions
  for each row execute function public.handle_updated_at();

-- ─── Google OAuth 설정 안내 (Supabase 대시보드에서 수행) ─────────────────
-- 1. Authentication → Providers → Google 활성화
-- 2. Google Cloud Console에서 OAuth 2.0 클라이언트 ID 생성
-- 3. 승인된 리디렉션 URI: https://YOUR_PROJECT.supabase.co/auth/v1/callback
-- 4. Client ID / Secret을 Supabase Google Provider에 입력
