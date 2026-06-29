package core

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"anox/api"
)

const (
	ConfigDir    = "data/configs"
	GlobalConfig = "_global"
	AnoxConfig   = "anox"
)

// ConfigStore manages all configuration files
type ConfigStore struct {
	mu       sync.RWMutex
	configs  map[string]*api.Config // service name -> config
	dataDir  string
	OnChange func(service string, config *api.Config)
}

// AnoxSettings represents Anox server settings
type AnoxSettings struct {
	Host   string `json:"host"`
	Port   string `json:"port"`
	Pass   string `json:"pass"`
	Secret string `json:"secret"`
}

// NewConfigStore creates a new config store
func NewConfigStore(dataDir string, onChange func(service string, config *api.Config)) (*ConfigStore, error) {
	if dataDir == "" {
		dataDir = ConfigDir
	}

	cs := &ConfigStore{
		configs:  make(map[string]*api.Config),
		dataDir:  dataDir,
		OnChange: onChange,
	}

	// Ensure config directory exists
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	// Load all existing configs
	if err := cs.loadAll(); err != nil {
		return nil, err
	}

	return cs, nil
}

// Get retrieves a config by service name
func (cs *ConfigStore) Get(service string) (*api.Config, error) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	config, exists := cs.configs[service]
	if !exists {
		return nil, fmt.Errorf("config not found: %s", service)
	}

	// Return a copy
	cfgCopy := &api.Config{
		Version: config.Version,
		Values:  make(map[string]string),
	}
	for k, v := range config.Values {
		cfgCopy.Values[k] = v
	}
	return cfgCopy, nil
}

// GetAll returns all configs
func (cs *ConfigStore) GetAll() map[string]*api.Config {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	result := make(map[string]*api.Config)
	for k, v := range cs.configs {
		cfgCopy := &api.Config{
			Version: v.Version,
			Values:  make(map[string]string),
		}
		for k2, v2 := range v.Values {
			cfgCopy.Values[k2] = v2
		}
		result[k] = cfgCopy
	}
	return result
}

// Set updates a config, creating if not exists
func (cs *ConfigStore) Set(service string, values map[string]string) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	config := &api.Config{
		Version: time.Now().Unix(),
		Values:  make(map[string]string),
	}
	for k, v := range values {
		config.Values[k] = v
	}

	cs.configs[service] = config

	// Persist to disk
	if err := cs.save(service, config); err != nil {
		return err
	}

	// Notify change
	if cs.OnChange != nil {
		go cs.OnChange(service, config)
	}

	return nil
}

// UpdateValues updates specific values in a config
func (cs *ConfigStore) UpdateValues(service string, values map[string]string) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	config, exists := cs.configs[service]
	if !exists {
		// Create new config
		config = &api.Config{
			Values: make(map[string]string),
		}
		cs.configs[service] = config
	}

	// Update values
	for k, v := range values {
		config.Values[k] = v
	}
	config.Version = time.Now().Unix()

	// Persist to disk
	if err := cs.save(service, config); err != nil {
		return err
	}

	// Notify change
	if cs.OnChange != nil {
		go cs.OnChange(service, config)
	}

	return nil
}

// Delete removes a config key
func (cs *ConfigStore) Delete(service, key string) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	config, exists := cs.configs[service]
	if !exists {
		return fmt.Errorf("config not found: %s", service)
	}

	delete(config.Values, key)
	config.Version = time.Now().Unix()

	// Persist to disk
	if err := cs.save(service, config); err != nil {
		return err
	}

	// Notify change
	if cs.OnChange != nil {
		go cs.OnChange(service, config)
	}

	return nil
}

// DeleteConfig removes an entire config file
func (cs *ConfigStore) DeleteConfig(service string) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	delete(cs.configs, service)

	// Delete file
	path := cs.filePath(service)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}

