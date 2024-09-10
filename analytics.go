// Package traefik_analytics is a Traefik plugin that collects request analytics
// and stores them in a PostgreSQL database.
package traefik_analytics

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"

	_ "github.com/lib/pq"
)

// Config holds the plugin configuration.
type Config struct {
	DatabaseDSN string `json:"databaseDSN,omitempty"`
}

// CreateConfig creates the default plugin configuration.
func CreateConfig() *Config {
	return &Config{
		DatabaseDSN: "",
	}
}

// Analytics is the plugin structure.
type Analytics struct {
	next     http.Handler
	name     string
	config   *Config
	dataChan chan RequestData
}

// New creates a new plugin instance.
func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	if config.DatabaseDSN == "" {
		return nil, fmt.Errorf("DatabaseDSN is required")
	}

	analytics := &Analytics{
		next:     next,
		name:     name,
		config:   config,
		dataChan: make(chan RequestData, 1000), // Buffered channel
	}

	// Start the processing worker
	go analytics.processingWorker()

	return analytics, nil
}

// ServeHTTP implements the http.Handler interface.
func (a *Analytics) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	start := time.Now()

	// Call the next handler
	a.next.ServeHTTP(rw, req)

	// Collect request data
	data := RequestData{
		IP:             req.RemoteAddr,
		UserAgent:      req.UserAgent(),
		Path:           req.URL.Path,
		Time:           start,
		Method:         req.Method,
		Protocol:       req.Proto,
		Host:           req.Host,
		AcceptLanguage: req.Header.Get("Accept-Language"),
		Referer:        req.Referer(),
		ContentType:    req.Header.Get("Content-Type"),
		ContentLength:  req.ContentLength,
		ResponseTime:   time.Since(start),
	}

	// Send data to processing goroutine
	select {
	case a.dataChan <- data:
		// Data sent successfully
	default:
		log.Println("Analytics channel full, discarding data")
	}
}

// RequestData holds the collected request information.
type RequestData struct {
	IP             string
	UserAgent      string
	Path           string
	Time           time.Time
	Method         string
	Protocol       string
	Host           string
	AcceptLanguage string
	Referer        string
	ContentType    string
	ContentLength  int64
	ResponseTime   time.Duration
}

// processingWorker handles database insertions.
func (a *Analytics) processingWorker() {
	for {
		err := a.runWorker()
		if err != nil {
			log.Printf("Worker encountered an error: %v", err)
			time.Sleep(5 * time.Second) // Wait before retrying
		}
	}
}

// runWorker performs the actual database operations.
func (a *Analytics) runWorker() error {
	db, err := sql.Open("postgres", a.config.DatabaseDSN)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %v", err)
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		return fmt.Errorf("failed to ping database: %v", err)
	}

	stmt, err := db.Prepare(`
        INSERT INTO request_logs (
            ip, user_agent, path, request_time, method, protocol, host,
            accept_language, referer, content_type, content_length, response_time
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
    `)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %v", err)
	}
	defer stmt.Close()

	for data := range a.dataChan {
		_, err := stmt.Exec(
			data.IP, data.UserAgent, data.Path, data.Time, data.Method,
			data.Protocol, data.Host, data.AcceptLanguage, data.Referer,
			data.ContentType, data.ContentLength, data.ResponseTime,
		)
		if err != nil {
			log.Printf("Failed to insert data: %v", err)
			// Continue processing other requests
		}
	}

	return nil
}
