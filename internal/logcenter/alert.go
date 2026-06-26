package logcenter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"anox/api"
	"anox/internal/logcenter/sms"
)

const (
	webhookTimeout     = 10 * time.Second
	wechatTextMaxBytes = 2048
	wechatMDMaxBytes   = 4096
)

var webhookHTTPClient = &http.Client{Timeout: webhookTimeout}

// AlertEngine handles log alerting with deduplication
type AlertEngine struct {
	mu           sync.RWMutex
	config       *api.AlertConfig
	smsSender    sms.Sender
	recentAlerts map[string]time.Time // fingerprint -> last alert time
	window       time.Duration
}

// NewAlertEngine creates a new alert engine
func NewAlertEngine() *AlertEngine {
	return &AlertEngine{
		recentAlerts: make(map[string]time.Time),
		window:       5 * time.Minute,
	}
}

// UpdateConfig updates the alert configuration
func (ae *AlertEngine) UpdateConfig(config *api.AlertConfig) {
	ae.mu.Lock()
	defer ae.mu.Unlock()
	if config == nil {
		ae.config = nil
		ae.smsSender = nil
		return
	}
	cfg := *config
	ae.config = &cfg
	ae.smsSender = sms.NewSenderFromAlertConfig(&cfg)
	if config.DeduplicateWindow > 0 {
		ae.window = time.Duration(config.DeduplicateWindow) * time.Second
	}
	log.Printf("[AlertEngine] Config updated: enabled=%v min_level=%s channels=%v",
		cfg.Enabled, cfg.MinLevel, cfg.Channels)
}

// Check checks a log entry against alert rules
func (ae *AlertEngine) Check(entry *api.LogEntry) {
	ae.mu.RLock()
	config := ae.config
	ae.mu.RUnlock()

	if config == nil || !config.Enabled {
		return
	}

	if !ae.shouldAlert(entry.Level, config.MinLevel) {
		return
	}

	fingerprint := ae.buildFingerprint(entry)

	if config.Deduplicate {
		if ae.isDuplicate(fingerprint) {
			return
		}
		ae.recordAlert(fingerprint)
	}

	log.Printf("[AlertEngine] Alert triggered: service=%s instance=%s level=%s action=%s",
		entry.Service, entry.Instance, entry.Level, entry.Action)

	for _, channel := range config.Channels {
		channel := strings.TrimSpace(channel)
		if channel == "" {
			continue
		}
		go ae.sendAlert(channel, entry, config)
	}
}

func (ae *AlertEngine) shouldAlert(level, minLevel api.LogLevel) bool {
	levels := []api.LogLevel{
		api.LogLevelDebug,
		api.LogLevelInfo,
		api.LogLevelImportant,
		api.LogLevelEmergency,
	}

	var levelIdx, minLevelIdx int
	for i, l := range levels {
		if l == level {
			levelIdx = i
		}
		if l == minLevel {
			minLevelIdx = i
		}
	}

	return levelIdx >= minLevelIdx
}

func (ae *AlertEngine) buildFingerprint(entry *api.LogEntry) string {
	msg := entry.Message
	if len(msg) > 50 {
		msg = msg[:50]
	}
	return fmt.Sprintf("%s|%s|%s|%s", entry.Service, entry.Level, entry.Action, msg)
}

func (ae *AlertEngine) isDuplicate(fingerprint string) bool {
	ae.mu.RLock()
	lastAlert, exists := ae.recentAlerts[fingerprint]
	ae.mu.RUnlock()

	if !exists {
		return false
	}

	return time.Since(lastAlert) < ae.window
}

func (ae *AlertEngine) recordAlert(fingerprint string) {
	ae.mu.Lock()
	ae.recentAlerts[fingerprint] = time.Now()
	ae.mu.Unlock()
}

func (ae *AlertEngine) sendAlert(channel string, entry *api.LogEntry, config *api.AlertConfig) {
	switch channel {
	case "wechat":
		ae.sendWechat(entry, config.WechatURL)
	case "dingtalk":
		ae.sendDingtalk(entry, config.DingtalkURL)
	case "feishu":
		ae.sendFeishu(entry, config.FeishuURL)
	case "sms":
		ae.sendSMS(entry, config)
	default:
		log.Printf("[AlertEngine] Unknown channel: %s", channel)
	}
}

func formatAlertTime(t time.Time) string {
	if t.IsZero() {
		return time.Now().Format("2006-01-02 15:04:05")
	}
	return t.Format("2006-01-02 15:04:05")
}

func levelColor(level api.LogLevel) string {
	switch level {
	case api.LogLevelEmergency, api.LogLevelImportant:
		return "warning"
	default:
		return "comment"
	}
}

func formatWechatMarkdown(entry *api.LogEntry) string {
	content := fmt.Sprintf(
		"### Anox 告警通知\n"+
			"> **服务**：%s\n"+
			"> **实例**：%s\n"+
			"> **级别**：<font color=\"%s\">%s</font>\n"+
			"> **动作**：%s\n"+
			"> **消息**：%s\n"+
			"> **时间**：%s",
		entry.Service,
		entry.Instance,
		levelColor(entry.Level),
		entry.Level,
		entry.Action,
		entry.Message,
		formatAlertTime(entry.Time),
	)
	return truncateUTF8Bytes(content, wechatMDMaxBytes)
}

func formatPlainAlert(entry *api.LogEntry) string {
	content := fmt.Sprintf(
		"Anox 告警通知\n\n服务：%s\n实例：%s\n级别：%s\n动作：%s\n消息：%s\n时间：%s",
		entry.Service,
		entry.Instance,
		entry.Level,
		entry.Action,
		entry.Message,
		formatAlertTime(entry.Time),
	)
	return truncateUTF8Bytes(content, wechatTextMaxBytes)
}

