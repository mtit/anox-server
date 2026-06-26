package sdk

import (
	"strconv"
)

// ConfigAccessor provides config access utilities
type ConfigAccessor struct {
	client *Client
}

// GetString gets a string config value
func (c *ConfigAccessor) GetString(key string) string {
	return c.client.GetConfig(key)
}

// GetServiceString gets a service-specific string config value
func (c *ConfigAccessor) GetServiceString(key string) string {
	return c.client.GetServiceConfig(key)
}

// GetGlobalString gets a global string config value
func (c *ConfigAccessor) GetGlobalString(key string) string {
	return c.client.GetGlobalConfig(key)
}

// GetInt gets an int config value
func (c *ConfigAccessor) GetInt(key string, defaultVal int) int {
	val := c.client.GetConfig(key)
	if val == "" {
		return defaultVal
	}

	i, err := strconv.Atoi(val)
	if err != nil {
		return defaultVal
	}

	return i
}

// GetInt64 gets an int64 config value
func (c *ConfigAccessor) GetInt64(key string, defaultVal int64) int64 {
	val := c.client.GetConfig(key)
	if val == "" {
		return defaultVal
	}

	i, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return defaultVal
	}

	return i
}

// GetFloat64 gets a float64 config value
func (c *ConfigAccessor) GetFloat64(key string, defaultVal float64) float64 {
	val := c.client.GetConfig(key)
	if val == "" {
		return defaultVal
	}

	f, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return defaultVal
	}

	return f
}

// GetBool gets a bool config value
func (c *ConfigAccessor) GetBool(key string, defaultVal bool) bool {
	val := c.client.GetConfig(key)
	if val == "" {
		return defaultVal
	}

	b, err := strconv.ParseBool(val)
	if err != nil {
		return defaultVal
	}

	return b
}

// Config provides a convenient way to access config values
// Usage: client.Config().GetString("key")
func (c *Client) Config() *ConfigAccessor {
	return &ConfigAccessor{client: c}
}
