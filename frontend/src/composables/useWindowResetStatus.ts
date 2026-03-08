import { useI18n } from 'vue-i18n'
import type { WindowResetStatus } from '@/types'

/**
 * Composable for formatting window reset status from backend-computed data.
 * @param keyPrefix - i18n key prefix (e.g., 'userSubscriptions.resetStatus' or 'admin.subscriptions.resetStatus')
 */
export function useWindowResetStatus(keyPrefix: string) {
  const { t } = useI18n()

  function formatCountdown(resetAt: string): string {
    const now = new Date()
    const end = new Date(resetAt)
    const diffMs = end.getTime() - now.getTime()
    if (diffMs <= 0) return '0m'

    const totalSeconds = Math.floor(diffMs / 1000)
    const days = Math.floor(totalSeconds / 86400)
    const hours = Math.floor((totalSeconds % 86400) / 3600)
    const minutes = Math.floor((totalSeconds % 3600) / 60)

    if (days > 0) return `${days}d ${hours}h`
    if (hours > 0) return `${hours}h ${minutes}m`
    return `${minutes}m`
  }

  function formatDateTime(dateStr: string): string {
    const d = new Date(dateStr)
    if (isNaN(d.getTime())) return dateStr
    const month = d.getMonth() + 1
    const day = d.getDate()
    const hours = String(d.getHours()).padStart(2, '0')
    const minutes = String(d.getMinutes()).padStart(2, '0')
    return `${month}/${day} ${hours}:${minutes}`
  }

  function formatWindowStatus(rs?: WindowResetStatus): string | null {
    if (!rs) return null
    switch (rs.status) {
      case 'awaiting_first_use':
        return t(`${keyPrefix}.awaitingFirstUse`)
      case 'active':
        if (!rs.reset_at) return null
        return t(`${keyPrefix}.active`, { time: formatCountdown(rs.reset_at) })
      case 'active_final_window': {
        const reason = t(`${keyPrefix}.activeFinalWindow`)
        if (!rs.reset_at) return reason
        const resetEnd = new Date(rs.reset_at)
        const now = new Date()
        const resetTimeStr = resetEnd.getTime() > now.getTime()
          ? t(`${keyPrefix}.active`, { time: formatCountdown(rs.reset_at) })
          : t(`${keyPrefix}.resetTimeAt`, { time: formatDateTime(rs.reset_at) })
        return `${resetTimeStr}\n${reason}`
      }
      case 'expired_will_reset':
        return t(`${keyPrefix}.expiredWillReset`)
      case 'expired_subscription':
        return t(`${keyPrefix}.expiredSubscription`)
      default:
        return null
    }
  }

  function getResetStatusClass(rs?: WindowResetStatus): string {
    if (!rs) return ''
    switch (rs.status) {
      case 'active_final_window':
        return 'text-orange-600 dark:text-orange-400 font-medium'
      case 'expired_subscription':
        return 'text-red-600 dark:text-red-400 font-medium'
      default:
        return ''
    }
  }

  return { formatWindowStatus, getResetStatusClass }
}
