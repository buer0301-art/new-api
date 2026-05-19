import assert from 'node:assert/strict'
import { describe, test } from 'node:test'
import {
  buildRuleFromRows,
  stringifyPerRequestRules,
} from './per-request-pricing.ts'

describe('per-request pricing resolution normalization', () => {
  test('stringify keeps rules whose default resolution matches a normalized price key', () => {
    const serialized = stringifyPerRequestRules({
      'video-test-model': {
        media_type: 'video',
        unit: 'second',
        prices: {
          '180*640': 0.03,
        },
        default_resolution: '180x640',
        fallback_enabled: false,
      },
    })

    assert.deepEqual(JSON.parse(serialized), {
      'video-test-model': {
        media_type: 'video',
        unit: 'second',
        prices: {
          '180*640': 0.03,
        },
        default_resolution: '180*640',
        fallback_enabled: false,
      },
    })
  })

  test('buildRuleFromRows preserves the configured row when default resolution only differs by separator', () => {
    const rule = buildRuleFromRows(
      'video',
      [
        {
          id: 'fallback',
          resolution: '1K',
          price: '0.08',
          enabled: true,
        },
        {
          id: 'custom',
          resolution: '180*640',
          price: '0.03',
          enabled: true,
        },
      ],
      '180x640'
    )

    assert.equal(rule?.default_resolution, '180*640')
  })
})
