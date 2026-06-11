import assert from 'node:assert/strict';
import { describe, test } from 'node:test';
import { getDashboardDefaultDateRangeStrings } from './dashboard-date-range.js';

function getLocalDateText(date) {
  const year = date.getFullYear().toString();
  const month = (date.getMonth() + 1).toString().padStart(2, '0');
  const day = date.getDate().toString().padStart(2, '0');

  return `${year}-${month}-${day}`;
}

describe('classic dashboard date range', () => {
  test('builds default dashboard range for the current day', () => {
    const { start_timestamp, end_timestamp } =
      getDashboardDefaultDateRangeStrings();
    const today = getLocalDateText(new Date());

    assert.match(start_timestamp, / 00:00:00$/);
    assert.match(end_timestamp, / 23:59:59$/);
    assert.equal(start_timestamp.slice(0, 10), today);
    assert.equal(end_timestamp.slice(0, 10), today);
  });
});
