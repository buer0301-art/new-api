/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Dialog } from '@/components/dialog'
import { getUser, updateUser } from '../../api'
import { ERROR_MESSAGES } from '../../constants'
import type { User } from '../../types'
import { useUsers } from '../users-provider'

interface SetUserInviterDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  user: User
}

export function SetUserInviterDialog(props: SetUserInviterDialogProps) {
  const { t } = useTranslation()
  const { triggerRefresh } = useUsers()
  const [inviterIdInput, setInviterIdInput] = useState('')
  const [loading, setLoading] = useState(false)

  const closeDialog = () => {
    if (loading) {
      return
    }
    setInviterIdInput('')
    props.onOpenChange(false)
  }

  const handleConfirm = async () => {
    const trimmedInviterId = inviterIdInput.trim()
    if (!trimmedInviterId) {
      toast.error(t('Please enter inviter user ID'))
      return
    }

    const inviterId = Number.parseInt(trimmedInviterId, 10)
    if (!Number.isInteger(inviterId) || inviterId <= 0) {
      toast.error(t('Please enter a valid number'))
      return
    }
    if (inviterId === props.user.id) {
      toast.error(t('Inviter cannot be the current user'))
      return
    }

    setLoading(true)
    try {
      const currentUser = await getUser(props.user.id)
      if (!currentUser.success || !currentUser.data) {
        toast.error(currentUser.message || t('Failed to load'))
        return
      }
      if ((currentUser.data.inviter_id || 0) > 0) {
        toast.error(t('Only users without an inviter can set one here.'))
        setInviterIdInput('')
        props.onOpenChange(false)
        triggerRefresh()
        return
      }

      const result = await updateUser({
        id: currentUser.data.id,
        username: currentUser.data.username,
        display_name: currentUser.data.display_name || currentUser.data.username,
        group: currentUser.data.group,
        remark: currentUser.data.remark || '',
        inviter_id: inviterId,
      })
      if (result.success) {
        toast.success(t('Updated successfully'))
        setInviterIdInput('')
        props.onOpenChange(false)
        triggerRefresh()
      } else {
        toast.error(result.message || t('Failed to update user'))
      }
    } catch {
      toast.error(t(ERROR_MESSAGES.UNEXPECTED))
    } finally {
      setLoading(false)
    }
  }

  return (
    <Dialog
      open={props.open}
      onOpenChange={(open) => {
        if (!open) {
          closeDialog()
          return
        }
        props.onOpenChange(true)
      }}
      title={t('Set Inviter')}
      description={t(
        'Set inviter for this user by entering the inviter user ID.'
      )}
      contentHeight='auto'
      bodyClassName='space-y-4'
      footer={
        <>
          <Button variant='outline' onClick={closeDialog} disabled={loading}>
            {t('Cancel')}
          </Button>
          <Button onClick={handleConfirm} disabled={loading}>
            {loading ? t('Processing...') : t('Confirm')}
          </Button>
        </>
      }
    >
      <div className='space-y-2'>
        <Label>{t('User ID')}</Label>
        <Input
          type='number'
          min={1}
          step={1}
          placeholder={t('Enter inviter user ID')}
          value={inviterIdInput}
          onChange={(e) => setInviterIdInput(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter') {
              void handleConfirm()
            }
          }}
        />
      </div>
    </Dialog>
  )
}
