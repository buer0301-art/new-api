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
import { describe, expect, test } from 'bun:test'

import i18n from 'i18next'
import { createElement } from 'react'
import { renderToStaticMarkup } from 'react-dom/server'
import { initReactI18next } from 'react-i18next'

import { useTaskLogsColumns } from './task-logs-columns'

await i18n.use(initReactI18next).init({
  lng: 'en',
  resources: { en: { translation: {} } },
  interpolation: { escapeValue: false },
})

function getColumns(isAdmin) {
  let columns = []

  function ColumnProbe() {
    columns = useTaskLogsColumns(isAdmin)
    return null
  }

  renderToStaticMarkup(createElement(ColumnProbe))
  return columns
}

function getColumnIds(isAdmin) {
  return getColumns(isAdmin).map((column) => column.id || column.accessorKey)
}

describe('task log columns', () => {
  test('shows every classic task-log field for admins', () => {
    expect(getColumnIds(true)).toEqual([
      'submit_time',
      'source',
      'task_id',
      'duration',
      'platform',
      'status',
      'fail_reason',
    ])
  })

  test('keeps admin-only fields hidden from regular users', () => {
    expect(getColumnIds(false)).toEqual([
      'submit_time',
      'task_id',
      'duration',
      'platform',
      'status',
      'fail_reason',
    ])
  })

  test('combines submit and finish times into one column', () => {
    const timeColumn = getColumns(false).find(
      (column) => column.accessorKey === 'submit_time'
    )
    const cell = timeColumn?.cell

    expect(typeof cell).toBe('function')
    const markup = renderToStaticMarkup(
      createElement(cell, {
        row: {
          original: {
            submit_time: 1704067200,
            finish_time: 1735689600,
          },
          getValue: () => 1704067200,
        },
      })
    )

    expect(markup).toContain('2024')
    expect(markup).toContain('2025')
  })

  test('renders numeric platform values as channel names', () => {
    const platformColumn = getColumns(true).find(
      (column) => column.accessorKey === 'platform'
    )
    const cell = platformColumn?.cell

    expect(typeof cell).toBe('function')
    const markup = renderToStaticMarkup(
      cell({
        row: {
          original: {
            platform: '1',
            action: 'generate',
          },
          getValue: () => '1',
        },
      })
    )

    expect(markup).toContain('OpenAI')
    expect(markup).toContain('Image to Video')
    expect(markup).not.toContain('>1<')
  })

  test('opens task details from the task ID instead of only copying it', () => {
    const taskIdColumn = getColumns(true).find(
      (column) => column.accessorKey === 'task_id'
    )
    const cell = taskIdColumn?.cell

    expect(typeof cell).toBe('function')
    const markup = renderToStaticMarkup(
      createElement(cell, {
        row: {
          original: {
            task_id: 'task_123',
            request_id: 'request_456',
          },
          getValue: () => 'task_123',
        },
      })
    )

    expect(markup).toContain('<button')
    expect(markup).toContain('aria-label="Details"')
    expect(markup).toContain('request_456')
  })

  test('combines task status and progress into one column', () => {
    const statusColumn = getColumns(false).find(
      (column) => column.accessorKey === 'status'
    )
    const cell = statusColumn?.cell

    expect(typeof cell).toBe('function')
    const markup = renderToStaticMarkup(
      cell({
        row: {
          original: {
            status: 'SUCCESS',
            progress: '100%',
          },
          getValue: () => 'SUCCESS',
        },
      })
    )

    expect(markup).toContain('Success')
    expect(markup).toContain('100%')
  })

  test('opens successful video results in a preview dialog trigger', () => {
    const detailsColumn = getColumns(true).find(
      (column) => column.accessorKey === 'fail_reason'
    )
    const cell = detailsColumn?.cell

    expect(typeof cell).toBe('function')
    const markup = renderToStaticMarkup(
      createElement(cell, {
        row: {
          original: {
            task_id: 'task_123',
            platform: '1',
            action: 'generate',
            status: 'SUCCESS',
            result_url: 'https://example.com/result.mp4',
          },
          getValue: () => '',
        },
      })
    )

    expect(markup).toContain('<button')
    expect(markup).toContain('Click to preview video')
    expect(markup).not.toContain('target="_blank"')
  })
})
