package common

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/stretchr/testify/require"
)

func TestInitConstantEnvEnablesErrorLogByDefault(t *testing.T) {
	t.Setenv("ERROR_LOG_ENABLED", "")

	initConstantEnv()

	require.True(t, constant.ErrorLogEnabled)
}

func TestInitConstantEnvAllowsDisablingErrorLog(t *testing.T) {
	t.Setenv("ERROR_LOG_ENABLED", "false")

	initConstantEnv()

	require.False(t, constant.ErrorLogEnabled)
}
