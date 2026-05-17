package controller

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestParseServerLogLineParsesApplicationLog(t *testing.T) {
	entry, ok := parseServerLogLine(`[ERR] 2026/05/17 - 10:11:12 | req-123 | upstream timeout`)

	require.True(t, ok)
	require.Equal(t, "ERR", entry.Level)
	require.Equal(t, "2026/05/17 - 10:11:12", entry.Timestamp)
	require.Equal(t, "req-123", entry.RequestID)
	require.Equal(t, "upstream timeout", entry.Message)
}

func TestParseServerLogLineParsesGinLog(t *testing.T) {
	entry, ok := parseServerLogLine(`[GIN] 2026/05/17 - 10:11:12 | relay | req-456 | 500 | 1.2s | 127.0.0.1 | POST /v1/chat/completions`)

	require.True(t, ok)
	require.Equal(t, "GIN", entry.Level)
	require.Equal(t, "2026/05/17 - 10:11:12", entry.Timestamp)
	require.Equal(t, "req-456", entry.RequestID)
	require.Equal(t, "relay | 500 | 1.2s | 127.0.0.1 | POST /v1/chat/completions", entry.Message)
}

func TestSearchServerLogFilesFiltersAndReturnsNewestFirst(t *testing.T) {
	logDir := t.TempDir()
	oldFile := filepath.Join(logDir, "oneapi-20260516120000.log")
	newFile := filepath.Join(logDir, "oneapi-20260517120000.log")

	require.NoError(t, os.WriteFile(oldFile, []byte(
		"[INFO] 2026/05/16 - 12:00:00 | req-1 | old match\n"+
			"[ERR] 2026/05/16 - 12:00:01 | req-2 | old timeout\n",
	), 0o644))
	require.NoError(t, os.WriteFile(newFile, []byte(
		"[INFO] 2026/05/17 - 12:00:00 | req-1 | first match\n"+
			"[ERR] 2026/05/17 - 12:00:01 | req-1 | latest timeout\n"+
			"[WARN] 2026/05/17 - 12:00:02 | req-3 | unrelated timeout\n",
	), 0o644))

	results, err := searchServerLogFiles(logDir, []LogFileInfo{
		{Name: filepath.Base(newFile), ModTime: time.Now()},
		{Name: filepath.Base(oldFile), ModTime: time.Now().Add(-time.Hour)},
	}, "req-1", "timeout", 10)

	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, "req-1", results[0].RequestID)
	require.Equal(t, "latest timeout", results[0].Message)
	require.Equal(t, filepath.Base(newFile), results[0].FileName)
}

func TestSearchServerLogFilesHonorsLimitWithNewestMatches(t *testing.T) {
	logDir := t.TempDir()
	fileName := "oneapi-20260517120000.log"
	require.NoError(t, os.WriteFile(filepath.Join(logDir, fileName), []byte(
		"[INFO] 2026/05/17 - 12:00:00 | req-1 | first\n"+
			"[INFO] 2026/05/17 - 12:00:01 | req-1 | second\n"+
			"[INFO] 2026/05/17 - 12:00:02 | req-1 | third\n",
	), 0o644))

	results, err := searchServerLogFiles(logDir, []LogFileInfo{{Name: fileName}}, "req-1", "", 2)

	require.NoError(t, err)
	require.Len(t, results, 2)
	require.Equal(t, "third", results[0].Message)
	require.Equal(t, "second", results[1].Message)
}

func TestSearchServerLogFilesMatchesAnyKeywordSeparatedByWhitespaceOrComma(t *testing.T) {
	logDir := t.TempDir()
	fileName := "oneapi-20260517120000.log"
	require.NoError(t, os.WriteFile(filepath.Join(logDir, fileName), []byte(
		"[ERR] 2026/05/17 - 12:00:00 | req-1 | upstream timeout\n"+
			"[WARN] 2026/05/17 - 12:00:01 | req-2 | quota refund failed\n"+
			"[INFO] 2026/05/17 - 12:00:02 | req-3 | channel recovered\n",
	), 0o644))

	results, err := searchServerLogFiles(logDir, []LogFileInfo{{Name: fileName}}, "", "timeout，refund channel", 10)

	require.NoError(t, err)
	require.Len(t, results, 3)
	require.Equal(t, "channel recovered", results[0].Message)
	require.Equal(t, "quota refund failed", results[1].Message)
	require.Equal(t, "upstream timeout", results[2].Message)
}
