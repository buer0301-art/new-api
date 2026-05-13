import assert from 'node:assert/strict'
import { describe, test } from 'node:test'
import { normalizeTopupIframeUrl } from './topup-url.ts'

describe('normalizeTopupIframeUrl', () => {
  test('returns empty string for empty topup links', () => {
    assert.equal(normalizeTopupIframeUrl(''), '')
    assert.equal(normalizeTopupIframeUrl('   '), '')
    assert.equal(normalizeTopupIframeUrl(undefined), '')
  })

  test('preserves valid http and https links', () => {
    assert.equal(
      normalizeTopupIframeUrl('https://pay.example.com/redeem?user=1'),
      'https://pay.example.com/redeem?user=1'
    )
    assert.equal(
      normalizeTopupIframeUrl('http://localhost:3000/topup'),
      'http://localhost:3000/topup'
    )
  })

  test('adds https protocol to bare domains', () => {
    assert.equal(
      normalizeTopupIframeUrl('pay.example.com/redeem'),
      'https://pay.example.com/redeem'
    )
  })

  test('rejects non-web protocols', () => {
    assert.equal(normalizeTopupIframeUrl('javascript:alert(1)'), '')
    assert.equal(normalizeTopupIframeUrl('mailto:billing@example.com'), '')
  })
})
