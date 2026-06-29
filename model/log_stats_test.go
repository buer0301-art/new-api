package model

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSumUsedQuotaIncludesFilteredTokenTotal(t *testing.T) {
	truncateTables(t)

	now := time.Now().Unix()
	require.NoError(t, LOG_DB.Create(&Log{
		Username:         "alice",
		TokenName:        "prod",
		ModelName:        "gpt-4o-mini",
		ChannelId:        10,
		Group:            "default",
		Type:             LogTypeConsume,
		CreatedAt:        now - 30,
		Quota:            100,
		PromptTokens:     1_200_000,
		CompletionTokens: 300_000,
		CacheReadTokens:  50_000,
		CacheWriteTokens: 60_000,
	}).Error)
	require.NoError(t, LOG_DB.Create(&Log{
		Username:         "alice",
		TokenName:        "prod",
		ModelName:        "gpt-4o-mini",
		ChannelId:        10,
		Group:            "default",
		Type:             LogTypeConsume,
		CreatedAt:        now - 20,
		Quota:            200,
		PromptTokens:     200_000,
		CompletionTokens: 100_000,
		CacheReadTokens:  10_000,
		CacheWriteTokens: 20_000,
	}).Error)
	require.NoError(t, LOG_DB.Create(&Log{
		Username:         "alice",
		TokenName:        "prod",
		ModelName:        "gpt-4o-mini",
		ChannelId:        11,
		Group:            "default",
		Type:             LogTypeConsume,
		CreatedAt:        now - 20,
		Quota:            500,
		PromptTokens:     9_000_000,
		CompletionTokens: 9_000_000,
		CacheReadTokens:  9_000,
		CacheWriteTokens: 9_000,
	}).Error)
	require.NoError(t, LOG_DB.Create(&Log{
		Username:         "alice",
		TokenName:        "prod",
		ModelName:        "gpt-4o-mini",
		ChannelId:        10,
		Group:            "other",
		Type:             LogTypeConsume,
		CreatedAt:        now - 20,
		Quota:            500,
		PromptTokens:     8_000_000,
		CompletionTokens: 8_000_000,
		CacheReadTokens:  8_000,
		CacheWriteTokens: 8_000,
	}).Error)

	stat, err := SumUsedQuota(
		LogTypeConsume,
		now-60,
		now,
		"gpt-4o-mini",
		"alice",
		"prod",
		10,
		"default",
		"",
		"",
	)

	require.NoError(t, err)
	assert.Equal(t, 300, stat.Quota)
	assert.Equal(t, 1_800_000, stat.Token)
	assert.Equal(t, 60_000, stat.CacheReadToken)
	assert.Equal(t, 80_000, stat.CacheWriteToken)
	assert.Equal(t, 2, stat.Rpm)
	assert.Equal(t, 1_800_000, stat.Tpm)
}

func TestSumUsedQuotaRpmTpmRespectTimeRange(t *testing.T) {
	truncateTables(t)

	now := time.Now().Unix()
	require.NoError(t, LOG_DB.Create(&Log{
		Username:         "alice",
		TokenName:        "prod",
		ModelName:        "gpt-4o-mini",
		ChannelId:        10,
		Group:            "default",
		Type:             LogTypeConsume,
		CreatedAt:        now - 30,
		Quota:            100,
		PromptTokens:     1_000,
		CompletionTokens: 2_000,
	}).Error)

	stat, err := SumUsedQuota(
		LogTypeConsume,
		now-10,
		now,
		"gpt-4o-mini",
		"alice",
		"prod",
		10,
		"default",
		"",
		"",
	)

	require.NoError(t, err)
	assert.Equal(t, 0, stat.Quota)
	assert.Equal(t, 0, stat.Token)
	assert.Equal(t, 0, stat.Rpm)
	assert.Equal(t, 0, stat.Tpm)

	stat, err = SumUsedQuota(
		LogTypeConsume,
		now-60,
		now-40,
		"gpt-4o-mini",
		"alice",
		"prod",
		10,
		"default",
		"",
		"",
	)

	require.NoError(t, err)
	assert.Equal(t, 0, stat.Quota)
	assert.Equal(t, 0, stat.Token)
	assert.Equal(t, 0, stat.Rpm)
	assert.Equal(t, 0, stat.Tpm)
}

func TestSumUsedQuotaReturnsZeroForNonConsumeType(t *testing.T) {
	truncateTables(t)

	now := time.Now().Unix()
	require.NoError(t, LOG_DB.Create(&Log{
		Username:         "alice",
		TokenName:        "prod",
		ModelName:        "gpt-4o-mini",
		Type:             LogTypeConsume,
		CreatedAt:        now - 30,
		Quota:            100,
		PromptTokens:     1_000,
		CompletionTokens: 2_000,
	}).Error)

	stat, err := SumUsedQuota(
		LogTypeTopup,
		now-60,
		now,
		"gpt-4o-mini",
		"alice",
		"prod",
		0,
		"",
		"",
		"",
	)

	require.NoError(t, err)
	assert.Equal(t, 0, stat.Quota)
	assert.Equal(t, 0, stat.Token)
	assert.Equal(t, 0, stat.CacheReadToken)
	assert.Equal(t, 0, stat.CacheWriteToken)
	assert.Equal(t, 0, stat.Rpm)
	assert.Equal(t, 0, stat.Tpm)
}

func TestRecordConsumeLogStoresCacheTokenFields(t *testing.T) {
	truncateTables(t)
	originalLogConsumeEnabled := common.LogConsumeEnabled
	common.LogConsumeEnabled = true
	t.Cleanup(func() {
		common.LogConsumeEnabled = originalLogConsumeEnabled
	})

	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	ctx.Set("username", "alice")

	RecordConsumeLog(ctx, 1, RecordConsumeLogParams{
		ChannelId:        10,
		PromptTokens:     100,
		CompletionTokens: 20,
		CacheReadTokens:  30,
		CacheWriteTokens: 40,
		ModelName:        "gpt-4o-mini",
		TokenName:        "prod",
		Quota:            12,
		Content:          "test",
		TokenId:          7,
		Group:            "default",
	})

	var log Log
	require.NoError(t, LOG_DB.Where("user_id = ? AND type = ?", 1, LogTypeConsume).First(&log).Error)
	assert.Equal(t, 30, log.CacheReadTokens)
	assert.Equal(t, 40, log.CacheWriteTokens)
}
