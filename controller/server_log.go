package controller

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
)

const (
	defaultServerLogSearchLimit = 200
	maxServerLogSearchLimit     = 1000
	maxServerLogLineSize        = 10 * 1024 * 1024
)

var serverLogKeywordSeparator = regexp.MustCompile(`[,\s，、]+`)

// ServerLogEntry 表示一次运行日志命中结果。
type ServerLogEntry struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	RequestID string `json:"request_id"`
	Message   string `json:"message"`
	RawLine   string `json:"raw_line"`
	FileName  string `json:"file_name"`
}

// ServerLogSearchResponse 表示运行日志检索结果。
type ServerLogSearchResponse struct {
	Entries []ServerLogEntry `json:"entries"`
	Limit   int              `json:"limit"`
}

func parseServerLogLine(line string) (ServerLogEntry, bool) {
	if !strings.HasPrefix(line, "[") {
		return ServerLogEntry{}, false
	}

	levelEnd := strings.Index(line, "]")
	if levelEnd <= 1 || len(line) <= levelEnd+2 {
		return ServerLogEntry{}, false
	}

	level := strings.TrimSpace(line[1:levelEnd])
	payload := strings.TrimSpace(line[levelEnd+1:])
	parts := strings.Split(payload, " | ")
	if len(parts) < 3 {
		return ServerLogEntry{}, false
	}

	entry := ServerLogEntry{
		Timestamp: strings.TrimSpace(parts[0]),
		Level:     level,
		RawLine:   line,
	}

	if level == "GIN" {
		if len(parts) < 4 {
			return ServerLogEntry{}, false
		}
		entry.RequestID = strings.TrimSpace(parts[2])
		messageParts := append([]string{strings.TrimSpace(parts[1])}, parts[3:]...)
		entry.Message = strings.Join(messageParts, " | ")
		return entry, true
	}

	entry.RequestID = strings.TrimSpace(parts[1])
	entry.Message = strings.Join(parts[2:], " | ")
	return entry, true
}

func splitServerLogKeywords(keyword string) []string {
	parts := serverLogKeywordSeparator.Split(strings.ToLower(keyword), -1)
	keywords := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		keywords = append(keywords, part)
	}
	return keywords
}

func serverLogLineMatchesKeywords(line string, keywords []string) bool {
	if len(keywords) == 0 {
		return true
	}
	normalizedLine := strings.ToLower(line)
	for _, keyword := range keywords {
		if strings.Contains(normalizedLine, keyword) {
			return true
		}
	}
	return false
}

func searchServerLogFiles(logDir string, files []LogFileInfo, requestID string, keyword string, limit int) ([]ServerLogEntry, error) {
	if limit <= 0 {
		limit = defaultServerLogSearchLimit
	}
	if limit > maxServerLogSearchLimit {
		limit = maxServerLogSearchLimit
	}
	keywords := splitServerLogKeywords(keyword)

	var results []ServerLogEntry
	for _, file := range files {
		path := filepath.Join(logDir, file.Name)
		fd, err := os.Open(path)
		if err != nil {
			return nil, err
		}

		var matches []ServerLogEntry
		scanner := bufio.NewScanner(fd)
		scanner.Buffer(make([]byte, 64*1024), maxServerLogLineSize)
		for scanner.Scan() {
			line := scanner.Text()
			if requestID != "" && !strings.Contains(line, requestID) {
				continue
			}
			if !serverLogLineMatchesKeywords(line, keywords) {
				continue
			}

			entry, ok := parseServerLogLine(line)
			if !ok {
				entry = ServerLogEntry{
					Message: line,
					RawLine: line,
				}
			}
			entry.FileName = file.Name

			if len(matches) == limit {
				copy(matches, matches[1:])
				matches[len(matches)-1] = entry
				continue
			}
			matches = append(matches, entry)
		}
		scanErr := scanner.Err()
		closeErr := fd.Close()
		if scanErr != nil {
			return nil, scanErr
		}
		if closeErr != nil {
			return nil, closeErr
		}

		for i := len(matches) - 1; i >= 0; i-- {
			results = append(results, matches[i])
			if len(results) == limit {
				return results, nil
			}
		}
	}

	return results, nil
}

// SearchServerLogs 检索服务器运行日志。
func SearchServerLogs(c *gin.Context) {
	requestID := strings.TrimSpace(c.Query("request_id"))
	keyword := strings.TrimSpace(c.Query("keyword"))
	if requestID == "" && keyword == "" {
		common.ApiErrorMsg(c, "request_id or keyword is required")
		return
	}

	if *common.LogDir == "" {
		common.ApiErrorMsg(c, "log directory not configured")
		return
	}

	limit := defaultServerLogSearchLimit
	if rawLimit := strings.TrimSpace(c.Query("limit")); rawLimit != "" {
		parsedLimit, err := strconv.Atoi(rawLimit)
		if err != nil || parsedLimit <= 0 {
			common.ApiErrorMsg(c, "limit must be a positive integer")
			return
		}
		limit = parsedLimit
	}
	if limit > maxServerLogSearchLimit {
		limit = maxServerLogSearchLimit
	}

	files, err := getLogFiles()
	if err != nil {
		common.ApiError(c, err)
		return
	}

	entries, err := searchServerLogFiles(*common.LogDir, files, requestID, keyword, limit)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	common.ApiSuccess(c, ServerLogSearchResponse{
		Entries: entries,
		Limit:   limit,
	})
}