// GetAnoxSettings returns Anox server settings from anox config
func (cs *ConfigStore) GetAnoxSettings() (*AnoxSettings, error) {
	config, err := cs.Get(AnoxConfig)
	if err != nil {
		// Return defaults
		return &AnoxSettings{
			Host:   getEnvOrDefault("ANOX_HOST", "0.0.0.0"),
			Port:   getEnvOrDefault("ANOX_PORT", "8848"),
			Pass:   getEnvOrDefault("ANOX_PASS", "admin"),
			Secret: defaultAnoxSecret(),
		}, nil
	}

	settings := &AnoxSettings{
		Host:   getEnvOrDefault("ANOX_HOST", config.Values["host"]),
		Port:   getEnvOrDefault("ANOX_PORT", config.Values["port"]),
		Pass:   getEnvOrDefault("ANOX_PASS", config.Values["pass"]),
		Secret: config.Values["secret"],
	}

	// Apply defaults if empty
	if settings.Host == "" {
		settings.Host = "0.0.0.0"
	}
	if settings.Port == "" {
		settings.Port = "8848"
	}
	if settings.Pass == "" {
		settings.Pass = "admin"
	}
	if settings.Secret == "" {
		settings.Secret = defaultAnoxSecret()
	}

	return settings, nil
}

// SaveAnoxSettings saves Anox server settings
func (cs *ConfigStore) SaveAnoxSettings(settings *AnoxSettings) error {
	values := map[string]string{
		"host":   settings.Host,
		"port":   settings.Port,
		"pass":   settings.Pass,
		"secret": settings.Secret,
	}
	return cs.UpdateValues(AnoxConfig, values)
}

// EnsureAnoxSettingsDefaults writes host/port/pass/secret into anox.json when missing.
func (cs *ConfigStore) EnsureAnoxSettingsDefaults() error {
	defaults := map[string]string{
		"host":   defaultAnoxHost(),
		"port":   defaultAnoxPort(),
		"pass":   defaultAnoxPass(),
		"secret": generateAnoxSecret(),
	}

	config, err := cs.Get(AnoxConfig)
	if err != nil {
		return cs.UpdateValues(AnoxConfig, defaults)
	}

	updates := make(map[string]string)
	for key, value := range defaults {
		if config.Values[key] == "" {
			updates[key] = value
		}
	}

	if len(updates) == 0 {
		return nil
	}

	return cs.UpdateValues(AnoxConfig, updates)
}

func defaultAnoxHost() string {
	if v := os.Getenv("ANOX_HOST"); v != "" {
		return v
	}
	if v := os.Getenv("HOST"); v != "" {
		return v
	}
	return "0.0.0.0"
}

func defaultAnoxPort() string {
	if v := os.Getenv("ANOX_PORT"); v != "" {
		return v
	}
	if v := os.Getenv("PORT"); v != "" {
		return v
	}
	return "8848"
}

func defaultAnoxPass() string {
	if v := os.Getenv("ANOX_PASS"); v != "" {
		return v
	}
	if v := os.Getenv("PASS"); v != "" {
		return v
	}
	return "admin"
}

func defaultAnoxSecret() string {
	if v := os.Getenv("ANOX_SECRET"); v != "" {
		return v
	}
	if v := os.Getenv("SECRET"); v != "" {
		return v
	}
	return "anox-default-secret"
}

func generateAnoxSecret() string {
	if v := defaultAnoxSecret(); v != "anox-default-secret" {
		return v
	}
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return defaultAnoxSecret()
	}
	return base64.RawURLEncoding.EncodeToString(buf)
}

// SyncEnvWithConfig syncs environment variables with anox config
func (cs *ConfigStore) SyncEnvWithConfig() error {
	host := os.Getenv("HOST")
	port := os.Getenv("PORT")
	pass := os.Getenv("PASS")
	secret := os.Getenv("SECRET")

	if host == "" && port == "" && pass == "" && secret == "" {
		return nil // No env overrides
	}

	// Load existing or create new
	config, err := cs.Get(AnoxConfig)
	if err != nil {
		config = &api.Config{Values: make(map[string]string)}
	}

	// Apply env overrides
	if host != "" {
		config.Values["host"] = host
	}
	if port != "" {
		config.Values["port"] = port
	}
	if pass != "" {
		config.Values["pass"] = pass
	}
	if secret != "" {
		config.Values["secret"] = secret
	}

	// Check if we need to update
	needsUpdate := false
	for k, v := range config.Values {
		envVal := os.Getenv(strings.ToUpper(k))
		if envVal != "" && envVal != v {
			config.Values[k] = envVal
			needsUpdate = true
		}
	}

	if needsUpdate || err != nil { // err means config didn't exist
		return cs.Set(AnoxConfig, config.Values)
	}

	return nil
}

