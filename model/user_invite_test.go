package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func withInviteSettings(t *testing.T, inviterQuota int, inviteeQuota int) {
	t.Helper()

	oldInviterQuota := common.QuotaForInviter
	oldInviteeQuota := common.QuotaForInvitee
	oldNewUserQuota := common.QuotaForNewUser
	paymentSetting := operation_setting.GetPaymentSetting()
	oldComplianceConfirmed := paymentSetting.ComplianceConfirmed
	oldComplianceTermsVersion := paymentSetting.ComplianceTermsVersion

	common.QuotaForInviter = inviterQuota
	common.QuotaForInvitee = inviteeQuota
	common.QuotaForNewUser = 0
	paymentSetting.ComplianceConfirmed = true
	paymentSetting.ComplianceTermsVersion = operation_setting.CurrentComplianceTermsVersion

	t.Cleanup(func() {
		common.QuotaForInviter = oldInviterQuota
		common.QuotaForInvitee = oldInviteeQuota
		common.QuotaForNewUser = oldNewUserQuota
		paymentSetting.ComplianceConfirmed = oldComplianceConfirmed
		paymentSetting.ComplianceTermsVersion = oldComplianceTermsVersion
	})
}

func TestUserInsertRecordsInviterRelationshipWithoutRewardQuota(t *testing.T) {
	truncateTables(t)
	withInviteSettings(t, 0, 0)

	inviter := User{
		Username:    "inviter",
		Password:    "password123",
		DisplayName: "inviter",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		AffCode:     "abcd",
	}
	require.NoError(t, DB.Create(&inviter).Error)

	invitee := User{
		Username:    "invitee",
		Password:    "password123",
		DisplayName: "invitee",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
	}
	require.NoError(t, invitee.Insert(inviter.Id))

	var createdInvitee User
	require.NoError(t, DB.First(&createdInvitee, invitee.Id).Error)
	assert.Equal(t, inviter.Id, createdInvitee.InviterId)

	var updatedInviter User
	require.NoError(t, DB.First(&updatedInviter, inviter.Id).Error)
	assert.Equal(t, 1, updatedInviter.AffCount)
	assert.Equal(t, 0, updatedInviter.AffQuota)
	assert.Equal(t, 0, updatedInviter.AffHistoryQuota)
}

func TestUserInsertWithTxRecordsInviterRelationship(t *testing.T) {
	truncateTables(t)
	withInviteSettings(t, 0, 0)

	inviter := User{
		Username:    "tx_inviter",
		Password:    "password123",
		DisplayName: "tx_inviter",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		AffCode:     "txcd",
	}
	require.NoError(t, DB.Create(&inviter).Error)

	invitee := User{
		Username:    "tx_invitee",
		DisplayName: "tx_invitee",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
	}
	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		return invitee.InsertWithTx(tx, inviter.Id)
	}))

	var createdInvitee User
	require.NoError(t, DB.First(&createdInvitee, invitee.Id).Error)
	assert.Equal(t, inviter.Id, createdInvitee.InviterId)
}

func TestUserSetInviterRecordsRelationshipAndRewards(t *testing.T) {
	truncateTables(t)
	withInviteSettings(t, 15, 7)

	inviter := User{
		Username:    "set_inviter_user",
		Password:    "password123",
		DisplayName: "set_inviter_user",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		AffCode:     "siu1",
	}
	require.NoError(t, DB.Create(&inviter).Error)

	invitee := User{
		Username:    "set_invitee_user",
		Password:    "password123",
		DisplayName: "set_invitee_user",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		AffCode:     "siv1",
	}
	require.NoError(t, DB.Create(&invitee).Error)

	require.NoError(t, invitee.SetInviter(inviter.Id))

	var updatedInvitee User
	require.NoError(t, DB.First(&updatedInvitee, invitee.Id).Error)
	assert.Equal(t, inviter.Id, updatedInvitee.InviterId)
	assert.Equal(t, 7, updatedInvitee.Quota)

	var updatedInviter User
	require.NoError(t, DB.First(&updatedInviter, inviter.Id).Error)
	assert.Equal(t, 1, updatedInviter.AffCount)
	assert.Equal(t, 15, updatedInviter.AffQuota)
	assert.Equal(t, 15, updatedInviter.AffHistoryQuota)
}

func TestUserSetInviterRejectsExistingInviter(t *testing.T) {
	truncateTables(t)
	withInviteSettings(t, 0, 0)

	firstInviter := User{
		Username:    "first_inviter",
		Password:    "password123",
		DisplayName: "first_inviter",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		AffCode:     "fiv1",
	}
	require.NoError(t, DB.Create(&firstInviter).Error)

	secondInviter := User{
		Username:    "second_inviter",
		Password:    "password123",
		DisplayName: "second_inviter",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		AffCode:     "siv2",
	}
	require.NoError(t, DB.Create(&secondInviter).Error)

	invitee := User{
		Username:    "existing_invitee",
		Password:    "password123",
		DisplayName: "existing_invitee",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		AffCode:     "eiv1",
	}
	require.NoError(t, DB.Create(&invitee).Error)
	require.NoError(t, invitee.SetInviter(firstInviter.Id))

	err := invitee.SetInviter(secondInviter.Id)
	require.Error(t, err)
	assert.EqualError(t, err, "inviter already set")
}
