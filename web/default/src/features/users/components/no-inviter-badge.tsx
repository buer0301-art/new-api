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
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { StatusBadge } from '@/components/status-badge'
import type { User } from '../types'
import { SetUserInviterDialog } from './dialogs/set-user-inviter-dialog'

export function NoInviterBadge(props: { user: User }) {
  const { t } = useTranslation()
  const [dialogOpen, setDialogOpen] = useState(false)

  return (
    <>
      <Tooltip>
        <TooltipTrigger
          render={
            <StatusBadge
              label={t('No Inviter')}
              variant='neutral'
              copyable={false}
              className='cursor-pointer hover:text-foreground'
              onClick={() => setDialogOpen(true)}
            />
          }
        />
        <TooltipContent>
          <p className='text-xs'>{t('Set Inviter')}</p>
        </TooltipContent>
      </Tooltip>
      <SetUserInviterDialog
        open={dialogOpen}
        onOpenChange={setDialogOpen}
        user={props.user}
      />
    </>
  )
}
