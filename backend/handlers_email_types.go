package main

// EmailItem — 플랫폼 공통 이메일 구조체
type EmailItem struct {
	Subject    string `json:"subject"`
	Sender     string `json:"sender"`
	ReceivedAt string `json:"received_at"`
	Body       string `json:"body"`
	IsRead     bool   `json:"is_read"`
	HasAttach  bool   `json:"has_attachments"`
}
