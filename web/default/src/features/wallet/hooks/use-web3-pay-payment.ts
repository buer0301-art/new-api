/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { useCallback, useState } from 'react'
import i18next from 'i18next'
import { toast } from 'sonner'
import { isApiSuccess, requestWeb3PayPayment } from '../api'
import type { Web3PayOrder } from '../types'

function isSafeHostedCheckoutUrl(value: string): boolean {
  try {
    const url = new URL(value.trim())
    return url.protocol === 'http:' || url.protocol === 'https:'
  } catch {
    return false
  }
}

export function useWeb3PayPayment() {
  const [processing, setProcessing] = useState(false)

  const processWeb3PayPayment = useCallback(async (topupAmount: number) => {
    setProcessing(true)

    try {
      const response = await requestWeb3PayPayment({
        amount: Math.floor(topupAmount),
      })

      if (
        isApiSuccess(response) &&
        response.data &&
        (response.data.paymentOptions?.length || response.data.payUrl)
      ) {
        return response.data as Web3PayOrder
      }

      const message =
        typeof response.data === 'string'
          ? response.data
          : response.message || i18next.t('Payment request failed')
      toast.error(message)
      return null
    } catch (_error) {
      toast.error(i18next.t('Payment request failed'))
      return null
    } finally {
      setProcessing(false)
    }
  }, [])

  return { processing, processWeb3PayPayment, isSafeHostedCheckoutUrl }
}
