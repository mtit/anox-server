package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"anox/api"
	"anox/internal/core"
	"anox/internal/logcenter"
	"anox/internal/registry"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// WSServer handles WebSocket connections
type WSServer struct {
	configStore  *core.ConfigStore
	registryMgr  *registry.Manager
	logCollector *logcenter.Collector
	connections  sync.Map // instanceID -> *Node
	watchers     sync.Map // conn -> struct{}
}

func NewWSServer(
	configStore *core.ConfigStore,
	registryMgr *registry.Manager,
	logCollector *logcenter.Collector,
) *WSServer {
	return &WSServer{
		configStore:  configStore,
		registryMgr:  registryMgr,
		logCollector: logCollector,
	}
}

func (ws *WSServer) MountRoutes(router *gin.Engine) {
	router.GET("/ws", ws.handleWebSocket)
}

func (ws *WSServer) handleWebSocket(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("[WebSocket] Upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	var node *registry.Node
	watchingServices := false
	defer func() {
		if watchingServices {
			ws.watchers.Delete(conn)
		}
	}()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[WebSocket] Read error: %v", err)
			}
			break
		}

		var msg map[string]interface{}
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("[WebSocket] Failed to parse message: %v", err)
			continue
		}

		msgType, ok := msg["type"].(string)
		if !ok {
			log.Printf("[WebSocket] Message without type field")
			continue
		}

		switch msgType {
		case "watch_services":
			watchingServices = ws.handleWatchServices(conn)
		case "register":
			node = ws.handleRegister(conn, msg)
		case "ping":
			ws.handlePing(conn, node, msg)
		case "log":
			ws.handleLog(conn, msg)
		case "logs_batch":
			ws.handleLogsBatch(conn, msg)
		case "pull_config":
			ws.handlePullConfig(conn, node, msg)
		default:
			log.Printf("[WebSocket] Unknown message type: %s", msgType)
		}
	}

	if node != nil {
		ws.registryMgr.Unregister(node.ID)
	}
}

func (ws *WSServer) handleWatchServices(conn *websocket.Conn) bool {
	ws.watchers.Store(conn, struct{}{})
	if err := conn.WriteJSON(api.ServicesSnapshotMessage{
		Type:     "services_snapshot",
		Services: ws.registryMgr.GetAllServices(),
	}); err != nil {
		log.Printf("[WebSocket] Failed to send services snapshot: %v", err)
		ws.watchers.Delete(conn)
		return false
	}
	return true
}

func (ws *WSServer) handleRegister(conn *websocket.Conn, msg map[string]interface{}) *registry.Node {
	serviceName, ok := msg["service_name"].(string)
	if !ok || serviceName == "" {
		ws.sendError(conn, "register", "missing service_name")
		return nil
	}

	node := ws.registryMgr.Register(serviceName, conn)
	ws.connections.Store(node.ID, node)

	if host := parseRegisterString(msg, "http_host"); host != "" {
		node.HttpHost = host
	}
	if port := parseRegisterString(msg, "http_port"); port != "" {
		node.HttpPort = port
	}

	response := api.RegisterResponse{Type: "register_response", InstanceID: node.ID, Success: true}
	if err := conn.WriteJSON(response); err != nil {
		log.Printf("[WebSocket] Failed to send register response: %v", err)
	}

	log.Printf("[WebSocket] Service registered: %s, Instance: %s, HTTP: %s:%s", serviceName, node.ID, node.HttpHost, node.HttpPort)
	return node
}

func parseRegisterString(msg map[string]interface{}, key string) string {
	v, ok := msg[key]
	if !ok || v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case float64:
		return fmt.Sprintf("%.0f", val)
	default:
		return fmt.Sprintf("%v", val)
	}
}

func (ws *WSServer) handlePing(conn *websocket.Conn, node *registry.Node, msg map[string]interface{}) {
	if node == nil {
		ws.sendError(conn, "ping", "not registered")
		return
	}

	var heartbeat api.HeartbeatMessage
	if v, ok := msg["cpu_cores"].(float64); ok {
		heartbeat.CPUCores = int(v)
	}
	if v, ok := msg["cpu_percent"].(float64); ok {
		heartbeat.CPUPercent = v
	}
	if v, ok := msg["memory_total_mb"].(float64); ok {
		heartbeat.MemoryTotalMB = int64(v)
	}
	if v, ok := msg["memory_avail_mb"].(float64); ok {
		heartbeat.MemoryAvailMB = int64(v)
	}
	if v, ok := msg["global_version"].(float64); ok {
		heartbeat.GlobalVersion = int64(v)
	}
	if v, ok := msg["service_version"].(float64); ok {
		heartbeat.ServiceVersion = int64(v)
	}

	node.UpdateHeartbeat(&heartbeat)
	ws.NotifyServiceChange("upsert", node.ServiceInstance, len(ws.registryMgr.GetServiceNodes(node.ServiceName)))

	response := api.HeartbeatResponse{Type: "pong"}
	globalConfig, err := ws.configStore.Get(core.GlobalConfig)
	if err == nil {
		response.GlobalVersion = globalConfig.Version
		response.NeedUpdateGlobal = globalConfig.Version != heartbeat.GlobalVersion
	}
	serviceConfig, err := ws.configStore.Get(node.ServiceName)
	if err == nil {
		response.ServiceVersion = serviceConfig.Version
		response.NeedUpdateService = serviceConfig.Version != heartbeat.ServiceVersion
	}

	if err := conn.WriteJSON(response); err != nil {
		log.Printf("[WebSocket] Failed to send ping response: %v", err)
	}
}

