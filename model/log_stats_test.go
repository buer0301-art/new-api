package model

import (
	"testing"
	"time"

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
	require.Equal(t, 300, stat.Quota)
	require.Equal(t, 1_800_000, stat.Token)
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
	require.Equal(t, 0, stat.Quota)
	require.Equal(t, 0, stat.Token)
	require.Equal(t, 0, stat.Rpm)
	require.Equal(t, 0, stat.Tpm)
}
