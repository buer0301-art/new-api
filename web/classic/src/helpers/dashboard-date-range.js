function timestampToString(timestamp) {
  const date = new Date(timestamp * 1000);
  const year = date.getFullYear().toString();
  const month = (date.getMonth() + 1).toString().padStart(2, '0');
  const day = date.getDate().toString().padStart(2, '0');
  const hour = date.getHours().toString().padStart(2, '0');
  const minute = date.getMinutes().toString().padStart(2, '0');
  const second = date.getSeconds().toString().padStart(2, '0');

  return `${year}-${month}-${day} ${hour}:${minute}:${second}`;
}

function getTodayStartTimestamp() {
  const now = new Date();
  now.setHours(0, 0, 0, 0);
  return Math.floor(now.getTime() / 1000);
}

function getTodayEndTimestamp() {
  const now = new Date();
  now.setHours(23, 59, 59, 999);
  return Math.floor(now.getTime() / 1000);
}

export function getDashboardDefaultDateRangeStrings() {
  return {
    start_timestamp: timestampToString(getTodayStartTimestamp()),
    end_timestamp: timestampToString(getTodayEndTimestamp()),
  };
}
