import { setRequestLocale } from 'next-intl/server';

import { CompanionDetailPage } from '@/screens/companion-detail';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string; id: string }>;
}

/** `/companions/[id]` — one companion: access, renew, remove (B-N4-04). */
export default async function CompanionDetailRoute({ params }: Props) {
  const { locale, id } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="companionDetail">
      <CompanionDetailPage id={Number(id)} />
    </RouteMessages>
  );
}
