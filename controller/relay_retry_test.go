package controller

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func newRetryTestContext() *gin.Context {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	return ctx
}

func TestGetRetryDecisionRetriesTooManyRequestsWithRemainingRetries(t *testing.T) {
	ctx := newRetryTestContext()
	err := types.NewOpenAIError(errors.New("The usage limit has been reached"), types.ErrorCodeBadResponseStatusCode, http.StatusTooManyRequests)

	decision := getRetryDecision(ctx, err, 1)

	require.True(t, decision.shouldRetry)
	require.Equal(t, "retryable_status_code", decision.reason)
}

func TestGetRetryDecisionSkipsWhenRetriesExhausted(t *testing.T) {
	ctx := newRetryTestContext()
	err := types.NewOpenAIError(errors.New("The usage limit has been reached"), types.ErrorCodeBadResponseStatusCode, http.StatusTooManyRequests)

	decision := getRetryDecision(ctx, err, 0)

	require.False(t, decision.shouldRetry)
	require.Equal(t, "retry_times_exhausted", decision.reason)
}

func TestGetRetryDecisionSkipsSpecificChannel(t *testing.T) {
	ctx := newRetryTestContext()
	ctx.Set("specific_channel_id", "38")
	err := types.NewOpenAIError(errors.New("The usage limit has been reached"), types.ErrorCodeBadResponseStatusCode, http.StatusTooManyRequests)

	decision := getRetryDecision(ctx, err, 1)

	require.False(t, decision.shouldRetry)
	require.Equal(t, "specific_channel", decision.reason)
}

func TestGetRetryDecisionSkipsExplicitSkipRetryError(t *testing.T) {
	ctx := newRetryTestContext()
	err := types.NewOpenAIError(
		errors.New("The usage limit has been reached"),
		types.ErrorCodeBadResponseStatusCode,
		http.StatusTooManyRequests,
		types.ErrOptionWithSkipRetry(),
	)

	decision := getRetryDecision(ctx, err, 1)

	require.False(t, decision.shouldRetry)
	require.Equal(t, "skip_retry_error", decision.reason)
}

func TestRetryDecisionUsesVisibleLogWhenSkipped(t *testing.T) {
	decision := retryDecision{reason: "retry_times_exhausted"}

	require.Equal(t, relayRetryLogLevelError, getRelayRetryLogLevel(decision))
}

func TestRetryDecisionUsesDebugLogWhenRetrying(t *testing.T) {
	decision := retryDecision{shouldRetry: true, reason: "retryable_status_code"}

	require.Equal(t, relayRetryLogLevelDebug, getRelayRetryLogLevel(decision))
}
