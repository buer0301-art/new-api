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
import type { ColumnDef } from '@tanstack/react-table'
import { FileText, HelpCircle, Music, Sparkles } from 'lucide-react'
/* eslint-disable react-refresh/only-export-components */
import { useState, useMemo } from 'react'
import { useTranslation } from 'react-i18next'

import { StatusBadge } from '@/components/status-badge'
import { formatTimestampToDate } from '@/lib/format'

import { TASK_ACTIONS, TASK_STATUS } from '../../constants'
import {
  taskActionMapper,
  taskPlatformMapper,
  taskStatusMapper,
} from '../../lib/mappers'
import { getTaskPlatformLabel, getTaskResultUrl } from '../../lib/task-log'
import type { TaskLog } from '../../types'
import {
  AudioPreviewDialog,
  type AudioClip,
} from '../dialogs/audio-preview-dialog'
import { FailReasonDialog } from '../dialogs/fail-reason-dialog'
import { TaskDetailsDialog } from '../dialogs/task-details-dialog'
import { VideoPreviewDialog } from '../dialogs/video-preview-dialog'
import { useUsageLogsContext } from '../usage-logs-provider'
import { createDurationColumn } from './column-helpers'

function parseTaskData(data: unknown): unknown[] {
  if (Array.isArray(data)) return data
  if (typeof data === 'string') {
    try {
      const parsed = JSON.parse(data)
      return Array.isArray(parsed) ? parsed : []
    } catch {
      return []
    }
  }
  return []
}

function AudioPreviewCell({ log }: { log: TaskLog }) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const clips = useMemo(() => {
    const data = parseTaskData(log.data)
    return data.filter(
      (c) =>
        c && typeof c === 'object' && (c as Record<string, unknown>).audio_url
    )
  }, [log.data])

  if (clips.length === 0) return null

  return (
    <>
      <button
        type='button'
        className='group flex items-center gap-1 text-left text-xs'
        onClick={() => setOpen(true)}
      >
        <Music className='text-muted-foreground size-3' />
        <span className='text-foreground leading-snug group-hover:underline'>
          {t('Click to preview audio')}
        </span>
      </button>
      <AudioPreviewDialog
        open={open}
        onOpenChange={setOpen}
        clips={clips as AudioClip[]}
      />
    </>
  )
}

function getTaskActionIcon(action: string) {
  if (action === TASK_ACTIONS.MUSIC) return Music
  if (action === TASK_ACTIONS.LYRICS) return FileText
  if (
    action === TASK_ACTIONS.GENERATE ||
    action === TASK_ACTIONS.TEXT_GENERATE ||
    action === TASK_ACTIONS.FIRST_TAIL_GENERATE ||
    action === TASK_ACTIONS.REFERENCE_GENERATE ||
    action === TASK_ACTIONS.REMIX_GENERATE
  ) {
    return Sparkles
  }
  return HelpCircle
}

