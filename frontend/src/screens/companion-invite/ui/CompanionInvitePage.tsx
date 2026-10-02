'use client';

import { InviteCompanionFlow } from '@/features/invite-companion';
import { useRouter } from '@/shared/i18n';

/** `/companions/new` — the «افزودن همدم» wizard (B-N4-04, Hamdam_Type … Hamdam_Done). */
export function CompanionInvitePage() {
  const router = useRouter();
  return <InviteCompanionFlow onExit={() => router.push('/companions')} />;
}
