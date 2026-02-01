/**
 * Copy text to clipboard with fallback for older browsers and non-secure contexts
 *
 * @param text - The text to copy to clipboard
 * @returns Promise resolving to true if successful, false otherwise
 */
export async function copyToClipboard(text: string): Promise<boolean> {
  // Try modern Clipboard API first
  if (navigator.clipboard && window.isSecureContext) {
    try {
      await navigator.clipboard.writeText(text)
      return true
    } catch {
      // Fall through to legacy method
    }
  }

  // Fallback for older browsers or non-secure contexts
  const textArea = document.createElement('textarea')
  textArea.value = text
  textArea.style.position = 'fixed'
  textArea.style.left = '-9999px'
  textArea.style.top = '-9999px'
  document.body.appendChild(textArea)
  textArea.focus()
  textArea.select()

  try {
    const successful = document.execCommand('copy')
    return successful
  } catch {
    return false
  } finally {
    document.body.removeChild(textArea)
  }
}
