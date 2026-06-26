package logcenter

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"anox/api"
)

const (
	LogsDir = "logs"
)

// Storage handles log file storage
type Storage struct {
	mu          sync.RWMutex
	fileHandles map[string]*logFileHandle // path -> handle
	maxHandles  int
	logsDir     string
}

type logFileHandle struct {
	file      *os.File
	writer    *bufio.Writer
	lastUsed  time.Time
	mu        sync.Mutex
}

// NewStorage creates a new log storage
func NewStorage(logsDir string) *Storage {
	if logsDir == "" {
		logsDir = LogsDir
	}
	
	s := &Storage{
		fileHandles: make(map[string]*logFileHandle),
		maxHandles:  100,
		logsDir:     logsDir,
	}
	
	// Ensure logs directory exists
	os.MkdirAll(logsDir, 0755)
	
	// Start cleanup goroutine
	go s.cleanupLoop()
	
	return s
}

// Store stores a log entry to file
func (s *Storage) Store(entry *api.LogEntry) error {
	if entry.Time.IsZero() {
		entry.Time = time.Now()
	}

	path := s.getLogPath(entry)
	
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}
	
	// Get or create file handle
	handle := s.getHandle(path)
	
	// Write log entry
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal log entry: %w", err)
	}
	
	handle.mu.Lock()
	defer handle.mu.Unlock()
	
	if _, err := handle.writer.Write(data); err != nil {
		return fmt.Errorf("failed to write log entry: %w", err)
	}
	
	if err := handle.writer.WriteByte('\n'); err != nil {
		return fmt.Errorf("failed to write newline: %w", err)
	}
	
	// Flush to ensure durability
	if err := handle.writer.Flush(); err != nil {
		return fmt.Errorf("failed to flush log entry: %w", err)
	}
	
	handle.lastUsed = time.Now()
	
	return nil
}

// Search searches logs by criteria
func (s *Storage) Search(service, instance, date, hour, keyword string) ([]*api.LogEntry, error) {
	var results []*api.LogEntry
	
	// Build search path
	var searchPaths []string
	
	if service == "" {
		// Search all services
		services, err := s.listServices()
		if err != nil {
			return nil, err
		}
		for _, svc := range services {
			paths := s.buildSearchPaths(svc, instance, date, hour)
			searchPaths = append(searchPaths, paths...)
		}
	} else {
		searchPaths = s.buildSearchPaths(service, instance, date, hour)
	}
	
	// Search each file
	for _, path := range searchPaths {
		entries, err := s.searchFile(path, keyword)
		if err != nil {
			// Log error but continue searching other files
			continue
		}
		results = append(results, entries...)
	}
	
	return results, nil
}

// ListServices returns all services that have log directories
func (s *Storage) ListServices() ([]string, error) {
	return s.listServices()
}

// ListInstances returns all instances for a service
func (s *Storage) ListInstances(service string) ([]string, error) {
	return s.listInstances(service)
}

// ListDates returns all dates for a service and instance
func (s *Storage) ListDates(service, instance string) ([]string, error) {
	return s.listDates(service, instance)
}

// ListHours returns all hours for a service, instance and date
func (s *Storage) ListHours(service, instance, date string) ([]string, error) {
	return s.listHours(service, instance, date)
}

func (s *Storage) getLogPath(entry *api.LogEntry) string {
	// Format: logs/{service}/{instance}/{date}/{hour}.log
	date := entry.Time.Format("2006-01-02")
	hour := fmt.Sprintf("%02d", entry.Time.Hour())
	
	return filepath.Join(s.logsDir, entry.Service, entry.Instance, date, hour+".log")
}

func (s *Storage) buildSearchPaths(service, instance, date, hour string) []string {
	var paths []string
	
	// Build base path
	base := filepath.Join(s.logsDir, service)
	
	if instance == "" {
		// Search all instances
		instances, _ := s.listInstances(service)
		for _, inst := range instances {
			paths = append(paths, s.buildSearchPathsForInstance(base, inst, date, hour)...)
		}
	} else {
		paths = s.buildSearchPathsForInstance(base, instance, date, hour)
	}
	
	return paths
}

func (s *Storage) buildSearchPathsForInstance(base, instance, date, hour string) []string {
	var paths []string
	
	instancePath := filepath.Join(base, instance)
	
	if date == "" {
		// Search all dates
		dates, _ := s.listDatesForPath(instancePath)
		for _, d := range dates {
			paths = append(paths, s.buildSearchPathsForDate(instancePath, d, hour)...)
		}
	} else {
		paths = s.buildSearchPathsForDate(instancePath, date, hour)
	}
	
	return paths
}