func truncateUTF8Bytes(s string, maxBytes int) string {
	if len(s) <= maxBytes {
		return s
	}
	for maxBytes > 0 && (s[maxBytes]&0xC0) == 0x80 {
		maxBytes--
	}
	return s[:maxBytes] + "..."
}

type webhookResponse struct {
	ErrCode       int    `json:"errcode"`
	ErrMsg        string `json:"errmsg"`
	Code          int    `json:"code"`
	Msg           string `json:"msg"`
	StatusCode    int    `json:"StatusCode"`
	StatusMessage string `json:"StatusMessage"`
}

func postWebhook(channel, webhookURL string, body interface{}) error {
	if webhookURL == "" {
		return fmt.Errorf("webhook URL is empty")
	}

	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal request body: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, webhookURL, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	resp, err := webhookHTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("http status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var result webhookResponse
	if len(respBody) > 0 {
		if err := json.Unmarshal(respBody, &result); err != nil {
			return fmt.Errorf("parse response: %w, body=%s", err, strings.TrimSpace(string(respBody)))
		}
	}

	switch channel {
	case "wechat", "dingtalk":
		if result.ErrCode != 0 {
			return fmt.Errorf("api error errcode=%d errmsg=%s", result.ErrCode, result.ErrMsg)
		}
	case "feishu":
		if result.Code != 0 {
			return fmt.Errorf("api error code=%d msg=%s", result.Code, result.Msg)
		}
		if result.StatusCode != 0 {
			return fmt.Errorf("api error StatusCode=%d StatusMessage=%s", result.StatusCode, result.StatusMessage)
		}
	}

	return nil
}

// sendWechat sends alert via WeChat Work webhook
// https://developer.work.weixin.qq.com/document/path/99110
func (ae *AlertEngine) sendWechat(entry *api.LogEntry, webhookURL string) {
	if webhookURL == "" {
		log.Printf("[AlertEngine] WeChat webhook URL not configured")
		return
	}

	body := map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]string{
			"content": formatWechatMarkdown(entry),
		},
	}

	if err := postWebhook("wechat", webhookURL, body); err != nil {
		log.Printf("[AlertEngine] WeChat alert failed for %s/%s: %v", entry.Service, entry.Instance, err)
		return
	}

	log.Printf("[AlertEngine] WeChat alert sent for %s/%s", entry.Service, entry.Instance)
}

// sendDingtalk sends alert via DingTalk custom robot webhook
func (ae *AlertEngine) sendDingtalk(entry *api.LogEntry, webhookURL string) {
	if webhookURL == "" {
		log.Printf("[AlertEngine] DingTalk webhook URL not configured")
		return
	}

	targetURL := webhookURL

	content := fmt.Sprintf(
		"### Anox 告警通知\n\n"+
			"- **服务**：%s\n"+
			"- **实例**：%s\n"+
			"- **级别**：%s\n"+
			"- **动作**：%s\n"+
			"- **消息**：%s\n"+
			"- **时间**：%s",
		entry.Service,
		entry.Instance,
		entry.Level,
		entry.Action,
		entry.Message,
		formatAlertTime(entry.Time),
	)

	body := map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]string{
			"title": "Anox 告警",
			"text":  content,
		},
	}

	if err := postWebhook("dingtalk", targetURL, body); err != nil {
		log.Printf("[AlertEngine] DingTalk alert failed for %s/%s: %v", entry.Service, entry.Instance, err)
		return
	}

	log.Printf("[AlertEngine] DingTalk alert sent for %s/%s", entry.Service, entry.Instance)
}

// sendFeishu sends alert via Feishu/Lark custom bot webhook
func (ae *AlertEngine) sendFeishu(entry *api.LogEntry, webhookURL string) {
	if webhookURL == "" {
		log.Printf("[AlertEngine] Feishu webhook URL not configured")
		return
	}

	content := formatPlainAlert(entry)
	body := map[string]interface{}{
		"msg_type": "text",
		"content": map[string]string{
			"text": content,
		},
	}

	if err := postWebhook("feishu", webhookURL, body); err != nil {
		log.Printf("[AlertEngine] Feishu alert failed for %s/%s: %v", entry.Service, entry.Instance, err)
		return
	}

	log.Printf("[AlertEngine] Feishu alert sent for %s/%s", entry.Service, entry.Instance)
}

// sendSMS sends alert via configured SMS provider.
func (ae *AlertEngine) sendSMS(entry *api.LogEntry, config *api.AlertConfig) {
	ae.mu.RLock()
	sender := ae.smsSender
	ae.mu.RUnlock()

	if sender == nil {
		log.Printf("[AlertEngine] SMS not configured (provider=%s)", config.SMSProvider)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), webhookTimeout)
	defer cancel()

	if err := sender.SendAlert(ctx, entry); err != nil {
		log.Printf("[AlertEngine] SMS alert failed for %s/%s via %s: %v",
			entry.Service, entry.Instance, sender.Provider(), err)
		return
	}

	log.Printf("[AlertEngine] SMS alert sent for %s/%s via %s",
		entry.Service, entry.Instance, sender.Provider())
}

// CleanupOldAlerts cleans up old alert records
func (ae *AlertEngine) CleanupOldAlerts() {
	ae.mu.Lock()
	defer ae.mu.Unlock()

	threshold := time.Now().Add(-2 * ae.window)
	for fp, lastAlert := range ae.recentAlerts {
		if lastAlert.Before(threshold) {
			delete(ae.recentAlerts, fp)
		}
	}
}
