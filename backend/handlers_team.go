// handlers_team.go — Team 워크스페이스 + 멤버 초대 (cross-platform)
// 사장님 요구: Team ₩79,000/월 — 5인+ 워크스페이스 실 구현
//
// 흐름:
//   1. Team 플랜 구독한 사용자 = workspace owner
//   2. owner 가 멤버 이메일 초대 → invite token 생성 → 이메일 발송
//   3. 멤버가 invite link 클릭 → workspace 가입
//   4. 모든 멤버는 workspace 한도 공유 (3000회/일 공동)
//   5. owner 만 결제·멤버 관리

package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Workspace — 팀 단위
type Workspace struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	OwnerEmail string    `json:"owner_email"`
	Plan       string    `json:"plan"`         // "team" / "enterprise"
	SeatLimit  int       `json:"seat_limit"`   // Team = 5
	CreatedAt  time.Time `json:"created_at"`
}

// WorkspaceMember
type WorkspaceMember struct {
	WorkspaceID string    `json:"workspace_id"`
	Email       string    `json:"email"`
	Role        string    `json:"role"`        // owner / admin / member
	JoinedAt    time.Time `json:"joined_at"`
	InvitedBy   string    `json:"invited_by"`
}

// WorkspaceInvite — 초대 토큰
type WorkspaceInvite struct {
	Token       string    `json:"token"`
	WorkspaceID string    `json:"workspace_id"`
	Email       string    `json:"email"`
	InvitedBy   string    `json:"invited_by"`
	ExpiresAt   time.Time `json:"expires_at"`
	Used        bool      `json:"used"`
}

var (
	wsStoreMu       sync.RWMutex
	workspaces      = make(map[string]*Workspace)         // id → Workspace
	wsMembers       = make(map[string][]WorkspaceMember)  // workspace_id → members
	wsInvites       = make(map[string]*WorkspaceInvite)   // token → invite
	userToWorkspace = make(map[string]string)             // email → workspace_id
)

func wsStorePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".nexus", "team_workspaces.json")
}

type wsStoreFile struct {
	Workspaces      map[string]*Workspace         `json:"workspaces"`
	Members         map[string][]WorkspaceMember  `json:"members"`
	Invites         map[string]*WorkspaceInvite   `json:"invites"`
	UserToWorkspace map[string]string             `json:"user_to_workspace"`
}

func loadWorkspaceStore() {
	data, err := os.ReadFile(wsStorePath())
	if err != nil {
		return
	}
	var f wsStoreFile
	if json.Unmarshal(data, &f) != nil {
		return
	}
	wsStoreMu.Lock()
	defer wsStoreMu.Unlock()
	if f.Workspaces != nil {
		workspaces = f.Workspaces
	}
	if f.Members != nil {
		wsMembers = f.Members
	}
	if f.Invites != nil {
		wsInvites = f.Invites
	}
	if f.UserToWorkspace != nil {
		userToWorkspace = f.UserToWorkspace
	}
}

func saveWorkspaceStore() {
	wsStoreMu.RLock()
	f := wsStoreFile{
		Workspaces:      workspaces,
		Members:         wsMembers,
		Invites:         wsInvites,
		UserToWorkspace: userToWorkspace,
	}
	wsStoreMu.RUnlock()
	data, _ := json.Marshal(f)
	p := wsStorePath()
	os.MkdirAll(filepath.Dir(p), 0755)
	os.WriteFile(p, data, 0600)
}

func newWorkspaceID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return "ws-" + hex.EncodeToString(b)
}

func newInviteToken() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// ── 핵심 API ─────────────────────────────────────────────

