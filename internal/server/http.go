package server

import (
	"net/http"
	"os"
	"time"

	"anox/api"
	"anox/internal/core"
	"anox/internal/logcenter"
	"anox/internal/registry"
	"github.com/gin-gonic/gin"
)

// Server handles HTTP API requests
type HTTPServer struct {
	router       *gin.Engine
	configStore  *core.ConfigStore
	registryMgr  *registry.Manager
	logStorage   *logcenter.Storage
	logCollector *logcenter.Collector
	alertEngine  *logcenter.AlertEngine
	anoxSettings *core.AnoxSettings
}

// NewHTTPServer creates a new HTTP server
func NewHTTPServer(
	configStore *core.ConfigStore,
	registryMgr *registry.Manager,
	logStorage *logcenter.Storage,
	logCollector *logcenter.Collector,
	alertEngine *logcenter.AlertEngine,
	anoxSettings *core.AnoxSettings,
) *HTTPServer {
	gin.SetMode(gin.ReleaseMode)

	s := &HTTPServer{
		router:       gin.Default(),
		configStore:  configStore,
		registryMgr:  registryMgr,
		logStorage:   logStorage,
		logCollector: logCollector,
		alertEngine:  alertEngine,
		anoxSettings: anoxSettings,
	}

	s.setupRoutes()
	return s
}

func (s *HTTPServer) setupRoutes() {
	// CORS middleware
	s.router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// API routes
	apiGroup := s.router.Group("/api")
	{
		apiGroup.POST("/login", s.handleLogin)

		protected := apiGroup.Group("")
		protected.Use(s.authMiddleware())
		{
			protected.GET("/overview", s.handleOverview)
			protected.GET("/system-metrics", s.handleSystemMetrics)
			protected.GET("/services", s.handleListServices)
			protected.GET("/services/:name", s.handleGetService)
			protected.GET("/configs", s.handleListConfigs)
			protected.GET("/configs/:name", s.handleGetConfig)
			protected.PUT("/configs/:name", s.handleUpdateConfig)
			protected.DELETE("/configs/:name/keys/:key", s.handleDeleteConfigKey)
			protected.GET("/logs/services", s.handleLogServices)
			protected.GET("/logs/instances", s.handleLogInstances)
			protected.GET("/logs/dates", s.handleLogDates)
			protected.GET("/logs/hours", s.handleLogHours)
			protected.POST("/logs/search", s.handleSearchLogs)
			protected.GET("/alerts/config", s.handleGetAlertConfig)
			protected.PUT("/alerts/config", s.handleUpdateAlertConfig)
		}
	}

	// Static frontend assets
	s.router.Static("/assets", "./web/dist/assets")
	s.router.StaticFile("/logo.png", "./web/dist/logo.png")

	// SPA history mode: unmatched GET requests return index.html
	s.router.NoRoute(func(c *gin.Context) {
		if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}

		// Serve root-level static files from dist (favicon, etc.)
		if c.Request.URL.Path != "/" {
			filePath := "./web/dist" + c.Request.URL.Path
			if info, err := os.Stat(filePath); err == nil && !info.IsDir() {
				c.File(filePath)
				return
			}
		}

		c.File("./web/dist/index.html")
	})
}

// Run starts the HTTP server
func (s *HTTPServer) Run(addr string) error {
	return s.router.Run(addr)
}

// Router returns the gin router for WebSocket mounting
func (s *HTTPServer) Router() *gin.Engine {
	return s.router
}

func (s *HTTPServer) handleOverview(c *gin.Context) {
	// Get system info
	cpuCores, memTotal := getSystemInfo()

	c.JSON(200, gin.H{
		"service_count":  s.registryMgr.GetTotalServiceCount(),
		"instance_count": s.registryMgr.GetTotalInstanceCount(),
		"cpu_cores":      cpuCores,
		"memory_mb":      memTotal,
		"host":           s.anoxSettings.Host,
		"port":           s.anoxSettings.Port,
	})
}

func (s *HTTPServer) handleSystemMetrics(c *gin.Context) {
	cpuPercent, memUsed, memTotal := getSystemMetrics()

	c.JSON(200, gin.H{
		"cpu_percent":     cpuPercent,
		"memory_used_mb":  memUsed,
		"memory_total_mb": memTotal,
		"timestamp":       time.Now().Unix(),
	})
}

