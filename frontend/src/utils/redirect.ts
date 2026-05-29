/**
 * Post-auth redirect sanitizer.
 *
 * Only same-origin, internal absolute paths (a single leading "/") are honored,
 * so an attacker cannot smuggle an external URL through the ?redirect query and
 * bounce a freshly authenticated user off-site. Anything else falls back to the
 * default landing page (/chat).
 */
export function sanitizeRedirectPath(
  path?: unknown,
  fallback = '/chat'
): string {
  // Vue Router query values are `string | null | (string | null)[]`; a duplicated
  // ?redirect param arrives as an array. Only honor a real, non-empty string.
  if (typeof path !== 'string' || path === '') return fallback
  // Must be an absolute internal path.
  if (!path.startsWith('/')) return fallback
  // Reject protocol-relative ("//host") which browsers treat as external.
  if (path.startsWith('//')) return fallback
  // Reject anything carrying a scheme/host (e.g. "/\evil.com", "/x://y").
  if (path.includes('://')) return fallback
  // Reject control characters that could break out of the path.
  if (path.includes('\n') || path.includes('\r') || path.includes('\t')) return fallback
  if (path.includes('\\')) return fallback
  return path
}
