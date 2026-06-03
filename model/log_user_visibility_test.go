package model

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetUserLogsHidesErrorLogs(t *testing.T) {
	truncateTables(t)

	require.NoError(t, LOG_DB.Create(&Log{
		UserId:    1,
		Username:  "alice",
		Type:      LogTypeConsume,
		Content:   "success",
		ModelName: "gpt-4o-mini",
	}).Error)
	require.NoError(t, LOG_DB.Create(&Log{
		UserId:    1,
		Username:  "alice",
		Type:      LogTypeError,
		Content:   "upstream failed",
		ModelName: "gpt-4o-mini",
	}).Error)

	logs, total, err := GetUserLogs(1, LogTypeUnknown, 0, 0, "", "", 0, 20, "", "", "")

	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, logs, 1)
	require.Equal(t, LogTypeConsume, logs[0].Type)
}

func TestGetUserLogsRejectsExplicitErrorType(t *testing.T) {
	truncateTables(t)

	require.NoError(t, LOG_DB.Create(&Log{
		UserId:    1,
		Username:  "alice",
		Type:      LogTypeError,
		Content:   "upstream failed",
		ModelName: "gpt-4o-mini",
	}).Error)

	logs, total, err := GetUserLogs(1, LogTypeError, 0, 0, "", "", 0, 20, "", "", "")

	require.NoError(t, err)
	require.EqualValues(t, 0, total)
	require.Empty(t, logs)
}

func TestGetAllLogsStillShowsErrorLogsForAdmins(t *testing.T) {
	truncateTables(t)

	require.NoError(t, LOG_DB.Create(&Log{
		UserId:    1,
		Username:  "alice",
		Type:      LogTypeError,
		Content:   "upstream failed",
		ModelName: "gpt-4o-mini",
	}).Error)

	logs, total, err := GetAllLogs(LogTypeUnknown, 0, 0, "", "", "", 0, 20, 0, "", "", "")

	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, logs, 1)
	require.Equal(t, LogTypeError, logs[0].Type)
}
