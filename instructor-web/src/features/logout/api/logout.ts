import { api } from '@/shared/api';
import { clearAuthToken } from '@/shared/session';

/**
 * POST /api/v1/auth/logout revokes the token server-side; the local session is
 * cleared whatever the network says (a stale token is worse than a failed call).
 */
export async function logout(): Promise<void> {
  try {
    await api.post('/auth/logout', undefined, { api: 'public', skipAuthRedirect: true });
  } catch {
    // ignore: cleared below
  } finally {
    clearAuthToken();
  }
}
