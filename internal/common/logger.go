package common

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// LogFile is the path to the request log file.
const LogFile = "log.txt"

var (
	logMu sync.Mutex
)

// LogEntry represents a single request log entry.
type LogEntry struct {
	Timestamp  string `json:"timestamp"`
	IP         string `json:"ip"`
	Method     string `json:"method"`
	Path       string `json:"path"`
	StatusCode int    `json:"status_code"`
	Duration   string `json:"duration"`
	RequestID  string `json:"request_id,omitempty"`
}

// WriteLog appends a log line to the log file in a human-readable format.
func WriteLog(entry *LogEntry) error {
	line := fmt.Sprintf("[%s] %s %s %s > %d (%s)\n",
		entry.Timestamp,
		entry.IP,
		entry.Method,
		entry.Path,
		entry.StatusCode,
		entry.Duration,
	)

	logMu.Lock()
	defer logMu.Unlock()

	f, err := os.OpenFile(LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(line)
	return err
}

// DownstreamEntry represents a downstream API call log entry.
type DownstreamEntry struct {
	Timestamp  string `json:"timestamp"`
	Model      string `json:"model"`
	URL        string `json:"url"`
	ReqBody    string `json:"request_body"`
	RespBody   string `json:"response_body"`
	Status     string `json:"status"`
	PromptTokens int    `json:"prompt_tokens"`
	OutputTokens int    `json:"output_tokens"`
	TotalTokens  int    `json:"total_tokens"`
	Error      string `json:"error,omitempty"`
}

// WriteDownstreamLog appends a downstream request/response log entry.
func WriteDownstreamLog(entry *DownstreamEntry) error {
	line := fmt.Sprintf("[DOWNSTREAM] %s | model=%s | url=%s | status=%s | tokens=%d+%d=%d | req=%.200s | resp=%.300s\n",
		entry.Timestamp,
		entry.Model,
		entry.URL,
		entry.Status,
		entry.PromptTokens,
		entry.OutputTokens,
		entry.TotalTokens,
		entry.ReqBody,
		entry.RespBody,
	)
	if entry.Error != "" {
		line = fmt.Sprintf("[DOWNSTREAM] %s | model=%s | url=%s | status=%s | error=%s | req=%.200s\n",
			entry.Timestamp,
			entry.Model,
			entry.URL,
			entry.Status,
			entry.Error,
			entry.ReqBody,
		)
	}

	logMu.Lock()
	defer logMu.Unlock()

	f, err := os.OpenFile(LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(line)
	return err
}

// RequestLogger returns a Gin middleware that logs every request to log.txt.
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Process the request
		c.Next()

		// Build the log entry after response is written
		entry := &LogEntry{
			Timestamp:  start.Format(time.RFC3339),
			IP:         c.ClientIP(),
			Method:     c.Request.Method,
			Path:       c.Request.URL.Path,
			StatusCode: c.Writer.Status(),
			Duration:   time.Since(start).Round(time.Millisecond).String(),
		}

		if err := WriteLog(entry); err != nil {
			fmt.Fprintf(os.Stderr, "failed to write log: %v\n", err)
		}
	}
}
