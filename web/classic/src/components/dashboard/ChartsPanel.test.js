import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { describe, test } from 'node:test';

const source = readFileSync(
  new URL('./ChartsPanel.jsx', import.meta.url),
  'utf8',
);

describe('classic dashboard ChartsPanel', () => {
  test('renders dashboard analytics charts with the real VChart renderer', () => {
    assert.match(source, /import \{ VChart \} from '@visactor\/react-vchart'/);
    assert.doesNotMatch(source, /import SafeVChart/);
    assert.match(source, /<VChart spec=\{spec_line\} option=\{CHART_CONFIG\} \/>/);
  });
});
