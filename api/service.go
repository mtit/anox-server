package api

import (
	"time"
)

// ServiceInstance represents a registered service instance
type ServiceInstance struct {
	ID            string    `json:"id"`
	ServiceName   string    `json:"service_name"`
	RegisteredAt  time.Time `json:"registered_at"`
	LastHeartbeat time.Time `json:"last_heartbeat"`
	
	// System metrics from heartbeat
	CPUCores      int     `json:"cpu_cores"`
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryTotalMB int64   `json:"memory_total_mb"`
	MemoryAvailMB int64   `json:"memory_avail_mb"`
	
	// Config versions from heartbeat
	GlobalVersion  int64 `json:"global_version"`
	ServiceVersion int64 `json:"service_version"`

	// HTTP endpoint reported at registration (for gateway routing)
	HttpHost string `json:"http_host"`
	HttpPort string `json:"http_port"`
	
	// WebSocket connection (not serialized)
	Conn interface{} `json:"-"`
}

// HeartbeatMessage is sent by client every 15 seconds
type HeartbeatMessage struct {
	Type           string  `json:"type"`           // "ping"
	CPUCores       int     `json:"cpu_cores"`
	CPUPercent     float64 `json:"cpu_percent"`
	MemoryTotalMB  int64   `json:"memory_total_mb"`
	MemoryAvailMB  int64   `json:"memory_avail_mb"`
	GlobalVersion  int64   `json:"global_version"`
	ServiceVersion int64   `json:"service_version"`
}

// HeartbeatResponse is sent by server in response to ping
type HeartbeatResponse struct {
	Type           string `json:"type"`           // "pong"
	NeedUpdateGlobal  bool `json:"need_update_global"`
	NeedUpdateService bool `json:"need_update_service"`
	GlobalVersion  int64  `json:"global_version"`
	ServiceVersion int64  `json:"service_version"`
}

// RegisterMessage is sent by client when connecting
type RegisterMessage struct {
	Type        string `json:"type"`         // "register"
	ServiceName string `json:"service_name"`
	HttpHost    string `json:"http_host"`    // HTTP listen host for gateway routing
	HttpPort    string `json:"http_port"`    // HTTP listen port for gateway routing
}

// RegisterResponse is sent by server in response to register
type RegisterResponse struct {
	Type       string `json:"type"`       // "register_response"
	InstanceID string `json:"instance_id"`
	Success    bool   `json:"success"`
	Message    string `json:"message,omitempty"`
}

// ServiceSummary is used for API responses
type ServiceSummary struct {
	Name         string               `json:"name"`
	InstanceCount int                 `json:"instance_count"`
	Instances   []*ServiceInstance   `json:"instances,omitempty"`
}