// GetAlertConfig returns alert configuration
func (cs *ConfigStore) GetAlertConfig() (*api.AlertConfig, error) {
	config, err := cs.Get(AnoxConfig)
	if err != nil {
		return &api.AlertConfig{
			Enabled:           false,
			MinLevel:          api.LogLevelEmergency,
			Deduplicate:       true,
			DeduplicateWindow: 300,
		}, nil
	}

	alertConfig := &api.AlertConfig{
		Enabled:           config.Values["alert_enabled"] == "true",
		MinLevel:          api.LogLevel(config.Values["alert_min_level"]),
		Channels:          parseChannels(config.Values["alert_channels"]),
		Deduplicate:       config.Values["alert_deduplicate"] != "false",
		DeduplicateWindow: parseInt(config.Values["alert_dedup_window"], 300),
		// Webhook URLs - simplified configuration
		WechatURL:   config.Values["wechat_url"],
		DingtalkURL: config.Values["dingtalk_url"],
		FeishuURL:   config.Values["feishu_url"],
		// SMS configuration
		SMSProvider:        config.Values["sms_provider"],
		SMSAccessKeyID:     config.Values["sms_access_key_id"],
		SMSAccessKeySecret: config.Values["sms_access_key_secret"],
		SMSSignName:        config.Values["sms_sign_name"],
		SMSTemplateCode:    config.Values["sms_template_code"],
		SMSPhoneNumbers:    config.Values["sms_phone_numbers"],
	}

	if alertConfig.MinLevel == "" {
		alertConfig.MinLevel = api.LogLevelEmergency
	}

	return alertConfig, nil
}

// SaveAlertConfig saves alert configuration
func (cs *ConfigStore) SaveAlertConfig(alertConfig *api.AlertConfig) error {
	values := map[string]string{
		"alert_enabled":      strconv.FormatBool(alertConfig.Enabled),
		"alert_min_level":    string(alertConfig.MinLevel),
		"alert_channels":     strings.Join(alertConfig.Channels, ","),
		"alert_deduplicate":  strconv.FormatBool(alertConfig.Deduplicate),
		"alert_dedup_window": strconv.Itoa(alertConfig.DeduplicateWindow),
		// Webhook URLs - simplified configuration
		"wechat_url":   alertConfig.WechatURL,
		"dingtalk_url": alertConfig.DingtalkURL,
		"feishu_url":   alertConfig.FeishuURL,
		// SMS configuration
		"sms_provider":          alertConfig.SMSProvider,
		"sms_access_key_id":     alertConfig.SMSAccessKeyID,
		"sms_access_key_secret": alertConfig.SMSAccessKeySecret,
		"sms_sign_name":         alertConfig.SMSSignName,
		"sms_template_code":     alertConfig.SMSTemplateCode,
		"sms_phone_numbers":     alertConfig.SMSPhoneNumbers,
	}

	return cs.UpdateValues(AnoxConfig, values)
}

// filePath returns the file path for a service config
func (cs *ConfigStore) filePath(service string) string {
	return filepath.Join(cs.dataDir, service+".json")
}

// loadAll loads all config files from disk
func (cs *ConfigStore) loadAll() error {
	files, err := os.ReadDir(cs.dataDir)
	if err != nil {
		return err
	}

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		service := strings.TrimSuffix(file.Name(), ".json")
		path := filepath.Join(cs.dataDir, file.Name())

		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read config %s: %w", service, err)
		}

		var config api.Config
		if err := json.Unmarshal(data, &config); err != nil {
			return fmt.Errorf("failed to parse config %s: %w", service, err)
		}

		cs.configs[service] = &config
	}

	return nil
}

// save persists a config to disk
func (cs *ConfigStore) save(service string, config *api.Config) error {
	path := cs.filePath(service)

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// getEnvOrDefault returns env value or default
func getEnvOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func parseChannels(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	channels := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			channels = append(channels, part)
		}
	}
	return channels
}

func parseList(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, ",")
}

func parseInt(s string, defaultVal int) int {
	if s == "" {
		return defaultVal
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return defaultVal
	}
	return v
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
