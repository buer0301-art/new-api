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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetUpLoggerDoesNotLogRequestBodyOrSensitiveHeaders(t *testing.T) {
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
	SetUpLogger(router)
	router.POST("/v1/chat/completions", func(c *gin.Context) {
		body, err := io.ReadAll(c.Request.Body)
		require.NoError(t, err)
		require.Contains(t, string(body), "secret prompt")
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(
		http.MethodPost,
		"/v1/chat/completions",
		strings.NewReader(`{"model":"test","prompt":"secret prompt","messages":[{"role":"user","content":"secret message"}]}`),
	)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer sk-secret")
	req.Header.Set("Cookie", "session=secret")

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusNoContent, resp.Code)
	logOutput := logBuffer.String()
	assert.Contains(t, logOutput, "[GIN]")
	assert.NotContains(t, logOutput, "[REQ]")
	assert.NotContains(t, logOutput, "secret prompt")
	assert.NotContains(t, logOutput, "secret message")
	assert.NotContains(t, logOutput, "Authorization")
	assert.NotContains(t, logOutput, "Bearer sk-secret")
	assert.NotContains(t, logOutput, "Cookie")
	assert.NotContains(t, logOutput, "session=secret")
}
