package sms

import (
	"testing"
	"time"

	"anox/api"
)

func TestParsePhoneNumbers(t *testing.T) {
	phones := ParsePhoneNumbers("13800138000, 13900139000 , , ")
	if len(phones) != 2 {
		t.Fatalf("expected 2 phones, got %d", len(phones))
	}
	if phones[0] != "13800138000" || phones[1] != "13900139000" {
		t.Fatalf("unexpected phones: %#v", phones)
	}
}

func TestBuildTemplateParams(t *testing.T) {
	entry := &api.LogEntry{
		Time:     time.Date(2026, 6, 26, 10, 0, 0, 0, time.UTC),
		Service:  "user-service",
		Instance: "user-service-1",
		Level:    api.LogLevelEmergency,
		Action:   "panic",
		Message:  "this is a very long alert message that should be truncated",
	}

	params := BuildTemplateParams(entry)
	if params["service"] != "user-service" {
		t.Fatalf("unexpected service: %s", params["service"])
	}
	if len([]rune(params["message"])) > 33 {
		t.Fatalf("message should be truncated: %s", params["message"])
	}
	if params["time"] != "2026-06-26 10:00:00" {
		t.Fatalf("unexpected time: %s", params["time"])
	}
}

func TestSignAliyunRequest(t *testing.T) {
	params := map[string]string{
		"AccessKeyId":      "testid",
		"Action":           "SendSms",
		"Format":           "JSON",
		"PhoneNumbers":     "13800138000",
		"RegionId":         "cn-hangzhou",
		"SignName":         "签名",
		"SignatureMethod":  "HMAC-SHA1",
		"SignatureNonce":   "nonce",
		"SignatureVersion": "1.0",
		"TemplateCode":     "SMS_123",
		"TemplateParam":    `{"code":"1234"}`,
		"Timestamp":        "2026-06-26T00:00:00Z",
		"Version":          "2017-05-25",
	}

	sig, err := signAliyunRequest("POST", params, "testsecret")
	if err != nil {
		t.Fatalf("sign failed: %v", err)
	}
	if sig == "" {
		t.Fatal("expected non-empty signature")
	}

	sig2, err := signAliyunRequest("POST", params, "testsecret")
	if err != nil {
		t.Fatalf("sign failed: %v", err)
	}
	if sig != sig2 {
		t.Fatal("signature should be deterministic for same inputs except nonce - this test uses fixed nonce")
	}
}

func TestNewSenderFromAlertConfig(t *testing.T) {
	if NewSenderFromAlertConfig(nil) != nil {
		t.Fatal("expected nil sender for nil config")
	}

	sender := NewSenderFromAlertConfig(&api.AlertConfig{
		SMSAccessKeyID:     "id",
		SMSAccessKeySecret: "secret",
		SMSSignName:        "Anox",
		SMSTemplateCode:    "SMS_001",
		SMSPhoneNumbers:    "13800138000",
	})
	if sender == nil {
		t.Fatal("expected aliyun sender")
	}
	if sender.Provider() != ProviderAliyun {
		t.Fatalf("expected aliyun provider, got %s", sender.Provider())
	}
}

func TestNewSenderUnsupportedProvider(t *testing.T) {
	_, err := NewSender(Config{
		Provider:        "unknown",
		AccessKeyID:     "id",
		AccessKeySecret: "secret",
		SignName:        "Anox",
		TemplateCode:    "SMS_001",
		PhoneNumbers:    []string{"13800138000"},
	})
	if err == nil {
		t.Fatal("expected error for unsupported provider")
	}
}
