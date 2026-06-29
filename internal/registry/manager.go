package registry

import (
	"log"
	"sync"
	"time"

	"anox/api"
	"github.com/gorilla/websocket"
)

// Manager manages all registered service instances
type Manager struct {
	mu           sync.RWMutex
	services     map[string]map[string]*Node // service name -> instance id -> node
	onDisconnect func(instance *api.ServiceInstance)
	onChange     func(event string, instance *api.ServiceInstance, instanceCount int)
}

// NewManager creates a new registry manager
func NewManager(onDisconnect func(instance *api.ServiceInstance)) *Manager {
	m := &Manager{
		services:     make(map[string]map[string]*Node),
		onDisconnect: onDisconnect,
	}

	go m.cleanupLoop()
	return m
}

func (m *Manager) SetOnChange(fn func(event string, instance *api.ServiceInstance, instanceCount int)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onChange = fn
}

// Register registers a new service instance
func (m *Manager) Register(serviceName string, conn *websocket.Conn, httpHost, httpPort string) *Node {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.services[serviceName]; !ok {
		m.services[serviceName] = make(map[string]*Node)
	}

	node := NewNode(serviceName, conn, httpHost, httpPort)
	m.services[serviceName][node.ID] = node
	instanceCount := len(m.services[serviceName])

	log.Printf("[Registry] Service registered: %s, Instance: %s", serviceName, node.ID)
	if m.onChange != nil {
		go m.onChange("upsert", node.ServiceInstance, instanceCount)
	}

	return node
}

// Unregister removes a service instance
func (m *Manager) Unregister(instanceID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for serviceName, instances := range m.services {
		if node, ok := instances[instanceID]; ok {
			delete(instances, instanceID)
			instanceCount := len(instances)

			if len(instances) == 0 {
				delete(m.services, serviceName)
			}

			log.Printf("[Registry] Service unregistered: %s, Instance: %s", serviceName, instanceID)

			if m.onDisconnect != nil {
				go m.onDisconnect(node.ServiceInstance)
			}
			if m.onChange != nil {
				go m.onChange("remove", node.ServiceInstance, instanceCount)
			}
			return
		}
	}
}

// GetNode returns a node by instance ID
func (m *Manager) GetNode(instanceID string) *Node {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, instances := range m.services {
		if node, ok := instances[instanceID]; ok {
			return node
		}
	}
	return nil
}

// GetServiceNodes returns all nodes for a service
func (m *Manager) GetServiceNodes(serviceName string) []*Node {
	m.mu.RLock()
	defer m.mu.RUnlock()

	instances, ok := m.services[serviceName]
	if !ok {
		return nil
	}

	nodes := make([]*Node, 0, len(instances))
	for _, node := range instances {
		nodes = append(nodes, node)
	}
	return nodes
}

// GetAllServices returns all service summaries
func (m *Manager) GetAllServices() []*api.ServiceSummary {
	m.mu.RLock()
	defer m.mu.RUnlock()

	summaries := make([]*api.ServiceSummary, 0, len(m.services))
	for serviceName, instances := range m.services {
		serviceNodes := make([]*api.ServiceInstance, 0, len(instances))
		for _, node := range instances {
			serviceNodes = append(serviceNodes, node.ServiceInstance)
		}
		summaries = append(summaries, &api.ServiceSummary{
			Name:          serviceName,
			InstanceCount: len(instances),
			Instances:     serviceNodes,
		})
	}
	return summaries
}

// GetAllInstances returns all instances across all services
func (m *Manager) GetAllInstances() []*api.ServiceInstance {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var instances []*api.ServiceInstance
	for _, serviceInstances := range m.services {
		for _, node := range serviceInstances {
			instances = append(instances, node.ServiceInstance)
		}
	}
	return instances
}

// GetTotalServiceCount returns total number of services
func (m *Manager) GetTotalServiceCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.services)
}

// GetTotalInstanceCount returns total number of instances
func (m *Manager) GetTotalInstanceCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	count := 0
	for _, instances := range m.services {
		count += len(instances)
	}
	return count
}

// BroadcastToService sends a message to all instances of a service
func (m *Manager) BroadcastToService(serviceName string, msg interface{}) {
	nodes := m.GetServiceNodes(serviceName)
	for _, node := range nodes {
		if err := node.SendMessage(msg); err != nil {
			log.Printf("[Registry] Failed to send message to %s: %v", node.ID, err)
		}
	}
}

// BroadcastToAll sends a message to all connected instances
func (m *Manager) BroadcastToAll(msg interface{}) {
	m.mu.RLock()
	services := make(map[string]map[string]*Node)
	for k, v := range m.services {
		services[k] = v
	}
	m.mu.RUnlock()

	for serviceName, instances := range services {
		for _, node := range instances {
			if err := node.SendMessage(msg); err != nil {
				log.Printf("[Registry] Failed to broadcast to %s/%s: %v", serviceName, node.ID, err)
			}
		}
	}
}

func (m *Manager) cleanupLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		m.removeDeadInstances()
	}
}

func (m *Manager) removeDeadInstances() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for serviceName, instances := range m.services {
		for id, node := range instances {
			if !node.IsAlive() {
				delete(instances, id)
				instanceCount := len(instances)
				log.Printf("[Registry] Removed dead instance: %s/%s", serviceName, id)
				if m.onDisconnect != nil {
					go m.onDisconnect(node.ServiceInstance)
				}
				if m.onChange != nil {
					go m.onChange("remove", node.ServiceInstance, instanceCount)
				}
			}
		}
		if len(instances) == 0 {
			delete(m.services, serviceName)
		}
	}
}