func (s *Storage) buildSearchPathsForDate(instancePath, date, hour string) []string {
	var paths []string
	
	datePath := filepath.Join(instancePath, date)
	
	if hour == "" {
		// Search all hours
		hours, _ := s.listHoursForPath(datePath)
		for _, h := range hours {
			paths = append(paths, filepath.Join(datePath, h+".log"))
		}
	} else {
		paths = append(paths, filepath.Join(datePath, hour+".log"))
	}
	
	return paths
}

func (s *Storage) getHandle(path string) *logFileHandle {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Return existing handle if available
	if handle, ok := s.fileHandles[path]; ok {
		handle.lastUsed = time.Now()
		return handle
	}
	
	// Check if we need to evict handles
	if len(s.fileHandles) >= s.maxHandles {
		s.evictOldest()
	}
	
	// Open new file
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		// Return a dummy handle that will fail gracefully
		return &logFileHandle{
			lastUsed: time.Now(),
		}
	}
	
	handle := &logFileHandle{
		file:     file,
		writer:   bufio.NewWriter(file),
		lastUsed: time.Now(),
	}
	
	s.fileHandles[path] = handle
	return handle
}

func (s *Storage) evictOldest() {
	var oldestPath string
	var oldestTime time.Time
	
	for path, handle := range s.fileHandles {
		if oldestPath == "" || handle.lastUsed.Before(oldestTime) {
			oldestPath = path
			oldestTime = handle.lastUsed
		}
	}
	
	if oldestPath != "" {
		if handle, ok := s.fileHandles[oldestPath]; ok {
			handle.mu.Lock()
			handle.writer.Flush()
			handle.file.Close()
			handle.mu.Unlock()
		}
		delete(s.fileHandles, oldestPath)
	}
}

func (s *Storage) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		s.closeIdleHandles()
	}
}

func (s *Storage) closeIdleHandles() {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	threshold := time.Now().Add(-5 * time.Minute)
	
	for path, handle := range s.fileHandles {
		if handle.lastUsed.Before(threshold) {
			handle.mu.Lock()
			handle.writer.Flush()
			handle.file.Close()
			handle.mu.Unlock()
			delete(s.fileHandles, path)
		}
	}
}

func (s *Storage) searchFile(path, keyword string) ([]*api.LogEntry, error) {
	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, nil
	}
	
	// Open file
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	
	var reader io.Reader = file
	
	// Check if file is gzipped
	if strings.HasSuffix(path, ".gz") {
		gz, err := gzip.NewReader(file)
		if err != nil {
			return nil, err
		}
		defer gz.Close()
		reader = gz
	}
	
	scanner := bufio.NewScanner(reader)
	var results []*api.LogEntry
	
	for scanner.Scan() {
		line := scanner.Text()
		
		// Keyword filter
		if keyword != "" && !strings.Contains(line, keyword) {
			continue
		}
		
		var entry api.LogEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		
		results = append(results, &entry)
	}
	
	return results, scanner.Err()
}

func (s *Storage) listServices() ([]string, error) {
	entries, err := os.ReadDir(s.logsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	
	var services []string
	for _, entry := range entries {
		if entry.IsDir() {
			services = append(services, entry.Name())
		}
	}
	
	return services, nil
}

func (s *Storage) listInstances(service string) ([]string, error) {
	path := filepath.Join(s.logsDir, service)
	entries, err := os.ReadDir(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	
	var instances []string
	for _, entry := range entries {
		if entry.IsDir() {
			instances = append(instances, entry.Name())
		}
	}
	
	return instances, nil
}

func (s *Storage) listDates(service, instance string) ([]string, error) {
	return s.listDatesForPath(filepath.Join(s.logsDir, service, instance))
}

func (s *Storage) listDatesForPath(path string) ([]string, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	
	var dates []string
	for _, entry := range entries {
		if entry.IsDir() {
			dates = append(dates, entry.Name())
		}
	}
	
	return dates, nil
}

func (s *Storage) listHours(service, instance, date string) ([]string, error) {
	return s.listHoursForPath(filepath.Join(s.logsDir, service, instance, date))
}

func (s *Storage) listHoursForPath(path string) ([]string, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	
	var hours []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".log") {
			hour := strings.TrimSuffix(entry.Name(), ".log")
			hours = append(hours, hour)
		}
	}
	
	return hours, nil
}
