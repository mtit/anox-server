package registry

import (
	"crypto/rand"
	"fmt"
	"sync"
	"time"

	"anox/api"
	"github.com/gorilla/websocket"
)

// Node represents a single service instance node
type Node struct {
	*api.ServiceInstance
	conn *websocket.Conn
	mu   sync.RWMutex
}

// NewNode creates a new node
func NewNode(serviceName string, conn *websocket.Conn, httpHost, httpPort string) *Node {
	return &Node{
		ServiceInstance: &api.ServiceInstance{
			ID:            generateInstanceID(serviceName),
			ServiceName:   serviceName,
			RegisteredAt:  time.Now(),
			LastHeartbeat: time.Now(),
			HttpHost:      httpHost,
			HttpPort:      httpPort,
			Conn:          conn,
		},
		conn: conn,
	}
}

// UpdateHeartbeat updates the node's heartbeat with metrics
func (n *Node) UpdateHeartbeat(heartbeat *api.HeartbeatMessage) {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.LastHeartbeat = time.Now()
	n.CPUCores = heartbeat.CPUCores
	n.CPUPercent = heartbeat.CPUPercent
	n.MemoryTotalMB = heartbeat.MemoryTotalMB
	n.MemoryAvailMB = heartbeat.MemoryAvailMB
	n.GlobalVersion = heartbeat.GlobalVersion
	n.ServiceVersion = heartbeat.ServiceVersion
}

// GetConn returns the websocket connection
func (n *Node) GetConn() *websocket.Conn {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.conn
}

// SetConn sets the websocket connection
func (n *Node) SetConn(conn *websocket.Conn) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.conn = conn
}

// IsAlive checks if the node is still alive (heartbeat within 30 seconds)
func (n *Node) IsAlive() bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return time.Since(n.LastHeartbeat) < 30*time.Second
}

// SendMessage sends a message to the node via WebSocket
func (n *Node) SendMessage(msg interface{}) error {
	n.mu.RLock()
	conn := n.conn
	n.mu.RUnlock()

	if conn == nil {
		return fmt.Errorf("no connection available")
	}

	return conn.WriteJSON(msg)
}

// generateInstanceID generates a unique instance ID
func generateInstanceID(serviceName string) string {
	return fmt.Sprintf("%s-%s%s", serviceName, time.Now().Format("20060102150405"), randomSuffix(6))
}

func randomSuffix(length int) string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	buf := make([]byte, length)
	if _, err := rand.Read(buf); err != nil {
		now := time.Now().UnixNano()
		for i := range buf {
			buf[i] = byte(now >> (i * 8))
		}
	}

	for i, b := range buf {
		buf[i] = chars[int(b)%len(chars)]
	}
	return string(buf)
}