export function useTaskLogsColumns(isAdmin: boolean): ColumnDef<TaskLog>[] {
  const { t } = useTranslation()
  const columns: ColumnDef<TaskLog>[] = [
    {
      accessorKey: 'submit_time',
      header: t('Time'),
      cell: ({ row }) => {
        const log = row.original
        const submitTime = row.getValue('submit_time') as number

        return (
          <div className='flex min-w-0 flex-col gap-0.5'>
            <span
              className='truncate font-mono text-xs tabular-nums'
              title={t('Submit Time')}
            >
              {formatTimestampToDate(submitTime, 'seconds')}
            </span>
            <span
              className='text-muted-foreground/60 truncate font-mono text-[11px] tabular-nums'
              title={t('Finish Time')}
            >
              {log.finish_time
                ? formatTimestampToDate(log.finish_time, 'seconds')
                : '-'}
            </span>
          </div>
        )
      },
      size: 170,
      meta: { label: t('Time') },
    },
  ]

  if (isAdmin) {
    const sourceLabel = `${t('Channel')} / ${t('User')}`
    columns.push({
      id: 'source',
      header: sourceLabel,
      accessorFn: (row) =>
        `${row.channel_id}:${row.username || row.user_id || ''}`,
      cell: function SourceCell({ row }) {
        const { sensitiveVisible, setSelectedUserId, setUserInfoDialogOpen } =
          useUsageLogsContext()
        const log = row.original
        const displayName = log.username || String(log.user_id || '?')

        return (
          <div className='flex max-w-[150px] min-w-0 flex-col gap-0.5'>
            {log.channel_id ? (
              <StatusBadge
                label={`#${log.channel_id}`}
                autoColor={String(log.channel_id)}
                copyText={String(log.channel_id)}
                size='sm'
                showDot={false}
                className='font-mono'
              />
            ) : (
              <span className='text-muted-foreground/60 text-xs'>-</span>
            )}
            <button
              type='button'
              className='text-muted-foreground truncate text-left text-xs hover:underline'
              onClick={(e) => {
                e.stopPropagation()
                setSelectedUserId(log.user_id)
                setUserInfoDialogOpen(true)
              }}
            >
              {sensitiveVisible ? displayName : '••••'}
            </button>
          </div>
        )
      },
      size: 145,
      meta: { label: sourceLabel },
    })
  }

  columns.push(
    {
      accessorKey: 'task_id',
      header: t('Task ID'),
      cell: function TaskIdCell({ row }) {
        const log = row.original
        const taskId = row.getValue('task_id') as string
        const [dialogOpen, setDialogOpen] = useState(false)

        if (!taskId) {
          return <span className='text-muted-foreground/60 text-xs'>-</span>
        }

        return (
          <>
            <div className='flex max-w-[210px] min-w-0 flex-col gap-0.5'>
              <button
                type='button'
                aria-label={t('Details')}
                title={t('Details')}
                className='flex max-w-full text-left'
                onClick={() => setDialogOpen(true)}
              >
                <StatusBadge
                  label={taskId}
                  variant='neutral'
                  size='sm'
                  copyable={false}
                  className='border-border/60 bg-muted/30 !text-foreground max-w-full cursor-pointer truncate rounded-md border px-1.5 py-0.5 font-mono hover:underline'
                />
              </button>
              {log.request_id ? (
                <StatusBadge
                  label={`${t('Request ID')}: ${log.request_id}`}
                  copyText={log.request_id}
                  variant='neutral'
                  size='sm'
                  className='border-border/40 bg-muted/20 text-muted-foreground max-w-full truncate rounded-md border px-1.5 py-0.5 font-mono !text-[10px] [&_span]:!text-[10px]'
                />
              ) : (
                <span className='text-muted-foreground/50 text-[11px]'>-</span>
              )}
            </div>
            <TaskDetailsDialog
              task={log}
              open={dialogOpen}
              onOpenChange={setDialogOpen}
            />
          </>
        )
      },
      size: 210,
      meta: { mobileTitle: true },
    },
    {
      ...createDurationColumn<TaskLog>({
        submitTimeKey: 'submit_time',
        finishTimeKey: 'finish_time',
        unit: 'seconds',
        headerLabel: t('Duration'),
        warningThresholdSec: 300,
      }),
      size: 90,
      maxSize: 100,
    },
    {
      accessorKey: 'platform',
      header: `${t('Platform')} / ${t('Type')}`,
      cell: ({ row }) => {
        const log = row.original
        const platform = row.getValue('platform') as string
        const platformLabel = getTaskPlatformLabel(platform)
        const ActionIcon = getTaskActionIcon(log.action)

        return (
          <div className='flex max-w-[165px] min-w-0 flex-col gap-0.5'>
            <StatusBadge
              label={t(taskPlatformMapper.getLabel(platform, platformLabel))}
              variant={taskPlatformMapper.getVariant(platform)}
              size='sm'
              copyable={false}
              className='-ml-1.5 max-w-full !text-xs [&_span]:!text-xs'
            />
            <StatusBadge
              label={t(taskActionMapper.getLabel(log.action))}
              variant={taskActionMapper.getVariant(log.action)}
              icon={ActionIcon}
              size='sm'
              copyable={false}
              className='-ml-1.5 max-w-full !text-xs [&_span]:!text-xs'
            />
          </div>
        )
      },
      size: 160,
      meta: { label: `${t('Platform')} / ${t('Type')}` },
    },
    {
      accessorKey: 'status',
      header: `${t('Status')} / ${t('Progress')}`,
      cell: ({ row }) => {
        const log = row.original
        const status = row.getValue('status') as string
        return (
          <div className='flex max-w-[125px] min-w-0 flex-col gap-0.5'>
            <StatusBadge
              label={t(
                taskStatusMapper.getLabel(status, status || 'Submitting')
              )}
              variant={taskStatusMapper.getVariant(status)}
              size='sm'
              copyable={false}
              className='-ml-1.5 max-w-full !text-xs [&_span]:!text-xs'
            />
            {log.progress ? (
              <span className='border-border/60 bg-muted/30 inline-flex w-fit max-w-full items-center truncate rounded-md border px-1.5 py-0.5 font-mono text-[11px]'>
                {log.progress}
              </span>
            ) : (
              <span className='text-muted-foreground/50 text-[11px]'>-</span>
            )}
          </div>
        )
      },
      size: 120,
      meta: { label: `${t('Status')} / ${t('Progress')}` },
    },
    {
      accessorKey: 'fail_reason',
      header: t('Details'),
      cell: function DetailsCell({ row }) {
        const log = row.original
        const failReason = row.getValue('fail_reason') as string
        const status = log.status
        const [failReasonDialogOpen, setFailReasonDialogOpen] = useState(false)
        const [videoDialogOpen, setVideoDialogOpen] = useState(false)

        const isSunoSuccess =
          log.platform === 'suno' && status === TASK_STATUS.SUCCESS
        if (isSunoSuccess) {
          const data = parseTaskData(log.data)
          if (
            data.some(
              (c) =>
                c &&
                typeof c === 'object' &&
                (c as Record<string, unknown>).audio_url
            )
          ) {
            return <AudioPreviewCell log={log} />
          }
        }

        const isVideoTask =
          log.action === TASK_ACTIONS.GENERATE ||
          log.action === TASK_ACTIONS.TEXT_GENERATE ||
          log.action === TASK_ACTIONS.FIRST_TAIL_GENERATE ||
          log.action === TASK_ACTIONS.REFERENCE_GENERATE ||
          log.action === TASK_ACTIONS.REMIX_GENERATE
        const isSuccess = status === TASK_STATUS.SUCCESS
        const resultUrl = getTaskResultUrl(log)

        if (isSuccess && isVideoTask && resultUrl) {
          return (
            <>
              <button
                type='button'
                onClick={() => setVideoDialogOpen(true)}
                className='text-foreground text-xs hover:underline'
              >
                {t('Click to preview video')}
              </button>
              {videoDialogOpen ? (
                <VideoPreviewDialog
                  videoUrl={resultUrl}
                  taskId={log.task_id}
                  open={videoDialogOpen}
                  onOpenChange={setVideoDialogOpen}
                />
              ) : null}
            </>
          )
        }

        if (!failReason) {
          return <span className='text-muted-foreground/60 text-xs'>-</span>
        }

        return (
          <>
            <button
              type='button'
              className='group flex max-w-[200px] items-center gap-1 text-left text-xs'
              onClick={() => setFailReasonDialogOpen(true)}
              title={t('Click to view full error message')}
            >
              <span className='truncate leading-snug text-red-600 group-hover:underline dark:text-red-400'>
                {failReason}
              </span>
            </button>
            <FailReasonDialog
              failReason={failReason}
              open={failReasonDialogOpen}
              onOpenChange={setFailReasonDialogOpen}
            />
          </>
        )
      },
      size: 165,
      maxSize: 180,
    }
  )

  return columns
}
