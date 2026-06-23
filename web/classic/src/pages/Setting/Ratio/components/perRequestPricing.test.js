import assert from 'node:assert/strict';
import { describe, test } from 'node:test';
import {
  buildRuleFromRows,
  stringifyPerRequestRules,
} from './perRequestPricing.js';

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
    });

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
    });
  });

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
      '180x640',
    );

    assert.equal(rule?.default_resolution, '180*640');
  });

  test('buildRuleFromRows preserves the selected video billing unit', () => {
    const rule = buildRuleFromRows(
      'video',
      [
        {
          id: 'full-hd',
          resolution: '1080',
          price: '1',
          enabled: true,
        },
      ],
      '1080',
      'request',
    );

    assert.equal(rule?.unit, 'request');
  });

  test('stringify keeps video request unit rules', () => {
    const serialized = stringifyPerRequestRules({
      'video-test-model': {
        media_type: 'video',
        unit: 'request',
        prices: {
          1080: 1,
        },
        default_resolution: '1080',
        fallback_enabled: false,
      },
    });

    assert.equal(JSON.parse(serialized)['video-test-model'].unit, 'request');
  });
});
