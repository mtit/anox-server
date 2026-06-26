package api

// Config represents a configuration file structure
type Config struct {
	Version int64             `json:"version"`
	Values  map[string]string `json:"values"`
}

// ConfigUpdateMessage is sent via WebSocket when config changes
type ConfigUpdateMessage struct {
	Type    string `json:"type"`    // "config_update"
	Service string `json:"service"` // "_global" or service name
	Version int64  `json:"version"`
}

// ConfigPullRequest is sent by client to pull config
type ConfigPullRequest struct {
	Type    string `json:"type"`    // "pull_config"
	Service string `json:"service"` // "_global" or service name
}

// ConfigPullResponse is sent by server in response to pull request
type ConfigPullResponse struct {
	Type    string            `json:"type"`    // "config_response"
	Service string            `json:"service"`
	Version int64             `json:"version"`
	Values  map[string]string `json:"values"`
}
