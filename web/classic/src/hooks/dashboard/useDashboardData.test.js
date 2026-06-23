import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { describe, test } from 'node:test';

const source = readFileSync(
  new URL('./useDashboardData.js', import.meta.url),
  'utf8',
);

describe('classic dashboard search modal', () => {
  test('closes before waiting for search data refresh', () => {
    const handlerStart = source.indexOf('const handleSearchConfirm = useCallback');
    const closeIndex = source.indexOf('setSearchModalVisible(false);', handlerStart);
    const refreshIndex = source.indexOf('const data = await refresh();', handlerStart);

    assert.notEqual(handlerStart, -1);
    assert.notEqual(closeIndex, -1);
    assert.notEqual(refreshIndex, -1);
    assert.ok(closeIndex < refreshIndex);
  });
});
