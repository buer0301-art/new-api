package controller

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestParsePerfMetricsRangeReadsExplicitTimestamps(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest(
		"GET",
		"/api/perf-metrics/summary?start_timestamp=1716076800&end_timestamp=1716163199",
		nil,
	)

	startTs, endTs, ok := parsePerfMetricsRange(ctx)

	require.True(t, ok)
	require.Equal(t, int64(1716076800), startTs)
	require.Equal(t, int64(1716163199), endTs)
}

func TestParsePerfMetricsRangeSwapsReverseInput(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest(
		"GET",
		"/api/perf-metrics/summary?start_timestamp=1716163199&end_timestamp=1716076800",
		nil,
	)

	startTs, endTs, ok := parsePerfMetricsRange(ctx)

	require.True(t, ok)
	require.Equal(t, int64(1716076800), startTs)
	require.Equal(t, int64(1716163199), endTs)
}

func TestParsePerfMetricsRangeRejectsMissingValues(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest("GET", "/api/perf-metrics/summary", nil)

	startTs, endTs, ok := parsePerfMetricsRange(ctx)

	require.False(t, ok)
	require.Zero(t, startTs)
	require.Zero(t, endTs)
}
