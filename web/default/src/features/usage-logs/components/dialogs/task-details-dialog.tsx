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
import { Check, Copy } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Dialog } from '@/components/dialog'
import { Button } from '@/components/ui/button'
import { useCopyToClipboard } from '@/hooks/use-copy-to-clipboard'

import type { TaskLog } from '../../types'

interface TaskDetailsDialogProps {
  task: TaskLog
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function TaskDetailsDialog(props: TaskDetailsDialogProps) {
  const { t } = useTranslation()
  const { copiedText, copyToClipboard } = useCopyToClipboard({ notify: false })
  const content = JSON.stringify(props.task, null, 2)

  return (
    <Dialog
      open={props.open}
      onOpenChange={props.onOpenChange}
      title={t('Details')}
      description={`${t('Task ID:')} ${props.task.task_id}`}
      contentClassName='sm:max-w-3xl'
      contentHeight='min(65vh, 560px)'
      footer={
        <Button
          variant='outline'
          onClick={() => copyToClipboard(content)}
          className='gap-2'
        >
          {copiedText === content ? (
            <Check className='size-4 text-green-600' />
          ) : (
            <Copy className='size-4' />
          )}
          {t('Copy to clipboard')}
        </Button>
      }
    >
      <pre className='bg-muted/40 text-foreground max-h-[calc(65vh-1rem)] overflow-auto rounded-md border p-3 font-mono text-xs leading-relaxed break-words whitespace-pre-wrap'>
        {content}
      </pre>
    </Dialog>
  )
}
