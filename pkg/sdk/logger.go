package sdk

import (
	"fmt"
	"log"
	"runtime/debug"
	"sync"
	"time"

	"anox/api"
)

// Logger handles log collection and async sending
type Logger struct {
	client    *Client
	buffer    []api.LogEntry
	mu        sync.Mutex
	batchSize int
	timeout   time.Duration
	stopCh    chan struct{}
}

// newLogger creates a new logger instance
func newLogger(client *Client) *Logger {
	l := &Logger{
		client:    client,
		buffer:    make([]api.LogEntry, 0, 10),
		batchSize: 10,
		timeout:   20 * time.Second,
		stopCh:    make(chan struct{}),
	}
	
	// Start the flush goroutine
	go l.flushLoop()
	
	return l
}

// Log sends a log entry
func (l *Logger) Log(entry api.LogEntry, display bool) {
	// Display locally if requested
	if display {
		l.displayLog(entry)
	}
	
	// Add to buffer
	l.mu.Lock()
	l.buffer = append(l.buffer, entry)
	shouldFlush := len(l.buffer) >= l.batchSize
	l.mu.Unlock()
	
	// Flush if buffer is full
	if shouldFlush {
		l.Flush()
	}
}

// LogDebug logs a debug message
func (l *Logger) LogDebug(action string, message string, ctx map[string]string, display bool) {
	entry := api.LogEntry{
		Time:     time.Now(),
		Service:  l.client.GetServiceName(),
		Instance: l.client.GetInstanceID(),
		Level:    api.LogLevelDebug,
		Action:   action,
		Message:  message,
		Context:  ctx,
	}
	l.Log(entry, display)
}

// LogInfo logs an info message
func (l *Logger) LogInfo(action string, message string, ctx map[string]string, display bool) {
	entry := api.LogEntry{
		Time:     time.Now(),
		Service:  l.client.GetServiceName(),
		Instance: l.client.GetInstanceID(),
		Level:    api.LogLevelInfo,
		Action:   action,
		Message:  message,
		Context:  ctx,
	}
	l.Log(entry, display)
}

// LogImportant logs an important message
func (l *Logger) LogImportant(action string, message string, ctx map[string]string, display bool) {
	entry := api.LogEntry{
		Time:     time.Now(),
		Service:  l.client.GetServiceName(),
		Instance: l.client.GetInstanceID(),
		Level:    api.LogLevelImportant,
		Action:   action,
		Message:  message,
		Context:  ctx,
	}
	l.Log(entry, display)
}

// LogEmergency logs an emergency message with full stack trace
func (l *Logger) LogEmergency(action string, message string, ctx map[string]string, display bool) {
	stacks := []string{string(debug.Stack())}
	
	entry := api.LogEntry{
		Time:     time.Now(),
		Service:  l.client.GetServiceName(),
		Instance: l.client.GetInstanceID(),
		Level:    api.LogLevelEmergency,
		Action:   action,
		Message:  message,
		Stacks:   stacks,
		Context:  ctx,
	}
	l.Log(entry, display)
}

// LogError logs an error message
func (l *Logger) LogError(action string, err error, ctx map[string]string, display bool) {
	stacks := []string{string(debug.Stack())}
	
	entry := api.LogEntry{
		Time:     time.Now(),
		Service:  l.client.GetServiceName(),
		Instance: l.client.GetInstanceID(),
		Level:    api.LogLevelEmergency,
		Action:   action,
		Message:  err.Error(),
		Stacks:   stacks,
		Context:  ctx,
	}
	l.Log(entry, display)
}

// Flush sends all buffered logs immediately
func (l *Logger) Flush() {
	l.mu.Lock()
	
	if len(l.buffer) == 0 {
		l.mu.Unlock()
		return
	}
	
	// Copy buffer
	logs := make([]api.LogEntry, len(l.buffer))
	copy(logs, l.buffer)
	l.buffer = l.buffer[:0]
	
	l.mu.Unlock()
	
	// Send logs
	l.sendLogs(logs)
}

// Close stops the logger and flushes remaining logs
func (l *Logger) Close() {
	close(l.stopCh)
	l.Flush()
}

func (l *Logger) flushLoop() {
	ticker := time.NewTicker(l.timeout)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			l.Flush()
		case <-l.stopCh:
			return
		}
	}
}

func (l *Logger) sendLogs(logs []api.LogEntry) {
	if len(logs) == 0 {
		return
	}

	// Convert to JSON-friendly format
	instanceID := l.client.GetInstanceID()
	logsData := make([]map[string]interface{}, len(logs))
	for i, logEntry := range logs {
		logsData[i] = map[string]interface{}{
			"time":     logEntry.Time.Format(time.RFC3339Nano),
			"service":  logEntry.Service,
			"instance": instanceID,
			"level":    logEntry.Level,
			"action":   logEntry.Action,
			"message":  logEntry.Message,
			"trace_id": logEntry.TraceID,
			"stacks":   logEntry.Stacks,
			"context":  logEntry.Context,
		}
	}

	msg := map[string]interface{}{
		"type": "logs_batch",
		"logs": logsData,
	}

	if err := l.client.writeJSON(msg); err != nil {
		log.Printf("[Anox SDK] Failed to send logs: %v", err)
		l.mu.Lock()
		l.buffer = append(logs, l.buffer...)
		l.mu.Unlock()
		go l.client.reconnect()
	}
}

func (l *Logger) displayLog(entry api.LogEntry) {
	var levelColor string
	switch entry.Level {
	case api.LogLevelDebug:
		levelColor = "\033[36m" // Cyan
	case api.LogLevelInfo:
		levelColor = "\033[32m" // Green
	case api.LogLevelImportant:
		levelColor = "\033[33m" // Yellow
	case api.LogLevelEmergency:
		levelColor = "\033[31m" // Red
	default:
		levelColor = "\033[0m" // Reset
	}
	resetColor := "\033[0m"
	
	fmt.Printf("%s[%s]%s %s[%s]%s %s - %s\n",
		levelColor,
		entry.Level,
		resetColor,
		levelColor,
		entry.Action,
		resetColor,
		entry.Time.Format("2006-01-02 15:04:05"),
		entry.Message,
	)
	
	if len(entry.Stacks) > 0 {
		for _, stack := range entry.Stacks {
			fmt.Printf("%sStack:%s\n%s\n", levelColor, resetColor, stack)
		}
	}
}
