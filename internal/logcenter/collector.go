package logcenter

import (
	"log"
	"time"

	"anox/api"
)

// Collector collects log entries from services
type Collector struct {
	storage  *Storage
	alert    *AlertEngine
	batchCh  chan *api.LogEntry
	stopCh   chan struct{}
	batchSize    int
	batchTimeout time.Duration
}

// NewCollector creates a new log collector
func NewCollector(storage *Storage, alert *AlertEngine) *Collector {
	c := &Collector{
		storage:      storage,
		alert:        alert,
		batchCh:      make(chan *api.LogEntry, 10000),
		stopCh:       make(chan struct{}),
		batchSize:    10,
		batchTimeout: 20 * time.Second,
	}
	
	// Start the batch processor
	go c.processBatch()
	
	return c
}

// Submit submits a log entry to the collector
func (c *Collector) Submit(entry *api.LogEntry) {
	select {
	case c.batchCh <- entry:
	default:
		// Channel full, drop the log entry
		log.Printf("[LogCollector] Channel full, dropping log from %s/%s", entry.Service, entry.Instance)
	}
}

// SubmitBatch submits multiple log entries at once
func (c *Collector) SubmitBatch(entries []*api.LogEntry) {
	for _, entry := range entries {
		c.Submit(entry)
	}
}

// Stop stops the collector
func (c *Collector) Stop() {
	close(c.stopCh)
}

func (c *Collector) processBatch() {
	batch := make([]*api.LogEntry, 0, c.batchSize)
	timer := time.NewTimer(c.batchTimeout)
	defer timer.Stop()

	resetTimer := func() {
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
		timer.Reset(c.batchTimeout)
	}

	for {
		select {
		case entry := <-c.batchCh:
			batch = append(batch, entry)
			
			// Process if batch is full
			if len(batch) >= c.batchSize {
				c.flush(batch)
				batch = make([]*api.LogEntry, 0, c.batchSize)
				resetTimer()
			}
			
		case <-timer.C:
			// Process on timeout
			if len(batch) > 0 {
				c.flush(batch)
				batch = make([]*api.LogEntry, 0, c.batchSize)
			}
			resetTimer()
			
		case <-c.stopCh:
			// Flush remaining
			if len(batch) > 0 {
				c.flush(batch)
			}
			return
		}
	}
}

func (c *Collector) flush(batch []*api.LogEntry) {
	if len(batch) == 0 {
		return
	}

	// Store logs
	for _, entry := range batch {
		if err := c.storage.Store(entry); err != nil {
			log.Printf("[LogCollector] Failed to store log: %v", err)
		}
	}

	// Check for alerts
	if c.alert != nil {
		for _, entry := range batch {
			c.alert.Check(entry)
		}
	}

	log.Printf("[LogCollector] Flushed %d log entries", len(batch))
}
