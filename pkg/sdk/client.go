package sdk

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"anox/api"
)

const (
	registerTimeout   = 10 * time.Second
	maxReconnectDelay = 30 * time.Second
)

// Client is the main SDK client for interacting with Anox
type Client struct {
	anoxURL        string
	serviceName    string
	instanceID     string
	httpHost       string
	httpPort       string
	conn           *websocket.Conn
	heartbeatStop  chan struct{}
	mu             sync.RWMutex
	writeMu        sync.Mutex
	reconnectMu    sync.Mutex
	registerDone   chan error
	closed         bool

	// Config
	globalConfig   map[string]string
	serviceConfig  map[string]string
	globalVersion  int64
	serviceVersion int64

	// Logger
	logger *Logger
}

// Config holds client configuration
type Config struct {
	AnoxURL     string // Anox server WebSocket URL (e.g., "ws://localhost:8848/ws")
	ServiceName string // Service name for registration
	HttpHost    string // HTTP listen host reported to Anox for gateway routing
	HttpPort    string // HTTP listen port reported to Anox for gateway routing
}

// NewClient creates a new SDK client
func NewClient(cfg Config) (*Client, error) {
	httpHost := cfg.HttpHost
	if httpHost == "" {
		httpHost = "127.0.0.1"
	}

	client := &Client{
		anoxURL:       cfg.AnoxURL,
		serviceName:   cfg.ServiceName,
		httpHost:      httpHost,
		httpPort:      cfg.HttpPort,
		globalConfig:  make(map[string]string),
		serviceConfig: make(map[string]string),
		heartbeatStop: make(chan struct{}),
	}

	if err := client.connect(); err != nil {
		return nil, fmt.Errorf("failed to connect to Anox: %w", err)
	}

	go client.messageLoop()
	go client.heartbeatLoop()

	if err := client.register(); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to register: %w", err)
	}

	client.logger = newLogger(client)
	client.pullInitialConfigs()

	return client, nil
}

// Close closes the client connection
func (c *Client) Close() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true
	c.mu.Unlock()

	close(c.heartbeatStop)

	c.mu.Lock()
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
	c.mu.Unlock()

	if c.logger != nil {
		c.logger.Close()
	}

	return nil
}

// GetConfig gets a config value by key
func (c *Client) GetConfig(key string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if val, ok := c.serviceConfig[key]; ok {
		return val
	}

	if val, ok := c.globalConfig[key]; ok {
		return val
	}

	return ""
}

// GetServiceConfig gets a service-specific config value
func (c *Client) GetServiceConfig(key string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if val, ok := c.serviceConfig[key]; ok {
		return val
	}

	return ""
}

// GetGlobalConfig gets a global config value
func (c *Client) GetGlobalConfig(key string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if val, ok := c.globalConfig[key]; ok {
		return val
	}

	return ""
}

// Log sends a log entry to Anox
func (c *Client) Log(ctx context.Context, message string, level api.LogLevel, action string, display bool) {
	entry := api.LogEntry{
		Time:     time.Now(),
		Service:  c.serviceName,
		Instance: c.instanceID,
		Level:    level,
		Action:   action,
		Message:  message,
		Context:  extractContext(ctx),
	}

	c.logger.Log(entry, display)
}

// LogWithStack sends a log entry with stack trace
func (c *Client) LogWithStack(ctx context.Context, message string, level api.LogLevel, action string, stacks []string, display bool) {
	entry := api.LogEntry{
		Time:     time.Now(),
		Service:  c.serviceName,
		Instance: c.instanceID,
		Level:    level,
		Action:   action,
		Message:  message,
		Stacks:   stacks,
		Context:  extractContext(ctx),
	}

	c.logger.Log(entry, display)
}

// Logger returns the logger instance
func (c *Client) Logger() *Logger {
	return c.logger
}

// SendHeartbeat immediately reports CPU and memory metrics to Anox.
func (c *Client) SendHeartbeat() error {
	return c.sendHeartbeat()
}

