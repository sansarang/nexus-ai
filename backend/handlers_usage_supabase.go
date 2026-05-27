//go:build windows

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// supabaseFetchCount: Supabase REST API로 오늘 사용량 조회
func supabaseFetchCount(jwt, userID, feature, today string) (int, error) {
	endpoint := fmt.Sprintf(
		"%s/rest/v1/usage_logs?user_id=eq.%s&feature=eq.%s&date=eq.%s&select=count",
		supabaseProjectURL,
		url.QueryEscape(userID), url.QueryEscape(feature), today,
	)
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Authorization", "Bearer "+jwt)
	req.Header.Set("apikey", supabaseAnonKey)

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return 0, fmt.Errorf("supabase HTTP %d", resp.StatusCode)
	}
	var rows []struct {
		Count int `json:"count"`
	}
	json.NewDecoder(resp.Body).Decode(&rows)
	if len(rows) > 0 {
		return rows[0].Count, nil
	}
	return 0, nil
}

// supabaseIncrementRPC: increment_usage RPC로 원자적 카운터 증가
// Supabase에 아래 함수가 등록되어 있어야 함:
//
//	CREATE OR REPLACE FUNCTION increment_usage(p_user_id text, p_feature text, p_date text)
//	RETURNS void AS $$
//	  INSERT INTO usage_logs (user_id, feature, date, count)
//	  VALUES (p_user_id, p_feature, p_date, 1)
//	  ON CONFLICT (user_id, feature, date)
//	  DO UPDATE SET count = usage_logs.count + 1, updated_at = now();
//	$$ LANGUAGE sql SECURITY DEFINER;
func supabaseIncrementRPC(jwt, userID, feature, today string) error {
	body, _ := json.Marshal(map[string]string{
		"p_user_id": userID,
		"p_feature": feature,
		"p_date":    today,
	})
	req, err := http.NewRequest("POST", supabaseProjectURL+"/rest/v1/rpc/increment_usage", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+jwt)
	req.Header.Set("apikey", supabaseAnonKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("rpc HTTP %d", resp.StatusCode)
	}
	return nil
}
