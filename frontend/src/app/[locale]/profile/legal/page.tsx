import { setRequestLocale } from 'next-intl/server';

import { LegalPage } from '@/screens/about';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
  searchParams: Promise<{ tab?: string | string[] }>;
}

/** `/profile/legal?tab=terms|privacy` — Terms & privacy (B-N1-12, nbl_Me_Legal). */
export default async function ProfileLegalRoute({ params, searchParams }: Props) {
  const [{ locale }, { tab }] = await Promise.all([params, searchParams]);
  setRequestLocale(locale);
  return (
    <RouteMessages route="profileLegal">
      <LegalPage initialTab={typeof tab === 'string' ? tab : undefined} />
    </RouteMessages>
  );
}
