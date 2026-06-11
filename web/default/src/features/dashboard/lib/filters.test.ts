import assert from 'node:assert/strict'
import { describe, test } from 'node:test'
import { DEFAULT_DASHBOARD_CHART_PREFERENCES } from '../constants.ts'
import { buildDefaultDashboardFilters } from './filters.ts'

describe('dashboard filters', () => {
  test('builds the default dashboard range for the current day', () => {
    const filters = buildDefaultDashboardFilters(
      DEFAULT_DASHBOARD_CHART_PREFERENCES
    )

    assert.equal(filters.start_timestamp?.getHours(), 0)
    assert.equal(filters.start_timestamp?.getMinutes(), 0)
    assert.equal(filters.start_timestamp?.getSeconds(), 0)
    assert.equal(filters.end_timestamp?.getHours(), 23)
    assert.equal(filters.end_timestamp?.getMinutes(), 59)
    assert.equal(filters.end_timestamp?.getSeconds(), 59)
    assert.equal(
      filters.start_timestamp?.toDateString(),
      new Date().toDateString()
    )
    assert.equal(filters.end_timestamp?.toDateString(), new Date().toDateString())
  })
})
