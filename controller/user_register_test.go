package controller

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	appI18n "github.com/QuantumNous/new-api/i18n"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegisterRejectsOverlongUsernameWithFriendlyMessage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	require.NoError(t, appI18n.Init())

	oldRegisterEnabled := common.RegisterEnabled
	oldPasswordRegisterEnabled := common.PasswordRegisterEnabled
	oldEmailVerificationEnabled := common.EmailVerificationEnabled
	oldGenerateDefaultToken := constant.GenerateDefaultToken
	common.RegisterEnabled = true
	common.PasswordRegisterEnabled = true
	common.EmailVerificationEnabled = false
	constant.GenerateDefaultToken = false
	t.Cleanup(func() {
		common.RegisterEnabled = oldRegisterEnabled
		common.PasswordRegisterEnabled = oldPasswordRegisterEnabled
		common.EmailVerificationEnabled = oldEmailVerificationEnabled
		constant.GenerateDefaultToken = oldGenerateDefaultToken
	})

	body, err := common.Marshal(map[string]any{
		"username": "daliu9303012@gmail.com",
		"password": "772545663",
		"email":    "daliu9303012@gmail.com",
	})
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/api/user/register", bytes.NewReader(body))
	context.Request.Header.Set("Content-Type", "application/json")

	Register(context)

	var response struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	assert.False(t, response.Success)
	assert.Contains(t, response.Message, "Username must be at most 20 characters")
	assert.NotContains(t, response.Message, "Field validation")
	assert.NotContains(t, response.Message, "User.Username")
}