// createTeamWorkspace — Paddle 결제 완료 시 호출 (handlers_paddle 또는 manual)
// 또는 사용자가 처음 Team 플랜 구독 시 자동 생성
func createTeamWorkspace(ownerEmail, name string, seatLimit int) *Workspace {
	if seatLimit <= 0 {
		seatLimit = 5 // Team 기본
	}
	ws := &Workspace{
		ID:         newWorkspaceID(),
		Name:       name,
		OwnerEmail: ownerEmail,
		Plan:       "team",
		SeatLimit:  seatLimit,
		CreatedAt:  time.Now(),
	}
	wsStoreMu.Lock()
	workspaces[ws.ID] = ws
	wsMembers[ws.ID] = []WorkspaceMember{
		{WorkspaceID: ws.ID, Email: ownerEmail, Role: "owner", JoinedAt: time.Now()},
	}
	userToWorkspace[ownerEmail] = ws.ID
	wsStoreMu.Unlock()
	go saveWorkspaceStore()
	return ws
}

// getWorkspaceForUser — 사용자가 속한 workspace 반환
func getWorkspaceForUser(email string) *Workspace {
	wsStoreMu.RLock()
	defer wsStoreMu.RUnlock()
	wsID, ok := userToWorkspace[email]
	if !ok {
		return nil
	}
	return workspaces[wsID]
}

// inviteMember — owner 가 새 멤버 초대
func inviteMember(workspaceID, ownerEmail, targetEmail string) (*WorkspaceInvite, error) {
	wsStoreMu.RLock()
	ws := workspaces[workspaceID]
	memberCount := len(wsMembers[workspaceID])
	wsStoreMu.RUnlock()
	if ws == nil {
		return nil, errResult("workspace not found")
	}
	if ws.OwnerEmail != ownerEmail {
		return nil, errResult("only owner can invite")
	}
	if memberCount >= ws.SeatLimit {
		return nil, errResult("seat limit reached")
	}
	invite := &WorkspaceInvite{
		Token:       newInviteToken(),
		WorkspaceID: workspaceID,
		Email:       strings.ToLower(strings.TrimSpace(targetEmail)),
		InvitedBy:   ownerEmail,
		ExpiresAt:   time.Now().Add(7 * 24 * time.Hour),
		Used:        false,
	}
	wsStoreMu.Lock()
	wsInvites[invite.Token] = invite
	wsStoreMu.Unlock()
	go saveWorkspaceStore()
	return invite, nil
}

// acceptInvite — 멤버가 초대 link 클릭 시
func acceptInvite(token, memberEmail string) (*Workspace, error) {
	wsStoreMu.Lock()
	defer wsStoreMu.Unlock()
	inv := wsInvites[token]
	if inv == nil {
		return nil, errResult("invite not found")
	}
	if inv.Used {
		return nil, errResult("invite already used")
	}
	if time.Now().After(inv.ExpiresAt) {
		return nil, errResult("invite expired")
	}
	if strings.ToLower(strings.TrimSpace(memberEmail)) != inv.Email {
		return nil, errResult("invite email mismatch")
	}
	ws := workspaces[inv.WorkspaceID]
	if ws == nil {
		return nil, errResult("workspace not found")
	}
	// 시트 한도 재확인
	if len(wsMembers[ws.ID]) >= ws.SeatLimit {
		return nil, errResult("seat limit reached")
	}
	wsMembers[ws.ID] = append(wsMembers[ws.ID], WorkspaceMember{
		WorkspaceID: ws.ID,
		Email:       memberEmail,
		Role:        "member",
		JoinedAt:    time.Now(),
		InvitedBy:   inv.InvitedBy,
	})
	userToWorkspace[memberEmail] = ws.ID
	inv.Used = true
	go saveWorkspaceStore()
	return ws, nil
}

