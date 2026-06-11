export function formatTokenCount(value) {
  const tokens = Number(value);
  if (!Number.isFinite(tokens)) return '0';

  const absTokens = Math.abs(tokens);
  const sign = tokens < 0 ? '-' : '';

  if (absTokens >= 1_000_000_000) {
    return `${sign}${(absTokens / 1_000_000_000).toFixed(2)}B`;
  }
  if (absTokens >= 1_000_000) {
    return `${sign}${(absTokens / 1_000_000).toFixed(2)}M`;
  }

  return `${sign}${Math.round(absTokens).toLocaleString()}`;
}
