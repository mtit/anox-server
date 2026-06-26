package core

import (
	"log"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
)

// HotReloadManager handles hot reloading of Anox configuration
type HotReloadManager struct {
	configStore *ConfigStore
	watcher     *fsnotify.Watcher
	stopCh      chan struct{}
	onReload    func()
}

// NewHotReloadManager creates a new hot reload manager
func NewHotReloadManager(configStore *ConfigStore, onReload func()) (*HotReloadManager, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	return &HotReloadManager{
		configStore: configStore,
		watcher:     watcher,
		stopCh:      make(chan struct{}),
		onReload:    onReload,
	}, nil
}

// Start begins watching for config changes
func (hrm *HotReloadManager) Start() error {
	// Watch the config directory
	configPath := filepath.Join(hrm.configStore.dataDir, "_global.json")
	
	// Ensure the file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Create default global config if not exists
		hrm.configStore.Set(GlobalConfig, map[string]string{
			"log_level": "info",
		})
	}
	
	// Watch the directory to catch file creations
	if err := hrm.watcher.Add(hrm.configStore.dataDir); err != nil {
		return err
	}

	go hrm.watch()
	return nil
}

// Stop stops the hot reload manager
func (hrm *HotReloadManager) Stop() {
	close(hrm.stopCh)
	hrm.watcher.Close()
}

func (hrm *HotReloadManager) watch() {
	for {
		select {
		case event, ok := <-hrm.watcher.Events:
			if !ok {
				return
			}
			
			// Check if _global.json was modified
			if filepath.Base(event.Name) == "_global.json" {
				if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) {
					log.Println("[HotReload] Global config changed, reloading...")
					hrm.handleGlobalConfigChange()
				}
			}
			
		case err, ok := <-hrm.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("[HotReload] Watcher error: %v", err)
			
		case <-hrm.stopCh:
			return
		}
	}
}

func (hrm *HotReloadManager) handleGlobalConfigChange() {
	// Reload the global config
	hrm.configStore.mu.Lock()
	defer hrm.configStore.mu.Unlock()
	
	// Reload from disk
	service := GlobalConfig
	path := hrm.configStore.filePath(service)
	
	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return
	}
	
	// Note: In a production system, you might want to validate
	// the config before applying changes. For now, we just notify.
	
	// Call the onReload callback if provided
	if hrm.onReload != nil {
		hrm.onReload()
	}
}
