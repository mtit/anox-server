package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"anox/api"
	"anox/internal/core"
	"anox/internal/logcenter"
	"anox/internal/registry"
	"anox/internal/server"
)

func main() {
	log.Println("[Anox] Starting Anox Server...")

	// Initialize config store
	configStore, err := core.NewConfigStore("data/configs", nil)
	if err != nil {
		log.Fatalf("[Anox] Failed to initialize config store: %v", err)
	}

	// Ensure default server settings exist in anox.json
	if err := configStore.EnsureAnoxSettingsDefaults(); err != nil {
		log.Printf("[Anox] Warning: failed to ensure anox settings defaults: %v", err)
	}

	// Sync environment variables with config
	if err := configStore.SyncEnvWithConfig(); err != nil {
		log.Printf("[Anox] Warning: failed to sync env with config: %v", err)
	}

	// Get Anox settings
	anoxSettings, err := configStore.GetAnoxSettings()
	if err != nil {
		log.Fatalf("[Anox] Failed to get settings: %v", err)
	}

	log.Printf("[Anox] Configuration loaded. Host: %s, Port: %s", anoxSettings.Host, anoxSettings.Port)

	// Initialize registry manager
	registryMgr := registry.NewManager(func(instance *api.ServiceInstance) {
		log.Printf("[Anox] Instance disconnected: %s/%s", instance.ServiceName, instance.ID)
	})

	// Initialize log storage
	logStorage := logcenter.NewStorage("logs")

	// Initialize alert engine
	alertEngine := logcenter.NewAlertEngine()
	
	// Load alert config
	alertConfig, _ := configStore.GetAlertConfig()
	alertEngine.UpdateConfig(alertConfig)
	log.Printf("[Anox] Alert configuration loaded: enabled=%v channels=%v",
		alertConfig.Enabled, alertConfig.Channels)

	// Initialize log collector
	logCollector := logcenter.NewCollector(logStorage, alertEngine)

	// Initialize hot reload manager
	hotReload, err := core.NewHotReloadManager(configStore, func() {
		log.Println("[Anox] Global configuration hot reloaded")
	})
	if err != nil {
		log.Fatalf("[Anox] Failed to initialize hot reload: %v", err)
	}
	
	if err := hotReload.Start(); err != nil {
		log.Printf("[Anox] Warning: failed to start hot reload: %v", err)
	}

	// Create WebSocket server
	wsServer := server.NewWSServer(configStore, registryMgr, logCollector)

	// Create HTTP server
	httpServer := server.NewHTTPServer(
		configStore,
		registryMgr,
		logStorage,
		logCollector,
		alertEngine,
		anoxSettings,
	)

	// Mount WebSocket routes
	wsServer.MountRoutes(httpServer.GetRouter())

	// Set up config change callback
	configStore.OnChange = func(service string, config *api.Config) {
		wsServer.NotifyConfigChange(service, config)
	}

	// Handle graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Start HTTP server in a goroutine
	addr := anoxSettings.Host + ":" + anoxSettings.Port
	go func() {
		log.Printf("[Anox] Server listening on %s", addr)
		if err := httpServer.Run(addr); err != nil {
			log.Fatalf("[Anox] Server error: %v", err)
		}
	}()

	// Wait for shutdown signal
	<-sigCh
	log.Println("[Anox] Shutting down...")

	// Cleanup
	hotReload.Stop()
	logCollector.Stop()

	log.Println("[Anox] Server stopped")
}
