-- 팀 워크스페이스 스키마

CREATE TABLE IF NOT EXISTS teams (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  name text NOT NULL,
  owner_id uuid NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
  invite_code text UNIQUE NOT NULL,
  created_at timestamptz DEFAULT now()
);

CREATE TABLE IF NOT EXISTS team_members (
  team_id uuid REFERENCES teams(id) ON DELETE CASCADE,
  user_id uuid REFERENCES auth.users(id) ON DELETE CASCADE,
  role text NOT NULL DEFAULT 'member' CHECK (role IN ('admin', 'member')),
  joined_at timestamptz DEFAULT now(),
  PRIMARY KEY (team_id, user_id)
);

CREATE TABLE IF NOT EXISTS team_workflows (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  team_id uuid REFERENCES teams(id) ON DELETE CASCADE,
  shared_by uuid REFERENCES auth.users(id) ON DELETE SET NULL,
  name text NOT NULL,
  description text DEFAULT '',
  workflow_json text NOT NULL,
  created_at timestamptz DEFAULT now()
);

CREATE TABLE IF NOT EXISTS team_settings (
  team_id uuid PRIMARY KEY REFERENCES teams(id) ON DELETE CASCADE,
  persona_id text DEFAULT 'general',
  persona_name text DEFAULT 'Nexus',
  primary_color text DEFAULT '#7c3aed',
  system_prompt text DEFAULT '',
  updated_by uuid REFERENCES auth.users(id) ON DELETE SET NULL,
  updated_at timestamptz DEFAULT now()
);

-- RLS 활성화
ALTER TABLE teams ENABLE ROW LEVEL SECURITY;
ALTER TABLE team_members ENABLE ROW LEVEL SECURITY;
ALTER TABLE team_workflows ENABLE ROW LEVEL SECURITY;
ALTER TABLE team_settings ENABLE ROW LEVEL SECURITY;

-- 팀 조회: 소속 멤버만
CREATE POLICY "teams_select" ON teams FOR SELECT
  USING (auth.uid() IN (SELECT user_id FROM team_members WHERE team_id = id));

-- 팀 생성: owner만
CREATE POLICY "teams_insert" ON teams FOR INSERT
  WITH CHECK (auth.uid() = owner_id);

-- 멤버 조회: 같은 팀 멤버
CREATE POLICY "members_select" ON team_members FOR SELECT
  USING (auth.uid() IN (SELECT user_id FROM team_members WHERE team_id = team_id));

-- 멤버 가입
CREATE POLICY "members_insert" ON team_members FOR INSERT
  WITH CHECK (auth.uid() = user_id);

-- 워크플로우 조회: 팀 멤버
CREATE POLICY "workflows_select" ON team_workflows FOR SELECT
  USING (auth.uid() IN (SELECT user_id FROM team_members WHERE team_id = team_id));

-- 워크플로우 공유: 팀 멤버
CREATE POLICY "workflows_insert" ON team_workflows FOR INSERT
  WITH CHECK (auth.uid() IN (SELECT user_id FROM team_members WHERE team_id = team_id));

-- 팀 설정 조회: 팀 멤버
CREATE POLICY "settings_select" ON team_settings FOR SELECT
  USING (auth.uid() IN (SELECT user_id FROM team_members WHERE team_id = team_id));

-- 팀 설정 수정: admin만
CREATE POLICY "settings_upsert" ON team_settings FOR ALL
  USING (auth.uid() IN (
    SELECT user_id FROM team_members WHERE team_id = team_id AND role = 'admin'
  ));
