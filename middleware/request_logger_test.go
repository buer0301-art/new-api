package middleware

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestRequestDetailLoggerLogsIncomingRequestAndRestoresBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var logBuffer bytes.Buffer
	common.LogWriterMu.Lock()
	originalWriter := gin.DefaultWriter
	gin.DefaultWriter = &logBuffer
	common.LogWriterMu.Unlock()
	t.Cleanup(func() {
		common.LogWriterMu.Lock()
		gin.DefaultWriter = originalWriter
		common.LogWriterMu.Unlock()
	})

	router := gin.New()
	router.Use(RequestDetailLogger())
	router.POST("/v1/chat/completions", func(c *gin.Context) {
		body, err := io.ReadAll(c.Request.Body)
		require.NoError(t, err)
		require.JSONEq(t, `{"model":"test","stream":false}`, string(body))
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(
		http.MethodPost,
		"/v1/chat/completions?trace=1",
		strings.NewReader(`{"model":"test","stream":false}`),
	)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer sk-test")
	req.Header.Set("X-Custom-Header", "custom-value")
	req.RemoteAddr = "203.0.113.9:12345"

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusNoContent, resp.Code)
	logOutput := logBuffer.String()
	require.Contains(t, logOutput, "[REQ]")
	require.Contains(t, logOutput, "method=POST")
	require.Contains(t, logOutput, "path=/v1/chat/completions")
	require.Contains(t, logOutput, "uri=/v1/chat/completions?trace=1")
	require.Contains(t, logOutput, "query=trace=1")
	require.Contains(t, logOutput, "client_ip=203.0.113.9")
	require.Contains(t, logOutput, "Content-Type:[application/json]")
	require.Contains(t, logOutput, "X-Custom-Header:[custom-value]")
	require.Contains(t, logOutput, `body="{\"model\":\"test\",\"stream\":false}"`)
	require.Contains(t, logOutput, "Authorization:[Bearer sk-test]")
}

func TestRequestDetailLoggerKeepsFullBodyForHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var logBuffer bytes.Buffer
	common.LogWriterMu.Lock()
	originalWriter := gin.DefaultWriter
	gin.DefaultWriter = &logBuffer
	common.LogWriterMu.Unlock()
	t.Cleanup(func() {
		common.LogWriterMu.Lock()
		gin.DefaultWriter = originalWriter
		common.LogWriterMu.Unlock()
	})

	router := gin.New()
	router.Use(RequestDetailLogger())
	router.POST("/v1/files", func(c *gin.Context) {
		body, err := io.ReadAll(c.Request.Body)
		require.NoError(t, err)
		require.Len(t, body, 1024+128)
		c.Status(http.StatusOK)
	})

	payload := strings.Repeat("a", 1024+128)
	req := httptest.NewRequest(http.MethodPost, "/v1/files", strings.NewReader(payload))
	req.Header.Set("Content-Type", "text/plain")

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
	logOutput := logBuffer.String()
	require.Contains(t, logOutput, strings.Repeat("a", 1024+128))
}
