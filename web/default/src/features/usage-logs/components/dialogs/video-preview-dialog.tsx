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
import { Check, Copy, ExternalLink, Video } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Dialog } from '@/components/dialog'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { useCopyToClipboard } from '@/hooks/use-copy-to-clipboard'

interface VideoPreviewDialogProps {
  videoUrl: string
  taskId: string
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function VideoPreviewDialog(props: VideoPreviewDialogProps) {
  const { t } = useTranslation()
  const { copiedText, copyToClipboard } = useCopyToClipboard({ notify: false })
  const [isLoading, setIsLoading] = useState(true)
  const [hasError, setHasError] = useState(false)

  return (
    <Dialog
      open={props.open}
      onOpenChange={props.onOpenChange}
      title={`${t('Video')} · ${t('Preview')}`}
      description={`${t('Task ID:')} ${props.taskId}`}
      contentClassName='sm:max-w-4xl'
      contentHeight='auto'
      bodyClassName='space-y-3'
      footer={
        <>
          <Button
            variant='outline'
            onClick={() => copyToClipboard(props.videoUrl)}
            className='gap-2'
          >
            {copiedText === props.videoUrl ? (
              <Check className='size-4 text-green-600' />
            ) : (
              <Copy className='size-4' />
            )}
            {t('Copy Link')}
          </Button>
          <Button
            onClick={() =>
              window.open(props.videoUrl, '_blank', 'noopener,noreferrer')
            }
            className='gap-2'
          >
            <ExternalLink className='size-4' />
            {t('Open in new tab')}
          </Button>
        </>
      }
    >
      <div className='bg-muted/30 relative flex aspect-video min-h-[240px] items-center justify-center overflow-hidden rounded-md border'>
        {isLoading && !hasError ? (
          <Skeleton className='absolute inset-0 size-full rounded-none' />
        ) : null}

        {hasError ? (
          <div className='text-muted-foreground flex max-w-full flex-col items-center gap-3 p-6 text-center'>
            <Video className='size-10' aria-hidden='true' />
            <p className='max-w-full font-mono text-xs break-all'>
              {props.videoUrl}
            </p>
          </div>
        ) : (
          <video
            src={props.videoUrl}
            controls
            preload='metadata'
            aria-label={t('Video')}
            className='size-full object-contain'
            onLoadStart={() => setIsLoading(true)}
            onLoadedData={() => setIsLoading(false)}
            onError={() => {
              setIsLoading(false)
              setHasError(true)
            }}
          />
        )}
      </div>
    </Dialog>
  )
}
