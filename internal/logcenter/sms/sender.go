package sms

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"anox/api"
)

const (
	ProviderAliyun  = "aliyun"
	ProviderTencent = "tencent"
)

// Config holds credentials and template settings for an SMS provider.
type Config struct {
	Provider        string
	AccessKeyID     string
	AccessKeySecret string
	SignName        string
	TemplateCode    string
	PhoneNumbers    []string
}

// Sender sends templated SMS alert messages.
type Sender interface {
	Provider() string
	SendAlert(ctx context.Context, entry *api.LogEntry) error
}

// NewSender creates an SMS sender for the configured provider.
func NewSender(cfg Config) (Sender, error) {
	provider := strings.ToLower(strings.TrimSpace(cfg.Provider))
	if provider == "" {
		provider = ProviderAliyun
	}

	switch provider {
	case ProviderAliyun:
		return NewAliyunSender(cfg)
	case ProviderTencent:
		return nil, fmt.Errorf("sms provider %q is not implemented yet", provider)
	default:
		return nil, fmt.Errorf("unsupported sms provider: %s", provider)
	}
}

// NewSenderFromAlertConfig builds a sender from alert settings.
// Returns nil when SMS is not configured.
func NewSenderFromAlertConfig(cfg *api.AlertConfig) Sender {
	if cfg == nil {
		return nil
	}

	phones := ParsePhoneNumbers(cfg.SMSPhoneNumbers)
	if len(phones) == 0 {
		return nil
	}
	if cfg.SMSAccessKeyID == "" || cfg.SMSAccessKeySecret == "" {
		return nil
	}
	if cfg.SMSSignName == "" || cfg.SMSTemplateCode == "" {
		return nil
	}

	sender, err := NewSender(Config{
		Provider:        cfg.SMSProvider,
		AccessKeyID:     cfg.SMSAccessKeyID,
		AccessKeySecret: cfg.SMSAccessKeySecret,
		SignName:        cfg.SMSSignName,
		TemplateCode:    cfg.SMSTemplateCode,
		PhoneNumbers:    phones,
	})
	if err != nil {
		return nil
	}
	return sender
}

// ParsePhoneNumbers splits comma-separated phone numbers.
func ParsePhoneNumbers(raw string) []string {
	parts := strings.Split(raw, ",")
	phones := make([]string, 0, len(parts))
	for _, part := range parts {
		phone := strings.TrimSpace(part)
		if phone != "" {
			phones = append(phones, phone)
		}
	}
	return phones
}

// BuildTemplateParams maps a log entry to SMS template variables.
// Templates should declare variables: service, instance, level, action, message, time.
func BuildTemplateParams(entry *api.LogEntry) map[string]string {
	t := entry.Time
	if t.IsZero() {
		t = time.Now()
	}
	return map[string]string{
		"service":  entry.Service,
		"instance": entry.Instance,
		"level":    string(entry.Level),
		"action":   entry.Action,
		"message":  truncateRunes(entry.Message, 30),
		"time":     t.Format("2006-01-02 15:04:05"),
	}
}

func truncateRunes(s string, maxRunes int) string {
	if utf8.RuneCountInString(s) <= maxRunes {
		return s
	}
	runes := []rune(s)
	return string(runes[:maxRunes]) + "..."
}
