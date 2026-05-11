//go:build windows

package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// ─── Paddle Webhook 이벤트 구조체 ───────────────────────────────────────────

type paddleWebhookEvent struct {
	EventType string          `json:"event_type"`
	Data      json.RawMessage `json:"data"`
}

type paddleSubscriptionData struct {
	ID         string `json:"id"`
	CustomerID string `json:"customer_id"`
	Status     string `json:"status"` // active | canceled | past_due | trialing
	Items      []struct {
		Price struct {
			ID string `json:"id"`
		} `json:"price"`
	} `json:"items"`
	CurrentBillingPeriod struct {
		EndsAt string `json:"ends_at"`
	} `json:"current_billing_period"`
	CustomData struct {
		UserID string `json:"user_id"`
	} `json:"custom_data"`
}

// ─── Paddle 서명 검증 ────────────────────────────────────────────────────────

func verifyPaddleSignature(r *http.Request, body []byte) bool {
	secret := os.Getenv("PADDLE_WEBHOOK_SECRET")
	if secret == "" {
		return true // 개발 환경: 검증 생략
	}
	sigHeader := r.Header.Get("Paddle-Signature")
	if sigHeader == "" {
		return false
	}

	// 형식: ts=TIMESTAMP;h1=HMAC
	var ts, h1 string
	for _, part := range strings.Split(sigHeader, ";") {
		if strings.HasPrefix(part, "ts=") {
			ts = strings.TrimPrefix(part, "ts=")
		}
		if strings.HasPrefix(part, "h1=") {
			h1 = strings.TrimPrefix(part, "h1=")
		}
	}
	if ts == "" || h1 == "" {
		return false
	}

	payload := ts + ":" + string(body)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(h1))
}

// ─── Supabase 구독 상태 업데이트 ────────────────────────────────────────────

type supabaseSubUpsert struct {
	UserID               string  `json:"user_id"`
	PaddleSubscriptionID string  `json:"paddle_subscription_id"`
	PaddleCustomerID     string  `json:"paddle_customer_id"`
	Status               string  `json:"status"`
	CurrentPeriodEnd     *string `json:"current_period_end"`
	UpdatedAt            string  `json:"updated_at"`
}

func updateSupabaseSubscription(payload supabaseSubUpsert) error {
	supabaseURL := os.Getenv("SUPABASE_URL")
	serviceKey := os.Getenv("SUPABASE_SERVICE_ROLE_KEY")
	if supabaseURL == "" || serviceKey == "" {
		return fmt.Errorf("SUPABASE_URL / SUPABASE_SERVICE_ROLE_KEY 환경변수 미설정")
	}

	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	url := supabaseURL + "/rest/v1/subscriptions"
	req, err := http.NewRequest("POST", url, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apikey", serviceKey)
	req.Header.Set("Authorization", "Bearer "+serviceKey)
	req.Header.Set("Prefer", "resolution=merge-duplicates")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("supabase error %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

// ─── 웹훅 핸들러 ─────────────────────────────────────────────────────────────

func handlePaddleWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "body read error", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if !verifyPaddleSignature(r, body) {
		http.Error(w, "invalid signature", http.StatusUnauthorized)
		return
	}

	var event paddleWebhookEvent
	if err := json.Unmarshal(body, &event); err != nil {
		http.Error(w, "json parse error", http.StatusBadRequest)
		return
	}

	switch event.EventType {
	case "subscription.created", "subscription.updated", "subscription.activated":
		var sub paddleSubscriptionData
		if err := json.Unmarshal(event.Data, &sub); err != nil {
			http.Error(w, "data parse error", http.StatusBadRequest)
			return
		}
		userID := sub.CustomData.UserID
		if userID == "" {
			w.WriteHeader(http.StatusOK)
			return
		}

		dbStatus := "active"
		if sub.Status == "canceled" || sub.Status == "cancelled" {
			dbStatus = "expired"
		} else if sub.Status == "past_due" {
			dbStatus = "expired"
		}

		var periodEnd *string
		if sub.CurrentBillingPeriod.EndsAt != "" {
			v := sub.CurrentBillingPeriod.EndsAt
			periodEnd = &v
		}

		payload := supabaseSubUpsert{
			UserID:               userID,
			PaddleSubscriptionID: sub.ID,
			PaddleCustomerID:     sub.CustomerID,
			Status:               dbStatus,
			CurrentPeriodEnd:     periodEnd,
			UpdatedAt:            time.Now().UTC().Format(time.RFC3339),
		}
		if err := updateSupabaseSubscription(payload); err != nil {
			fmt.Println("[paddle webhook] supabase update error:", err)
		}

	case "subscription.canceled":
		var sub paddleSubscriptionData
		if err := json.Unmarshal(event.Data, &sub); err != nil {
			w.WriteHeader(http.StatusOK)
			return
		}
		userID := sub.CustomData.UserID
		if userID == "" {
			w.WriteHeader(http.StatusOK)
			return
		}
		payload := supabaseSubUpsert{
			UserID:               userID,
			PaddleSubscriptionID: sub.ID,
			PaddleCustomerID:     sub.CustomerID,
			Status:               "expired",
			CurrentPeriodEnd:     nil,
			UpdatedAt:            time.Now().UTC().Format(time.RFC3339),
		}
		if err := updateSupabaseSubscription(payload); err != nil {
			fmt.Println("[paddle webhook] supabase update error:", err)
		}
	}

	w.WriteHeader(http.StatusOK)
}