func (c *Client) connect() error {
	u, err := url.Parse(c.anoxURL)
	if err != nil {
		return err
	}

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return err
	}

	c.mu.Lock()
	c.conn = conn
	c.mu.Unlock()

	return nil
}

// writeJSON sends a JSON message on the websocket.
// Gorilla websocket does not allow concurrent writers.
func (c *Client) writeJSON(v interface{}) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()

	if conn == nil {
		return fmt.Errorf("not connected")
	}

	return conn.WriteJSON(v)
}

func (c *Client) register() error {
	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()
	if conn == nil {
		return fmt.Errorf("not connected")
	}

	done := make(chan error, 1)
	c.mu.Lock()
	c.registerDone = done
	c.mu.Unlock()

	registerMsg := api.RegisterMessage{
		Type:        "register",
		ServiceName: c.serviceName,
		HttpHost:    c.httpHost,
		HttpPort:    c.httpPort,
	}

	if err := c.writeJSON(registerMsg); err != nil {
		c.mu.Lock()
		c.registerDone = nil
		c.mu.Unlock()
		return err
	}

	select {
	case err := <-done:
		return err
	case <-time.After(registerTimeout):
		c.mu.Lock()
		c.registerDone = nil
		c.mu.Unlock()
		return fmt.Errorf("register timeout")
	}
}

func (c *Client) handleRegisterResponse(msg map[string]interface{}) {
	var regErr error

	success, _ := msg["success"].(bool)
	if !success {
		message, _ := msg["message"].(string)
		regErr = fmt.Errorf("registration failed: %s", message)
	} else {
		instanceID, _ := msg["instance_id"].(string)
		c.mu.Lock()
		c.instanceID = instanceID
		c.mu.Unlock()
		log.Printf("[Anox SDK] Registered as %s/%s", c.serviceName, c.instanceID)
	}

	c.mu.Lock()
	done := c.registerDone
	c.registerDone = nil
	c.mu.Unlock()

	if done != nil {
		done <- regErr
	}
}

func (c *Client) heartbeatLoop() {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := c.sendHeartbeat(); err != nil {
				log.Printf("[Anox SDK] Heartbeat failed: %v", err)
				go c.reconnect()
			}
		case <-c.heartbeatStop:
			return
		}
	}
}

func (c *Client) sendHeartbeat() error {
	cpuPercent, memTotal, memAvail := getSystemMetrics()

	c.mu.RLock()
	globalVersion := c.globalVersion
	serviceVersion := c.serviceVersion
	c.mu.RUnlock()

	heartbeat := api.HeartbeatMessage{
		Type:           "ping",
		CPUCores:       runtime.NumCPU(),
		CPUPercent:     cpuPercent,
		MemoryTotalMB:  memTotal,
		MemoryAvailMB:  memAvail,
		GlobalVersion:  globalVersion,
		ServiceVersion: serviceVersion,
	}

	return c.writeJSON(heartbeat)
}

func (c *Client) messageLoop() {
	for {
		select {
		case <-c.heartbeatStop:
			return
		default:
		}

		c.mu.RLock()
		conn := c.conn
		c.mu.RUnlock()

		if conn == nil {
			time.Sleep(time.Second)
			continue
		}

		_, message, err := conn.ReadMessage()
		if err != nil {
			if !c.isClosed() {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("[Anox SDK] Connection lost: %v", err)
				}
				c.clearConnection()
				go c.reconnect()
			}
			time.Sleep(time.Second)
			continue
		}

		var msg map[string]interface{}
		if err := json.Unmarshal(message, &msg); err != nil {
			continue
		}

		msgType, ok := msg["type"].(string)
		if !ok {
			continue
		}

		switch msgType {
		case "register_response":
			c.handleRegisterResponse(msg)
		case "pong":
			c.handlePong(msg)
		case "config_update":
			c.handleConfigUpdate(msg)
		case "config_response":
			c.handleConfigResponse(msg)
		}
	}
}

func (c *Client) isClosed() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.closed
}

func (c *Client) clearConnection() {
	c.mu.Lock()
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
	c.mu.Unlock()
}

