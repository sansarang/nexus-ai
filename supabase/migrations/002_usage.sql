-- ═══════════════════════════════════════════════════════
--  사용량 추적 테이블
--  Supabase SQL Editor에서 실행하세요
-- ═══════════════════════════════════════════════════════

create table if not exists public.usage_logs (
  id            uuid primary key default gen_random_uuid(),
  user_id       uuid not null references auth.users(id) on delete cascade,
  date          date not null default current_date,
  free_count    int  not null default 0,   -- Groq 무료 호출
  premium_count int  not null default 0,   -- Claude/Perplexity/Tavily
  updated_at    timestamptz not null default now(),
  unique (user_id, date)
);

alter table public.usage_logs enable row level security;

-- 본인 사용량만 조회 가능
create policy "users_read_own_usage"
  on public.usage_logs for select
  using (auth.uid() = user_id);

-- Edge Function(service role)만 쓰기 가능
create policy "service_role_write_usage"
  on public.usage_logs for all
  using (auth.role() = 'service_role');

-- ── 사용량 현황 뷰 (대시보드용) ────────────────────────────
create or replace view public.usage_summary as
select
  u.user_id,
  u.date,
  u.free_count,
  u.premium_count,
  (u.free_count + u.premium_count) as total_count,
  500 - u.free_count    as free_left,
  50  - u.premium_count as premium_left
from public.usage_logs u;

-- ── 일별 사용량 초과 방지 함수 ──────────────────────────────
create or replace function public.check_usage_limit(
  p_user_id uuid,
  p_tier text  -- 'free' or 'premium'
) returns boolean language plpgsql security definer as $$
declare
  v_count int;
  v_limit int;
begin
  v_limit := case p_tier when 'premium' then 50 else 500 end;

  select coalesce(
    case p_tier
      when 'premium' then premium_count
      else free_count
    end, 0
  ) into v_count
  from public.usage_logs
  where user_id = p_user_id and date = current_date;

  return coalesce(v_count, 0) < v_limit;
end;
$$;
