package middleware

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
)

const RouteTagKey = "route_tag"

func RouteTag(tag string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(RouteTagKey, tag)
		c.Next()
	}
}

func SetUpLogger(server *gin.Engine) {
	server.Use(RequestDetailLogger())
	server.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		var requestID string
		if param.Keys != nil {
			requestID, _ = param.Keys[common.RequestIdKey].(string)
		}
		tag, _ := param.Keys[RouteTagKey].(string)
		if tag == "" {
			tag = "web"
		}
		return fmt.Sprintf("[GIN] %s | %s | %s | %3d | %13v | %15s | %7s %s\n",
			param.TimeStamp.Format("2006/01/02 - 15:04:05"),
			tag,
			requestID,
			param.StatusCode,
			param.Latency,
			param.ClientIP,
			param.Method,
			param.Path,
		)
	}))
}

func RequestDetailLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		body, readErr := readAndRestoreRequestBody(c.Request)
		writeRequestDetailLog(c, body, readErr)
		c.Next()
	}
}

func readAndRestoreRequestBody(req *http.Request) (string, error) {
	if req == nil || req.Body == nil {
		return "", nil
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		req.Body = io.NopCloser(bytes.NewReader(nil))
		return "", err
	}

	req.Body = io.NopCloser(bytes.NewReader(body))
	req.ContentLength = int64(len(body))
	return string(body), nil
}

func writeRequestDetailLog(c *gin.Context, body string, readErr error) {
	req := c.Request
	if req == nil || req.URL == nil {
		return
	}

	requestID := c.GetString(common.RequestIdKey)
	tag := c.GetString(RouteTagKey)
	if tag == "" {
		tag = "web"
	}

	fields := []string{
		fmt.Sprintf("tag=%s", tag),
		fmt.Sprintf("request_id=%s", requestID),
		fmt.Sprintf("method=%s", req.Method),
		fmt.Sprintf("path=%s", req.URL.Path),
		fmt.Sprintf("uri=%s", req.RequestURI),
		fmt.Sprintf("query=%s", req.URL.RawQuery),
		fmt.Sprintf("client_ip=%s", c.ClientIP()),
		fmt.Sprintf("host=%s", req.Host),
		fmt.Sprintf("remote_addr=%s", req.RemoteAddr),
		fmt.Sprintf("headers=%v", req.Header),
	}

	if readErr != nil {
		fields = append(fields, fmt.Sprintf("body_read_error=%q", readErr.Error()))
	} else {
		fields = append(fields, fmt.Sprintf("body=%q", body))
	}

	common.LogWriterMu.RLock()
	_, _ = fmt.Fprintf(gin.DefaultWriter, "[REQ] %s | %s \n", time.Now().Format("2006/01/02 - 15:04:05"), strings.Join(fields, " | "))
	common.LogWriterMu.RUnlock()
}