func (ws *WSServer) handleLog(conn *websocket.Conn, msg map[string]interface{}) {
	entry := parseLogEntry(msg)
	ws.logCollector.Submit(&entry)
}

func (ws *WSServer) handleLogsBatch(conn *websocket.Conn, msg map[string]interface{}) {
	logsRaw, ok := msg["logs"].([]interface{})
	if !ok {
		return
	}
	for _, logRaw := range logsRaw {
		logMap, ok := logRaw.(map[string]interface{})
		if !ok {
			continue
		}
		entry := parseLogEntry(logMap)
		ws.logCollector.Submit(&entry)
	}
}

func parseLogEntry(msg map[string]interface{}) api.LogEntry {
	var entry api.LogEntry
	entry.Time = parseLogTime(msg["time"])
	if v, ok := msg["service"].(string); ok {
		entry.Service = v
	}
	if v, ok := msg["instance"].(string); ok {
		entry.Instance = v
	}
	if v, ok := msg["level"].(string); ok {
		entry.Level = api.LogLevel(v)
	}
	if v, ok := msg["action"].(string); ok {
		entry.Action = v
	}
	if v, ok := msg["message"].(string); ok {
		entry.Message = v
	}
	if v, ok := msg["trace_id"].(string); ok {
		entry.TraceID = v
	}
	if v, ok := msg["context"].(map[string]interface{}); ok {
		entry.Context = make(map[string]string)
		for k, val := range v {
			if s, ok := val.(string); ok {
				entry.Context[k] = s
			}
		}
	}
	if v, ok := msg["stacks"].([]interface{}); ok {
		for _, stack := range v {
			if s, ok := stack.(string); ok {
				entry.Stacks = append(entry.Stacks, s)
			}
		}
	}
	return entry
}

func parseLogTime(v interface{}) time.Time {
	if s, ok := v.(string); ok && s != "" {
		if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
			return t
		}
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			return t
		}
	}
	return time.Now()
}

func (ws *WSServer) handlePullConfig(conn *websocket.Conn, node *registry.Node, msg map[string]interface{}) {
	service, ok := msg["service"].(string)
	if !ok || service == "" {
		ws.sendError(conn, "pull_config", "missing service")
		return
	}

	config, err := ws.configStore.Get(service)
	if err != nil {
		ws.sendError(conn, "pull_config", err.Error())
		return
	}

	response := api.ConfigPullResponse{Type: "config_response", Service: service, Version: config.Version, Values: config.Values}
	if err := conn.WriteJSON(response); err != nil {
		log.Printf("[WebSocket] Failed to send config response: %v", err)
	}
}

func (ws *WSServer) sendError(conn *websocket.Conn, msgType, errorMsg string) {
	response := map[string]interface{}{"type": msgType + "_error", "error": errorMsg, "success": false}
	if err := conn.WriteJSON(response); err != nil {
		log.Printf("[WebSocket] Failed to send error response: %v", err)
	}
}

func (ws *WSServer) NotifyConfigChange(service string, config *api.Config) {
	message := api.ConfigUpdateMessage{Type: "config_update", Service: service, Version: config.Version}
	if service == core.GlobalConfig {
		ws.registryMgr.BroadcastToAll(message)
	} else {
		ws.registryMgr.BroadcastToService(service, message)
	}
}

func (ws *WSServer) NotifyServiceChange(event string, instance *api.ServiceInstance, instanceCount int) {
	msg := api.ServiceEventMessage{Type: "service_event", Event: event, Service: instance.ServiceName, Instance: instance, Instances: instanceCount}
	ws.watchers.Range(func(key, value interface{}) bool {
		conn, ok := key.(*websocket.Conn)
		if !ok {
			return true
		}
		if err := conn.WriteJSON(msg); err != nil {
			log.Printf("[WebSocket] Failed to notify watcher: %v", err)
			ws.watchers.Delete(conn)
			conn.Close()
		}
		return true
	})
}