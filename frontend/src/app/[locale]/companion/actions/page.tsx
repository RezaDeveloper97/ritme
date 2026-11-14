import { setRequestLocale } from 'next-intl/server';

import { CompanionActionsPage } from '@/screens/companion-actions';

import { RouteMessages } from '../../../RouteMessages';

interface Props {
  params: Promise<{ locale: string }>;
}

/** `/companion/actions` — «اقدام‌های همراهی» (companion v2 compCards), companion nav. */
export default async function CompanionActionsRoute({ params }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  return (
    <RouteMessages route="companionActions">
      <CompanionActionsPage />
    </RouteMessages>
  );
}