// removeMember — owner 가 멤버 제거
func removeMember(workspaceID, ownerEmail, targetEmail string) error {
	wsStoreMu.Lock()
	defer wsStoreMu.Unlock()
	ws := workspaces[workspaceID]
	if ws == nil || ws.OwnerEmail != ownerEmail {
		return errResult("forbidden")
	}
	if targetEmail == ownerEmail {
		return errResult("cannot remove owner")
	}
	out := make([]WorkspaceMember, 0, len(wsMembers[workspaceID]))
	for _, m := range wsMembers[workspaceID] {
		if m.Email != targetEmail {
			out = append(out, m)
		}
	}
	wsMembers[workspaceID] = out
	delete(userToWorkspace, targetEmail)
	go saveWorkspaceStore()
	return nil
}

func errResult(msg string) error {
	return &teamError{msg: msg}
}

type teamError struct{ msg string }

func (e *teamError) Error() string { return e.msg }

// ── HTTP 엔드포인트 ────────────────────────────────────────

// handleTeamCreate — POST /api/team/create (Paddle 웹훅 또는 수동)
func handleTeamCreate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		OwnerEmail string `json:"owner_email"`
		Name       string `json:"name"`
		SeatLimit  int    `json:"seat_limit"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.OwnerEmail == "" {
		writeJSON(w, 400, map[string]any{"success": false, "message": "owner_email required"})
		return
	}
	if req.Name == "" {
		req.Name = "Team"
	}
	ws := createTeamWorkspace(req.OwnerEmail, req.Name, req.SeatLimit)
	json200(w, map[string]any{"success": true, "workspace": ws})
}

// handleTeamMembers — GET /api/team/members?workspace_id=xxx
func handleTeamMembers(w http.ResponseWriter, r *http.Request) {
	wsID := r.URL.Query().Get("workspace_id")
	if wsID == "" {
		// 현재 사용자 workspace 자동 조회
		email := r.URL.Query().Get("email")
		ws := getWorkspaceForUser(email)
		if ws == nil {
			json200(w, map[string]any{"success": false, "message": "not in any workspace"})
			return
		}
		wsID = ws.ID
	}
	wsStoreMu.RLock()
	members := wsMembers[wsID]
	ws := workspaces[wsID]
	wsStoreMu.RUnlock()
	json200(w, map[string]any{
		"success":    true,
		"workspace":  ws,
		"members":    members,
		"used_seats": len(members),
	})
}

// handleTeamInvite — POST /api/team/invite
func handleTeamInvite(w http.ResponseWriter, r *http.Request) {
	var req struct {
		WorkspaceID string `json:"workspace_id"`
		OwnerEmail  string `json:"owner_email"`
		TargetEmail string `json:"target_email"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	invite, err := inviteMember(req.WorkspaceID, req.OwnerEmail, req.TargetEmail)
	if err != nil {
		writeJSON(w, 400, map[string]any{"success": false, "message": err.Error()})
		return
	}
	// 초대 URL — nexusai.ai.kr/team/accept?token=...
	inviteURL := "https://nexusai.ai.kr/team/accept?token=" + invite.Token
	json200(w, map[string]any{
		"success":     true,
		"invite":      invite,
		"invite_url":  inviteURL,
		"expires_in":  "7 days",
	})
}

// handleTeamAccept — POST /api/team/accept
func handleTeamAccept(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token       string `json:"token"`
		MemberEmail string `json:"member_email"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	ws, err := acceptInvite(req.Token, req.MemberEmail)
	if err != nil {
		writeJSON(w, 400, map[string]any{"success": false, "message": err.Error()})
		return
	}
	json200(w, map[string]any{"success": true, "workspace": ws})
}

// handleTeamRemove — POST /api/team/remove
func handleTeamRemove(w http.ResponseWriter, r *http.Request) {
	var req struct {
		WorkspaceID string `json:"workspace_id"`
		OwnerEmail  string `json:"owner_email"`
		TargetEmail string `json:"target_email"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if err := removeMember(req.WorkspaceID, req.OwnerEmail, req.TargetEmail); err != nil {
		writeJSON(w, 400, map[string]any{"success": false, "message": err.Error()})
		return
	}
	json200(w, map[string]any{"success": true})
}
