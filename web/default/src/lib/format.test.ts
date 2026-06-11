import assert from 'node:assert/strict'
import { describe, test } from 'node:test'
import { formatTokenCount } from './format.ts'

describe('formatTokenCount', () => {
  test('formats token totals with M and B units', () => {
    assert.equal(formatTokenCount(0), '0')
    assert.equal(formatTokenCount(999_999), '999,999')
    assert.equal(formatTokenCount(1_000_000), '1.00M')
    assert.equal(formatTokenCount(1_298_640), '1.30M')
    assert.equal(formatTokenCount(1_000_000_000), '1.00B')
    assert.equal(formatTokenCount(1_234_567_890), '1.23B')
  })
})
