package sms

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"anox/api"
)

const (
	aliyunEndpoint = "https://dysmsapi.aliyuncs.com/"
	aliyunVersion  = "2017-05-25"
	aliyunRegion   = "cn-hangzhou"
)

// AliyunSender sends SMS via Alibaba Cloud dysmsapi.
type AliyunSender struct {
	accessKeyID     string
	accessKeySecret string
	signName        string
	templateCode    string
	phoneNumbers    []string
	httpClient      *http.Client
}

// NewAliyunSender creates an Aliyun SMS sender.
func NewAliyunSender(cfg Config) (*AliyunSender, error) {
	if cfg.AccessKeyID == "" || cfg.AccessKeySecret == "" {
		return nil, fmt.Errorf("aliyun sms: access key is required")
	}
	if cfg.SignName == "" || cfg.TemplateCode == "" {
		return nil, fmt.Errorf("aliyun sms: sign name and template code are required")
	}
	if len(cfg.PhoneNumbers) == 0 {
		return nil, fmt.Errorf("aliyun sms: phone numbers are required")
	}

	return &AliyunSender{
		accessKeyID:     cfg.AccessKeyID,
		accessKeySecret: cfg.AccessKeySecret,
		signName:        cfg.SignName,
		templateCode:    cfg.TemplateCode,
		phoneNumbers:    cfg.PhoneNumbers,
		httpClient:      &http.Client{Timeout: 10 * time.Second},
	}, nil
}

func (s *AliyunSender) Provider() string {
	return ProviderAliyun
}

func (s *AliyunSender) SendAlert(ctx context.Context, entry *api.LogEntry) error {
	params, err := json.Marshal(BuildTemplateParams(entry))
	if err != nil {
		return fmt.Errorf("marshal template params: %w", err)
	}

	return s.send(ctx, strings.Join(s.phoneNumbers, ","), string(params))
}

func (s *AliyunSender) send(ctx context.Context, phoneNumbers, templateParam string) error {
	query := map[string]string{
		"AccessKeyId":      s.accessKeyID,
		"Action":           "SendSms",
		"Format":           "JSON",
		"PhoneNumbers":     phoneNumbers,
		"RegionId":         aliyunRegion,
		"SignName":         s.signName,
		"SignatureMethod":  "HMAC-SHA1",
		"SignatureNonce":   fmt.Sprintf("%d", time.Now().UnixNano()),
		"SignatureVersion": "1.0",
		"TemplateCode":     s.templateCode,
		"TemplateParam":    templateParam,
		"Timestamp":        time.Now().UTC().Format("2006-01-02T15:04:05Z"),
		"Version":          aliyunVersion,
	}

	signature, err := signAliyunRequest("POST", query, s.accessKeySecret)
	if err != nil {
		return err
	}
	query["Signature"] = signature

	form := url.Values{}
	for key, value := range query {
		form.Set(key, value)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, aliyunEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("http status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var result struct {
		Code    string `json:"Code"`
		Message string `json:"Message"`
		BizID   string `json:"BizId"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("parse response: %w, body=%s", err, strings.TrimSpace(string(body)))
	}
	if result.Code != "OK" {
		return fmt.Errorf("aliyun sms error code=%s message=%s", result.Code, result.Message)
	}

	return nil
}

func signAliyunRequest(method string, params map[string]string, accessKeySecret string) (string, error) {
	keys := make([]string, 0, len(params))
	for key := range params {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var canonical strings.Builder
	for i, key := range keys {
		if i > 0 {
			canonical.WriteByte('&')
		}
		canonical.WriteString(percentEncode(key))
		canonical.WriteByte('=')
		canonical.WriteString(percentEncode(params[key]))
	}

	stringToSign := method + "&" + percentEncode("/") + "&" + percentEncode(canonical.String())

	mac := hmac.New(sha1.New, []byte(accessKeySecret+"&"))
	if _, err := mac.Write([]byte(stringToSign)); err != nil {
		return "", fmt.Errorf("sign request: %w", err)
	}
	return base64.StdEncoding.EncodeToString(mac.Sum(nil)), nil
}

func percentEncode(value string) string {
	encoded := url.QueryEscape(value)
	encoded = strings.ReplaceAll(encoded, "+", "%20")
	encoded = strings.ReplaceAll(encoded, "*", "%2A")
	encoded = strings.ReplaceAll(encoded, "%7E", "~")
	return encoded
}
