package api

import "time"

// LogLevel represents the severity level of a log entry
type LogLevel string

const (
	LogLevelDebug     LogLevel = "Debug"
	LogLevelInfo      LogLevel = "Info"
	LogLevelImportant LogLevel = "Important"
	LogLevelEmergency LogLevel = "Emergency"
)

// LogEntry represents a single log entry from a service
type LogEntry struct {
	Time     time.Time         `json:"time"`
	Service  string            `json:"service"`
	Instance string            `json:"instance"`
	Level    LogLevel          `json:"level"`
	Action   string            `json:"action"`
	Message  string            `json:"message"`
	TraceID  string            `json:"trace_id,omitempty"`
	Stacks   []string          `json:"stacks,omitempty"`
	Context  map[string]string `json:"context,omitempty"`
}

// AlertConfig represents the alert configuration for log center
type AlertConfig struct {
	Enabled           bool     `json:"enabled"`
	MinLevel          LogLevel `json:"min_level"`     // Minimum level to trigger alert
	Channels          []string `json:"channels"`      // "sms", "wechat", "dingtalk", "feishu"
	Deduplicate       bool     `json:"deduplicate"`   // Enable deduplication
	DeduplicateWindow int      `json:"deduplicate_window"` // Window in seconds

	// Webhook URLs - simplified configuration
	WechatURL   string `json:"wechat_url,omitempty"`   // 企业微信机器人 webhook URL
	DingtalkURL string `json:"dingtalk_url,omitempty"` // 钉钉机器人 webhook URL
	FeishuURL   string `json:"feishu_url,omitempty"`   // 飞书机器人 webhook URL

	// SMS configuration
	SMSProvider        string `json:"sms_provider,omitempty"` // aliyun, tencent, ...
	SMSAccessKeyID     string `json:"sms_access_key_id,omitempty"`
	SMSAccessKeySecret string `json:"sms_access_key_secret,omitempty"`
	SMSSignName        string `json:"sms_sign_name,omitempty"`
	SMSTemplateCode    string `json:"sms_template_code,omitempty"`
	SMSPhoneNumbers    string `json:"sms_phone_numbers,omitempty"` // comma separated
}

// AlertFingerprint is used for deduplication
type AlertFingerprint struct {
	Service string   `json:"service"`
	Level   LogLevel `json:"level"`
	Action  string   `json:"action"`
	Message string   `json:"message"`
}
