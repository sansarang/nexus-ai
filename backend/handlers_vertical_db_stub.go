//go:build !windows

package main

// 비-Windows(개발/Mac) 빌드용 스텁.
// 실제 구현은 handlers_vertical_db.go(//go:build windows)에 있으나,
// cross-platform 파일(phase_b_domain_apis.go)이 loadVerticalAPIKeys()를 참조하므로
// darwin/linux 빌드가 깨지지 않도록 동일한 타입·로더를 제공한다.
// 동작은 Windows와 동일: ~/.nexus/vertical_apis.json 에서 키를 읽는다.

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type VerticalAPIKeys struct {
	LawGOKR     string `json:"law_go_kr"`
	DataGOKR    string `json:"data_go_kr"`
	YouTubeV3   string `json:"youtube_v3"`
	GitHubToken string `json:"github_token"`
}

func loadVerticalAPIKeys() VerticalAPIKeys {
	home, _ := os.UserHomeDir()
	data, err := os.ReadFile(filepath.Join(home, ".nexus", "vertical_apis.json"))
	if err != nil {
		return VerticalAPIKeys{}
	}
	var keys VerticalAPIKeys
	json.Unmarshal(data, &keys)
	return keys
}