func (c *Client) reconnect() {
	c.reconnectMu.Lock()
	defer c.reconnectMu.Unlock()

	if c.isClosed() {
		return
	}

	c.clearConnection()

	delay := 2 * time.Second
	for {
		if c.isClosed() {
			return
		}

		if err := c.connect(); err != nil {
			log.Printf("[Anox SDK] Reconnect failed: %v, retry in %v", err, delay)
			time.Sleep(delay)
			delay = minDuration(delay*2, maxReconnectDelay)
			continue
		}

		if err := c.register(); err != nil {
			log.Printf("[Anox SDK] Re-register failed: %v, retry in %v", err, delay)
			c.clearConnection()
			time.Sleep(delay)
			delay = minDuration(delay*2, maxReconnectDelay)
			continue
		}

		log.Printf("[Anox SDK] Reconnected successfully")
		c.pullInitialConfigs()
		if err := c.sendHeartbeat(); err != nil {
			log.Printf("[Anox SDK] Initial heartbeat after reconnect failed: %v", err)
		}
		if c.logger != nil {
			go c.logger.Flush()
		}
		return
	}
}

func minDuration(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}

func (c *Client) pullInitialConfigs() {
	c.pullConfig("_global")
	c.pullConfig(c.serviceName)
}

func (c *Client) handlePong(msg map[string]interface{}) {
	needUpdateGlobal, _ := msg["need_update_global"].(bool)
	needUpdateService, _ := msg["need_update_service"].(bool)

	if needUpdateGlobal {
		c.pullConfig("_global")
	}
	if needUpdateService {
		c.pullConfig(c.serviceName)
	}
}

func (c *Client) handleConfigUpdate(msg map[string]interface{}) {
	service, ok := msg["service"].(string)
	if !ok {
		return
	}

	c.pullConfig(service)
}

func (c *Client) handleConfigResponse(msg map[string]interface{}) {
	service, _ := msg["service"].(string)
	version, _ := msg["version"].(float64)
	values, _ := msg["values"].(map[string]interface{})

	c.mu.Lock()
	defer c.mu.Unlock()

	if service == "_global" {
		c.globalVersion = int64(version)
		c.globalConfig = make(map[string]string)
		for k, v := range values {
			if s, ok := v.(string); ok {
				c.globalConfig[k] = s
			}
		}
	} else if service == c.serviceName {
		c.serviceVersion = int64(version)
		c.serviceConfig = make(map[string]string)
		for k, v := range values {
			if s, ok := v.(string); ok {
				c.serviceConfig[k] = s
			}
		}
	}

	log.Printf("[Anox SDK] Config updated: %s (version: %d)", service, int64(version))
}

func (c *Client) pullConfig(service string) {
	msg := api.ConfigPullRequest{
		Type:    "pull_config",
		Service: service,
	}

	if err := c.writeJSON(msg); err != nil {
		log.Printf("[Anox SDK] Failed to pull config: %v", err)
	}
}

func extractContext(ctx context.Context) map[string]string {
	result := make(map[string]string)

	if ctx != nil {
		if traceID, ok := ctx.Value("trace_id").(string); ok {
			result["trace_id"] = traceID
		}
		if userID, ok := ctx.Value("user_id").(string); ok {
			result["user_id"] = userID
		}
		if ip, ok := ctx.Value("ip").(string); ok {
			result["ip"] = ip
		}
	}

	return result
}

// GetServiceName returns the service name
func (c *Client) GetServiceName() string {
	return c.serviceName
}

// GetInstanceID returns the instance ID
func (c *Client) GetInstanceID() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.instanceID
}

// SetTraceID sets the trace ID in context
func SetTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, "trace_id", traceID)
}

// SetUserID sets the user ID in context
func SetUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, "user_id", userID)
}

// SetIP sets the IP in context
func SetIP(ctx context.Context, ip string) context.Context {
	return context.WithValue(ctx, "ip", ip)
}

// anoxDebug prints debug log to console (only if ANOX_DEBUG env is set)
func anoxDebug(format string, args ...interface{}) {
	if os.Getenv("ANOX_DEBUG") != "" {
		log.Printf("[Anox SDK DEBUG] "+format, args...)
	}
}