func (s *HTTPServer) handleListServices(c *gin.Context) {
	services := s.registryMgr.GetAllServices()
	c.JSON(200, gin.H{"services": services})
}

func (s *HTTPServer) handleGetService(c *gin.Context) {
	name := c.Param("name")
	nodes := s.registryMgr.GetServiceNodes(name)

	if len(nodes) == 0 {
		c.JSON(404, gin.H{"error": "service not found"})
		return
	}

	var instances []*api.ServiceInstance
	for _, node := range nodes {
		instances = append(instances, node.ServiceInstance)
	}

	c.JSON(200, gin.H{
		"name":      name,
		"instances": instances,
	})
}

func (s *HTTPServer) handleListConfigs(c *gin.Context) {
	configs := s.configStore.GetAll()
	c.JSON(200, gin.H{"configs": configs})
}

func (s *HTTPServer) handleGetConfig(c *gin.Context) {
	name := c.Param("name")
	config, err := s.configStore.Get(name)
	if err != nil {
		c.JSON(404, gin.H{"error": "config not found"})
		return
	}

	c.JSON(200, gin.H{"config": config})
}

func (s *HTTPServer) handleUpdateConfig(c *gin.Context) {
	name := c.Param("name")

	var req struct {
		Values map[string]string `json:"values"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "invalid request"})
		return
	}

	// Merge with existing config
	existing, err := s.configStore.Get(name)
	if err == nil {
		// Merge existing values
		for k, v := range existing.Values {
			if _, ok := req.Values[k]; !ok {
				req.Values[k] = v
			}
		}
	}

	if err := s.configStore.Set(name, req.Values); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"message": "config updated"})
}

func (s *HTTPServer) handleDeleteConfigKey(c *gin.Context) {
	name := c.Param("name")
	key := c.Param("key")

	if err := s.configStore.Delete(name, key); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"message": "key deleted"})
}

func (s *HTTPServer) handleLogServices(c *gin.Context) {
	services, err := s.logStorage.ListServices()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"services": services})
}

func (s *HTTPServer) handleLogInstances(c *gin.Context) {
	service := c.Query("service")
	if service == "" {
		c.JSON(400, gin.H{"error": "service parameter required"})
		return
	}

	instances, err := s.logStorage.ListInstances(service)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"instances": instances})
}

func (s *HTTPServer) handleLogDates(c *gin.Context) {
	service := c.Query("service")
	instance := c.Query("instance")

	if service == "" || instance == "" {
		c.JSON(400, gin.H{"error": "service and instance parameters required"})
		return
	}

	dates, err := s.logStorage.ListDates(service, instance)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"dates": dates})
}

func (s *HTTPServer) handleLogHours(c *gin.Context) {
	service := c.Query("service")
	instance := c.Query("instance")
	date := c.Query("date")

	if service == "" || instance == "" || date == "" {
		c.JSON(400, gin.H{"error": "service, instance and date parameters required"})
		return
	}

	hours, err := s.logStorage.ListHours(service, instance, date)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"hours": hours})
}

func (s *HTTPServer) handleSearchLogs(c *gin.Context) {
	var req struct {
		Service  string `json:"service"`
		Instance string `json:"instance"`
		Date     string `json:"date"`
		Hour     string `json:"hour"`
		Keyword  string `json:"keyword"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "invalid request"})
		return
	}

	if req.Service == "" || req.Instance == "" || req.Date == "" {
		c.JSON(400, gin.H{"error": "service, instance and date are required"})
		return
	}

	logs, err := s.logStorage.Search(req.Service, req.Instance, req.Date, req.Hour, req.Keyword)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"logs": logs})
}

func (s *HTTPServer) handleGetAlertConfig(c *gin.Context) {
	config, err := s.configStore.GetAlertConfig()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"config": config})
}

func (s *HTTPServer) handleUpdateAlertConfig(c *gin.Context) {
	var config api.AlertConfig

	if err := c.BindJSON(&config); err != nil {
		c.JSON(400, gin.H{"error": "invalid request"})
		return
	}

	if err := s.configStore.SaveAlertConfig(&config); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	// Update alert engine
	s.alertEngine.UpdateConfig(&config)

	c.JSON(200, gin.H{"message": "alert config updated"})
}

// GetRouter returns the gin router
func (s *HTTPServer) GetRouter() *gin.Engine {
	return s.router
}
