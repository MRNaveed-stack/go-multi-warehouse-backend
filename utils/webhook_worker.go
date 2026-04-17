package utils

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"pureGo/config"
	"time"
)

type WebhookPayLoad struct {
	Event string      `json:"event"`
	Data  interface{} `json:"data"`
}

type WebhookJobData struct {
	URL     string
	Secret  string
	Payload WebhookPayLoad
}

func DispatchWebHook(eventType string, data interface{}) {
	rows, err := config.DB.Query("SELECT url, secret FROM webhooks WHERE event_type = $1 AND is_active = true", eventType)
	if err != nil {
		log.Printf("webhook query failed for event %s: %v", eventType, err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var url, secret string
		if err := rows.Scan(&url, &secret); err != nil {
			log.Printf("webhook scan failed: %v", err)
			continue
		}

		JobQueue <- Job{
			Type: "WEBHOOK_SEND",
			Data: WebhookJobData{
				URL:    url,
				Secret: secret,
				Payload: WebhookPayLoad{
					Event: eventType,
					Data:  data,
				},
			},
		}
	}

	if err := rows.Err(); err != nil {
		log.Printf("webhook iteration failed: %v", err)
	}
}

func SendHttpRequest(url string, payload interface{}, secret string) {
	body, err := json.Marshal(payload)
	if err != nil {
		log.Printf("webhook marshal failed: %v", err)
		return
	}

	h := hmac.New(sha256.New, []byte(secret))
	h.Write(body)
	signature := hex.EncodeToString(h.Sum(nil))

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		log.Printf("webhook request creation failed: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Hub-Signature", signature)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("webhook request failed: %v", err)
		return
	}
	defer resp.Body.Close()
}
