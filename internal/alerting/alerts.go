package alerting

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/smtp"
	"time"
	"tkngate/internal/config"
	"tkngate/internal/logging"
)

// DispatchAlert fires asynchronous alerts to configured destinations (Slack, Email, Webhook)
func DispatchAlert(scope string, identifier string, zone string, spent, limit float64) {
	if !config.Cfg.Alerts.Enabled {
		return
	}

	go func() {
		msg := fmt.Sprintf("🚨 BUDGET ALERT: %s [%s] entered %s zone (Spent: $%.2f / Limit: $%.2f)", scope, identifier, zone, spent, limit)
		
		if config.Cfg.Alerts.SlackWebhook != "" {
			sendSlackWebhook(config.Cfg.Alerts.SlackWebhook, msg)
		}

		if config.Cfg.Alerts.GenericWebhook != "" {
			sendGenericWebhook(config.Cfg.Alerts.GenericWebhook, scope, identifier, zone, spent, limit)
		}

		if config.Cfg.Alerts.EmailSmtpHost != "" {
			sendEmail(msg)
		}
	}()
}

func sendSlackWebhook(url, text string) {
	payload := map[string]string{"text": text}
	body, _ := json.Marshal(payload)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		logging.Logger.Error("Failed to send Slack alert", "error", err)
		return
	}
	defer resp.Body.Close()
}

func sendGenericWebhook(url, scope, identifier string, zone string, spent, limit float64) {
	payload := map[string]interface{}{
		"event":      "budget_threshold_crossed",
		"scope":      scope,
		"identifier": identifier,
		"zone":       zone,
		"spent_usd":  spent,
		"limit_usd":  limit,
		"timestamp":  time.Now().Format(time.RFC3339),
	}
	body, _ := json.Marshal(payload)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		logging.Logger.Error("Failed to send generic webhook alert", "error", err)
		return
	}
	defer resp.Body.Close()
}

func sendEmail(text string) {
	conf := config.Cfg.Alerts
	auth := smtp.PlainAuth("", conf.EmailSmtpUser, conf.EmailSmtpPass, conf.EmailSmtpHost)
	to := []string{conf.EmailTo}
	msg := []byte("To: " + conf.EmailTo + "\r\n" +
		"Subject: Tkngate Budget Alert\r\n" +
		"\r\n" + text + "\r\n")
		
	addr := fmt.Sprintf("%s:%d", conf.EmailSmtpHost, conf.EmailSmtpPort)
	err := smtp.SendMail(addr, auth, conf.EmailFrom, to, msg)
	if err != nil {
		logging.Logger.Error("Failed to send Email alert", "error", err)
	}
}
